package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

// Languages lists the supported --language values.
var Languages = []string{"javascript", "python"}

// locator carries the exact recorded element-resolution form.
type locator struct {
	selector string
	semantic []kv // recorded semantic keys, in semanticKeys order
	index    int  // nonzero selects the nth findAll match
}

type kv struct {
	key   string
	value string
}

func (l *locator) key() string {
	parts := []string{l.selector, strconv.Itoa(l.index)}
	for _, s := range l.semantic {
		parts = append(parts, s.key+"="+s.value)
	}
	return strings.Join(parts, "\x00")
}

// call is one client-API statement, language-neutral.
type call struct {
	recv    string   // "page", "keyboard", "mouse", "touch"
	element *locator // non-nil replaces recv with a find on the page
	method  string   // client method name in camelCase
	args    []interface{}
}

// emission is a rendered timeline entry: a call or a comment.
type emission struct {
	comment string
	call    *call
}

// semanticKeys is the locator key order used by find across clients.
var semanticKeys = []string{"role", "text", "label", "placeholder", "alt", "title", "testid", "xpath", "near"}

// elementArgs maps replayable element methods to their param names in
// client-argument order. nil-valued optional params are dropped from the end.
var elementArgs = map[string][]string{
	"click": {}, "dblclick": {}, "clear": {}, "unset": {}, "hover": {},
	"focus": {}, "tap": {}, "scrollIntoView": {},
	"fill": {"value"}, "type": {"text"}, "press": {"key"},
	"selectOption": {"value"}, "setFiles": {"files"},
	"dispatchEvent": {"eventType", "eventInit"},
}

// pageArgs maps replayable page methods to client method name and param order.
var pageArgs = map[string]struct {
	method string
	params []string
}{
	"navigate":        {"go", []string{"url"}},
	"reload":          {"reload", nil},
	"back":            {"back", nil},
	"forward":         {"forward", nil},
	"screenshot":      {"screenshot", nil},
	"eval":            {"evaluate", []string{"expression"}},
	"setContent":      {"setContent", []string{"html"}},
	"waitForURL":      {"waitForURL", []string{"pattern"}},
	"waitForLoad":     {"waitForLoad", []string{"state"}},
	"waitForFunction": {"waitForFunction", []string{"fn"}},
	"wait":            {"wait", []string{"ms"}},
	"scroll":          {"scroll", []string{"direction", "amount", "selector"}},
}

var keyboardArgs = map[string][]string{
	"press": {"key"}, "down": {"key"}, "up": {"key"}, "type": {"text"},
}

var mouseArgs = map[string][]string{
	"click": {"x", "y"}, "move": {"x", "y"}, "down": {}, "up": {},
	"wheel": {"deltaX", "deltaY"},
}

var touchArgs = map[string][]string{
	"tap":       {"x", "y"},
	"swipe":     {"x1", "y1", "x2", "y2", "duration"},
	"longPress": {"x", "y", "duration"},
}

// skipMethods are pure queries and visual noise: replaying them changes
// nothing on the page, so the script omits them.
var skipMethods = map[string]bool{
	"element.text": true, "element.innerText": true, "element.html": true,
	"element.value": true, "element.attr": true, "element.bounds": true,
	"element.boundingBox": true, "element.isVisible": true, "element.isHidden": true,
	"element.isEnabled": true, "element.isSet": true, "element.isEditable": true,
	"element.role": true, "element.label": true, "element.screenshot": true,
	"element.find": true, "element.findAll": true, "element.highlight": true,
	"page.title": true, "page.url": true, "page.content": true,
	"page.viewport": true, "page.window": true, "page.find": true,
	"page.findAll": true, "page.a11yTree": true, "page.pdf": true,
	"page.frames": true, "page.frame": true, "page.count": true,
}

var skipPrefixes = []string{"recording.", "run.", "check.", "browser.", "context."}

