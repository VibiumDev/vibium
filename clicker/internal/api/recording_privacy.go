package api

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"time"
)

const redacted = "[REDACTED]"

// Recording events can contain both HAR and optional raw BiDi data. Sanitize
// the serialized copy so neither representation bypasses credential masking,
// and the live browser state and recorder's internal data remain unchanged.
// This is structural redaction, not a detector for secrets in page prose/images.
func marshalRecordingEvent(event map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	redactRecordingValue(copy)
	return marshalEvent(copy)
}

func secretField(name string) bool {
	name = strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(name))
	switch name {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "password", "passwd", "apikey", "xapikey", "accesstoken", "refreshtoken", "idtoken", "token", "clientsecret":
		return true
	}
	return false
}

func redactRecordingValue(value interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		// HAR query/header pairs and BiDi header values share a name/value shape.
		name, _ := v["name"].(string)
		if secretField(name) {
			if _, ok := v["value"]; ok {
				v["value"] = redacted
			}
		}
		for key, child := range v {
			if secretField(key) {
				v[key] = redacted
				continue
			}
			if key == "cookies" {
				if cookies, ok := child.([]interface{}); ok {
					for _, c := range cookies {
						if cookie, ok := c.(map[string]interface{}); ok {
							cookie["value"] = redacted
						}
					}
				}
			}
			if raw, ok := child.(string); ok && (strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://")) {
				if u, err := url.Parse(raw); err == nil {
					changed := u.User != nil
					u.User = nil
					q := u.Query()
					for name := range q {
						if secretField(name) {
							q.Set(name, redacted)
							changed = true
						}
					}
					if changed {
						u.RawQuery = q.Encode()
						v[key] = u.String()
					}
				}
			}
			redactRecordingValue(child)
		}
	case []interface{}:
		// Playwright DOM snapshots represent elements as [tag, attributes, ...].
		if len(v) >= 2 {
			if tag, ok := v[0].(string); ok && strings.EqualFold(tag, "input") {
				if attrs, ok := v[1].(map[string]interface{}); ok {
					kind, _ := attrs["type"].(string)
					auto, _ := attrs["autocomplete"].(string)
					if strings.EqualFold(kind, "password") || auto == "current-password" || auto == "new-password" {
						for _, key := range []string{"value", "__playwright_value_"} {
							if _, ok := attrs[key]; ok {
								attrs[key] = redacted
							}
						}
					}
				}
			}
		}
		for _, child := range v {
			redactRecordingValue(child)
		}
	}
}

// RegisterSecret masks a known credential wherever it occurs in textual trace
// evidence, including observations and earlier actions in the same recording.
func (t *Recorder) RegisterSecret(secret string) {
	if secret == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.secrets == nil {
		t.secrets = map[string]bool{}
	}
	t.secrets[secret] = true
}

func (t *Recorder) redactKnown(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Longest first handles credentials that share prefixes.
		values := make([]string, 0, len(t.secrets))
		for secret := range t.secrets {
			values = append(values, secret)
		}
		sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
		for _, secret := range values {
			if len(secret) >= 4 {
				v = strings.ReplaceAll(v, secret, redacted)
			} else if v == secret {
				v = redacted
			}
		}
		return v
	case map[string]interface{}:
		for k, child := range v {
			// Preserve trace schema and linkage, while redacting arbitrary payloads.
			switch k {
			case "type", "method", "class", "callId", "parentId", "pageId", "frameId", "contextId", "snapshotName", "beforeSnapshot", "afterSnapshot", "status":
				continue
			}
			v[k] = t.redactKnown(child)
		}
	case []interface{}:
		for i, child := range v {
			v[i] = t.redactKnown(child)
		}
	}
	return value
}

