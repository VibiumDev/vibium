package api

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// RefSource says what a mapped element is backed by.
type RefSource string

const (
	SourceDOM   RefSource = "dom"   // a DOM element found by css or a structured locator
	SourceTree  RefSource = "tree"  // a DOM element matched from the semantic tree
	SourceImage RefSource = "image" // a region matched to a reference image
	SourceVLM   RefSource = "vlm"   // a region a vision model pointed at
)

// RefEntry is one mapped element. Selector is empty for a visual region, which
// is driven by coordinates. Box is in viewport CSS pixels at find time.
type RefEntry struct {
	Ref      string                 `json:"ref"`
	Selector string                 `json:"selector,omitempty"`
	Label    string                 `json:"label"`
	Tag      string                 `json:"tag,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Role     string                 `json:"role,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Box      BoxInfo                `json:"box"`
	Source   RefSource              `json:"source"`
	Method   string                 `json:"method,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
	State    *PageState             `json:"-"` // page state when a visual region was found
	Image    []byte                 `json:"-"` // reference image, kept for re-detection
	Strategy Strategy               `json:"-"`
	Checked  bool                   `json:"-"` // its element is stored, so a stale ref can be told
}

// Visual reports whether interaction has to go through coordinates.
func (e *RefEntry) Visual() bool { return e.Selector == "" }

// Center is the default interaction point.
func (e *RefEntry) Center() (int, int) {
	return int(math.Round(e.Box.X + e.Box.Width/2)), int(math.Round(e.Box.Y + e.Box.Height/2))
}

// PageState is what a visual match's coordinates were valid against.
type PageState struct {
	URL     string  `json:"url"`
	ScrollX float64 `json:"scrollX"`
	ScrollY float64 `json:"scrollY"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
}

// RefTable allocates @eN references for one browsing context. Numbering only
// grows: a selector seen before keeps its ref, a region at the same box keeps
// its ref, and everything else gets the next number.
type RefTable struct {
	mu      sync.Mutex
	entries []*RefEntry
	byKey   map[string]*RefEntry
}

func (t *RefTable) key(e *RefEntry) string {
	if e.Selector != "" {
		return "sel:" + e.Selector
	}
	return fmt.Sprintf("box:%s:%d,%d,%d,%d", e.Source, int(math.Round(e.Box.X)), int(math.Round(e.Box.Y)), int(math.Round(e.Box.Width)), int(math.Round(e.Box.Height)))
}

// Register stores e and returns the entry that owns its ref. A known key
// refreshes the stored entry's label, box, and provenance in place.
func (t *RefTable) Register(e RefEntry) *RefEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.byKey == nil {
		t.byKey = map[string]*RefEntry{}
	}
	k := t.key(&e)
	if cur, ok := t.byKey[k]; ok {
		ref := cur.Ref
		*cur = e
		cur.Ref = ref
		return cur
	}
	stored := e
	stored.Ref = fmt.Sprintf("@e%d", len(t.entries)+1)
	t.entries = append(t.entries, &stored)
	t.byKey[k] = &stored
	return &stored
}

// Lookup resolves a ref such as "@e3".
func (t *RefTable) Lookup(ref string) (*RefEntry, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, e := range t.entries {
		if e.Ref == ref {
			return e, true
		}
	}
	return nil, false
}

// Entries returns the table in ref order.
func (t *RefTable) Entries() []*RefEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*RefEntry, len(t.entries))
	copy(out, t.entries)
	return out
}

func (t *RefTable) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.entries)
}

// RefTables holds one RefTable per browsing context.
type RefTables struct {
	mu     sync.Mutex
	tables map[string]*RefTable
}

// For returns the table for a context, creating it on first use.
func (r *RefTables) For(context string) *RefTable {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tables == nil {
		r.tables = map[string]*RefTable{}
	}
	t, ok := r.tables[context]
	if !ok {
		t = &RefTable{}
		r.tables[context] = t
	}
	return t
}

// Drop forgets one context's table: its refs named elements of a document
// that a navigation replaced.
func (r *RefTables) Drop(context string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tables, context)
}

// Reset forgets every table.
func (r *RefTables) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tables = nil
}

// IsRef reports whether a selector names a mapped element.
func IsRef(selector string) bool {
	return strings.HasPrefix(selector, "@e") && len(selector) > 2
}
