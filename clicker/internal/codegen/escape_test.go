package codegen

import "testing"

func TestJSString(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", `'plain'`},
		{"it's", `'it\'s'`},
		{`back\slash`, `'back\\slash'`},
		{"line\nbreak", `'line\nbreak'`},
		{"tab\there", `'tab\there'`},
		{"return\r", `'return\r'`},
		{"nul\x00", "'nul\\u0000'"},
		{"bell\x07", "'bell\\u0007'"},
		{"sep\u2028end", "'sep\\u2028end'"},
		{"ünïcødé ✓ 日本語", `'ünïcødé ✓ 日本語'`},
		{`<a href="x">`, `'<a href="x">'`},
	}
	for _, c := range cases {
		if got := jsString(c.in); got != c.want {
			t.Errorf("jsString(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestPyString(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", `'plain'`},
		{"it's", `'it\'s'`},
		{`back\slash`, `'back\\slash'`},
		{"line\nbreak", `'line\nbreak'`},
		{"tab\there", `'tab\there'`},
		{"return\r", `'return\r'`},
		{"nul\x00", "'nul\\x00'"},
		{"bell\x07", `'bell\x07'`},
		{"ünïcødé ✓ 日本語", `'ünïcødé ✓ 日本語'`},
		{`"double"`, `'"double"'`},
	}
	for _, c := range cases {
		if got := pyString(c.in); got != c.want {
			t.Errorf("pyString(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestSnakeCase(t *testing.T) {
	cases := map[string]string{
		"click":          "click",
		"selectOption":   "select_option",
		"scrollIntoView": "scroll_into_view",
		"dragTo":         "drag_to",
		"setViewport":    "set_viewport",
		"waitForURL":     "wait_for_url",
		"waitUntil":      "wait_until",
		"longPress":      "long_press",
	}
	for in, want := range cases {
		if got := snakeCase(in); got != want {
			t.Errorf("snakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}
