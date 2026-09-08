package main

import (
	"io"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// promptArgs only expands an unambiguous, single multiword argument. Flags are
// parsed with Run's actual schema, so option values never become the prompt.
func promptArgs(root, run *cobra.Command, args []string) []string {
	if cmd, _, err := root.Find(args); err == nil && cmd != root {
		return args
	}
	flags := pflag.NewFlagSet("prompt", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.AddFlagSet(root.PersistentFlags())
	flags.AddFlagSet(run.Flags())
	if err := flags.Parse(args); err != nil {
		return args
	}
	positional := flags.Args()
	if len(positional) != 1 || strings.TrimSpace(positional[0]) == "" || strings.IndexFunc(positional[0], unicode.IsSpace) < 0 {
		return args
	}
	return append([]string{"run"}, args...)
}
