package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// walkCommands visits every descendant of root, depth first, with its full
// space-joined path.
func walkCommands(root *cobra.Command, fn func(path string, c *cobra.Command)) {
	var walk func(c *cobra.Command, prefix string)
	walk = func(c *cobra.Command, prefix string) {
		for _, sub := range c.Commands() {
			path := strings.TrimSpace(prefix + " " + sub.Name())
			fn(path, sub)
			walk(sub, path)
		}
	}
	walk(root, "")
}

// A subcommand that redeclares a root persistent flag shadows it: cobra drops
// the root's flag from Global Flags and the local wording and behavior win.
// That is how `mcp --headless` diverged from every other command (#452). The
// allowlist is empty on purpose; a legitimate exception should be argued into
// this test, not slipped past it.
func TestNoLocalFlagShadowsRootPersistent(t *testing.T) {
	root, _ := newRootCmd("vibium")

	global := map[string]bool{}
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		global[f.Name] = true
	})
	if len(global) == 0 {
		t.Fatal("no root persistent flags found; the walk below would pass vacuously")
	}

	walkCommands(root, func(path string, c *cobra.Command) {
		check := func(f *pflag.Flag) {
			if global[f.Name] {
				t.Errorf("%q declares local flag --%s, shadowing the root persistent flag of the same name", path, f.Name)
			}
		}
		c.Flags().VisitAll(check)
		c.PersistentFlags().VisitAll(check)
	})
}

// The commands that disable cobra flag parsing re-run the global-flag bridge
// themselves (parseFlagsAllowNegative), and get --help/--session/--channel
// coverage from tests/cli/help-flags.test.js and global-flags.test.js. A new
// member joins that contract deliberately: add it here and to those suites,
// or it ships with #422/#423/#482 all over again.
func TestDisableFlagParsingSetIsExact(t *testing.T) {
	root, _ := newRootCmd("vibium")

	want := []string{"fill", "geolocation", "sleep", "type"}
	var got []string
	walkCommands(root, func(path string, c *cobra.Command) {
		if c.DisableFlagParsing {
			got = append(got, path)
		}
	})
	sort.Strings(got)

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("DisableFlagParsing commands = %v, want %v\n"+
			"If you added one on purpose, extend tests/cli/help-flags.test.js and tests/cli/global-flags.test.js to cover it, then update this list.", got, want)
	}
}
