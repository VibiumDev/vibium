package main

import (
	"github.com/spf13/cobra"
)

func newPermissionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission [name] [granted|denied|prompt]",
		Short: "Grant, deny or reset a browser permission for an origin",
		Example: `  vibium permission geolocation granted
  # → Permission geolocation set to granted for https://example.com

  vibium permission notifications denied --origin https://example.com
  # Answer for an origin other than the current page's

  vibium permission camera prompt
  # Back to asking`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			callArgs := map[string]interface{}{"name": args[0], "state": args[1]}
			if origin, _ := cmd.Flags().GetString("origin"); origin != "" {
				callArgs["origin"] = origin
			}
			result, err := daemonCall("browser_set_permission", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	cmd.Flags().String("origin", "", "Origin the permission applies to (default: the current page's origin)")
	return cmd
}
