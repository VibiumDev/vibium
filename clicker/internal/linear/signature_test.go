package linear

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSignatureRenderIncludesArtifacts(t *testing.T) {
	s := Signature{
		Engine:    "vibium 2026.9.16",
		Binary:    "/usr/local/bin/vibium",
		Command:   "vibium run walk",
		Playbook:  "walk",
		Model:     "openai-compatible/grok-4",
		Host:      "https://example.com/roster",
		Artifacts: []string{"/tmp/out/run.json", "/tmp/out/home.png"},
	}
	got := s.Render()
	for _, want := range []string{
		"## Signed by Vibium",
		"Not a hand-written bug report",
		"`vibium 2026.9.16`",
		"`/usr/local/bin/vibium`",
		"`vibium run walk`",
		"`walk`",
		"`openai-compatible/grok-4`",
		"https://example.com/roster",
		"`/tmp/out/run.json`",
		"`/tmp/out/home.png`",
		"### Artifacts",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "lin_api_") || strings.Contains(got, "API_KEY") {
		t.Fatal("signature leaked a key")
	}
}

func TestListPackFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "shots"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shots", "00-home.png"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	files := ListPackFiles(dir)
	joined := strings.Join(files, "\n")
	if !strings.Contains(joined, "run.json") || !strings.Contains(joined, "00-home.png") {
		t.Fatalf("%v", files)
	}
	for _, f := range files {
		if !filepath.IsAbs(f) {
			t.Fatalf("not abs: %s", f)
		}
	}
}

func TestPackTitlePrefersBroken(t *testing.T) {
	p := Pack{Result: RunResult{
		Summary: "all good",
		Evidence: []EvidenceLine{
			{Summary: "WORKS - / home"},
			{Summary: "BROKEN - /settings reporting API 404"},
			{Summary: "CSS SHELL - dark"},
		},
	}}
	if !strings.Contains(p.Title(), "BROKEN") {
		t.Fatalf("title %q", p.Title())
	}
	none := Pack{Result: RunResult{
		Summary:  "ok",
		Evidence: []EvidenceLine{{Summary: "BROKEN - none"}},
	}}
	if strings.Contains(none.Title(), "BROKEN") {
		t.Fatalf("used none: %q", none.Title())
	}
}

func TestLoadPackEnvelope(t *testing.T) {
	dir := t.TempDir()
	raw := `{"ok":true,"result":{"status":"completed","goal":"walk","summary":"toured nav","evidence":[{"type":"observation","summary":"BROKEN - /x 404"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := LoadPack(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Result.Status != "completed" || p.Result.Summary != "toured nav" {
		t.Fatalf("%+v", p.Result)
	}
	body := p.Description()
	if !strings.Contains(body, "Signed by Vibium") || !strings.Contains(body, "run.json") {
		t.Fatalf("body:\n%s", body)
	}
}
