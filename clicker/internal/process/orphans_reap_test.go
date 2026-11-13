//go:build !windows

package process

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestReapOrphansSurvivesPSTruncation reproduces #701's third cause. On the
// arm64 CI runner, ps capped each output line at the terminal width (COLUMNS
// without a tty), and the vibium pipe's binary path — long on a self-hosted
// runner workdir — got cut mid-name to ".../vibiu". The ancestry walk then
// found no vibium owner for a live session's chromedriver and SIGKILLed it
// the moment a sibling vibium launched. The fake ps below behaves like the
// real one: full lines only when unlimited width (ww) is requested.
//
// Pids are far above any real process so the SIGKILL of the true orphan hits
// nothing (ESRCH, ignored by ReapOrphans).
func TestReapOrphansSurvivesPSTruncation(t *testing.T) {
	marker := "/home/admin/.cache/vibium/chrome-for-testing"
	// The vibium name must start past column 80, like the venv-installed
	// binary on the CI runner did.
	pipeCmd := "/home/admin/actions-runner/_work/vibium/vibium/clients/python/.venv/lib/python3/site-packages/vibium/bin/vibium pipe"
	table := fmt.Sprintf(
		"      1       0 /sbin/init\n"+
			"9190001       1 %s\n"+ // live vibium pipe, long path
			"9190002 9190001 %s/156.0/chromedriver-linux64/chromedriver --port=1\n"+ // its browser: NOT an orphan
			"9190003       1 %s/156.0/chromedriver-linux64/chromedriver --port=2\n", // true orphan (owner died)
		pipeCmd, marker, marker)

	dir := t.TempDir()
	fakePS := filepath.Join(dir, "ps")
	script := fmt.Sprintf(`#!/bin/sh
# Fake procps: honor COLUMNS unless unlimited width (ww) was requested.
table='%s'
case "$*" in
  *ww*) printf '%%s' "$table" ;;
  *)    printf '%%s' "$table" | cut -c1-"${COLUMNS:-80}" ;;
esac
`, table)
	if err := os.WriteFile(fakePS, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("COLUMNS", "80")

	got := ReapOrphans(marker)
	want := []int{9190003}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReapOrphans killed %v, want only the true orphan %v;\n"+
			"killing 9190002 means a truncated ps line hid its live vibium owner", got, want)
	}
}
