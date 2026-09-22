package linear

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Signature is the block appended to every Linear issue or comment Vibium files.
// It is how a reader tells a generated report from a hand-typed ticket.
type Signature struct {
	Engine    string   // vibium version, e.g. "vibium 2026.9.16" or "vibium dev"
	Binary    string   // absolute path of the vibium binary when known
	Command   string   // the CLI invocation, e.g. `vibium run walk`
	Playbook  string   // inspect, walk, or empty for a freeform goal
	Model     string   // provider/model when Run or Check used one
	Host      string   // page URL after the run, when known
	Artifacts []string // absolute paths of recordings, JSON, screenshots
}

// Render returns Markdown for the issue body. Never includes API keys.
func (s Signature) Render() string {
	var b strings.Builder
	b.WriteString("## Signed by Vibium\n\n")
	b.WriteString("Not a hand-written bug report. Opened from a Vibium recording.\n\n")
	if s.Engine != "" {
		fmt.Fprintf(&b, "- Engine: `%s`\n", s.Engine)
	}
	if s.Binary != "" {
		fmt.Fprintf(&b, "- Binary: `%s`\n", s.Binary)
	}
	if s.Command != "" {
		fmt.Fprintf(&b, "- Command: `%s`\n", s.Command)
	}
	if s.Playbook != "" {
		fmt.Fprintf(&b, "- Playbook: `%s`\n", s.Playbook)
	}
	if s.Model != "" {
		fmt.Fprintf(&b, "- Model: `%s`\n", s.Model)
	}
	if s.Host != "" {
		fmt.Fprintf(&b, "- URL: %s\n", s.Host)
	}
	if len(s.Artifacts) > 0 {
		b.WriteString("\n### Artifacts\n\n")
		for _, path := range s.Artifacts {
			fmt.Fprintf(&b, "- `%s`\n", path)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// EngineLabel is `vibium <version>` for the signature.
func EngineLabel(version string) string {
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	return "vibium " + version
}

// BinaryPath is the vibium executable when os.Executable works.
func BinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

// AbsArtifacts returns existing paths as absolute, skipping empties.
func AbsArtifacts(paths []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

// ListPackFiles lists regular files in dir (not recursive except shots/).
func ListPackFiles(dir string) []string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if e.Name() == "shots" {
				shots, err := os.ReadDir(p)
				if err != nil {
					continue
				}
				for _, s := range shots {
					if s.Type().IsRegular() {
						names = append(names, filepath.Join(p, s.Name()))
					}
				}
			}
			continue
		}
		if e.Type().IsRegular() {
			names = append(names, p)
		}
	}
	return AbsArtifacts(names)
}
