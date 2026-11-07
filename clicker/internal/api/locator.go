package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/ai"
)

// Strategy picks how a locator is resolved.
type Strategy string

const (
	StrategyAuto Strategy = "auto"
	StrategyCV   Strategy = "cv"
	StrategyVLM  Strategy = "vlm"
	StrategyTree Strategy = "tree"
)

var Strategies = []string{string(StrategyAuto), string(StrategyCV), string(StrategyVLM), string(StrategyTree)}

// MaxImageBytes bounds a reference image payload.
const MaxImageBytes = 4 * 1024 * 1024

// Locator is one element query. A bare string is a Description; every other
// field is an explicit locator with fixed meaning. Description and Image stand
// alone; the structured fields compose.
type Locator struct {
	Description string
	Bare        bool // Description came from a bare string, not an explicit description option
	Image       []byte
	Selector    string // CSS (wire "css" or the older "selector")
	Tag         string
	Name        string // accessible-name filter
	Role        string
	Text        string
	Label       string
	Placeholder string
	Alt         string
	Title       string
	Testid      string
	Xpath       string
	Scope       string
	HasText     string
	Has         string
	Strategy    Strategy
	Timeout     time.Duration
	HasTimeout  bool
	Limit       int
	Details     bool
	AI          ai.Overrides
}

// Kind classifies the locator for dispatch.
func (l Locator) Kind() string {
	switch {
	case len(l.Image) > 0:
		return "image"
	case l.Description != "":
		return "description"
	default:
		return "structured"
	}
}

func (l Locator) structured() bool {
	return l.Selector != "" || l.Tag != "" || l.Name != "" || l.Role != "" || l.Text != "" || l.Label != "" ||
		l.Placeholder != "" || l.Alt != "" || l.Title != "" || l.Testid != "" || l.Xpath != ""
}

// Semantic reports whether the structured fields need the semantic matcher
// rather than a plain CSS query.
func (l Locator) Semantic() bool {
	return l.Tag != "" || l.Name != "" || l.Role != "" || l.Text != "" || l.Label != "" ||
		l.Placeholder != "" || l.Alt != "" || l.Title != "" || l.Testid != "" || l.Xpath != ""
}

// Describe renders the locator for messages.
func (l Locator) Describe() string {
	switch l.Kind() {
	case "image":
		return "image"
	case "description":
		return fmt.Sprintf("%q", l.Description)
	}
	var parts []string
	for _, p := range []struct{ k, v string }{
		{"css", l.Selector}, {"tag", l.Tag}, {"name", l.Name}, {"role", l.Role}, {"text", l.Text}, {"label", l.Label},
		{"placeholder", l.Placeholder}, {"alt", l.Alt}, {"title", l.Title}, {"testid", l.Testid}, {"xpath", l.Xpath},
	} {
		if p.v != "" {
			parts = append(parts, p.k+"="+p.v)
		}
	}
	return strings.Join(parts, ", ")
}

// ParseLocator reads a locator from wire or tool params and validates the
// combination against the strategy table. Image arrives base64 encoded, with
// or without a data: prefix.
func ParseLocator(params map[string]interface{}) (Locator, error) {
	str := func(k string) string {
		s, _ := params[k].(string)
		return s
	}
	l := Locator{
		Description: strings.TrimSpace(str("description")),
		Selector:    str("css"),
		Tag:         strings.ToLower(strings.TrimSpace(str("tag"))),
		Name:        str("name"),
		Role:        str("role"),
		Text:        str("text"),
		Label:       str("label"),
		Placeholder: str("placeholder"),
		Alt:         str("alt"),
		Title:       str("title"),
		Testid:      str("testid"),
		Xpath:       str("xpath"),
		Scope:       str("scope"),
		HasText:     str("hasText"),
		Has:         str("has"),
		Timeout:     DefaultTimeout,
		Limit:       10,
	}
	if l.Selector == "" {
		l.Selector = str("selector")
	} else if s := str("selector"); s != "" && s != l.Selector {
		return l, fmt.Errorf("css and selector name the same locator; give one")
	}
	if ms, ok := params["timeout"].(float64); ok {
		l.Timeout = time.Duration(ms) * time.Millisecond
		l.HasTimeout = true
	}
	if n, ok := params["limit"].(float64); ok && n > 0 {
		l.Limit = int(n)
	}
	if d, ok := params["details"].(bool); ok {
		l.Details = d
	}
	if b, ok := params["bare"].(bool); ok {
		l.Bare = b
	}
	for name, target := range map[string]**string{"provider": &l.AI.Provider, "model": &l.AI.Model, "aiBaseURL": &l.AI.BaseURL, "reasoningEffort": &l.AI.ReasoningEffort} {
		if v, ok := params[name]; ok {
			s, ok := v.(string)
			if !ok {
				return l, fmt.Errorf("%s must be a string", name)
			}
			*target = &s
		}
	}

	switch v := params["image"].(type) {
	case nil:
	case string:
		data, err := decodeImageParam(v)
		if err != nil {
			return l, err
		}
		l.Image = data
	default:
		return l, fmt.Errorf("image must be a base64 string")
	}

	strategy := strings.ToLower(strings.TrimSpace(str("strategy")))
	if strategy == "" {
		strategy = string(StrategyAuto)
	}
	switch Strategy(strategy) {
	case StrategyAuto, StrategyCV, StrategyVLM, StrategyTree:
		l.Strategy = Strategy(strategy)
	default:
		return l, fmt.Errorf("strategy must be one of %s, got %q", strings.Join(Strategies, ", "), strategy)
	}

	return l, l.validate()
}

func (l Locator) validate() error {
	hasImage := len(l.Image) > 0
	hasDesc := l.Description != ""
	switch {
	case hasImage && hasDesc:
		return fmt.Errorf("image and description are separate locators; give one")
	case hasImage && l.structured():
		return fmt.Errorf("image is a standalone locator; it does not combine with %s", l.Describe())
	case hasDesc && l.structured():
		return fmt.Errorf("description is a standalone locator; it does not combine with %s", l.Describe())
	case !hasImage && !hasDesc && !l.structured():
		return fmt.Errorf("find needs a description, an image, or a locator (css, tag, name, role, text, label, placeholder, alt, title, testid, xpath)")
	}
	switch l.Strategy {
	case StrategyCV:
		if !hasImage {
			return fmt.Errorf("strategy cv needs an image locator")
		}
	case StrategyVLM:
		if !hasImage && !hasDesc {
			return fmt.Errorf("strategy vlm needs a description or an image; %s is resolved exactly, never by a model", l.Describe())
		}
	case StrategyTree:
		if hasImage {
			return fmt.Errorf("strategy tree does not apply to an image locator")
		}
	}
	return nil
}

func decodeImageParam(v string) ([]byte, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, fmt.Errorf("image is empty")
	}
	if strings.HasPrefix(v, "data:") {
		_, enc, ok := strings.Cut(v, ";base64,")
		if !ok {
			return nil, fmt.Errorf("image data URL must be base64")
		}
		v = enc
	}
	if len(v) > MaxImageBytes*4/3+4 {
		return nil, fmt.Errorf("image exceeds %d MB", MaxImageBytes/1024/1024)
	}
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		if data, err = base64.RawStdEncoding.DecodeString(v); err != nil {
			return nil, fmt.Errorf("image is not valid base64")
		}
	}
	if len(data) > MaxImageBytes {
		return nil, fmt.Errorf("image exceeds %d MB", MaxImageBytes/1024/1024)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(data)); err != nil || (format != "png" && format != "jpeg") {
		return nil, fmt.Errorf("image must be a PNG or JPEG")
	}
	return data, nil
}
