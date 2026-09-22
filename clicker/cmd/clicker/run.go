package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/daemon"
	runop "github.com/vibium/clicker/internal/run"
	"github.com/vibium/clicker/internal/verifier"
)

func newRunCmd() *cobra.Command {
	var output string
	var keepOpen bool
	var report string
	cmd := &cobra.Command{
		Use: `run "<goal>"`, Short: "Accomplish a goal in the live browser with configured model tools",
		Long: "Accomplish a live browser goal using VIBIUM_AI_* configuration.\nReturns completed or not_completed with evidence. Check setup with vibium ready ai.\nCloses only a browser it starts, after saving evidence; --keep-open preserves it.\n--report linear files a signed Linear issue (or linear:ENG-12 comments on one).",
		Example: `  vibium run "change my timezone to America/Chicago"
  # Accomplishes the goal in the existing browser and reports its result.
  vibium run "open https://example.com" -o run.zip --keep-open
  # Saves the live recording and leaves a newly started browser open.
  vibium run "open every nav item" -o walk.zip --report linear
  # Files a Linear issue signed by Vibium with the zip path.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := runGoal(cmd, args[0], output, keepOpen)
			if err != nil {
				printError(err)
				return
			}
			posted, reportErr := maybeReportLinear(cmd.Context(), report, output, result, "vibium run", "", "", "")
			if jsonOutput {
				out := map[string]any{"run": result}
				if posted != nil {
					out["report"] = posted
				}
				ok := reportErr == nil
				env := jsonEnvelope{OK: ok, Result: out}
				if reportErr != nil {
					env.Error = reportErr.Error()
				}
				printJSON(env)
				if reportErr != nil {
					os.Exit(1)
				}
				return
			}
			fmt.Printf("RUN: %s\n\n%s\n\n%s\n", result.Goal, result.Verdict(), result.Summary)
			for _, e := range result.Evidence {
				fmt.Printf("- %s\n", e.Summary)
			}
			if output != "" {
				fmt.Printf("Recording saved to %s\n", output)
			}
			if reportErr != nil {
				printError(reportErr)
				return
			}
			if posted != nil {
				fmt.Printf("Linear %s %s\n", posted.Action, posted.URL)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Save a new live recording ZIP; an active recording exports its current chunk without continuous video")
	cmd.Flags().BoolVar(&keepOpen, "keep-open", false, "Keep a browser started by Run open after saving evidence")
	cmd.Flags().StringVar(&report, "report", "", "File the result: linear, or linear:ENG-12 to comment")
	addModelFlags(cmd)
	return cmd
}
func runGoal(cmd *cobra.Command, goal, output string, keepOpen bool) (*runop.Result, error) {
	files := operationFiles{output: output}
	if err := files.validate(cmd); err != nil {
		return nil, err
	}
	config, err := verifier.ResolveConfig("run", modelOverrides(cmd))
	if err != nil {
		return nil, fmt.Errorf("%w; run vibium ready ai for setup checks", err)
	}
	req := runop.Request{Goal: goal, Output: files.output, Config: config}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	options := agent.OperationCLIOptions{LaunchOptions: requestedLaunchOptions(), KeepOpen: keepOpen}
	result, err := daemon.Run(req, options)
	if isConnectionError(err) {
		daemon.CleanStale()
		if err = autoStartDaemon(); err == nil {
			result, err = daemon.Run(req, options)
		}
	}
	return result, err
}
