package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/codegen"
)

func newCodegenCmd() *cobra.Command {
	var input, language, output string
	cmd := &cobra.Command{
		Use:   "codegen -i <recording.zip>",
		Short: "Generate a runnable script from a recording",
		Long: "Generate a runnable script that replays a recording's action stream.\n" +
			"Reads the zip offline; no browser or daemon is needed.\n" +
			"The script is a starting point: edit it, then run it with node or python.",
		Example: `  vibium codegen -i record.zip
  # Prints a JavaScript replay script to stdout:
  #   import { browser } from 'vibium';
  #   ...
  #   await vibe.go('https://var.parts');
  #   await vibe.find({ text: 'Vibium Battery Pack' }).click();
  vibium codegen -i record.zip --language python -o replay.py
  # Saves the Python version of the script as a new file.`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			script, err := runCodegen(input, language, output)
			if err != nil {
				printError(err)
				return
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: script})
				return
			}
			if output != "" {
				fmt.Printf("Script saved to %s\n", output)
				return
			}
			fmt.Print(script)
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", "", "Recording zip to read (required)")
	cmd.Flags().StringVarP(&language, "language", "l", "javascript", "Script language: javascript or python")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Save the script as a new file instead of printing it")
	cmd.MarkFlagRequired("input")
	return cmd
}

func runCodegen(input, language, output string) (string, error) {
	valid := false
	for _, l := range codegen.Languages {
		if language == l {
			valid = true
		}
	}
	if !valid {
		return "", fmt.Errorf("unsupported language %q (supported: %s)", language, strings.Join(codegen.Languages, ", "))
	}
	path, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	steps, err := codegen.ReadTrace(path)
	if err != nil {
		return "", err
	}
	script, err := codegen.Generate(steps, language)
	if err != nil {
		return "", err
	}
	if output != "" {
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			if os.IsExist(err) {
				return "", fmt.Errorf("--output path already exists; choose a new file")
			}
			return "", err
		}
		if _, err := f.WriteString(script); err != nil {
			f.Close()
			os.Remove(output)
			return "", err
		}
		if err := f.Close(); err != nil {
			os.Remove(output)
			return "", err
		}
	}
	return script, nil
}
