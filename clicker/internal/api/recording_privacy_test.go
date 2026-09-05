package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRecordingCredentialRedaction(t *testing.T) {
	var event map[string]interface{}
	raw := `{"type":"resource-snapshot","snapshot":{"request":{"url":"https://user:URL-PASSWORD@example.com/api?token=QUERY-SECRET&item=1","headers":[{"name":"Authorization","value":"AUTH-SECRET"},{"name":"X-API-Key","value":{"type":"string","value":"KEY-SECRET"}},{"name":"Accept","value":"application/json"}],"cookies":[{"name":"session","value":{"type":"string","value":"COOKIE-SECRET"}}],"queryString":[{"name":"token","value":"QUERY-SECRET"}]},"response":{"headers":[{"name":"Set-Cookie","value":"RESPONSE-SECRET"}]},"html":["INPUT",{"type":"password","value":"DOM-SECRET","__playwright_value_":"LIVE-SECRET"}],"password":"PARAM-SECRET"}}`
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatal(err)
	}
	data, err := marshalRecordingEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"URL-PASSWORD", "QUERY-SECRET", "AUTH-SECRET", "KEY-SECRET", "COOKIE-SECRET", "RESPONSE-SECRET", "DOM-SECRET", "LIVE-SECRET", "PARAM-SECRET"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("credential leaked: %s", secret)
		}
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "application/json") || !strings.Contains(string(data), "item=1") {
		t.Fatal("removed useful evidence")
	}
	original, _ := json.Marshal(event)
	if !strings.Contains(string(original), "AUTH-SECRET") {
		t.Fatal("mutated original event")
	}
}

func TestRecordingRedactsEarlierObservationsAndOmitsSensitiveVisuals(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "PROVIDER-KEY-SENTINEL")
	r := NewRecorder()
	r.Start(RecordingStartOptions{}, nil)
	r.RegisterSecret("PASSWORD-SENTINEL")
	r.mu.Lock()
	r.events = append(r.events,
		recordEvent{"type": "before", "callId": "call@1", "method": "vibium:element.fill", "params": map[string]interface{}{"value": "PASSWORD-SENTINEL"}},
		recordEvent{"type": "after", "callId": "call@1", "result": map[string]interface{}{"status": "passed", "observation": "PASSWORD-SENTINEL and PROVIDER-KEY-SENTINEL and LATE-HEADER-SECRET"}},
	)
	r.network = append(r.network, recordEvent{"type": "resource-snapshot", "snapshot": map[string]interface{}{"request": map[string]interface{}{"headers": []interface{}{map[string]interface{}{"name": "X-API-Key", "value": "LATE-HEADER-SECRET"}}}}})
	r.omitVisuals = true
	r.mu.Unlock()
	r.AddScreenshot([]byte("SECRET-IMAGE"), "page", 1, 1, time.Now())
	r.AddFrameSnapshot("call@1", "after", "page", "https://example.test", "html", []interface{}{"HTML", map[string]interface{}{}, "SECRET-DOM"}, nil, nil)
	data, err := r.Stop()
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range z.File {
		if strings.HasPrefix(file.Name, "resources/") || strings.HasPrefix(file.Name, "video/") {
			t.Fatal("sensitive visual resource exported")
		}
		reader, _ := file.Open()
		contents, _ := io.ReadAll(reader)
		reader.Close()
		for _, secret := range []string{"PASSWORD-SENTINEL", "PROVIDER-KEY-SENTINEL", "LATE-HEADER-SECRET", "SECRET-IMAGE", "SECRET-DOM"} {
			if bytes.Contains(contents, []byte(secret)) {
				t.Fatalf("%s leaked %s", file.Name, secret)
			}
		}
		if file.Name == "trace.trace" && (!bytes.Contains(contents, []byte("vibiumPrivacy")) || !bytes.Contains(contents, []byte(`"status":"passed"`))) {
			t.Fatal("lost privacy marker or result schema")
		}
	}
}
