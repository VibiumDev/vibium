package api

import (
	"errors"
	"strings"
	"testing"
)

func TestRefTableAllocation(t *testing.T) {
	var tables RefTables
	tab := tables.For("ctx-1")
	a := tab.Register(RefEntry{Selector: "#a", Label: "[a] \"A\"", Source: SourceDOM})
	b := tab.Register(RefEntry{Selector: "#b", Label: "[b]", Source: SourceDOM})
	if a.Ref != "@e1" || b.Ref != "@e2" {
		t.Fatalf("refs %s %s, want @e1 @e2", a.Ref, b.Ref)
	}
	// A second find of the same element keeps its ref and refreshes the label.
	again := tab.Register(RefEntry{Selector: "#a", Label: "[a] \"A2\"", Source: SourceTree})
	if again.Ref != "@e1" || again.Label != "[a] \"A2\"" || again.Source != SourceTree {
		t.Errorf("re-registering #a: %+v", again)
	}
	if tab.Len() != 2 {
		t.Errorf("len %d, want 2", tab.Len())
	}
	// Visual regions are identified by source and box.
	v := tab.Register(RefEntry{Box: BoxInfo{X: 10, Y: 20, Width: 30, Height: 40}, Source: SourceImage, Label: "[image]"})
	v2 := tab.Register(RefEntry{Box: BoxInfo{X: 10.2, Y: 19.8, Width: 30, Height: 40}, Source: SourceImage, Label: "[image]"})
	v3 := tab.Register(RefEntry{Box: BoxInfo{X: 10, Y: 20, Width: 30, Height: 40}, Source: SourceVLM, Label: "[region]"})
	if v.Ref != "@e3" || v2.Ref != "@e3" || v3.Ref != "@e4" {
		t.Errorf("visual refs %s %s %s, want @e3 @e3 @e4", v.Ref, v2.Ref, v3.Ref)
	}
	if !v.Visual() || a.Visual() {
		t.Error("Visual() should reflect the selector")
	}
	// Numbering continues after a map registers the same selectors.
	c := tab.Register(RefEntry{Selector: "#c", Source: SourceDOM})
	if c.Ref != "@e5" {
		t.Errorf("next ref %s, want @e5", c.Ref)
	}
	if e, ok := tab.Lookup("@e2"); !ok || e.Selector != "#b" {
		t.Errorf("lookup @e2: %v %v", e, ok)
	}
	if _, ok := tab.Lookup("@e9"); ok {
		t.Error("unknown ref found")
	}
	// Contexts are separate tables.
	other := tables.For("ctx-2").Register(RefEntry{Selector: "#z", Source: SourceDOM})
	if other.Ref != "@e1" {
		t.Errorf("other context ref %s, want @e1", other.Ref)
	}
	tables.Reset()
	if tables.For("ctx-1").Len() != 0 {
		t.Error("reset did not clear")
	}
}

func TestResolveRefParams(t *testing.T) {
	var tab RefTable
	dom := tab.Register(RefEntry{Selector: "#go", Source: SourceDOM})
	vis := tab.Register(RefEntry{Box: BoxInfo{X: 1, Y: 2, Width: 3, Height: 4}, Source: SourceImage})

	params := map[string]interface{}{"selector": dom.Ref}
	if err := ResolveRefParams(&tab, "vibium:element.fill", params); err != nil || params["selector"] != "#go" {
		t.Errorf("dom ref: %v %v", err, params)
	}

	params = map[string]interface{}{"selector": vis.Ref}
	if err := ResolveRefParams(&tab, "vibium:element.click", params); err != nil || params["visualRef"] != vis || params["selector"] != vis.Ref {
		t.Errorf("visual click: %v %v", err, params)
	}
	ep := ExtractElementParams(params)
	if ep.Visual != vis {
		t.Error("ExtractElementParams did not carry the visual entry")
	}

	params = map[string]interface{}{"selector": vis.Ref}
	err := ResolveRefParams(&tab, "vibium:element.fill", params)
	if !errors.Is(err, ErrVisualUnsupported) {
		t.Errorf("visual fill: %v", err)
	}
	params = map[string]interface{}{"source": vis.Ref, "target": "#x"}
	if err := ResolveRefParams(&tab, "vibium:element.dragTo", params); !errors.Is(err, ErrVisualUnsupported) {
		t.Errorf("visual drag source: %v", err)
	}
	params = map[string]interface{}{"selector": "@e42"}
	if err := ResolveRefParams(&tab, "browser_click", params); err == nil || !strings.Contains(err.Error(), "not a mapped element") {
		t.Errorf("unknown ref: %v", err)
	}
	params = map[string]interface{}{"selector": "#plain"}
	if err := ResolveRefParams(&tab, "browser_click", params); err != nil || params["selector"] != "#plain" {
		t.Errorf("plain selector touched: %v %v", err, params)
	}
	if err := ResolveRefParams(nil, "browser_click", map[string]interface{}{"selector": "@e1"}); err != nil {
		t.Errorf("nil table: %v", err)
	}
}

func TestVisualElementInfoNeedsNoPage(t *testing.T) {
	e := &RefEntry{Ref: "@e1", Box: BoxInfo{X: 10, Y: 20, Width: 30, Height: 40}, Label: "[image]"}
	x, y := e.Center()
	if x != 25 || y != 40 {
		t.Errorf("center (%d, %d), want (25, 40)", x, y)
	}
	j := RefJSON(e, true)
	if j["ref"] != "@e1" || j["source"] != RefSource("") || j["details"] == nil {
		t.Errorf("RefJSON: %v", j)
	}
	if _, ok := j["selector"]; ok {
		t.Error("visual entry must not report a selector")
	}
	if FormatRef(e) != "@e1 [image]" {
		t.Errorf("FormatRef: %q", FormatRef(e))
	}
}
