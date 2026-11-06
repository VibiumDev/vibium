package api

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const treeJSON = `{"result":{"platform":"android","provider":"uiautomator","generation":7,"screen":{"width":1080,"height":2400,"rotation":0},"nodes":[
{"id":"n1","role":"button","name":"Add to Cart","bounds":{"x":100,"y":2000,"width":880,"height":120},"clickable":true},
{"id":"n2","role":"button","name":"Buy now","bounds":{"x":100,"y":2140,"width":880,"height":120}},
{"id":"n3","role":"heading","name":"","text":"Widget","bounds":{"x":40,"y":200,"width":600,"height":80}},
{"id":"n4","role":"image","name":"Product photo","bounds":{"x":0,"y":0,"width":0,"height":0}}
]}}`

const noDOM = `{"type":"error","error":"unknown error","message":"script evaluation is not supported"}`
const roadieViewport = `{"result":{"width":1080,"height":2400}}`
const unknownCommand = `{"type":"error","error":"unknown command","message":"unsupported method: roadie:tree.get"}`

func TestNativeTreeCandidates(t *testing.T) {
	f := &scriptedSession{}
	f.step(treeJSON, nil)
	tree, err := fetchNativeTree(f)
	if err != nil {
		t.Fatal(err)
	}
	cands := tree.candidates()
	if len(cands) != 3 {
		t.Fatalf("candidates %d, want 3 (zero-size node dropped): %+v", len(cands), cands)
	}
	if cands[0].Selector != "" || cands[0].Source != SourceTree || cands[0].Method != nativeTreeMethod {
		t.Errorf("provenance: %+v", cands[0])
	}
	if cands[2].Name != "Widget" || cands[2].Role != "heading" {
		t.Errorf("text falls back to name: %+v", cands[2])
	}
	if cands[0].Details["node"] != "n1" || cands[0].Details["generation"] != 7 {
		t.Errorf("details: %v", cands[0].Details)
	}
}

func TestNativeTreeUnavailable(t *testing.T) {
	f := &scriptedSession{}
	f.step(unknownCommand, errors.New("unknown command: unsupported method: roadie:tree.get"))
	if _, err := fetchNativeTree(f); !errors.Is(err, ErrTreeUnavailable) {
		t.Errorf("got %v", err)
	}
}

func TestFindDescriptionOnDevice(t *testing.T) {
	// No script engine, a Roadie viewport, then the tree: a phone screen.
	f := &scriptedSession{}
	f.step(noDOM, errors.New("script evaluation is not supported"))
	f.step(roadieViewport, nil)
	f.step(treeJSON, nil)
	loc := Locator{Description: "the Add to Cart button", Strategy: StrategyTree, Timeout: time.Second, Limit: 10}
	res, err := FindElements(f, "screen", loc, false)
	if err != nil {
		t.Fatal(err)
	}
	m := res.Matches[0]
	if m.Selector != "" || m.Role != "button" || m.Box.X != 100 || m.Box.Y != 2000 {
		t.Errorf("match %+v", m)
	}
	e := m.Entry(res)
	if !e.Visual() || e.Method != nativeTreeMethod {
		t.Errorf("entry %+v", e)
	}

	// Explicit tree on a device without one is a capability error, not a miss.
	f = &scriptedSession{}
	f.step(noDOM, errors.New("script evaluation is not supported"))
	f.step(roadieViewport, nil)
	f.step(unknownCommand, errors.New("unknown command: unsupported method: roadie:tree.get"))
	if _, err := FindElements(f, "screen", loc, false); !errors.Is(err, ErrTreeUnavailable) {
		t.Errorf("tree strategy without a tree: %v", err)
	}

	// Ambiguity is reported, never guessed.
	f = &scriptedSession{}
	f.step(noDOM, errors.New("script evaluation is not supported"))
	f.step(roadieViewport, nil)
	f.step(treeJSON, nil)
	var amb *AmbiguousError
	if _, err := FindElements(f, "screen", Locator{Description: "the button", Strategy: StrategyTree, Timeout: time.Second}, false); !errors.As(err, &amb) || amb.Count != 2 {
		t.Errorf("ambiguous: %v", err)
	}
}

func TestBareSelectorLookingStringIsRefusedBeforeAnyCall(t *testing.T) {
	f := &scriptedSession{}
	for _, d := range []string{"h1", "#login", ".submit", "button.primary", "nav > a"} {
		_, err := FindElements(f, "ctx", Locator{Description: d, Bare: true, Strategy: StrategyAuto, Timeout: time.Second}, false)
		var refused *SelectorAsDescriptionError
		if !errors.As(err, &refused) {
			t.Errorf("%q: got %v", d, err)
		}
		for _, want := range []string{"vibium find css", "find({css:", "find(css=", "SelectorOptions().css(", "find description"} {
			if err != nil && !strings.Contains(err.Error(), want) {
				t.Errorf("%q: message lacks %q", d, want)
			}
		}
	}
	if f.calls != 0 {
		t.Errorf("the session was called %d times; a refused string must cost nothing", f.calls)
	}
}

func TestVerifyNativeRef(t *testing.T) {
	e := &RefEntry{Ref: "@e1", Role: "button", Name: "Add to Cart", Box: BoxInfo{X: 100, Y: 2000, Width: 880, Height: 120}, Source: SourceTree, Method: nativeTreeMethod}
	moved := `{"result":{"platform":"android","nodes":[{"id":"n1","role":"button","name":"Add to Cart","bounds":{"x":100,"y":1800,"width":880,"height":120}}]}}`
	f := &scriptedSession{}
	f.step(moved, nil)
	if err := verifyNativeRef(f, e); err != nil || e.Box.Y != 1800 {
		t.Errorf("moved node not followed: %v %+v", err, e.Box)
	}
	f = &scriptedSession{}
	f.step(`{"result":{"platform":"android","nodes":[]}}`, nil)
	var stale *StaleRefError
	if err := verifyNativeRef(f, e); !errors.As(err, &stale) {
		t.Errorf("gone node: %v", err)
	}
	f = &scriptedSession{}
	f.step(unknownCommand, errors.New("unknown command"))
	if err := verifyNativeRef(f, e); !errors.As(err, &stale) {
		t.Errorf("tree gone: %v", err)
	}
}
