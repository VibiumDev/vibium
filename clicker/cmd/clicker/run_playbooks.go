package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type runPlaybook struct {
	name  string
	short string
	goal  string
}

// Preset Run goals. Keep each under verifier.MaxClaim.
var runPlaybooks = []runPlaybook{
	{
		name:  "inspect",
		short: "Record header lockup and primary KPI without navigating",
		goal: "Do not sign in, fill forms, submit, or navigate away. Record: (1) the exact header brand text, " +
			"(2) whether a logo mark exists besides the product name, (3) whether the page is dark, " +
			"(4) whether an email or avatar is in the header, (5) the main heading or primary KPI or empty state. " +
			"If this is a login wall, 403, 404, or error, say so and stop.",
	},
	{
		name:  "walk",
		short: "Tour in-app nav; map works, broken, and CSS shell",
		goal: "Already authenticated in this app. Stay on this hostname.\n" +
			"Do NOT: sign out, fill forms, type passwords, submit, follow an identity provider " +
			"(Microsoft, Google, Okta, Auth0 Universal Login), open a different host, download files, or change settings.\n" +
			"DO: open primary nav, sidebar, tabs, and in-app links that stay on this hostname. After each new route, " +
			"note header brand, dark vs light, extra logos, email or avatar in the header, and whether the page is a " +
			"working view, empty state, 403, 404, error, or login wall.\n" +
			"Write three sections:\n" +
			"1. WORKS — URL, title, what the page is (one line each)\n" +
			"2. BROKEN — URL, and the failure\n" +
			"3. CSS SHELL — header lockup exact text; extra logo besides the product name (yes/no); dark page (yes/no); " +
			"email or avatar in header (yes/no); third-party SSO button (yes/no); body font if visible\n" +
			"Visit at most 8 distinct routes including the starting page. Prefer nav over random cards. " +
			"If a click would leave this host or open an identity provider, skip it. " +
			"End on a representative inner page, not a login screen.",
	},
}

func addRunPlaybooks(run *cobra.Command) {
	for _, playbook := range runPlaybooks {
		playbook := playbook
		var output string
		var keepOpen bool
		cmd := &cobra.Command{
			Use:   playbook.name,
			Short: playbook.short,
			Long:  playbook.short + ".\nUses a fixed goal. Pass -o and --keep-open the same way as vibium run \"<goal>\".",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				result, err := runGoal(cmd, playbook.goal, output, keepOpen)
				if err != nil {
					printError(err)
					return
				}
				printRunResult(playbook.name, result, output)
			},
		}
		addRunFlags(cmd, &output, &keepOpen)
		cmd.Example = fmt.Sprintf("  vibium run %s -o %s.zip --keep-open\n  # %s", playbook.name, playbook.name, playbook.short)
		run.AddCommand(cmd)
	}
}
