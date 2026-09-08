package verifier

import (
	"fmt"
	"strings"
)

// Playwright version 8 encodes subtree references as [[snapshotsAgo,nodeIndex]].
// Node indices are post-order, excluding reference nodes. Resolve only within
// the same trace stream/frame; never execute HTML or load referenced resources.
// Format: microsoft/playwright packages/isomorphic/trace/snapshotRenderer.ts.
func (s *TraceSource) snapshotText(target traceEvent) (string, error) {
	targetSnap, _ := target.data["snapshot"].(map[string]interface{})
	var snapshots []interface{}
	for _, e := range s.events {
		if e.stream != target.stream || e.data["type"] != "frame-snapshot" {
			continue
		}
		snap, _ := e.data["snapshot"].(map[string]interface{})
		if snap["frameId"] != targetSnap["frameId"] || snap["pageId"] != targetSnap["pageId"] {
			continue
		}
		snapshots = append(snapshots, snap["html"])
		if e.id == target.id {
			break
		}
	}
	if len(snapshots) == 0 {
		return "", fmt.Errorf("snapshot is unavailable")
	}
	nodes := map[int][]interface{}{}
	var index func(interface{}, int, *[]interface{}) error
	index = func(n interface{}, depth int, out *[]interface{}) error {
		if depth > 256 {
			return fmt.Errorf("snapshot nesting exceeds limit")
		}
		if _, ok := n.(string); ok {
			*out = append(*out, n)
			return nil
		}
		if a, ok := n.([]interface{}); ok && len(a) >= 2 {
			if _, ok := a[0].(string); ok {
				for _, child := range a[2:] {
					if err := index(child, depth+1, out); err != nil {
						return err
					}
				}
				*out = append(*out, n)
			}
		}
		return nil
	}
	var out strings.Builder
	visits := 0
	var visit func(interface{}, int, int) error
	visit = func(n interface{}, snapshot, depth int) error {
		visits++
		if depth > 256 || visits > 100000 || out.Len() > 1024*1024 {
			return fmt.Errorf("snapshot expansion exceeds limit")
		}
		if text, ok := n.(string); ok {
			out.WriteString(text)
			out.WriteByte('\n')
			return nil
		}
		a, ok := n.([]interface{})
		if !ok || len(a) == 0 {
			return fmt.Errorf("malformed DOM snapshot")
		}
		if ref, ok := a[0].([]interface{}); ok {
			if len(ref) != 2 {
				return fmt.Errorf("malformed snapshot reference")
			}
			delta, ok1 := ref[0].(float64)
			node, ok2 := ref[1].(float64)
			if !ok1 || !ok2 || delta < 0 || node < 0 || float64(int(delta)) != delta || float64(int(node)) != node {
				return fmt.Errorf("invalid snapshot reference")
			}
			i := snapshot - int(delta)
			if i < 0 || i >= len(snapshots) {
				return fmt.Errorf("snapshot reference is unavailable in this archive")
			}
			if _, ok := nodes[i]; !ok {
				var list []interface{}
				if err := index(snapshots[i], 0, &list); err != nil {
					return err
				}
				nodes[i] = list
			}
			if int(node) >= len(nodes[i]) {
				return fmt.Errorf("invalid snapshot node reference")
			}
			return visit(nodes[i][int(node)], i, depth+1)
		}
		tag, ok := a[0].(string)
		if !ok || len(a) < 2 {
			return fmt.Errorf("malformed snapshot element")
		}
		tag = strings.ToLower(tag)
		if tag == "script" || tag == "style" || tag == "noscript" {
			return nil
		}
		attrs, _ := a[1].(map[string]interface{})
		secret := strings.EqualFold(stringField(attrs, "type"), "password") || strings.Contains(stringField(attrs, "autocomplete"), "password")
		out.WriteString("<" + tag)
		for _, key := range []string{"id", "role", "aria-label", "aria-checked", "aria-selected", "aria-expanded", "aria-hidden", "hidden", "type", "name", "placeholder", "alt", "title", "href", "value", "__playwright_value_", "__playwright_checked_", "__playwright_selected_"} {
			if secret && (key == "value" || key == "__playwright_value_") {
				continue
			}
			if val, ok := attrs[key]; ok {
				clean := map[string]interface{}{key: val}
				sanitizeTrace(clean)
				out.WriteString(fmt.Sprintf(" %s=%q", key, fmt.Sprint(clean[key])))
			}
		}
		if secret {
			out.WriteString(" [password omitted]")
		}
		out.WriteString(">\n")
		for _, child := range a[2:] {
			if err := visit(child, snapshot, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(snapshots[len(snapshots)-1], len(snapshots)-1, 0); err != nil {
		return "", err
	}
	return out.String(), nil
}
