package apidrift

import (
	"os"
	"strings"
	"testing"
)

const fixture = "## Page\n" +
	"| # | Description | Wire Command | CLI | MCP | JS | Python | Java |\n" +
	"|---|---|---|---|---|---|---|---|\n" +
	"| 1 | Generate PDF | `vibium:page.pdf` | `vibium pdf` | `browser_pdf` | `page.pdf(opts?)` | `page.pdf(**opts)` | `page.pdf(options?)` |\n" +
	"| 2 | Find all | `vibium:page.findAll` | `vibium find --all <sel>` | `browser_find_all` | `page.findAll(sel)` | `page.find_all(sel)` | `page.findAll(sel)` |\n" +
	"| 3 | Capture a response | `vibium:capture` | — | ⬜ | `page.capture.response(pat)` | `page.capture.response(pat)` | `page.capture().response(pat)` |\n" +
	"| 4 | Request handle | — | — | — | `route.request` (property) | *passed via callback args* | `route.request()` |\n"

func TestParseFixture(t *testing.T) {
	rows, problems := Parse(fixture)
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}

	pdf := rows[0]
	if pdf.Section != "Page" || pdf.Desc != "Generate PDF" {
		t.Fatalf("row 1 parsed wrong: %+v", pdf)
	}
	for surface, want := range map[string]string{
		"wire": "vibium:page.pdf", "cli": "pdf", "mcp": "browser_pdf",
		"js": "page.pdf", "python": "page.pdf", "java": "page.pdf",
	} {
		if got := pdf.Cells[surface].Name; got != want {
			t.Errorf("row 1 %s name = %q, want %q", surface, got, want)
		}
	}

	if got := rows[1].Cells["cli"].Name; got != "find" {
		t.Errorf("cli with flag: got %q, want %q", got, "find")
	}
	if got := rows[2].Cells["python"].Name; got != "page.capture.response" {
		t.Errorf("nested receiver: got %q", got)
	}
	if rows[2].Cells["cli"].Status != NotApplicable || rows[2].Cells["mcp"].Status != Planned {
		t.Errorf("markers parsed wrong: %+v", rows[2].Cells)
	}
	if got := rows[3].Cells["js"].Name; got != "route.request" {
		t.Errorf("property with trailing note: got %q", got)
	}
	if rows[3].Cells["python"].Status != Note {
		t.Errorf("italic note parsed as %v", rows[3].Cells["python"].Status)
	}
}

func TestParseReportsProblems(t *testing.T) {
	bad := "## X\n" +
		"| # | Description | Wire Command | CLI | MCP | JS | Python | Java |\n" +
		"|---|---|---|---|---|---|---|---|\n" +
		"| 1 | Ok | `w` | — | — | `a.b()` | `a.b()` | `a.b()` |\n" +
		"| 1 | Dup num | `w` | — | — | `a.c()` | `a.c()` | `a.c()` |\n" +
		"| x | Bad num | `w` | — | — | `a.d()` | `a.d()` | `a.d()` |\n" +
		"| 4 | Bad cell | plain text | — | — | `a.e()` | `a.e()` | `a.e()` |\n" +
		"| 5 | Short row | `w` | — | — |\n"
	_, problems := Parse(bad)
	for _, want := range []string{"already used", "not a number", "expected", "columns"} {
		found := false
		for _, p := range problems {
			if strings.Contains(p, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no problem mentioning %q in %v", want, problems)
		}
	}
}

func TestReceiver(t *testing.T) {
	for name, want := range map[string][2]string{
		"page.pdf":              {"page", "pdf"},
		"page.capture.response": {"page.capture", "response"},
		"start":                 {"", "start"},
	} {
		recv, method := Receiver(name)
		if recv != want[0] || method != want[1] {
			t.Errorf("Receiver(%q) = %q,%q want %q,%q", name, recv, method, want[0], want[1])
		}
	}
}

// The real spec must always parse clean: this is the doc-rot gate.
func TestParseRealSpec(t *testing.T) {
	content, err := os.ReadFile("../../../docs/reference/api.md")
	if err != nil {
		t.Fatalf("reading api.md: %v", err)
	}
	rows, problems := Parse(string(content))
	if len(problems) != 0 {
		t.Fatalf("api.md has %d problems:\n%s", len(problems), strings.Join(problems, "\n"))
	}
	if len(rows) < 150 {
		t.Fatalf("only %d rows parsed, the spec has ~165+", len(rows))
	}
}

func TestCompare(t *testing.T) {
	rows, _ := Parse(fixture)
	actual := ClientSurface{
		"page":         {"pdf", "screenshot"},
		"page.capture": {"response"},
	}
	f := Compare(rows, "python", actual)

	if len(f.Missing) != 1 || !strings.Contains(f.Missing[0], "find_all") {
		t.Errorf("Missing = %v, want the find_all claim", f.Missing)
	}
	if len(f.Extra) != 1 || f.Extra[0] != "page.screenshot" {
		t.Errorf("Extra = %v, want page.screenshot", f.Extra)
	}
	if len(f.Unmapped) != 0 {
		t.Errorf("Unmapped = %v, want none", f.Unmapped)
	}

	// route.* is claimed by the JS column but the extractor reported no
	// route receiver: that is an extractor gap, not silence.
	f = Compare(rows, "js", actual)
	found := false
	for _, r := range f.Unmapped {
		if r == "route" {
			found = true
		}
	}
	if !found {
		t.Errorf("js Unmapped = %v, want route", f.Unmapped)
	}
}

func TestSymbolFromFluentAndFlat(t *testing.T) {
	row := "## X\n" +
		"| # | Description | Wire Command | CLI | MCP | JS | Python | Java |\n" +
		"|---|---|---|---|---|---|---|---|\n" +
		"| 1 | Capture | — | `vibium diff map` | `browser_diff_map` | — | — | `page.capture().response(pat, action)` |\n"
	rows, problems := Parse(row)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if got := rows[0].Cells["java"].Name; got != "page.capture.response" {
		t.Errorf("fluent java cell: got %q, want page.capture.response", got)
	}
	if got := rows[0].Cells["cli"].Name; got != "diff map" {
		t.Errorf("cli subcommand path: got %q, want %q", got, "diff map")
	}
}

func TestCompareFlatNamespace(t *testing.T) {
	rows, _ := Parse("## X\n" +
		"| # | Description | Wire Command | CLI | MCP | JS | Python | Java |\n" +
		"|---|---|---|---|---|---|---|---|\n" +
		"| 1 | A | — | `vibium daemon start` | `browser_pdf` | — | — | — |\n" +
		"| 2 | B | — | `vibium gone` | `browser_gone` | — | — | — |\n")

	f := Compare(rows, "cli", ClientSurface{"": {"daemon start", "extra thing"}})
	if len(f.Missing) != 1 || !strings.Contains(f.Missing[0], "gone") {
		t.Errorf("cli Missing = %v, want the gone claim", f.Missing)
	}
	if len(f.Extra) != 1 || f.Extra[0] != "extra thing" {
		t.Errorf("cli Extra = %v, want [extra thing] without a leading dot", f.Extra)
	}

	f = Compare(rows, "mcp", ClientSurface{"": {"browser_pdf"}})
	if len(f.Missing) != 1 || !strings.Contains(f.Missing[0], "browser_gone") {
		t.Errorf("mcp Missing = %v, want browser_gone", f.Missing)
	}
}
