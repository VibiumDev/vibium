package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckFileFlags(t *testing.T) {
	for _, args := range [][]string{
		{"-i", "record.zip"}, {"--input", "trace.zip", "--report", "verdict.json"},
		{"-o", "verification.zip", "--report", "verdict.json"},
	} {
		cmd := newCheckCmd()
		if err := cmd.ParseFlags(args); err != nil {
			t.Fatal(err)
		}
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		report, _ := cmd.Flags().GetString("report")
		files := operationFiles{input, output, report}
		if err := files.validate(cmd); err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{files.input, files.output, files.report} {
			if value != "" && !filepath.IsAbs(value) {
				t.Fatal("path not resolved in caller's cwd")
			}
		}
	}
	for _, args := range [][]string{
		{"--record", "record.zip"}, {"--trace", "trace.zip"},
		{"--input=", "--report=verdict.json"}, {"--output="}, {"--report="},
		{"-i", "input.zip", "-o", "output.zip"}, {"-o", "same.zip", "--report", "./same.zip"},
	} {
		cmd := newCheckCmd()
		if err := cmd.ParseFlags(args); err != nil {
			continue
		}
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		report, _ := cmd.Flags().GetString("report")
		files := operationFiles{input, output, report}
		if err := files.validate(cmd); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestCheckNeverOverwritesEvidence(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "input.zip")
	if err := os.WriteFile(original, []byte("immutable"), 0600); err != nil {
		t.Fatal(err)
	}
	hardlink := filepath.Join(dir, "hardlink.zip")
	if err := os.Link(original, hardlink); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "symlink.zip")
	if err := os.Symlink(original, symlink); err != nil {
		t.Fatal(err)
	}
	dangling := filepath.Join(dir, "dangling.zip")
	if err := os.Symlink(filepath.Join(dir, "absent"), dangling); err != nil {
		t.Fatal(err)
	}
	for _, destination := range []string{original, hardlink, symlink, dangling, dir} {
		for _, flag := range []string{"--report", "--output"} {
			cmd := newCheckCmd()
			if err := cmd.ParseFlags([]string{flag, destination}); err != nil {
				t.Fatal(err)
			}
			files := operationFiles{}
			if flag == "--report" {
				files.report = destination
			} else {
				files.output = destination
			}
			if err := files.validate(cmd); err == nil {
				t.Errorf("accepted %s %s", flag, destination)
			}
		}
	}
	data, _ := os.ReadFile(original)
	if string(data) != "immutable" {
		t.Fatal("changed input")
	}
}

func TestCheckKeepOpenOnlyForLiveChecks(t *testing.T) {
	cmd := newCheckCmd()
	if err := cmd.ParseFlags([]string{"--keep-open"}); err != nil {
		t.Fatal(err)
	}
	keepOpen, _ := cmd.Flags().GetBool("keep-open")
	if !keepOpen {
		t.Fatal("flag was not set")
	}
	_, err := runCheck(cmd, "claim", operationFiles{input: "record.zip"})
	if err == nil || !strings.Contains(err.Error(), "only applies to live verification") {
		t.Fatalf("unexpected error: %v", err)
	}
}
