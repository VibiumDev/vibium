package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// newCommandsCmd lists every runnable command path as JSON, for the API
// drift checker (cmd/apidrift) and any tooling that wants the real command
// tree instead of parsed help text.
func newCommandsCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "commands",
		Short:  "List every command path as JSON",
		Hidden: true,
		Example: `  vibium commands
  # {"":["a11y-tree","attr","back", ... ,"page new","wait","wait fn", ...]}`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			var paths []string
			var walk func(c *cobra.Command, prefix string)
			walk = func(c *cobra.Command, prefix string) {
				for _, sub := range c.Commands() {
					if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
						continue
					}
					path := strings.TrimSpace(prefix + " " + sub.Name())
					// A parent that groups subcommands (is, page) runs only
					// to print help; one that also takes arguments (wait
					// <sel> beside wait fn/url/load) or says so explicitly
					// (storage) is a real command too.
					standalone := !sub.HasSubCommands() ||
						strings.ContainsAny(sub.Use, "[<") ||
						sub.Annotations["standalone"] == "true"
					if sub.Runnable() && standalone {
						paths = append(paths, path)
					}
					walk(sub, path)
				}
			}
			walk(root, "")
			sort.Strings(paths)
			out, _ := json.Marshal(map[string][]string{"": paths})
			fmt.Println(string(out))
		},
	}
}
