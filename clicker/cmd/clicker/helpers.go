package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// printCheck prints an actionability check result with a checkmark or X.
func printCheck(name string, passed bool) {
	if passed {
		fmt.Printf("✓ %s: true\n", name)
	} else {
		fmt.Printf("✗ %s: false\n", name)
	}
}

// lateParseCommands records every command built with lateParse. The flag
// drift test compares it against the commands that disable flag parsing, so
// setting DisableFlagParsing by hand, without the parse and global-flag
// re-apply that must come with it (#482), fails the build.
var lateParseCommands = map[string]bool{}

// lateParse converts a command to late flag parsing, for positionals that can
// be negative numbers (`sleep -5`, `geolocation 37.8 -122.4`), which pflag
// would otherwise read as unknown flags. The command's Run receives the
// parsed positionals and still checks its own arity; everything else, flag
// parsing, --help interception, and the global-flag re-apply, happens here.
//
// This is the only supported way to disable flag parsing on a command.
func lateParse(cmd *cobra.Command) *cobra.Command {
	inner := cmd.Run
	cmd.DisableFlagParsing = true
	cmd.Args = cobra.ArbitraryArgs
	cmd.Run = func(c *cobra.Command, raw []string) {
		args, err := parseFlagsAllowNegative(c, raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		inner(c, args)
	}
	lateParseCommands[cmd.Name()] = true
	return cmd
}

// parseFlagsAllowNegative splits raw args into flags and positionals, treating a
// token that starts with '-' as a positional when it parses as a number.
//
// pflag has no negative-number support, so `sleep -5` and
// `geolocation 37.8 -122.4` are read as unknown flags. The two workarounds in
// use — DisableFlagParsing and SetInterspersed(false) — both fix that by
// breaking trailing flags, so `sleep 1 --json` printed plain text and
// `fill '#x' y --timeout 5s` failed on arity (#241).
//
// Commands reach this through lateParse, which owns the DisableFlagParsing
// and ArbitraryArgs settings it depends on.
func parseFlagsAllowNegative(cmd *cobra.Command, raw []string) ([]string, error) {
	// Touching InheritedFlags merges the parents' persistent flags (--json,
	// --headless, --verbose) into cmd.Flags(). cmd.ParseFlags is not usable
	// here: it returns early when DisableFlagParsing is set, which is exactly
	// the mode these commands run in.
	cmd.InheritedFlags()
	flags := cmd.Flags()

	var flagTokens, positionals []string

	for i := 0; i < len(raw); i++ {
		tok := raw[i]

		if tok == "--" {
			positionals = append(positionals, raw[i+1:]...)
			break
		}

		if len(tok) < 2 || !strings.HasPrefix(tok, "-") {
			positionals = append(positionals, tok)
			continue
		}

		// "-5" and "-122.4" are values, not flags.
		if _, err := strconv.ParseFloat(tok, 64); err == nil {
			positionals = append(positionals, tok)
			continue
		}

		flagTokens = append(flagTokens, tok)

		// A flag that needs a value consumes the next token, unless it was
		// given inline as --name=value.
		if strings.Contains(tok, "=") {
			continue
		}
		name := strings.TrimLeft(tok, "-")
		f := flags.Lookup(name)
		if f == nil && len(name) == 1 {
			// ShorthandLookup panics on anything longer than one character.
			f = flags.ShorthandLookup(name)
		}
		if f != nil && f.NoOptDefVal == "" && i+1 < len(raw) {
			i++
			flagTokens = append(flagTokens, raw[i])
		}
	}

	if err := flags.Parse(flagTokens); err != nil {
		return nil, err
	}

	// DisableFlagParsing also disables cobra's help interception, so -h and
	// --help land in the flag set unhandled and the command runs for real,
	// destructively for fill (#422, #423). Honor the flag here.
	if help, _ := flags.GetBool("help"); help {
		cmd.Help()
		os.Exit(0)
	}

	// The root's PersistentPreRunE ran before this parse, so it computed the
	// *Set booleans, the validations and the env-var bridges from unset
	// values. Re-apply them now that the flags hold what the user passed;
	// otherwise --session and --channel are accepted and silently ignored on
	// every command that reaches this helper (#482).
	if err := applyGlobalFlags(cmd); err != nil {
		return nil, err
	}

	return positionals, nil
}
