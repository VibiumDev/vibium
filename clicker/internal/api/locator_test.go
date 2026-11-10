package api

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"
)

func pngBase64(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 7), uint8(y * 11), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestParseLocatorForms(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		kind   string
		want   string // Describe()
	}{
		{"description", map[string]interface{}{"description": "the Add to Cart button"}, "description", `"the Add to Cart button"`},
		{"css", map[string]interface{}{"css": "h1"}, "structured", "css=h1"},
		{"legacy selector is css", map[string]interface{}{"selector": "h1"}, "structured", "css=h1"},
		{"tag", map[string]interface{}{"tag": "H1"}, "structured", "tag=h1"},
		{"role and text compose", map[string]interface{}{"role": "button", "text": "Add"}, "structured", "role=button, text=Add"},
		{"role and name compose", map[string]interface{}{"role": "link", "name": "More"}, "structured", "name=More, role=link"},
		{"css scopes a semantic locator", map[string]interface{}{"selector": "nav", "role": "link"}, "structured", "css=nav, role=link"},
		{"image", map[string]interface{}{"image": pngBase64(t, 16, 16)}, "image", "image"},
		{"image data url", map[string]interface{}{"image": "data:image/png;base64," + pngBase64(t, 16, 16)}, "image", "image"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := ParseLocator(tt.params)
			if err != nil {
				t.Fatalf("ParseLocator: %v", err)
			}
			if loc.Kind() != tt.kind || loc.Describe() != tt.want {
				t.Errorf("got kind %s %q, want %s %q", loc.Kind(), loc.Describe(), tt.kind, tt.want)
			}
			if loc.Strategy != StrategyAuto {
				t.Errorf("default strategy = %s, want auto", loc.Strategy)
			}
		})
	}
}

func TestParseLocatorOptions(t *testing.T) {
	loc, err := ParseLocator(map[string]interface{}{
		"description": "x", "strategy": "VLM", "timeout": 1500.0, "limit": 3.0, "details": true,
		"provider": "openai", "model": "gpt-vision", "aiBaseURL": "http://127.0.0.1:1", "reasoningEffort": "low",
	})
	if err != nil {
		t.Fatal(err)
	}
	if loc.Strategy != StrategyVLM || loc.Timeout != 1500*time.Millisecond || !loc.HasTimeout || loc.Limit != 3 || !loc.Details {
		t.Errorf("options not read: %+v", loc)
	}
	if loc.AI.Provider == nil || *loc.AI.Provider != "openai" || loc.AI.Model == nil || *loc.AI.Model != "gpt-vision" ||
		loc.AI.BaseURL == nil || *loc.AI.BaseURL != "http://127.0.0.1:1" || loc.AI.ReasoningEffort == nil || *loc.AI.ReasoningEffort != "low" {
		t.Errorf("AI overrides not read: %+v", loc.AI)
	}
	loc, err = ParseLocator(map[string]interface{}{"description": "h1", "bare": true})
	if err != nil || !loc.Bare {
		t.Errorf("bare flag not read: %v %+v", err, loc)
	}
	loc, err = ParseLocator(map[string]interface{}{"description": "x", "timeout": 0.0})
	if err != nil || loc.Timeout != 0 || !loc.HasTimeout {
		t.Errorf("explicit zero timeout must stay zero: %v %+v", err, loc)
	}
}

func TestParseLocatorRejects(t *testing.T) {
	img := pngBase64(t, 16, 16)
	tests := []struct {
		name   string
		params map[string]interface{}
		want   string
	}{
		{"nothing", map[string]interface{}{}, "find needs a description"},
		{"description with css", map[string]interface{}{"description": "x", "css": "h1"}, "standalone"},
		{"description with role", map[string]interface{}{"description": "x", "role": "button"}, "standalone"},
		{"image with text", map[string]interface{}{"image": img, "text": "x"}, "standalone"},
		{"image with description", map[string]interface{}{"image": img, "description": "x"}, "separate locators"},
		{"cv needs image", map[string]interface{}{"description": "x", "strategy": "cv"}, "strategy cv needs an image"},
		{"cv on css", map[string]interface{}{"css": "h1", "strategy": "cv"}, "strategy cv needs an image"},
		{"vlm on css", map[string]interface{}{"css": "h1", "strategy": "vlm"}, "never by a model"},
		{"vlm on role", map[string]interface{}{"role": "button", "strategy": "vlm"}, "never by a model"},
		{"tree on image", map[string]interface{}{"image": img, "strategy": "tree"}, "does not apply to an image"},
		{"unknown strategy", map[string]interface{}{"css": "h1", "strategy": "ai"}, "strategy must be one of"},
		{"bad image", map[string]interface{}{"image": "bm90IGFuIGltYWdl"}, "PNG or JPEG"},
		{"image not a string", map[string]interface{}{"image": 7.0}, "base64"},
		{"two css spellings", map[string]interface{}{"css": "h1", "selector": "h2"}, "give one"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLocator(tt.params)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestParseLocatorImageLimit(t *testing.T) {
	big := strings.Repeat("A", MaxImageBytes*4/3+8)
	if _, err := ParseLocator(map[string]interface{}{"image": big}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("oversized image accepted: %v", err)
	}
}

func TestLocatorStructuredSemantics(t *testing.T) {
	loc, _ := ParseLocator(map[string]interface{}{"css": "h1"})
	if loc.Semantic() {
		t.Error("css alone is not semantic")
	}
	loc, _ = ParseLocator(map[string]interface{}{"tag": "h1"})
	if !loc.Semantic() {
		t.Error("tag needs the semantic matcher")
	}
}
