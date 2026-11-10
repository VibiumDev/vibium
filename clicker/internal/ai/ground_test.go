package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var tinyPNG = []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 32))

func TestValidBoxes(t *testing.T) {
	req := GroundRequest{Width: 1000, Height: 500}
	in := []GroundBox{
		{X: 10, Y: 10, Width: 100, Height: 40, Role: "Button", Label: " Add "},
		{X: 12, Y: 12, Width: 100, Height: 40},    // duplicate of the first
		{X: -5, Y: 10, Width: 50, Height: 20},     // clipped a little, kept
		{X: 990, Y: 10, Width: 100, Height: 40},   // mostly outside, dropped
		{X: 10, Y: 10, Width: 1, Height: 40},      // degenerate
		{X: 0, Y: 0, Width: 1000, Height: 500},    // the whole screen
		{X: 300, Y: 300, Width: 50, Height: 50},   // fine
		{X: 2000, Y: 2000, Width: 50, Height: 50}, // outside
	}
	out := validBoxes(in, req, 10)
	if len(out) != 3 {
		t.Fatalf("kept %d boxes, want 3: %+v", len(out), out)
	}
	// Sorted by position: the clipped box at x=0 comes first.
	if out[0].X != 0 || out[0].Width != 45 {
		t.Errorf("spilling box not clipped: %+v", out[0])
	}
	if out[1].Role != "button" || out[1].Label != "Add" {
		t.Errorf("role/label not normalized: %+v", out[1])
	}
	if got := validBoxes(in, req, 1); len(got) != 1 {
		t.Errorf("limit 1 kept %d", len(got))
	}
	scoped := req
	scoped.Scope = &GroundBox{X: 250, Y: 250, Width: 200, Height: 200}
	if got := validBoxes(in, scoped, 10); len(got) != 1 || got[0].X != 300 {
		t.Errorf("scope kept %+v", got)
	}
}

func TestGroundOpenAI(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"report_matches","arguments":"{\"matches\":[{\"x\":10,\"y\":20,\"width\":30,\"height\":40,\"role\":\"button\",\"label\":\"Add to Cart\"},{\"x\":-1,\"y\":-1,\"width\":1,\"height\":1}],\"reason\":\"\"}"}}]}}]}`)
	}))
	defer srv.Close()
	c := Config{Role: "find", Provider: "openai-compatible", Model: "vision-x", BaseURL: srv.URL, APIKey: "k"}
	res, err := (&Model{}).Ground(context.Background(), c, GroundRequest{Description: "the Add to Cart button", Screenshot: tinyPNG, Reference: tinyPNG, Width: 800, Height: 600})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Boxes) != 1 || res.Boxes[0].X != 10 || res.Boxes[0].Label != "Add to Cart" {
		t.Errorf("boxes %+v", res.Boxes)
	}
	if res.Model != "vision-x" || res.Provider != "openai-compatible" {
		t.Errorf("provenance %+v", res)
	}
	if got["tool_choice"] != nil {
		t.Error("tool_choice must stay off for openai-compatible servers")
	}
	msgs := got["messages"].([]interface{})
	user := msgs[1].(map[string]interface{})["content"].([]interface{})
	images := 0
	for _, p := range user {
		if p.(map[string]interface{})["type"] == "image_url" {
			images++
		}
	}
	if images != 2 {
		t.Errorf("sent %d images, want screenshot and reference", images)
	}
	text := user[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "800 by 600") || !strings.Contains(text, "single element") {
		t.Errorf("prompt: %s", text)
	}
}

func TestGroundErrors(t *testing.T) {
	status := 400
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, `{"error":{"code":"invalid_value","param":"messages[1].content[1]"}}`)
	}))
	defer srv.Close()
	c := Config{Role: "find", Provider: "openai-compatible", Model: "text-only", BaseURL: srv.URL, APIKey: "k"}
	req := GroundRequest{Description: "x", Screenshot: tinyPNG, Width: 10, Height: 10}
	_, err := (&Model{}).Ground(context.Background(), c, req)
	if err == nil || !strings.Contains(err.Error(), "vision-capable") {
		t.Errorf("text-only model: %v", err)
	}
	_, err = (&Model{}).Ground(context.Background(), c, GroundRequest{Description: "x", Screenshot: []byte("GIF89a"), Width: 10, Height: 10})
	if err == nil || !strings.Contains(err.Error(), "PNG or JPEG") {
		t.Errorf("bad image: %v", err)
	}
	_, err = (&Model{}).Ground(context.Background(), Config{Role: "find"}, req)
	if err == nil {
		t.Error("unconfigured provider must fail")
	}
}

func TestResolveConfigFindRole(t *testing.T) {
	t.Setenv("VIBIUM_AI_PROVIDER", "openai")
	t.Setenv("VIBIUM_AI_MODEL", "gpt-vision")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	c, err := ResolveConfig("find", Overrides{})
	if err != nil || c.Role != "find" || c.Model != "gpt-vision" {
		t.Errorf("find role: %v %+v", err, c)
	}
	model := "other"
	c, err = ResolveConfig("find", Overrides{Model: &model})
	if err != nil || c.Model != "other" {
		t.Errorf("override: %v %+v", err, c)
	}
	if _, err := ResolveConfig("ground", Overrides{}); err == nil {
		t.Error("unknown role accepted")
	}
}
