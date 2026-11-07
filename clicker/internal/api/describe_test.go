package api

import (
	"reflect"
	"testing"
)

func TestParseDescription(t *testing.T) {
	tests := []struct {
		in     string
		name   string
		roles  []string
		visual bool
	}{
		{"the Add to Cart button", "add to cart", []string{"button"}, false},
		{"Add to Cart", "add to cart", nil, false},
		{"the main heading", "main", []string{"heading"}, false},
		{"the link labeled 'Learn more'", "learn more", []string{"link"}, false},
		{"the search box", "", []string{"searchbox", "textbox", "checkbox", "combobox"}, false},
		{"the red Add to Cart button", "add to cart", []string{"button"}, true},
		{"the settings icon in the top right", "settings in the", nil, true},
		{"h1", "h1", nil, false},
	}
	for _, tt := range tests {
		q := parseDescription(tt.in)
		if q.Name != tt.name || q.Visual != tt.visual {
			t.Errorf("%q: got name %q visual %v, want %q %v", tt.in, q.Name, q.Visual, tt.name, tt.visual)
		}
		if tt.roles == nil && len(q.Roles) != 0 || tt.roles != nil && !reflect.DeepEqual(q.Roles, tt.roles) {
			t.Errorf("%q: roles %v, want %v", tt.in, q.Roles, tt.roles)
		}
	}
}

func cand(sel, role, name string) Found {
	return Found{Selector: sel, Role: role, Name: name, Label: "[" + role + "] " + name}
}

func TestMatchDescription(t *testing.T) {
	cands := []Found{
		cand("#add", "button", "Add to Cart"),
		cand("#buy", "button", "Buy now"),
		cand("#title", "heading", "Add to Cart later?"),
		cand("#more", "link", "Learn more"),
		cand("#q", "searchbox", ""),
		cand("#img", "img", "Add to Cart"),
	}
	sel := func(fs []Found) []string {
		var out []string
		for _, f := range fs {
			out = append(out, f.Selector)
		}
		return out
	}
	tests := []struct {
		desc string
		want []string
	}{
		{"the Add to Cart button", []string{"#add"}},
		{"Add to Cart", []string{"#add", "#img"}},       // exact name, any role: both
		{"the Add to Cart heading", []string{"#title"}}, // containment with role
		{"the Learn more link", []string{"#more"}},
		{"the search box", []string{"#q"}},               // role only
		{"the Checkout button", nil},                     // nothing
		{"the red Add to Cart button", []string{"#add"}}, // the tree can still name it; the caller decides to skip
		{"the red button", nil},                          // role only, with a color the tree cannot judge
	}
	for _, tt := range tests {
		got := sel(matchDescription(parseDescription(tt.desc), cands))
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: got %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestLooksLikeCSS(t *testing.T) {
	yes := []string{"h1", "button.primary", "#login", ".submit", "a[href]", "nav > a", "//div", "input:focus", "div"}
	no := []string{"the Add to Cart button", "Add to Cart", "main heading", "Sign in", "", "h1 heading"}
	for _, s := range yes {
		if !looksLikeCSS(s) {
			t.Errorf("%q should look like CSS", s)
		}
	}
	for _, s := range no {
		if looksLikeCSS(s) {
			t.Errorf("%q should not look like CSS", s)
		}
	}
}
