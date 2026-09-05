package api

import (
	"encoding/json"
	"strings"
	"testing"
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
