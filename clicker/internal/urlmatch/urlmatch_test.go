package urlmatch

import (
	"encoding/json"
	"os"
	"testing"
)

type fixtureCase struct {
	Pattern string `json:"pattern"`
	URL     string `json:"url"`
	Match   bool   `json:"match"`
	Error   bool   `json:"error"`
	Why     string `json:"why"`
}

type fixture struct {
	Cases      []fixtureCase `json:"cases"`
	Deviations []fixtureCase `json:"vibiumDeviations"`
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/url-patterns.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if len(f.Cases) == 0 {
		t.Fatal("fixture has no cases")
	}
	return f
}

// The expectations in testdata were produced by running Playwright's own
// globToRegexPattern, so this asserts dialect compatibility rather than
// agreement with a second reading of the algorithm.
func TestMatchesPlaywrightDialect(t *testing.T) {
	f := loadFixture(t)
	for _, c := range f.Cases {
		t.Run(c.Pattern+" vs "+c.URL, func(t *testing.T) {
			got, err := Match(c.Pattern, c.URL)
			if c.Error {
				if err == nil {
					t.Fatalf("Match(%q, %q) = %v, want an error", c.Pattern, c.URL, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%q, %q) error = %v", c.Pattern, c.URL, err)
			}
			if got != c.Match {
				t.Errorf("Match(%q, %q) = %v, want %v", c.Pattern, c.URL, got, c.Match)
			}
		})
	}
}

func TestDocumentedDeviations(t *testing.T) {
	f := loadFixture(t)
	for _, c := range f.Deviations {
		t.Run(c.Pattern+" vs "+c.URL, func(t *testing.T) {
			got, err := Match(c.Pattern, c.URL)
			if err != nil {
				t.Fatalf("Match(%q, %q) error = %v", c.Pattern, c.URL, err)
			}
			if got != c.Match {
				t.Errorf("Match(%q, %q) = %v, want %v — %s", c.Pattern, c.URL, got, c.Match, c.Why)
			}
		})
	}
}

func TestEmptyPatternMatchesEverything(t *testing.T) {
	got, err := Match("", "http://example.com/anything")
	if err != nil {
		t.Fatalf("Match error = %v", err)
	}
	if !got {
		t.Error("an empty pattern should match everything")
	}
}

// Loose backs the documented "wait until the URL contains ..." behavior, which
// several commands and the README rely on.
func TestLoose(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		url     string
		want    bool
	}{
		{"plain substring", "/dashboard", "http://h/app/dashboard?x=1", true},
		{"plain substring absent", "/admin", "http://h/app/dashboard", false},
		{"bare word", "success", "http://h/checkout/success", true},
		{"glob pattern uses glob rules", "**/subpage", "http://h/subpage", true},
		{"glob pattern anchored", "**/subpage", "http://h/subpage/extra", false},
		{"glob with braces", "**/*.{png,jpg}", "http://h/a/b.png", true},
		{"empty matches", "", "http://h/x", true},
		{"unparseable glob falls back to substring", "{unclosed", "http://h/{unclosed/x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Loose(tt.pattern, tt.url); got != tt.want {
				t.Errorf("Loose(%q, %q) = %v, want %v", tt.pattern, tt.url, got, tt.want)
			}
		})
	}
}

func TestCompileErrors(t *testing.T) {
	for _, pattern := range []string{"{a,{b,c}}", "}bad", "{unclosed"} {
		if _, err := Compile(pattern); err == nil {
			t.Errorf("Compile(%q) = nil error, want a syntax error", pattern)
		}
	}
}

// A comma outside a group must stay literal. Playwright emits "\," here, which
// JavaScript accepts as an identity escape but RE2 rejects outright.
func TestLiteralCommaCompiles(t *testing.T) {
	got, err := Match("a,b", "a,b")
	if err != nil {
		t.Fatalf("Match error = %v", err)
	}
	if !got {
		t.Error(`Match("a,b", "a,b") = false, want true`)
	}
}

func TestCompileIsCached(t *testing.T) {
	a, err := Compile("**/cached")
	if err != nil {
		t.Fatalf("Compile error = %v", err)
	}
	b, _ := Compile("**/cached")
	if a != b {
		t.Error("Compile should return the memoized matcher for the same pattern")
	}
}
