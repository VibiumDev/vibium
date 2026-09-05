package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/daemon"
	"github.com/vibium/clicker/internal/verifier"
)

func newVerifyCmd() *cobra.Command {
	var record, trace, output string
	cmd := &cobra.Command{
		Use:   `verify "<claim>"`,
		Short: "Verify a claim in a live browser or an existing recording with an independent model",
		Example: `  vibium verify "changing my display name persists after refresh"
  vibium verify --record record.zip "checkout completed successfully"
  vibium verify --trace trace.zip --output verification.json "the order confirmation is visible"
  # Returns PASS, FAIL, or INCONCLUSIVE with concise evidence. Input zips stay unchanged.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("record") && cmd.Flags().Changed("trace") {
				printError(fmt.Errorf("use either --record or its alias --trace"))
				return
			}
			if cmd.Flags().Changed("trace") {
				record = trace
			}
			if (cmd.Flags().Changed("record") || cmd.Flags().Changed("trace")) && record == "" {
				printError(fmt.Errorf("record path must not be empty"))
				return
			}
			if record != "" {
				var err error
				record, err = filepath.Abs(record)
				if err != nil {
					printError(err)
					return
				}
				// Resolve on the calling side: a reused daemon can have a different cwd.
			}
			if output != "" {
				if _, err := os.Lstat(output); err == nil {
					printError(fmt.Errorf("output already exists; choose a new JSON file (input archives are read-only)"))
					return
				} else if !os.IsNotExist(err) {
					printError(err)
					return
				}
			}
			config, err := verifier.ConfigFromEnv()
			if err != nil {
				printError(err)
				return
			}
			req := verifier.Request{Claim: args[0], Record: record, Config: config}
			if err := req.Validate(); err != nil {
				printError(err)
				return
			}
			if record == "" {
				if _, err := daemonCall("browser_start", map[string]interface{}{}); err != nil {
					printError(err)
					return
				}
			}
			result, err := daemon.Verify(req)
			if record != "" && isConnectionError(err) {
				daemon.CleanStale()
				if err = autoStartDaemon(); err == nil {
					result, err = daemon.Verify(req)
				}
			}
			if err != nil {
				printError(err)
				return
			}
			if output != "" {
				data, _ := json.MarshalIndent(result, "", "  ")
				f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
				if err != nil {
					printError(err)
					return
				}
				_, err = f.Write(append(data, '\n'))
				closeErr := f.Close()
				if err == nil {
					err = closeErr
				}
				if err != nil {
					printError(err)
					return
				}
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: result})
				return
			}
			fmt.Printf("VERIFY: %s\n\n%s\n\n%s\n", result.Claim, result.Verdict(), result.Summary)
			for _, e := range result.Evidence {
				fmt.Printf("- %s\n", e.Summary)
			}
		},
	}
	cmd.Flags().StringVar(&record, "record", "", "Verify an existing record.zip as read-only evidence; no browser is launched")
	cmd.Flags().StringVar(&trace, "trace", "", "Alias for --record (Playwright trace.zip)")
	cmd.Flags().StringVar(&output, "output", "", "Save the result as JSON to a new file; never overwrite input evidence")
	return cmd
}
