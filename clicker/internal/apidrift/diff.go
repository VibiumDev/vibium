package apidrift

import (
	"fmt"
	"sort"
	"strings"
)

// ClientSurface is what a client extractor reports: dotted receiver path
// ("page", "page.capture", "el") to the public members it exports.
type ClientSurface map[string][]string

// Findings is the result of one client-vs-spec comparison.
type Findings struct {
	// Missing: the spec claims the symbol is implemented, the client does
	// not export it. These fail the check.
	Missing []string
	// Extra: the client exports it, no spec row names it. Reported so new
	// methods get documented, but they do not fail the check: the doc rot
	// this catches is loud already (the method is visible in the client).
	Extra []string
	// Unmapped: spec receivers the extractor did not report at all,
	// usually a missing entry in the extractor's receiver map.
	Unmapped []string
}

// Compare checks one client surface column against what the client exports.
func Compare(rows []Row, surface string, actual ClientSurface) Findings {
	var f Findings
	claimed := map[string]map[string]bool{}
	unmapped := map[string]bool{}

	for _, row := range rows {
		cell := row.Cells[surface]
		if cell.Status != Implemented || row.Planned {
			continue
		}
		recv, method := Receiver(cell.Name)
		// Undotted symbols (CLI subcommands, MCP tool names) live under the
		// extractor's "" key as one flat namespace. A client column's
		// module-level name has nothing to look up.
		if recv == "" {
			if _, flat := actual[""]; !flat {
				continue
			}
		}
		members, ok := actual[recv]
		if !ok {
			unmapped[recv] = true
			continue
		}
		if claimed[recv] == nil {
			claimed[recv] = map[string]bool{}
		}
		claimed[recv][method] = true
		if !contains(members, method) {
			f.Missing = append(f.Missing, fmt.Sprintf("row %d (%s): %s claims %s, surface has no %s",
				row.Num, row.Desc, surface, cell.Name, strings.TrimPrefix(recv+"."+method, ".")))
		}
	}

	for recv, members := range actual {
		for _, m := range members {
			if claimed[recv] != nil && claimed[recv][m] {
				continue
			}
			f.Extra = append(f.Extra, strings.TrimPrefix(recv+"."+m, "."))
		}
	}
	for recv := range unmapped {
		f.Unmapped = append(f.Unmapped, recv)
	}

	sort.Strings(f.Missing)
	sort.Strings(f.Extra)
	sort.Strings(f.Unmapped)
	return f
}

// Report renders findings for humans. ok reports whether the check passes.
func (f Findings) Report(surface string) (string, bool) {
	var b strings.Builder
	ok := true
	if len(f.Missing) > 0 {
		ok = false
		fmt.Fprintf(&b, "%s: %d symbols the spec claims but the client does not export:\n", surface, len(f.Missing))
		for _, m := range f.Missing {
			fmt.Fprintf(&b, "  %s\n", m)
		}
	}
	if len(f.Unmapped) > 0 {
		ok = false
		fmt.Fprintf(&b, "%s: %d spec receivers the extractor did not report (extend its receiver map):\n", surface, len(f.Unmapped))
		for _, r := range f.Unmapped {
			fmt.Fprintf(&b, "  %s\n", r)
		}
	}
	if len(f.Extra) > 0 {
		fmt.Fprintf(&b, "%s: %d exported members with no api.md row (document or alias them):\n", surface, len(f.Extra))
		for _, e := range f.Extra {
			fmt.Fprintf(&b, "  %s\n", e)
		}
	}
	if ok && len(f.Extra) == 0 {
		fmt.Fprintf(&b, "%s: in sync with api.md\n", surface)
	}
	return b.String(), ok
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
