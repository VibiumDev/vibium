package browser

import (
	"path/filepath"
	"strings"
	"testing"
)

// The Windows profile sweep must target firefox.exe scoped to this session's
// profile, so it reaps the re-parented browser (#622) without touching the
// user's own Firefox. The kill itself needs Windows; this pins the script the
// Windows path runs. The path is built with filepath.Join so filepath.Base
// splits it the same way on the test host as it does on the Windows runtime.
func TestFirefoxProfileKillScript(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "vibium-firefox-profile-1738012345")
	script := firefoxProfileKillScript(profile)

	for _, want := range []string{
		`Name='firefox.exe'`,                // scoped to Firefox, not every process
		"vibium-firefox-profile-1738012345", // scoped to this session's profile
		"-like",                             // substring match on the command line
		"Stop-Process",                      // the actual termination
		"-Force",                            // kill, do not ask
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q\ngot: %s", want, script)
		}
	}
}

// The match uses the profile basename only, not the full path: the basename
// is unique per session and carries no separators or drive colon to escape
// inside the -like pattern. A parent path component must not leak in.
func TestFirefoxProfileKillScriptUsesBasenameOnly(t *testing.T) {
	parent := t.TempDir()
	script := firefoxProfileKillScript(filepath.Join(parent, "vibium-firefox-profile-42"))

	if strings.Contains(script, parent) {
		t.Errorf("script embedded the parent path %q instead of the basename:\n%s", parent, script)
	}
	if !strings.Contains(script, "vibium-firefox-profile-42") {
		t.Errorf("script dropped the profile basename:\n%s", script)
	}
}
