// Package apidrift parses docs/reference/api.md as the canonical description
// of every Vibium surface (wire, CLI, MCP, JS, Python, Java) and compares it
// against what a client actually exports. The doc is the spec on purpose:
// the checker fails when the doc is malformed, so api.md cannot rot, and a
// client cannot silently drift from what the doc promises.
package apidrift

import (
	"fmt"
	"strconv"
	"strings"
)

// Surfaces are the api.md table columns after # and Description, in order.
var Surfaces = []string{"wire", "cli", "mcp", "js", "python", "java"}

// Status classifies one table cell.
type Status int

const (
	// Implemented: a backticked name, e.g. `page.pdf(opts?)`.
	Implemented Status = iota
	// Planned: the ⬜ marker.
	Planned
	// NotApplicable: the — marker.
	NotApplicable
	// Note: italic prose, e.g. *client-side event listener*. Real behavior
	// with no nameable symbol on this surface.
	Note
)

// Cell is one parsed table cell.
type Cell struct {
	Raw    string
	Status Status
	// Name is the parsed symbol for Implemented cells: a dotted path for
	// clients ("page.capture.response"), the command for wire
	// ("vibium:page.pdf"), the subcommand path for cli ("page new"), the
	// tool name for mcp ("browser_pdf"). Empty otherwise.
	Name string
}

// Row is one api.md table row.
type Row struct {
	Section string
	Num     int
	Desc    string
	Cells   map[string]Cell
	// Planned marks rows from a "(Planned)" section: their cells may show
	// intended signatures, which are design, not implementation claims.
	Planned bool
}

// Receiver returns the dotted receiver path and method name of a client
// cell name ("page.capture.response" -> "page.capture", "response").
func Receiver(name string) (string, string) {
	i := strings.LastIndex(name, ".")
	if i < 0 {
		return "", name
	}
	return name[:i], name[i+1:]
}

// Parse reads api.md content into rows. Problems are collected rather than
// aborting at the first, so one run reports every defect in the doc.
func Parse(content string) ([]Row, []string) {
	var rows []Row
	var problems []string
	seen := map[int]string{}
	section := ""

	for lineNo, line := range strings.Split(content, "\n") {
		loc := fmt.Sprintf("api.md:%d", lineNo+1)
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		fields := strings.Split(line, "|")
		// "| a | b |" splits into empty, cells..., empty.
		if len(fields) != len(Surfaces)+4 {
			problems = append(problems, fmt.Sprintf("%s: %d columns, want %d", loc, len(fields)-2, len(Surfaces)+2))
			continue
		}
		cells := fields[1 : len(fields)-1]
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if cells[0] == "#" {
			continue // header
		}

		num, err := strconv.Atoi(cells[0])
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: row number %q is not a number", loc, cells[0]))
			continue
		}
		if prev, dup := seen[num]; dup {
			problems = append(problems, fmt.Sprintf("%s: row number %d already used at %s", loc, num, prev))
		}
		seen[num] = loc

		row := Row{Section: section, Num: num, Desc: cells[1], Cells: map[string]Cell{},
			Planned: strings.Contains(section, "(Planned)")}
		if row.Desc == "" {
			problems = append(problems, fmt.Sprintf("%s: empty description", loc))
		}
		for i, surface := range Surfaces {
			cell, err := parseCell(surface, cells[2+i])
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %s cell: %v", loc, surface, err))
			}
			row.Cells[surface] = cell
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		problems = append(problems, "no table rows found")
	}
	return rows, problems
}

func parseCell(surface, raw string) (Cell, error) {
	cell := Cell{Raw: raw}
	switch {
	case raw == "":
		return cell, fmt.Errorf("empty")
	case raw == "—":
		cell.Status = NotApplicable
		return cell, nil
	case raw == "⬜":
		cell.Status = Planned
		return cell, nil
	case strings.HasPrefix(raw, "*") && strings.HasSuffix(raw, "*"):
		cell.Status = Note
		return cell, nil
	}

	start := strings.Index(raw, "`")
	end := strings.LastIndex(raw, "`")
	if start < 0 || end == start {
		return cell, fmt.Errorf("expected `name`, ⬜, —, or *note*, got %q", raw)
	}
	// Trailing prose after the backticks is allowed: `route.request` (property)
	name := raw[start+1 : end]
	cell.Status = Implemented
	cell.Name = symbolFrom(surface, name)
	if cell.Name == "" {
		return cell, fmt.Errorf("no symbol in %q", raw)
	}
	return cell, nil
}

// symbolFrom reduces a backticked cell to a comparable symbol.
func symbolFrom(surface, name string) string {
	switch surface {
	case "wire", "mcp":
		return name
	case "cli":
		// `vibium find --all <sel>` -> "find": the leading binary name goes,
		// then subcommand words up to the first flag or placeholder.
		fields := strings.Fields(name)
		if len(fields) == 0 || fields[0] != "vibium" {
			return ""
		}
		var words []string
		for _, f := range fields[1:] {
			if strings.HasPrefix(f, "<") || strings.HasPrefix(f, "-") || strings.HasPrefix(f, "[") || strings.HasPrefix(f, "@") {
				break
			}
			words = append(words, f)
		}
		return strings.Join(words, " ")
	default: // js, python, java
		// Java's fluent accessors read page.capture().response(...): the
		// empty parens are part of the path, not the call being named.
		name = strings.ReplaceAll(name, "()", "")
		if i := strings.Index(name, "("); i >= 0 {
			name = name[:i]
		}
		return strings.TrimSpace(name)
	}
}
