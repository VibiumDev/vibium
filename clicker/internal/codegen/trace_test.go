package codegen

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, traceLines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "record.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("trace.trace")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(strings.Join(traceLines, "\n"))); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadTrace(t *testing.T) {
	path := writeZip(t, []string{
		`{"version":8,"type":"context-options","browserName":"chromium"}`,
		`{"type":"before","callId":"call@1","class":"Tracing","method":"tracingGroup","title":"login","params":{"name":"login"}}`,
		`{"type":"before","callId":"call@2","class":"Page","method":"vibium:page.navigate","title":"Page.navigate","params":{"url":"https://example.com"},"parentId":"call@1"}`,
		`{"type":"after","callId":"call@2"}`,
		`{"type":"before","callId":"call@3","class":"Element","method":"vibium:element.click","title":"Element.click","params":{"selector":"#x"}}`,
		`{"type":"after","callId":"call@3","error":{"message":"timed out"}}`,
		`{"type":"after","callId":"call@1"}`,
	})
	steps, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 4 {
		t.Fatalf("got %d steps, want 4: %+v", len(steps), steps)
	}
	if steps[0].Kind != StepGroupStart || steps[0].Title != "login" {
		t.Errorf("step 0 = %+v, want group start titled login", steps[0])
	}
	if steps[1].Method != "vibium:page.navigate" || steps[1].Params["url"] != "https://example.com" {
		t.Errorf("step 1 = %+v, want navigate action", steps[1])
	}
	if steps[2].Error != "timed out" {
		t.Errorf("step 2 error = %q, want timed out", steps[2].Error)
	}
	if steps[3].Kind != StepGroupEnd {
		t.Errorf("step 3 = %+v, want group end", steps[3])
	}
}

func TestReadTraceRejectsWrongVersion(t *testing.T) {
	path := writeZip(t, []string{
		`{"version":7,"type":"context-options"}`,
	})
	if _, err := ReadTrace(path); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("got %v, want a version error", err)
	}
}

func TestReadTraceRejectsMissingFile(t *testing.T) {
	if _, err := ReadTrace(filepath.Join(t.TempDir(), "missing.zip")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestReadTraceEndToEndGenerate(t *testing.T) {
	path := writeZip(t, []string{
		`{"version":8,"type":"context-options"}`,
		`{"type":"before","callId":"call@1","class":"Page","method":"vibium:page.navigate","title":"Page.navigate","params":{"url":"https://example.com"}}`,
		`{"type":"after","callId":"call@1"}`,
		`{"type":"before","callId":"call@2","class":"Element","method":"vibium:element.fill","title":"Element.fill","params":{"selector":"#q","value":"it's \"quoted\""}}`,
		`{"type":"after","callId":"call@2"}`,
	})
	steps, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	js, err := Generate(steps, "javascript")
	if err != nil {
		t.Fatal(err)
	}
	want := "await vibe.find({ css: '#q' }).fill('it\\'s \"quoted\"');\n"
	if !strings.Contains(js, want) {
		t.Errorf("output lacks %q\ngot:\n%s", want, js)
	}
}
