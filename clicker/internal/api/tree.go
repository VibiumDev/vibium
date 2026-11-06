package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// ErrTreeUnavailable says the target exposes no UI hierarchy right now.
var ErrTreeUnavailable = errors.New("tree capability unavailable: this target exposes no DOM or UI tree")

// nativeTree is a snapshot of a device's UI hierarchy, as Roadie reports it
// over the roadie:tree.get extension command on the same socket Vibium
// drives the device with. Platform-specific attributes ride in Hints.
//
// Proposed contract (to be settled with Roadie): the command takes no
// params and answers {platform, provider, generation, screen: {width,
// height, rotation}, nodes: [{id, parent, role, name, text, bounds: {x, y,
// width, height}, enabled, clickable, hints}]}. Bounds are in the screen
// coordinate space pointer actions use. An unsupported command means the
// tree is unavailable; an empty nodes list is a valid empty tree.
type nativeTree struct {
	Platform   string       `json:"platform"`
	Provider   string       `json:"provider"`
	Generation int          `json:"generation"`
	Screen     nativeScreen `json:"screen"`
	Nodes      []nativeNode `json:"nodes"`
}

type nativeScreen struct {
	Width    int `json:"width"`
	Height   int `json:"height"`
	Rotation int `json:"rotation"`
}

type nativeNode struct {
	ID        string                 `json:"id"`
	Parent    string                 `json:"parent,omitempty"`
	Role      string                 `json:"role"`
	Name      string                 `json:"name"`
	Text      string                 `json:"text"`
	Bounds    BoxInfo                `json:"bounds"`
	Enabled   *bool                  `json:"enabled,omitempty"`
	Clickable *bool                  `json:"clickable,omitempty"`
	Hints     map[string]interface{} `json:"hints,omitempty"`
}

const nativeTreeMethod = "native-tree"

// fetchNativeTree asks the target for its UI hierarchy.
func fetchNativeTree(s Session) (*nativeTree, error) {
	resp, err := s.SendBidiCommand("roadie:tree.get", map[string]interface{}{})
	if err != nil {
		if strings.Contains(err.Error(), "unknown command") || strings.Contains(err.Error(), "unsupported method") {
			return nil, ErrTreeUnavailable
		}
		return nil, fmt.Errorf("%w: %v", ErrTreeUnavailable, err)
	}
	var out struct {
		Result nativeTree `json:"result"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, fmt.Errorf("%w: malformed tree", ErrTreeUnavailable)
	}
	return &out.Result, nil
}

// candidates flattens the tree into the records the description matcher
// reads. Nodes keep their provenance; no HTML tag or selector is invented.
func (t *nativeTree) candidates() []Found {
	var out []Found
	for _, n := range t.Nodes {
		if n.Bounds.Width <= 0 || n.Bounds.Height <= 0 {
			continue
		}
		if n.Enabled != nil && !*n.Enabled && n.Name == "" && n.Text == "" {
			continue
		}
		name := strings.TrimSpace(n.Name)
		if name == "" {
			name = strings.TrimSpace(n.Text)
		}
		role := strings.ToLower(strings.TrimSpace(n.Role))
		out = append(out, Found{
			Label:  visualLabel(name, role, n.Bounds),
			Role:   role,
			Name:   name,
			Text:   strings.TrimSpace(n.Text),
			Box:    n.Bounds,
			Source: SourceTree,
			Method: nativeTreeMethod,
			Details: map[string]interface{}{
				"platform": t.Platform, "provider": t.Provider, "generation": t.Generation, "node": n.ID,
			},
		})
	}
	return out
}

// findInNativeTree resolves a description against a device's UI tree.
func findInNativeTree(s Session, q descQuery, loc Locator, all bool) (FindResult, error) {
	tree, err := fetchNativeTree(s)
	if err != nil {
		return FindResult{}, err
	}
	if q.Visual && loc.Strategy == StrategyTree {
		return FindResult{Method: nativeTreeMethod}, &NotFoundError{Locator: loc, Timeout: loc.Timeout, Hint: "strategy tree only matches names and roles in the UI tree"}
	}
	matches := matchDescription(q, tree.candidates())
	res := FindResult{Method: nativeTreeMethod}
	switch {
	case len(matches) == 1 || (all && len(matches) > 0):
		if all && loc.Limit > 0 && len(matches) > loc.Limit {
			matches = matches[:loc.Limit]
		}
		for i := range matches {
			matches[i].Index = i
		}
		res.Matches = matches
		return res, nil
	case len(matches) > 1:
		return res, &AmbiguousError{Description: loc.Description, Count: len(matches)}
	case all:
		return res, nil
	}
	return res, &NotFoundError{Locator: loc, Timeout: loc.Timeout, Hint: fmt.Sprintf("no node in the %s UI tree is named that", tree.Platform)}
}

// verifyNativeRef re-reads the tree before a coordinate action and keeps the
// ref on the node it named, so a stale snapshot never drives a blind tap.
func verifyNativeRef(s Session, e *RefEntry) error {
	tree, err := fetchNativeTree(s)
	if err != nil {
		return &StaleRefError{Ref: e.Ref, Reason: "the UI tree is no longer available"}
	}
	wantX, wantY := e.Box.X+e.Box.Width/2, e.Box.Y+e.Box.Height/2
	best, bestDist := -1, math.Inf(1)
	cands := tree.candidates()
	for i, c := range cands {
		if c.Role != e.Role || normalizeName(c.Name) != normalizeName(e.Name) {
			continue
		}
		d := math.Hypot(c.Box.X+c.Box.Width/2-wantX, c.Box.Y+c.Box.Height/2-wantY)
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return &StaleRefError{Ref: e.Ref, Reason: "the screen changed and the element is gone"}
	}
	c := cands[best]
	e.Box, e.Label, e.Details = c.Box, c.Label, c.Details
	return nil
}