// Generate renders the recorded steps as a runnable script.
func Generate(steps []Step, language string) (string, error) {
	var r renderer
	switch language {
	case "javascript":
		r = jsRenderer{}
	case "python":
		r = pyRenderer{}
	default:
		return "", fmt.Errorf("unsupported language %q (supported: %s)", language, strings.Join(Languages, ", "))
	}

	var emissions []emission
	for _, s := range steps {
		switch s.Kind {
		case StepGroupStart:
			emissions = append(emissions, emission{comment: "group: " + s.Title})
		case StepGroupEnd:
		case StepAction:
			if s.Error != "" {
				emissions = append(emissions, emission{comment: fmt.Sprintf("skipped failed action: %s (%s)", s.Title, s.Error)})
				continue
			}
			c, skip, note := mapAction(s)
			if note != "" {
				emissions = append(emissions, emission{comment: note})
				continue
			}
			if skip {
				continue
			}
			emissions = append(emissions, emission{call: c})
		}
	}

	var b strings.Builder
	b.WriteString(r.header())
	elements := 0
	for i := 0; i < len(emissions); i++ {
		e := emissions[i]
		if e.comment != "" {
			b.WriteString(r.comment(e.comment) + "\n")
			continue
		}
		// Consecutive actions on the same locator share one element variable.
		run := 1
		if e.call.element != nil {
			for i+run < len(emissions) {
				next := emissions[i+run].call
				if next == nil || next.element == nil || next.element.key() != e.call.element.key() {
					break
				}
				run++
			}
		}
		if run > 1 {
			elements++
			name := fmt.Sprintf("el%d", elements)
			b.WriteString(r.bindElement(name, e.call.element))
			for j := 0; j < run; j++ {
				b.WriteString(r.elementCall(name, emissions[i+j].call))
			}
			i += run - 1
			continue
		}
		b.WriteString(r.statement(e.call))
	}
	b.WriteString(r.footer())
	return b.String(), nil
}

// mapAction classifies one recorded action. It returns the call to emit,
// skip=true for silent omissions, or a comment for unsupported actions.
func mapAction(s Step) (c *call, skip bool, note string) {
	name := strings.TrimPrefix(s.Method, "vibium:")
	if name == s.Method {
		return nil, true, "" // not a vibium command
	}
	if skipMethods[name] {
		return nil, true, ""
	}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(name, p) {
			return nil, true, ""
		}
	}

	group, method, ok := strings.Cut(name, ".")
	if !ok {
		return nil, false, "skipped unsupported action: " + s.Title
	}
	switch group {
	case "element":
		if params, ok := elementArgs[method]; ok {
			return &call{element: locatorFromParams(s.Params), method: method, args: argValues(s.Params, params)}, false, ""
		}
		if method == "set" {
			var args []interface{}
			if v, ok := s.Params["value"].(bool); ok && !v {
				args = []interface{}{false}
			}
			return &call{element: locatorFromParams(s.Params), method: "set", args: args}, false, ""
		}
		if method == "dragTo" {
			target, _ := s.Params["target"].(map[string]interface{})
			return &call{element: locatorFromParams(s.Params), method: "dragTo", args: []interface{}{locatorFromParams(target)}}, false, ""
		}
		if method == "waitFor" {
			return &call{element: locatorFromParams(s.Params), method: "waitUntil", args: argValues(s.Params, []string{"state"})}, false, ""
		}
	case "page":
		if m, ok := pageArgs[method]; ok {
			return &call{recv: "page", method: m.method, args: argValues(s.Params, m.params)}, false, ""
		}
		if method == "setViewport" {
			size := map[string]interface{}{"width": s.Params["width"], "height": s.Params["height"]}
			return &call{recv: "page", method: "setViewport", args: []interface{}{size}}, false, ""
		}
	case "keyboard":
		if params, ok := keyboardArgs[method]; ok {
			return &call{recv: "keyboard", method: method, args: argValues(s.Params, params)}, false, ""
		}
	case "mouse":
		if params, ok := mouseArgs[method]; ok {
			return &call{recv: "mouse", method: method, args: argValues(s.Params, params)}, false, ""
		}
	case "touch":
		if params, ok := touchArgs[method]; ok {
			return &call{recv: "touch", method: method, args: argValues(s.Params, params)}, false, ""
		}
	}
	return nil, false, "skipped unsupported action: " + s.Title
}

// argValues reads params in client-argument order, dropping absent optional
// trailing values.
func argValues(params map[string]interface{}, names []string) []interface{} {
	var args []interface{}
	for _, n := range names {
		v, ok := params[n]
		if !ok || v == nil {
			break
		}
		args = append(args, v)
	}
	return args
}

func locatorFromParams(params map[string]interface{}) *locator {
	l := &locator{}
	if s, ok := params["selector"].(string); ok {
		l.selector = s
	}
	for _, k := range semanticKeys {
		if v, ok := params[k].(string); ok {
			l.semantic = append(l.semantic, kv{key: k, value: v})
		}
	}
	if n, ok := params["index"].(float64); ok {
		l.index = int(n)
	}
	return l
}

// renderer emits language-specific source text.
type renderer interface {
	header() string
	footer() string
	comment(text string) string
	statement(c *call) string
	bindElement(name string, l *locator) string
	elementCall(name string, c *call) string
}

func formatNumber(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
