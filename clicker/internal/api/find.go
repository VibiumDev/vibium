package api

import (
	gocontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/ai"
)

// Found is one established match.
type Found struct {
	Selector string                 `json:"selector,omitempty"`
	Label    string                 `json:"label"`
	Tag      string                 `json:"tag,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Role     string                 `json:"role,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Box      BoxInfo                `json:"box"`
	Index    int                    `json:"index"`
	Source   RefSource              `json:"source"`
	Method   string                 `json:"method,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// Entry converts a match into a ref-table entry.
func (f Found) Entry(res FindResult) RefEntry {
	e := RefEntry{
		Selector: f.Selector, Label: f.Label, Tag: f.Tag, Text: f.Text, Role: f.Role, Name: f.Name,
		Box: f.Box, Source: f.Source, Method: f.Method, Details: f.Details, State: res.State,
	}
	if f.Source == SourceImage {
		e.Image = res.Image
	}
	return e
}

// FindResult is the outcome of one find, however it was resolved.
type FindResult struct {
	Matches  []Found
	Method   string // css, semantic, tree, sift, vlm
	VLM      bool   // a model was called
	Provider string
	Model    string
	Elapsed  time.Duration
	Note     string     // the model's reason for an empty or short list
	State    *PageState // page state visual boxes were measured against
	Image    []byte     // the reference image, for image matches
}

// Diagnostics is the opt-in detail block for --details.
func (r FindResult) Diagnostics() map[string]interface{} {
	d := map[string]interface{}{"method": r.Method, "vlm": r.VLM, "elapsedMs": r.Elapsed.Milliseconds()}
	if r.VLM {
		d["provider"], d["model"] = r.Provider, r.Model
	}
	if r.Note != "" {
		d["note"] = r.Note
	}
	return d
}

// NotFoundError says a locator matched nothing before its timeout. It is a
// miss, never a configuration or processing failure.
type NotFoundError struct {
	Locator Locator
	Timeout time.Duration
	Hint    string
}

func (e *NotFoundError) Error() string {
	msg := fmt.Sprintf("timeout after %s waiting for '%s': element not found", e.Timeout, e.Locator.Describe())
	if e.Hint != "" {
		msg += "; " + e.Hint
	}
	return msg
}

// AmbiguousError says a description names several elements equally well.
type AmbiguousError struct {
	Description string
	Count       int
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%d elements match %q equally; use --all, add a distinguishing word, or use a css or role locator", e.Count, e.Description)
}

// FindElements resolves a locator on the page. all=false returns at most one
// match and reports a miss as NotFoundError; all=true returns every match and
// reports a miss as an empty list.
func FindElements(s Session, context string, loc Locator, all bool) (FindResult, error) {
	start := time.Now()
	var res FindResult
	var err error
	switch loc.Kind() {
	case "image":
		res, err = findByImage(s, context, loc, all)
	case "description":
		res, err = findByDescription(s, context, loc, all)
	default:
		res, err = findStructured(s, context, loc, all)
	}
	res.Elapsed = time.Since(start)
	return res, err
}

func locateArgs(loc Locator, all bool) []map[string]interface{} {
	str := func(v string) map[string]interface{} { return map[string]interface{}{"type": "string", "value": v} }
	return []map[string]interface{}{
		str(loc.Scope), str(loc.Selector), str(loc.Tag), str(loc.Name), str(loc.Role), str(loc.Text), str(loc.Label),
		str(loc.Placeholder), str(loc.Alt), str(loc.Title), str(loc.Testid), str(loc.Xpath),
		str(loc.HasText), str(loc.Has),
		{"type": "boolean", "value": all},
		{"type": "number", "value": loc.Limit},
	}
}

// runFoundScript runs a script that returns a JSON array of element records.
func runFoundScript(s Session, context, script string, args []map[string]interface{}, source RefSource, method string) ([]Found, error) {
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		if commandCanceled(err) {
			return nil, err
		}
		return nil, nil
	}
	if invErr := checkInvalidSelector(resp); invErr != nil {
		return nil, invErr
	}
	var result struct {
		Result struct {
			Result struct {
				Type  string `json:"type"`
				Value string `json:"value,omitempty"`
			} `json:"result"`
		} `json:"result"`
	}
	if json.Unmarshal(resp, &result) != nil || result.Result.Result.Type != "string" || result.Result.Result.Value == "" {
		return nil, nil
	}
	var found []Found
	if json.Unmarshal([]byte(result.Result.Result.Value), &found) != nil {
		return nil, nil
	}
	for i := range found {
		found[i].Source, found[i].Method = source, method
	}
	return found, nil
}

func findStructured(s Session, context string, loc Locator, all bool) (FindResult, error) {
	method := "css"
	if loc.Semantic() {
		method = "semantic"
	}
	args := locateArgs(loc, all)
	deadline := time.Now().Add(loc.Timeout)
	for {
		found, err := runFoundScript(s, context, locateScript(), args, SourceDOM, method)
		if err != nil {
			return FindResult{}, err
		}
		if len(found) > 0 {
			return FindResult{Matches: found, Method: method}, nil
		}
		if time.Now().After(deadline) {
			if all {
				return FindResult{Method: method}, nil
			}
			return FindResult{}, &NotFoundError{Locator: loc, Timeout: loc.Timeout}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// localWindow bounds how long a deterministic pass keeps polling before the
// search moves on to a model: long enough for a page to settle, short enough
// that a miss does not sit through the whole timeout first.
const localWindow = 2 * time.Second

func findByDescription(s Session, context string, loc Locator, all bool) (FindResult, error) {
	// A bare string that reads like a selector is almost always an old
	// find("h1"). It is refused here, before any wait and before any model
	// call, so a script written for the CSS meaning fails fast and cheaply.
	// An explicit description option is taken at its word.
	if loc.Bare && looksLikeCSS(loc.Description) {
		return FindResult{}, &SelectorAsDescriptionError{Description: loc.Description}
	}
	q := parseDescription(loc.Description)
	cfg, cfgErr := ai.ResolveConfig("find", loc.AI)
	if loc.Strategy == StrategyVLM {
		if cfgErr != nil {
			return FindResult{}, fmt.Errorf("strategy vlm needs a configured vision model: %w", cfgErr)
		}
		return groundDescription(s, context, loc, cfg, all)
	}
	if !pageHasDOM(s, context) {
		// A device: its native UI tree when it has one, else the screenshot.
		res, err := findInNativeTree(s, q, loc, all)
		if err == nil || loc.Strategy == StrategyTree || !errors.Is(err, ErrTreeUnavailable) && cfgErr != nil {
			return res, err
		}
		if cfgErr != nil {
			return FindResult{}, fmt.Errorf("finding %q on this target needs a vision model: %w (%v)", loc.Description, cfgErr, err)
		}
		return groundDescription(s, context, loc, cfg, all)
	}

	deadline := time.Now().Add(loc.Timeout)
	window := deadline
	if loc.Strategy == StrategyAuto && cfgErr == nil && time.Until(deadline) > localWindow {
		window = time.Now().Add(localWindow)
	}
	ambiguous := 0
	if !q.Visual || loc.Strategy == StrategyTree {
		for {
			cands, err := runFoundScript(s, context, candidatesScript(), []map[string]interface{}{
				{"type": "string", "value": loc.Scope}, {"type": "number", "value": 500},
			}, SourceTree, "tree")
			if err != nil {
				return FindResult{}, err
			}
			matches := matchDescription(q, cands)
			if len(matches) == 1 || (all && len(matches) > 0) {
				if !all {
					scrollIntoView(s, context, matches[0].Selector)
				}
				if all && loc.Limit > 0 && len(matches) > loc.Limit {
					matches = matches[:loc.Limit]
				}
				for i := range matches {
					matches[i].Index = i
				}
				return FindResult{Matches: matches, Method: "tree"}, nil
			}
			if len(matches) > 1 {
				ambiguous = len(matches)
				break
			}
			if time.Now().After(window) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	if loc.Strategy == StrategyAuto && cfgErr == nil {
		return groundDescription(s, context, loc, cfg, all)
	}
	if ambiguous > 1 {
		return FindResult{}, &AmbiguousError{Description: loc.Description, Count: ambiguous}
	}
	if all {
		return FindResult{Method: "tree"}, nil
	}
	return FindResult{}, &NotFoundError{Locator: loc, Timeout: loc.Timeout, Hint: descriptionHint(loc, q, cfgErr)}
}

// descriptionHint explains a miss: what a model would have added, and the
// exact css form when the description reads like a selector.
func descriptionHint(loc Locator, q descQuery, cfgErr error) string {
	var parts []string
	if looksLikeCSS(loc.Description) {
		d := loc.Description
		parts = append(parts, fmt.Sprintf("find takes a description; for the CSS selector use `vibium find css %q`, `find({css: %q})` in JavaScript, `find(css=%q)` in Python, or `new SelectorOptions().css(%q)` in Java", d, d, d, d))
	}
	switch {
	case loc.Strategy == StrategyTree:
		parts = append(parts, "strategy tree only matches names and roles in the page tree")
	case q.Visual && cfgErr != nil:
		parts = append(parts, fmt.Sprintf("the description needs a vision model (%v)", cfgErr))
	case cfgErr != nil:
		parts = append(parts, fmt.Sprintf("no element is named that; a vision model would be consulted next, but none is configured (%v)", cfgErr))
	}
	return strings.Join(parts, "; ")
}

// pageHasDOM is false for a target that runs no script but answers a Roadie
// viewport query: a phone screen rather than a web page.
func pageHasDOM(s Session, context string) bool {
	if _, err := EvalSimpleScript(s, context, "() => 'ok'"); err == nil {
		return true
	}
	_, err := s.SendBidiCommand("roadie:screen.getViewport", map[string]interface{}{})
	return err != nil
}

func scrollIntoView(s Session, context, selector string) {
	if selector == "" {
		return
	}
	CallScript(s, context, `(selector) => {
		`+PierceQueryJS()+`
		const el = pierceQuery(document, selector);
		if (!el) return null;
		if (el.scrollIntoViewIfNeeded) el.scrollIntoViewIfNeeded(true);
		else el.scrollIntoView({ block: 'center', inline: 'nearest' });
		return 'ok';
	}`, []map[string]interface{}{{"type": "string", "value": selector}})
}

// groundDescription asks the configured vision model where the description is.
func groundDescription(s Session, context string, loc Locator, cfg ai.Config, all bool) (FindResult, error) {
	shot, err := captureScreen(s, context)
	if err != nil {
		return FindResult{}, err
	}
	req := ai.GroundRequest{Description: loc.Description, Screenshot: shot.data, Width: shot.width, Height: shot.height, All: all, Limit: loc.Limit}
	if loc.Scope != "" {
		if box, ok := scopeBox(s, context, loc.Scope); ok {
			req.Scope = &ai.GroundBox{X: box.X * shot.scaleX, Y: box.Y * shot.scaleY, Width: box.Width * shot.scaleX, Height: box.Height * shot.scaleY}
		}
	}
	return groundMatches(s, context, loc, cfg, req, shot, all)
}

func groundMatches(s Session, context string, loc Locator, cfg ai.Config, req ai.GroundRequest, shot *screenCapture, all bool) (FindResult, error) {
	out, err := (&ai.Model{}).Ground(gocontext.Background(), cfg, req)
	res := FindResult{Method: "vlm", VLM: true, Provider: out.Provider, Model: out.Model, Note: out.Reason, State: shot.state}
	if err != nil {
		return res, err
	}
	for i, b := range out.Boxes {
		box := BoxInfo{X: b.X / shot.scaleX, Y: b.Y / shot.scaleY, Width: b.Width / shot.scaleX, Height: b.Height / shot.scaleY}
		f := Found{Label: visualLabel(b.Label, b.Role, box), Role: b.Role, Name: b.Label, Box: box, Index: i, Source: SourceVLM, Method: "vlm"}
		if loc.Details {
			f.Details = map[string]interface{}{"provider": out.Provider, "model": out.Model, "modelBox": box}
		}
		res.Matches = append(res.Matches, snapToDOM(s, context, f))
	}
	if len(res.Matches) == 0 && !all {
		hint := "the model found no match"
		if out.Reason != "" {
			hint = "the model found no match: " + out.Reason
		}
		return res, &NotFoundError{Locator: loc, Timeout: loc.Timeout, Hint: hint}
	}
	return res, nil
}

// visualLabel names a coordinate-backed match the way DOM labels read.
func visualLabel(name, role string, box BoxInfo) string {
	if role == "" {
		role = "region"
	}
	if name != "" {
		return fmt.Sprintf("[%s] %q", role, name)
	}
	return fmt.Sprintf("[%s] %dx%d at (%d, %d)", role, int(box.Width), int(box.Height), int(box.X), int(box.Y))
}

// snapToDOM replaces a visual match with the DOM element under it when the
// page has one that covers it, so the ref gets native interaction.
func snapToDOM(s Session, context string, f Found) Found {
	found, err := runFoundScript(s, context, snapScript(), []map[string]interface{}{
		{"type": "number", "value": f.Box.X}, {"type": "number", "value": f.Box.Y},
		{"type": "number", "value": f.Box.Width}, {"type": "number", "value": f.Box.Height},
	}, f.Source, f.Method)
	if err != nil || len(found) == 0 {
		return f
	}
	el := found[0]
	if f.Details != nil {
		el.Details = f.Details
		el.Details["matchBox"] = f.Box
	}
	if el.Name == "" {
		el.Name = f.Name
	}
	el.Index = f.Index
	return el
}

// scopeBox reads the viewport box of a scope selector.
func scopeBox(s Session, context, scope string) (BoxInfo, bool) {
	found, err := runFoundScript(s, context, locateScript(), locateArgs(Locator{Selector: scope, Limit: 1}, false), SourceDOM, "css")
	if err != nil || len(found) == 0 {
		return BoxInfo{}, false
	}
	return found[0].Box, true
}
