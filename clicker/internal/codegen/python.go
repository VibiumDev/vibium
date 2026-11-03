package codegen

import (
	"fmt"
	"strings"
	"unicode"
)

type pyRenderer struct{}

func (pyRenderer) header() string {
	return `from vibium import browser

bro = browser.start()
vibe = bro.page()

`
}

func (pyRenderer) footer() string {
	return "\nbro.stop()\n"
}

func (pyRenderer) comment(text string) string {
	return "# " + text
}

func (r pyRenderer) statement(c *call) string {
	if c.element != nil {
		return fmt.Sprintf("%s.%s\n", r.find(c.element), r.invocation(c))
	}
	return fmt.Sprintf("%s.%s\n", r.receiver(c.recv), r.invocation(c))
}

func (r pyRenderer) bindElement(name string, l *locator) string {
	return fmt.Sprintf("%s = %s\n", name, r.find(l))
}

func (r pyRenderer) elementCall(name string, c *call) string {
	return fmt.Sprintf("%s.%s\n", name, r.invocation(c))
}

func (r pyRenderer) receiver(recv string) string {
	if recv == "page" {
		return "vibe"
	}
	return "vibe." + recv
}

func (r pyRenderer) invocation(c *call) string {
	args := make([]string, len(c.args))
	for i, a := range c.args {
		args[i] = r.literal(a)
	}
	return fmt.Sprintf("%s(%s)", snakeCase(c.method), strings.Join(args, ", "))
}

func (r pyRenderer) find(l *locator) string {
	if l.index != 0 {
		return fmt.Sprintf("vibe.find_all(%s)[%d]", r.locatorArg(l), l.index)
	}
	return fmt.Sprintf("vibe.find(%s)", r.locatorArg(l))
}

// locatorArg renders the recorded locator form: a positional CSS selector or
// semantic keyword arguments.
func (r pyRenderer) locatorArg(l *locator) string {
	if len(l.semantic) == 0 {
		return pyString(l.selector)
	}
	parts := make([]string, len(l.semantic))
	for i, s := range l.semantic {
		parts[i] = fmt.Sprintf("%s=%s", s.key, pyString(s.value))
	}
	return strings.Join(parts, ", ")
}

func (r pyRenderer) literal(v interface{}) string {
	switch x := v.(type) {
	case string:
		return pyString(x)
	case float64:
		return formatNumber(x)
	case bool:
		if x {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	case *locator:
		return r.find(x)
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
			parts = append(parts, fmt.Sprintf("%s: %s", pyString(k), r.literal(x[k])))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return pyString(fmt.Sprint(v))
}

// snakeCase converts a client camelCase method name to the Python form:
// selectOption → select_option, waitForURL → wait_for_url.
func snakeCase(name string) string {
	var b strings.Builder
	runes := []rune(name)
	for i, c := range runes {
		if unicode.IsUpper(c) {
			prevLower := i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]))
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || (i > 0 && unicode.IsUpper(runes[i-1]) && nextLower) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(c))
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

// pyString renders a single-quoted Python string literal.
func pyString(s string) string {
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
		default:
			if c < 0x20 {
				b.WriteString(fmt.Sprintf(`\x%02x`, c))
			} else {
				b.WriteRune(c)
			}
		}
	}
	b.WriteByte('\'')
	return b.String()
}
