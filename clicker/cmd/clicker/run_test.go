package main

import (
	"path/filepath"
	"testing"
)

func TestRunFileFlags(t *testing.T) {
	cmd := newRunCmd()
	if err := cmd.ParseFlags([]string{"-o", "run.zip", "--report", "result.json"}); err != nil {
		t.Fatal(err)
	}
	output, _ := cmd.Flags().GetString("output")
	report, _ := cmd.Flags().GetString("report")
	files := operationFiles{output: output, report: report}
	if err := files.validate(cmd); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{files.output, files.report} {
		if !filepath.IsAbs(value) {
			t.Fatal("path not resolved in caller's cwd")
		}
	}
	for _, args := range [][]string{
		{"-i", "record.zip"}, {"--input", "record.zip"},
		{"--report="}, {"--report", "same.zip", "-o", "./same.zip"},
	} {
		cmd := newRunCmd()
		if err := cmd.ParseFlags(args); err != nil {
			continue
		}
		output, _ := cmd.Flags().GetString("output")
		report, _ := cmd.Flags().GetString("report")
		files := operationFiles{output: output, report: report}
		if err := files.validate(cmd); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}
