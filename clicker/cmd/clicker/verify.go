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

type verifyFiles struct{ input, output, report string }

func (f *verifyFiles) validate(cmd *cobra.Command) error {
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

func newVerifyCmd() *cobra.Command {
	var files verifyFiles
	cmd := &cobra.Command{
		Use:   `verify "<claim>"`,
		Short: "Verify a claim in a live browser or an existing recording with an independent model",
		Example: `  vibium verify "changing my display name persists after refresh"
  # Returns PASS, FAIL, or INCONCLUSIVE with concise evidence.
  vibium verify "changing my display name persists after refresh" -o verification.zip
  # Saves the live verification in a new recording ZIP.
  vibium verify -i record.zip "checkout completed successfully"
  vibium verify --input trace.zip --report verification.json "the order confirmation is visible"
  # Reads a Vibium recording or Playwright trace without launching a browser.
  vibium verify "the cart contains one battery pack" -o verification.zip --report verdict.json --json
  # Saves a recording and JSON report; also prints the structured verdict.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := runVerify(cmd, args[0], files)
			if err != nil {
				printError(err)
				return
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: result})
				return
			}
			fmt.Printf("VERIFY: %s\n\n%s\n\n%s\n", result.Claim, result.Verdict(), result.Summary)
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
	return cmd
}

// Return errors before printing them so all artifact cleanup runs before the
// CLI's printError exits the process.
func runVerify(cmd *cobra.Command, claim string, files verifyFiles) (result *verifier.Result, err error) {
	if err = files.validate(cmd); err != nil {
		return
	}
	config, err := verifier.ConfigFromEnv()
	if err != nil {
		return result, err
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
	if files.input == "" {
		if _, err = daemonCall("browser_start", map[string]interface{}{}); err != nil {
			return
		}
	}
	result, err = daemon.Verify(req)
	if files.input != "" && isConnectionError(err) {
		daemon.CleanStale()
		if err = autoStartDaemon(); err == nil {
			result, err = daemon.Verify(req)
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
