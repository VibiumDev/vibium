package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/daemon"
	"github.com/vibium/clicker/internal/verifier"
)

type operationFiles struct{ input, output, report string }

func (f *operationFiles) validate(cmd *cobra.Command) error {
	for name, value := range map[string]*string{"input": &f.input, "output": &f.output, "report": &f.report} {
		if cmd.Flags().Changed(name) && *value == "" {
			return fmt.Errorf("--%s requires a nonempty path", name)
		}
		if *value == "" {
			continue
		}
		absolute, err := filepath.Abs(*value)
		if err != nil {
			return err
		}
		*value = absolute
		if name != "input" {
			if _, err := os.Lstat(absolute); err == nil {
				return fmt.Errorf("--%s path already exists; choose a new file", name)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
	}
	if f.input != "" && f.output != "" {
		return fmt.Errorf("--input and --output cannot be combined yet; use --report to save an archive verification verdict")
	}
	if f.output != "" && f.output == f.report {
		return fmt.Errorf("--output and --report require different paths")
	}
	return nil
}

func newCheckCmd() *cobra.Command {
	var files operationFiles
	cmd := &cobra.Command{
		Use:   `check "<claim>"`,
		Short: "Check a claim in a live browser or an existing recording with an independent model",
		Long:  "Check a claim in a live browser or an existing recording with an independent model.\nCheck provider setup first with: vibium ready ai.\nCloses a browser it starts after saving evidence; reuses and preserves an existing browser.\nUse --keep-open to leave a newly started browser open.",
		Example: `  vibium check "changing my display name persists after refresh"
  # Returns PASS, FAIL, or INCONCLUSIVE with concise evidence.
  vibium check "changing my display name persists after refresh" -o verification.zip
  # Saves the live verification in a new recording ZIP.
  vibium check "https://example.com is up" -o status.zip --keep-open
  # Leaves a browser started by Check open for inspection.
  vibium check -i record.zip "checkout completed successfully"
  vibium check --input trace.zip --report verification.json "the order confirmation is visible"
  # Reads a Vibium recording or Playwright trace without launching a browser.
  vibium check "the cart contains one battery pack" -o verification.zip --report verdict.json --json
  # Saves a recording and JSON report; also prints the structured verdict.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := runCheck(cmd, args[0], files)
			if err != nil {
				printError(err)
				return
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: result})
				return
			}
			fmt.Printf("CHECK: %s\n\n%s\n\n%s\n", result.Claim, result.Verdict(), result.Summary)
			for _, e := range result.Evidence {
				fmt.Printf("- %s\n", e.Summary)
			}
			if files.output != "" {
				fmt.Printf("Recording saved to %s\n", files.output)
			}
			if files.report != "" {
				fmt.Printf("Report saved to %s\n", files.report)
			}
		},
	}
	cmd.Flags().StringVarP(&files.input, "input", "i", "", "Read an existing Vibium recording or Playwright trace ZIP; no browser is launched")
	cmd.Flags().StringVarP(&files.output, "output", "o", "", "Save a new recording ZIP of live verification; an active recording is exported without stopping it (current chunk, no video)")
	cmd.Flags().StringVar(&files.report, "report", "", "Save the verdict and concise evidence as a new JSON file")
	cmd.Flags().Bool("keep-open", false, "Leave a browser started by Check open after the run (existing browsers are always preserved)")
	addModelFlags(cmd)
	return cmd
}

// Return errors before printing them so all artifact cleanup runs before the
// CLI's printError exits the process.
func runCheck(cmd *cobra.Command, claim string, files operationFiles) (result *verifier.Result, err error) {
	if err = files.validate(cmd); err != nil {
		return
	}
	keepOpen, _ := cmd.Flags().GetBool("keep-open")
	if files.input != "" && cmd.Flags().Changed("keep-open") {
		return nil, fmt.Errorf("--keep-open only applies to live verification; it cannot be combined with --input")
	}
	config, err := verifier.ResolveConfig("check", modelOverrides(cmd))
	if err != nil {
		return result, fmt.Errorf("%w; run vibium ready ai for setup checks", err)
	}
	req := verifier.Request{Claim: claim, Record: files.input, Output: files.output, Config: config}
	if err = req.Validate(); err != nil {
		return
	}
	var report *os.File
	if files.report != "" {
		report, err = os.OpenFile(files.report, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return
		}
		defer func() {
			closeErr := report.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				os.Remove(files.report)
			}
		}()
	}
	// Keep browser startup and cleanup inside the daemon's serialized Check
	// request. A separate browser_start call cannot safely establish ownership.
	run := func() (*verifier.Result, error) {
		if files.input == "" {
			return daemon.CheckWithBrowser(req, agent.OperationCLIOptions{LaunchOptions: requestedLaunchOptions(), KeepOpen: keepOpen})
		}
		return daemon.Check(req)
	}
	result, err = run()
	if isConnectionError(err) {
		daemon.CleanStale()
		if err = autoStartDaemon(); err == nil {
			result, err = run()
		}
	}
	if err != nil {
		return
	}
	if report != nil {
		err = json.NewEncoder(report).Encode(result)
	}
	return
}
