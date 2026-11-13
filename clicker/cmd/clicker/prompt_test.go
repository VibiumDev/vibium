package main

import (
	"github.com/spf13/cobra"
	"reflect"
	"testing"
)

func TestPromptShorthand(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		expanded bool
	}{
		{"sentence", []string{"open the account page"}, true},
		{"flags before prompt", []string{"--model", "model with spaces", "--headless", "change my name", "-o", "a file.zip"}, true},
		{"equals flag", []string{"--model=model with spaces", "change my name"}, true},
		{"terminator", []string{"--", "--open the page"}, true},
		{"known command", []string{"stop"}, false},
		{"known command arguments", []string{"check", "a claim"}, false},
		{"explicit one word", []string{"run", "stop"}, false},
		{"removed command", []string{"perform", "a goal"}, false},
		{"typo", []string{"staart"}, false},
		{"unquoted words", []string{"open", "the", "page"}, false},
		{"empty", []string{}, false},
		{"whitespace", []string{"  "}, false},
		{"option is not prompt", []string{"--model", "model with spaces"}, false},
		{"missing option value", []string{"a goal", "--model"}, false},
		{"unknown option", []string{"a goal", "--wat"}, false},
	}
	for _, row := range cases {
		t.Run(row.name, func(t *testing.T) {
			root := &cobra.Command{Use: "vibium"}
			root.PersistentFlags().Bool("headless", false, "")
			run := newRunCmd()
			root.AddCommand(run, &cobra.Command{Use: "stop"}, &cobra.Command{Use: "check"})
			want := row.args
			if row.expanded {
				want = append([]string{"run"}, want...)
			}
			if got := promptArgs(root, run, row.args); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}