func (t *Recorder) discoverSecrets(value interface{}) {
	add := func(v interface{}) {
		if s, ok := v.(string); ok && s != "" {
			t.secrets[s] = true
		}
		if m, ok := v.(map[string]interface{}); ok {
			if s, ok := m["value"].(string); ok && s != "" {
				t.secrets[s] = true
			}
		}
	}
	switch v := value.(type) {
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok && secretField(name) {
			add(v["value"])
		}
		for k, child := range v {
			if secretField(k) {
				add(child)
			}
			if k == "cookies" {
				if cookies, ok := child.([]interface{}); ok {
					for _, cookie := range cookies {
						if c, ok := cookie.(map[string]interface{}); ok {
							add(c["value"])
						}
					}
				}
			}
			if raw, ok := child.(string); ok && (strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")) {
				if u, err := url.Parse(raw); err == nil {
					if u.User != nil {
						if password, ok := u.User.Password(); ok {
							add(password)
						}
					}
					for name, values := range u.Query() {
						if secretField(name) {
							for _, value := range values {
								add(value)
							}
						}
					}
				}
			}
			t.discoverSecrets(child)
		}
	case []interface{}:
		if len(v) >= 2 {
			if attrs, ok := v[1].(map[string]interface{}); ok && (attrs["type"] == "password" || attrs["autocomplete"] == "current-password" || attrs["autocomplete"] == "new-password") {
				add(attrs["value"])
				add(attrs["__playwright_value_"])
			}
		}
		for _, child := range v {
			t.discoverSecrets(child)
		}
	}
}

// CaptureRecordingSecrets uses the existing session for a fixed, bounded DOM
// inspection. It never changes application state. If a sensitive field is
// present, omit visual artifacts for the recording: a pixel/video scrubber
// cannot reliably promise to remove plaintext revealed by the application.
func CaptureRecordingSecrets(s Session, recorder *Recorder, params map[string]interface{}) {
	if recorder == nil || !recorder.IsRecording() {
		return
	}
	context, err := s.GetContextID()
	if err != nil {
		return
	}
	recorder.mu.Lock()
	known := make([]string, 0, len(recorder.secrets))
	for value := range recorder.secrets {
		if len(value) >= 4 {
			known = append(known, value)
		}
	}
	recorder.mu.Unlock()
	script := `(selector, known) => { ` + PierceQueryJS() + `;
 const secret = el => el && (el.type === 'password' || /password/i.test(el.autocomplete || '') || /^(password|passwd|api[-_]?key|access[-_]?token|refresh[-_]?token|client[-_]?secret)$/i.test(el.name || el.id || ''));
 const values = []; let sensitive = false; let visited = 0;
 const visit = root => { for (const el of root.querySelectorAll('*')) { if (++visited > 20000) break; if (secret(el)) { sensitive = true; if (el.value) values.push(el.value); } if (el.shadowRoot) visit(el.shadowRoot); if (el.tagName === 'IFRAME') { try { if (el.contentDocument) visit(el.contentDocument); else sensitive = true; } catch { sensitive = true; } } } };
 visit(document);
 const visible = (document.body?.innerText || '').slice(0,200000); if (known.some(value => visible.includes(value))) sensitive = true;
 let target; try { target = selector ? pierceQuery(document, selector) : document.activeElement; } catch {}
 return JSON.stringify({values, sensitive, target: !!secret(target)});
 }`
	selector, _ := params["selector"].(string)
	resp, err := s.SendBidiCommandWithTimeout("script.callFunction", map[string]interface{}{
		"functionDeclaration": script, "target": map[string]interface{}{"context": context}, "awaitPromise": false,
		"arguments": []map[string]interface{}{{"type": "string", "value": selector}, {"type": "array", "value": privacyStringValues(known)}},
	}, 2*time.Second)
	if err != nil {
		return
	}
	text, err := parseScriptResult(resp)
	if err != nil {
		return
	}
	var found struct {
		Values    []string `json:"values"`
		Sensitive bool     `json:"sensitive"`
		Target    bool     `json:"target"`
	}
	if json.Unmarshal([]byte(text), &found) != nil {
		return
	}
	for _, secret := range found.Values {
		recorder.RegisterSecret(secret)
	}
	if found.Target {
		for _, key := range []string{"value", "text", "key"} {
			if value, ok := params[key].(string); ok {
				recorder.RegisterSecret(value)
			}
		}
	}
	if found.Sensitive {
		recorder.mu.Lock()
		recorder.omitVisuals = true
		recorder.videoUnavailable = "Visual evidence omitted: recording encountered sensitive form fields."
		recorder.mu.Unlock()
	}
}

func (t *Recorder) marshalPrivateEvent(event map[string]interface{}) ([]byte, error) {
	data, err := marshalRecordingEvent(event)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	if t.omitVisuals {
		delete(copy, "beforeSnapshot")
		delete(copy, "afterSnapshot")
		if copy["type"] == "context-options" {
			copy["vibiumPrivacy"] = "Visual evidence omitted because sensitive fields were encountered."
		}
	}
	t.redactKnown(copy)
	return marshalEvent(copy)
}

func privacyStringValues(values []string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		out = append(out, map[string]interface{}{"type": "string", "value": value})
	}
	return out
}
