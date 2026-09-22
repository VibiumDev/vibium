package linear

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunResult is the subset of vibium run JSON we file.
type RunResult struct {
	Status   string         `json:"status"`
	Goal     string         `json:"goal"`
	Summary  string         `json:"summary"`
	Evidence []EvidenceLine `json:"evidence"`
	Host     string         `json:"host,omitempty"`
}

// EvidenceLine is one observation.
type EvidenceLine struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

// Pack is a directory of run artifacts plus the parsed result.
type Pack struct {
	Dir      string
	Result   RunResult
	Playbook string
	Command  string
	Model    string
	Version  string
}

// LoadPack reads dir/run.json if present, else treats dir as missing JSON.
func LoadPack(dir string) (Pack, error) {
	pack := Pack{Dir: dir}
	path := filepath.Join(dir, "run.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pack, fmt.Errorf("no run.json in %s; pass a Vibium pack directory", dir)
		}
		return pack, err
	}
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return pack, fmt.Errorf("run.json: %w", err)
	}
	payload := envelope.Result
	if len(payload) == 0 {
		payload = raw
	}
	if err := json.Unmarshal(payload, &pack.Result); err != nil {
		return pack, fmt.Errorf("run.json result: %w", err)
	}
	return pack, nil
}

// Title picks a Linear title from BROKEN evidence, else the summary.
func (p Pack) Title() string {
	for _, e := range p.Result.Evidence {
		s := strings.TrimSpace(e.Summary)
		if looksBroken(s) {
			return clipTitle(s)
		}
	}
	if s := strings.TrimSpace(p.Result.Summary); s != "" {
		return clipTitle("Vibium run: " + s)
	}
	if p.Result.Host != "" {
		return clipTitle("Vibium run: " + p.Result.Host)
	}
	return "Vibium run"
}

func looksBroken(s string) bool {
	u := strings.ToUpper(s)
	if !strings.Contains(u, "BROKEN") {
		return false
	}
	if strings.Contains(u, "BROKEN: NONE") || strings.Contains(u, "BROKEN — NONE") || strings.Contains(u, "BROKEN - NONE") {
		return false
	}
	return true
}

func clipTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		return s[:197] + "..."
	}
	return s
}

// Description is the issue body: findings, then the signature with filepaths.
func (p Pack) Description() string {
	var b strings.Builder
	if p.Result.Status != "" {
		fmt.Fprintf(&b, "Run status: `%s`\n\n", p.Result.Status)
	}
	if p.Result.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Result.Summary)
	}
	if len(p.Result.Evidence) > 0 {
		b.WriteString("## Evidence\n\n")
		for _, e := range p.Result.Evidence {
			fmt.Fprintf(&b, "- %s\n", e.Summary)
		}
		b.WriteString("\n")
	}
	sig := Signature{
		Engine:    EngineLabel(p.Version),
		Binary:    BinaryPath(),
		Command:   p.Command,
		Playbook:  p.Playbook,
		Model:     p.Model,
		Host:      p.Result.Host,
		Artifacts: ListPackFiles(p.Dir),
	}
	if sig.Command == "" && p.Result.Goal != "" {
		sig.Command = "vibium run"
	}
	b.WriteString("---\n\n")
	b.WriteString(sig.Render())
	return b.String()
}

// CommentBody is a shorter signed comment for an existing issue.
func (p Pack) CommentBody() string {
	var b strings.Builder
	b.WriteString("— Vibium (generated)\n\n")
	if p.Result.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Result.Summary)
	}
	b.WriteString(Signature{
		Engine:    EngineLabel(p.Version),
		Binary:    BinaryPath(),
		Command:   p.Command,
		Playbook:  p.Playbook,
		Model:     p.Model,
		Host:      p.Result.Host,
		Artifacts: ListPackFiles(p.Dir),
	}.Render())
	return b.String()
}
