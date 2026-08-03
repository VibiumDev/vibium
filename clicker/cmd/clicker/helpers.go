package main

import (
	"fmt"
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

// parseFlagsAllowNegative splits raw args into flags and positionals, treating a
// token that starts with '-' as a positional when it parses as a number.
//
// pflag has no negative-number support, so `sleep -5` and
// `geolocation 37.8 -122.4` are read as unknown flags. The two workarounds in
// use — DisableFlagParsing and SetInterspersed(false) — both fix that by
// breaking trailing flags, so `sleep 1 --json` printed plain text and
// `fill '#x' y --timeout 5s` failed on arity (#241).
//
// Commands using this set DisableFlagParsing and cobra.ArbitraryArgs, then check
// their own arity on the returned positionals.
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
	return positionals, nil
}
