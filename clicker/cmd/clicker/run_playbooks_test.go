package main

import (
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/verifier"
)

func TestRunPlaybooks(t *testing.T) {
	run := newRunCmd()
	if len(runPlaybooks) == 0 {
		t.Fatal("expected at least one run playbook")
	}
	seen := map[string]bool{}
	for _, playbook := range runPlaybooks {
		if playbook.name == "" || strings.ContainsAny(playbook.name, " \t") {
			t.Fatalf("playbook name %q must be a single token", playbook.name)
		}
		if seen[playbook.name] {
			t.Fatalf("duplicate playbook %q", playbook.name)
		}
		seen[playbook.name] = true
		goal := strings.TrimSpace(playbook.goal)
		if goal == "" || len(playbook.goal) > verifier.MaxClaim {
			t.Fatalf("playbook %q goal must be nonempty and at most %d bytes, got %d", playbook.name, verifier.MaxClaim, len(playbook.goal))
		}
		cmd, _, err := run.Find([]string{playbook.name})
		if err != nil {
			t.Fatalf("find %s: %v", playbook.name, err)
		}
		if cmd.Name() != playbook.name {
			t.Fatalf("find %s got %q", playbook.name, cmd.Name())
		}
	}
	cmd, leftover, err := run.Find([]string{"not-a-playbook"})
	if err != nil {
		t.Fatalf("find unknown: %v", err)
	}
	if cmd.Name() == "not-a-playbook" {
		t.Fatal("unknown token must remain a freeform run goal, not a subcommand")
	}
	if len(leftover) != 1 || leftover[0] != "not-a-playbook" {
		t.Fatalf("unknown token leftover = %q", leftover)
	}
}
