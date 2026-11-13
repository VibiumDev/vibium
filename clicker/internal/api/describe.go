package api

import (
	"fmt"
	"regexp"
	"strings"
)

// descQuery is a description reduced to what the page tree can answer: a
// name and the roles it may have. Visual marks words the tree cannot judge.
type descQuery struct {
	Raw    string
	Name   string
	Roles  []string
	Visual bool
}

// visualWords are constraints only a picture can confirm.
var visualWords = map[string]bool{
	"red": true, "blue": true, "green": true, "yellow": true, "orange": true, "purple": true, "pink": true,
	"black": true, "white": true, "gray": true, "grey": true, "brown": true, "dark": true, "light": true,
	"colored": true, "coloured": true, "big": true, "small": true, "large": true, "tiny": true, "round": true,
	"square": true, "icon": true, "logo": true, "arrow": true, "star": true, "gear": true, "hamburger": true,
	"top": true, "bottom": true, "left": true, "right": true, "above": true, "below": true, "beside": true,
	"next": true, "near": true, "corner": true, "center": true, "middle": true, "upper": true, "lower": true,
	"leftmost": true, "rightmost": true, "topmost": true, "highlighted": true, "selected": true, "bold": true,
}

// roleWords map a word in a description to the roles it can mean.
var roleWords = map[string][]string{
	"button": {"button"}, "btn": {"button"},
	"link":    {"link"},
	"heading": {"heading"}, "header": {"heading", "banner"}, "headline": {"heading"},
	"textbox": {"textbox", "searchbox"}, "input": {"textbox", "searchbox", "combobox", "spinbutton", "checkbox", "radio", "slider"},
	"field": {"textbox", "searchbox", "combobox", "spinbutton"}, "box": {"textbox", "searchbox", "checkbox", "combobox"},
	"textarea": {"textbox"}, "search": {"searchbox", "textbox"},
	"checkbox": {"checkbox"}, "radio": {"radio"}, "switch": {"switch"}, "toggle": {"switch", "checkbox"},
	"tab": {"tab"}, "slider": {"slider"},
	"image": {"img"}, "img": {"img"}, "picture": {"img"}, "photo": {"img"}, "thumbnail": {"img"},
	"dropdown": {"combobox", "listbox"}, "select": {"combobox", "listbox"}, "menu": {"menu", "menubar"},
	"option": {"option", "menuitem"}, "item": {"listitem", "menuitem", "option"},
}

var fillerPhrases = []string{"that says", "that reads", "with the text", "with text", "labeled", "labelled", "named", "called", "saying", "reading", "titled"}

var nonWord = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
var spaces = regexp.MustCompile(`\s+`)

func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = nonWord.ReplaceAllString(s, " ")
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

func parseDescription(raw string) descQuery {
	q := descQuery{Raw: raw}
	text := strings.ToLower(strings.TrimSpace(raw))
	for _, f := range fillerPhrases {
		text = strings.ReplaceAll(text, " "+f+" ", " ")
	}
	words := strings.Fields(normalizeName(text))
	for len(words) > 0 && (words[0] == "the" || words[0] == "a" || words[0] == "an") {
		words = words[1:]
	}
	var name []string
	for _, w := range words {
		if visualWords[w] {
			q.Visual = true
			continue
		}
		if roles, ok := roleWords[w]; ok {
			for _, r := range roles {
				if !q.roleOK(r) || len(q.Roles) == 0 {
					q.Roles = append(q.Roles, r)
				}
			}
			continue
		}
		name = append(name, w)
	}
	q.Name = strings.Join(name, " ")
	return q
}

func (q descQuery) roleOK(role string) bool {
	if len(q.Roles) == 0 {
		return true
	}
	for _, r := range q.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// matchDescription picks the candidates the tree can vouch for: an exact name
// first, then a single whole-phrase containment, then a lone element of the
// named role. Anything weaker is left to the model.
func matchDescription(q descQuery, cands []Found) []Found {
	var exact, partial, byRole []Found
	for _, c := range cands {
		if !q.roleOK(c.Role) {
			continue
		}
		name := normalizeName(c.Name)
		if q.Name == "" {
			byRole = append(byRole, c)
			continue
		}
		switch {
		case name == q.Name:
			exact = append(exact, c)
		case name != "" && strings.Contains(" "+name+" ", " "+q.Name+" "):
			partial = append(partial, c)
		}
	}
	switch {
	case len(exact) > 0:
		return exact
	case len(partial) > 0:
		return partial
	case q.Name == "" && len(q.Roles) > 0 && !q.Visual:
		// "the red button" names only a role once the color is dropped;
		// the tree cannot vouch for the color, so it does not answer.
		return byRole
	}
	return nil
}

// SelectorAsDescriptionError says a bare find string reads like a CSS
// selector and was not searched for.
type SelectorAsDescriptionError struct {
	Description string
}

func (e *SelectorAsDescriptionError) Error() string {
	d := e.Description
	return fmt.Sprintf("%q reads like a CSS selector, and find takes a description; for the selector use `vibium find css %q`, `find({css: %q})` in JavaScript, `find(css=%q)` in Python, or `new SelectorOptions().css(%q)` in Java; for an element actually named %q use `vibium find description %q` or the description option", d, d, d, d, d, d, d)
}

var cssLike = regexp.MustCompile(`^(//|\.|#|\[)|[>#.\[\]:+~*=]`)

var htmlTags = map[string]bool{
	"a": true, "abbr": true, "article": true, "aside": true, "b": true, "body": true, "button": true, "canvas": true,
	"div": true, "em": true, "fieldset": true, "footer": true, "form": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"h5": true, "h6": true, "header": true, "html": true, "i": true, "iframe": true, "img": true, "input": true, "label": true,
	"li": true, "main": true, "nav": true, "ol": true, "option": true, "p": true, "pre": true, "section": true, "select": true,
	"span": true, "strong": true, "summary": true, "table": true, "tbody": true, "td": true, "textarea": true, "th": true,
	"thead": true, "tr": true, "ul": true, "video": true,
}

// looksLikeCSS reports whether a description reads like a selector that used
// to be the bare-string form of find.
func looksLikeCSS(desc string) bool {
	d := strings.TrimSpace(desc)
	if d == "" || strings.Contains(d, " ") && !cssLike.MatchString(d) {
		return false
	}
	return htmlTags[strings.ToLower(d)] || cssLike.MatchString(d)
}
