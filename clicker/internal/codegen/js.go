package codegen

import (
	"fmt"
	"sort"
	"strings"
)

type jsRenderer struct{}

func (jsRenderer) header() string {
	return `import { browser } from 'vibium';

const bro = await browser.start();
const vibe = await bro.page();

`
}

func (jsRenderer) footer() string {
	return "\nawait bro.stop();\n"
}

func (jsRenderer) comment(text string) string {
	return "// " + text
}

func (r jsRenderer) statement(c *call) string {
	if c.element != nil {
		return fmt.Sprintf("await %s.%s;\n", r.find(c.element), r.invocation(c))
	}
	return fmt.Sprintf("await %s.%s;\n", r.receiver(c.recv), r.invocation(c))
}

func (r jsRenderer) bindElement(name string, l *locator) string {
	if l.index != 0 {
		return fmt.Sprintf("const %s = %s;\n", name, r.find(l))
	}
	return fmt.Sprintf("const %s = await %s;\n", name, r.find(l))
}

func (r jsRenderer) elementCall(name string, c *call) string {
	return fmt.Sprintf("await %s.%s;\n", name, r.invocation(c))
}

func (r jsRenderer) receiver(recv string) string {
	if recv == "page" {
		return "vibe"
	}
	return "vibe." + recv
}

func (r jsRenderer) invocation(c *call) string {
	args := make([]string, len(c.args))
	for i, a := range c.args {
		args[i] = r.literal(a)
	}
	return fmt.Sprintf("%s(%s)", c.method, strings.Join(args, ", "))
}

// find renders the recorded locator form. A nonzero index replays the nth
// findAll match; that whole expression is already awaited.
func (r jsRenderer) find(l *locator) string {
	if l.index != 0 {
		return fmt.Sprintf("(await vibe.findAll(%s))[%d]", r.locatorArg(l), l.index)
	}
	return fmt.Sprintf("vibe.find(%s)", r.locatorArg(l))
}

func (r jsRenderer) locatorArg(l *locator) string {
	if l.bare() {
		return jsString(l.description)
	}
	var parts []string
	if l.selector != "" {
		parts = append(parts, "css: "+jsString(l.selector))
	}
	if l.description != "" {
		parts = append(parts, "description: "+jsString(l.description))
	}
	if l.image {
		parts = append(parts, "image: "+jsString(imagePlaceholder))
	}
	for _, s := range l.semantic {
		parts = append(parts, fmt.Sprintf("%s: %s", s.key, jsString(s.value)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (r jsRenderer) literal(v interface{}) string {
	switch x := v.(type) {
	case string:
		return jsString(x)
	case float64:
		return formatNumber(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case *locator:
		return "await " + r.find(x)
	case []interface{}:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = r.literal(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		keys := objectKeys(x)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", jsObjectKey(k), r.literal(x[k])))
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	case options:
		parts := make([]string, len(x))
		for i, o := range x {
			parts[i] = fmt.Sprintf("%s: %s", jsObjectKey(o.js), r.literal(o.jsVal))
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	}
	return jsString(fmt.Sprint(v))
}

// objectKeys orders width/height first so setViewport reads naturally, then
// the rest alphabetically for deterministic output.
func objectKeys(m map[string]interface{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		rank := func(k string) int {
			switch k {
			case "width":
				return 0
			case "height":
				return 1
			}
			return 2
		}
		if rank(keys[i]) != rank(keys[j]) {
			return rank(keys[i]) < rank(keys[j])
		}
		return keys[i] < keys[j]
	})
	return keys
}

var jsIdentifier = func(k string) bool {
	if k == "" {
		return false
	}
	for i, c := range k {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_', c == '$':
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func jsObjectKey(k string) string {
	if jsIdentifier(k) {
		return k
	}
	return jsString(k)
}

// jsString renders a single-quoted JavaScript string literal. U+2028 and
// U+2029 are line terminators inside JS string literals, so they are escaped
// along with the control characters.
func jsString(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, c := range s {
		switch c {
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\u2028', '\u2029':
			b.WriteString(fmt.Sprintf(`\u%04x`, c))
		default:
			if c < 0x20 {
				b.WriteString(fmt.Sprintf(`\u%04x`, c))
			} else {
				b.WriteRune(c)
			}
		}
	}
	b.WriteByte('\'')
	return b.String()
}
