// apidrift checks docs/reference/api.md, the canonical cross-surface API
// spec, for internal consistency and against what a client actually exports.
//
//	apidrift validate -spec docs/reference/api.md
//	# Parse the spec and fail on malformed rows.
//	# api.md: 187 rows, 16 sections, spec is well-formed
//
//	apidrift check -spec docs/reference/api.md -surface python -actual surface.json
//	# Compare one client column against an extractor dump ("-" reads stdin).
//	# python: in sync with api.md
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vibium/clicker/internal/apidrift"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: apidrift <validate|check> [flags]")
	}
	switch os.Args[1] {
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		spec := fs.String("spec", "", "path to api.md")
		fs.Parse(os.Args[2:])
		rows := load(*spec)
		sections := map[string]bool{}
		for _, r := range rows {
			sections[r.Section] = true
		}
		fmt.Printf("%s: %d rows, %d sections, spec is well-formed\n", *spec, len(rows), len(sections))
	case "check":
		fs := flag.NewFlagSet("check", flag.ExitOnError)
		spec := fs.String("spec", "", "path to api.md")
		surface := fs.String("surface", "", "spec column to check (js, python, java)")
		actualPath := fs.String("actual", "-", "extractor JSON dump, - for stdin")
		fs.Parse(os.Args[2:])
		rows := load(*spec)

		var data []byte
		var err error
		if *actualPath == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(*actualPath)
		}
		if err != nil {
			fail(fmt.Sprintf("reading actual surface: %v", err))
		}
		var actual apidrift.ClientSurface
		if err := json.Unmarshal(data, &actual); err != nil {
			fail(fmt.Sprintf("parsing actual surface: %v", err))
		}

		report, ok := apidrift.Compare(rows, *surface, actual).Report(*surface)
		fmt.Print(report)
		if !ok {
			os.Exit(1)
		}
	default:
		fail(fmt.Sprintf("unknown command %q (want validate or check)", os.Args[1]))
	}
}

func load(specPath string) []apidrift.Row {
	if specPath == "" {
		fail("-spec is required")
	}
	content, err := os.ReadFile(specPath)
	if err != nil {
		fail(fmt.Sprintf("reading spec: %v", err))
	}
	rows, problems := apidrift.Parse(string(content))
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, p)
		}
		fail(fmt.Sprintf("%d spec problems", len(problems)))
	}
	return rows
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "Error: "+msg)
	os.Exit(1)
}
