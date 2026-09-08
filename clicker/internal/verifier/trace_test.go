package verifier

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func archive(t *testing.T, files map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "trace.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for n, data := range files {
		w, err := z.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(data))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return p
}

const traceFixture = `{"type":"context-options","version":8,"browserName":"chromium","playwrightVersion":"1.58.0"}
{"type":"before","callId":"call@1","method":"tracingGroup","title":"Check checkout","params":{"method":"vibium:check.run","claim":"checkout works"}}
{"type":"before","callId":"call@2","parentId":"call@1","method":"click","params":{"selector":"#submit"}}
{"type":"after","callId":"call@2","afterSnapshot":"after@call@2"}
{"type":"after","callId":"call@1","result":{"status":"passed","summary":"Earlier assessment"}}
{"type":"frame-snapshot","snapshot":{"frameId":"frame","pageId":"page","snapshotName":"before","html":["HTML",{},["BODY",{},["H1",{},"Order confirmed"],["INPUT",{"type":"password","value":"PASSWORD"}],["SCRIPT",{},"SCRIPT-SECRET"]]]}}
{"type":"frame-snapshot","snapshot":{"frameId":"frame","pageId":"page","snapshotName":"after@call@2","html":["HTML",{},["BODY",{},[[1,1]],["P",{},"Order 123"]]]}}
{"type":"screencast-frame","pageId":"page","timestamp":12,"sha1":"screen.png"}
{"type":"console","text":"Order completed","messageType":"log"}
`

func TestTraceReadOnlyTools(t *testing.T) {
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aFOsAAAAASUVORK5CYII=")
	p := archive(t, map[string]string{"trace.trace": traceFixture, "resources/screen.png": string(png), "trace.network": `{"type":"resource-snapshot","snapshot":{"request":{"url":"https://user:secret@example.com/?token=SECRET","headers":[{"name":"Authorization","value":"SECRET"}]},"response":{"status":200,"content":{"text":"BODY-SECRET"}}}}`})
	before, _ := os.ReadFile(p)
	s, err := OpenTrace(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	run := func(name string, args map[string]interface{}) Observation {
		t.Helper()
		obs, err := s.Execute(context.Background(), name, args)
		if err != nil {
			t.Fatal(err)
		}
		return obs
	}
	if obs := run("trace_summary", nil); strings.Contains(obs.Text, "Order confirmed") {
		t.Fatal("initial context dumps archive")
	}
	if obs := run("trace_list_actions", nil); !strings.Contains(obs.Text, "call@1") {
		t.Fatal(obs.Text)
	}
	var screen string
	for _, e := range s.events {
		if e.data["type"] == "frame-snapshot" {
			obs := run("trace_inspect_snapshot", map[string]interface{}{"id": e.id})
			if !strings.Contains(obs.Text, "Order confirmed") || strings.Contains(obs.Text, "PASSWORD") || strings.Contains(obs.Text, "SCRIPT-SECRET") {
				t.Fatal(obs.Text)
			}
		}
		if e.data["type"] == "screencast-frame" {
			screen = e.id
		}
	}
	obs := run("trace_inspect_screenshot", map[string]interface{}{"id": screen})
	decoded, _ := base64.StdEncoding.DecodeString(obs.Image)
	if !bytes.Equal(png, decoded) || obs.MIME != "image/png" {
		t.Fatal("image corrupted")
	}
	if obs := run("trace_network", nil); strings.Contains(obs.Text, "SECRET") || strings.Contains(obs.Text, "secret") {
		t.Fatal(obs.Text)
	}
	for _, tc := range []struct {
		name string
		args map[string]interface{}
	}{
		{"browser_click", nil}, {"trace_inspect_screenshot", map[string]interface{}{"id": "../../secret"}}, {"trace_summary", map[string]interface{}{"path": "/etc/passwd"}}, {"trace_list_actions", map[string]interface{}{"offset": -1.0}}, {"trace_list_actions", map[string]interface{}{"offset": 1.5}},
	} {
		if _, err := s.Execute(context.Background(), tc.name, tc.args); err == nil {
			t.Fatalf("accepted %s", tc.name)
		}
	}
	after, _ := os.ReadFile(p)
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("modified input")
	}
}
func TestTraceRejectsMalformedArchives(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"no trace": {"readme": "hello"}, "future": {"trace.trace": `{"type":"context-options","version":99}`}, "invalid": {"trace.trace": "broken"}, "missing version": {"trace.trace": `{"type":"before","callId":"a"}`}, "traversal": {"../x": "x", "trace.trace": traceFixture},
	} {
		t.Run(name, func(t *testing.T) {
			if s, err := OpenTrace(context.Background(), archive(t, files)); err == nil {
				s.Close()
				t.Fatal("accepted malformed archive")
			}
		})
	}
}
func TestRecordedVerifierLoop(t *testing.T) {
	p := archive(t, map[string]string{"trace.trace": traceFixture})
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []message `json:"messages"`
			Tools    []struct {
				Function Tool `json:"function"`
			} `json:"tools"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		for _, tool := range body.Tools {
			if !strings.HasPrefix(tool.Function.Name, "trace_") {
				t.Error("live tool exposed")
			}
		}
		if requests == 0 {
			if len(body.Messages) != 3 || body.Messages[0].Content != traceInstruction {
				t.Error("wrong fresh context")
			}
			answer(w, nil, calls("trace_list_actions", 1))
		} else {
			answer(w, `{"status":"inconclusive","summary":"No confirmation evidence inspected","evidence":[]}`, nil)
		}
		requests++
	}))
	defer server.Close()
	req := testRequest(server.URL)
	req.Record = p
	result, err := CheckRecord(context.Background(), req)
	if err != nil || result.Status != "inconclusive" || requests != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTraceSearchIncludesResolvedDOM(t *testing.T) {
	p := archive(t, map[string]string{"trace.trace": traceFixture})
	source, err := OpenTrace(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	obs, err := source.Execute(context.Background(), "trace_search", map[string]interface{}{"query": "Order 123"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(obs.Text, "Order 123") || !strings.Contains(obs.Text, "frame-snapshot") {
		t.Fatal("DOM evidence not searchable", obs.Text)
	}
}
