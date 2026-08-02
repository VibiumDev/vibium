package api

import (
	"testing"

	"github.com/vibium/clicker/internal/urlmatch"
)

// WaitForURL is documented as a substring test ("Wait until the page URL
// contains a given substring" — agent/schema.go, cmd/clicker/wait_cmd.go,
// README.md), so it uses urlmatch.Loose rather than the anchored glob matcher.
//
// The expectations below are the behavior of the matchesPattern/globMatch pair
// this replaced, captured over every pattern the tests, docs and README use.
// They exist so the substring contract cannot be quietly traded away for
// anchored globs.
func TestWaitForURLPatternContract(t *testing.T) {
	tests := []struct {
		pattern string
		url     string
		want    bool
	}{
		{"**/login", "http://127.0.0.1:8080/login", true},
		{"**/login", "http://127.0.0.1:8080/secure", false},
		{"**/login", "http://127.0.0.1:8080/subpage", false},
		{"**/login", "http://127.0.0.1:8080/dashboard", false},
		{"**/login", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/login", "http://127.0.0.1:8080/example", false},
		{"**/login", "http://127.0.0.1:8080/", false},
		{"**/login", "http://127.0.0.1:8080/checkout/success", false},
		{"**/login", "http://127.0.0.1:8080/login?next=/secure", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/login", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/secure", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/subpage", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/dashboard", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/example", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/checkout/success", false},
		{"**/nonexistent-page-xyz", "http://127.0.0.1:8080/login?next=/secure", false},
		{"**/secure", "http://127.0.0.1:8080/login", false},
		{"**/secure", "http://127.0.0.1:8080/secure", true},
		{"**/secure", "http://127.0.0.1:8080/subpage", false},
		{"**/secure", "http://127.0.0.1:8080/dashboard", false},
		{"**/secure", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/secure", "http://127.0.0.1:8080/example", false},
		{"**/secure", "http://127.0.0.1:8080/", false},
		{"**/secure", "http://127.0.0.1:8080/checkout/success", false},
		{"**/secure", "http://127.0.0.1:8080/login?next=/secure", true},
		{"/login", "http://127.0.0.1:8080/login", true},
		{"/login", "http://127.0.0.1:8080/secure", false},
		{"/login", "http://127.0.0.1:8080/subpage", false},
		{"/login", "http://127.0.0.1:8080/dashboard", false},
		{"/login", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"/login", "http://127.0.0.1:8080/example", false},
		{"/login", "http://127.0.0.1:8080/", false},
		{"/login", "http://127.0.0.1:8080/checkout/success", false},
		{"/login", "http://127.0.0.1:8080/login?next=/secure", true},
		{"**/dashboard", "http://127.0.0.1:8080/login", false},
		{"**/dashboard", "http://127.0.0.1:8080/secure", false},
		{"**/dashboard", "http://127.0.0.1:8080/subpage", false},
		{"**/dashboard", "http://127.0.0.1:8080/dashboard", true},
		{"**/dashboard", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/dashboard", "http://127.0.0.1:8080/example", false},
		{"**/dashboard", "http://127.0.0.1:8080/", false},
		{"**/dashboard", "http://127.0.0.1:8080/checkout/success", false},
		{"**/dashboard", "http://127.0.0.1:8080/login?next=/secure", false},
		{"**/never-going-here", "http://127.0.0.1:8080/login", false},
		{"**/never-going-here", "http://127.0.0.1:8080/secure", false},
		{"**/never-going-here", "http://127.0.0.1:8080/subpage", false},
		{"**/never-going-here", "http://127.0.0.1:8080/dashboard", false},
		{"**/never-going-here", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/never-going-here", "http://127.0.0.1:8080/example", false},
		{"**/never-going-here", "http://127.0.0.1:8080/", false},
		{"**/never-going-here", "http://127.0.0.1:8080/checkout/success", false},
		{"**/never-going-here", "http://127.0.0.1:8080/login?next=/secure", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/login", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/secure", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/subpage", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/dashboard", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/example", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/checkout/success", false},
		{"**/never-matches-xyz", "http://127.0.0.1:8080/login?next=/secure", false},
		{"**/subpage", "http://127.0.0.1:8080/login", false},
		{"**/subpage", "http://127.0.0.1:8080/secure", false},
		{"**/subpage", "http://127.0.0.1:8080/subpage", true},
		{"**/subpage", "http://127.0.0.1:8080/dashboard", false},
		{"**/subpage", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/subpage", "http://127.0.0.1:8080/example", false},
		{"**/subpage", "http://127.0.0.1:8080/", false},
		{"**/subpage", "http://127.0.0.1:8080/checkout/success", false},
		{"**/subpage", "http://127.0.0.1:8080/login?next=/secure", false},
		{"/dashboard", "http://127.0.0.1:8080/login", false},
		{"/dashboard", "http://127.0.0.1:8080/secure", false},
		{"/dashboard", "http://127.0.0.1:8080/subpage", false},
		{"/dashboard", "http://127.0.0.1:8080/dashboard", true},
		{"/dashboard", "http://127.0.0.1:8080/app/dashboard?x=1", true},
		{"/dashboard", "http://127.0.0.1:8080/example", false},
		{"/dashboard", "http://127.0.0.1:8080/", false},
		{"/dashboard", "http://127.0.0.1:8080/checkout/success", false},
		{"/dashboard", "http://127.0.0.1:8080/login?next=/secure", false},
		{"/example", "http://127.0.0.1:8080/login", false},
		{"/example", "http://127.0.0.1:8080/secure", false},
		{"/example", "http://127.0.0.1:8080/subpage", false},
		{"/example", "http://127.0.0.1:8080/dashboard", false},
		{"/example", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"/example", "http://127.0.0.1:8080/example", true},
		{"/example", "http://127.0.0.1:8080/", false},
		{"/example", "http://127.0.0.1:8080/checkout/success", false},
		{"/example", "http://127.0.0.1:8080/login?next=/secure", false},
		{"success", "http://127.0.0.1:8080/login", false},
		{"success", "http://127.0.0.1:8080/secure", false},
		{"success", "http://127.0.0.1:8080/subpage", false},
		{"success", "http://127.0.0.1:8080/dashboard", false},
		{"success", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"success", "http://127.0.0.1:8080/example", false},
		{"success", "http://127.0.0.1:8080/", false},
		{"success", "http://127.0.0.1:8080/checkout/success", true},
		{"success", "http://127.0.0.1:8080/login?next=/secure", false},
		{"**/checkout/*", "http://127.0.0.1:8080/login", false},
		{"**/checkout/*", "http://127.0.0.1:8080/secure", false},
		{"**/checkout/*", "http://127.0.0.1:8080/subpage", false},
		{"**/checkout/*", "http://127.0.0.1:8080/dashboard", false},
		{"**/checkout/*", "http://127.0.0.1:8080/app/dashboard?x=1", false},
		{"**/checkout/*", "http://127.0.0.1:8080/example", false},
		{"**/checkout/*", "http://127.0.0.1:8080/", false},
		{"**/checkout/*", "http://127.0.0.1:8080/checkout/success", true},
		{"**/checkout/*", "http://127.0.0.1:8080/login?next=/secure", false},
	}

	for _, tt := range tests {
		if got := urlmatch.Loose(tt.pattern, tt.url); got != tt.want {
			t.Errorf("Loose(%q, %q) = %v, want %v", tt.pattern, tt.url, got, tt.want)
		}
	}
}
