package api

import (
	"encoding/json"
	"net/url"
	"strings"
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
