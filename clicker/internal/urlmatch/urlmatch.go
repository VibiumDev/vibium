// Package urlmatch implements URL glob matching for route interception and
// request/response observers.
//
// The dialect is Playwright's, so patterns written for Playwright behave the
// same here. Before this package the matching lived in each client and had
// drifted: the JS client supported {a,b} groups, the Python client used fnmatch
// (where * crosses "/"), and the Java client matched substrings.
package urlmatch

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// escapedChars are the characters that must be escaped to appear literally in a
// regular expression. Note "?" is here: in this dialect it is a literal, not a
// single-character wildcard.
var escapedChars = map[byte]bool{
	'$': true, '^': true, '+': true, '.': true, '*': true, '(': true, ')': true,
	'|': true, '\\': true, '?': true, '{': true, '}': true, '[': true, ']': true,
}

// Matcher is a compiled pattern.
type Matcher struct {
	pattern string
	re      *regexp.Regexp
}

// Pattern returns the glob the matcher was compiled from.
func (m *Matcher) Match(u string) bool { return m.re.MatchString(u) }

// String returns the original glob.
func (m *Matcher) String() string { return m.pattern }

var cache sync.Map // pattern -> *cacheEntry

type cacheEntry struct {
	m   *Matcher
	err error
}

// Compile builds a Matcher, memoizing the result per pattern.
func Compile(pattern string) (*Matcher, error) {
	if v, ok := cache.Load(pattern); ok {
		e := v.(*cacheEntry)
		return e.m, e.err
	}

	e := &cacheEntry{}
	expr, err := globToRegex(pattern)
	if err != nil {
		e.err = err
	} else if re, cerr := regexp.Compile(expr); cerr != nil {
		e.err = fmt.Errorf("invalid glob pattern %q: %w", pattern, cerr)
	} else {
		e.m = &Matcher{pattern: pattern, re: re}
	}

	cache.Store(pattern, e)
	return e.m, e.err
}

// Match reports whether url matches pattern. An empty pattern matches anything.
func Match(pattern, u string) (bool, error) {
	if pattern == "" {
		return true, nil
	}
	m, err := Compile(normalizePattern(pattern))
	if err != nil {
		return false, err
	}
	return m.Match(u), nil
}

// Loose is the predicate for the "wait until the URL contains ..." family,
// which is documented as a substring test. A pattern carrying glob syntax is
// matched as a glob; anything else is a plain substring test, so
// `vibium wait url "/dashboard"` keeps working.
func Loose(pattern, u string) bool {
	if pattern == "" {
		return true
	}
	if !strings.ContainsAny(pattern, "*{\\") {
		return strings.Contains(u, pattern)
	}
	ok, err := Match(pattern, u)
	if err != nil {
		// An unparseable glob falls back to the substring behavior rather than
		// failing a navigation wait.
		return strings.Contains(u, pattern)
	}
	return ok
}

// normalizePattern gives a plain absolute URL the same shape the browser
// reports, so route("http://example.com") matches "http://example.com/".
// Patterns containing glob syntax are left alone.
func normalizePattern(pattern string) string {
	if strings.ContainsAny(pattern, "*{}\\") {
		return pattern
	}
	u, err := url.Parse(pattern)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return pattern
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

// globToRegex is a transliteration of Playwright's globToRegexPattern
// (reference/playwright/packages/isomorphic/urlMatch.ts).
func globToRegex(glob string) (string, error) {
	var b strings.Builder
	b.WriteByte('^')

	inGroup := false
	for i := 0; i < len(glob); i++ {
		c := glob[i]

		if c == '\\' && i+1 < len(glob) {
			i++
			writeLiteral(&b, glob[i])
			continue
		}

		if c == '*' {
			charBefore := byte(0)
			if i > 0 {
				charBefore = glob[i-1]
			}
			starCount := 1
			for i+1 < len(glob) && glob[i+1] == '*' {
				starCount++
				i++
			}
			if starCount > 1 {
				charAfter := byte(0)
				if i+1 < len(glob) {
					charAfter = glob[i+1]
				}
				if charAfter == '/' {
					// "/**/" collapses so that https://a/**/b.js also matches https://a/b.js
					if charBefore == '/' {
						b.WriteString("((.+/)|)")
					} else {
						b.WriteString("(.*/)")
					}
					i++
				} else {
					b.WriteString("(.*)")
				}
			} else {
				b.WriteString("([^/]*)")
			}
			continue
		}

		switch c {
		case '{':
			if inGroup {
				return "", fmt.Errorf("invalid glob pattern %q: nested '{' is not supported", glob)
			}
			inGroup = true
			b.WriteByte('(')
		case '}':
			if !inGroup {
				return "", fmt.Errorf("invalid glob pattern %q: unmatched '}'", glob)
			}
			inGroup = false
			b.WriteByte(')')
		case ',':
			if inGroup {
				b.WriteByte('|')
			} else {
				// Playwright emits "\," here, which JS accepts as an identity
				// escape but RE2 rejects. A comma is not a metacharacter.
				b.WriteByte(',')
			}
		default:
			writeLiteral(&b, c)
		}
	}

	if inGroup {
		return "", fmt.Errorf("invalid glob pattern %q: unmatched '{'", glob)
	}

	b.WriteByte('$')
	return b.String(), nil
}

func writeLiteral(b *strings.Builder, c byte) {
	if escapedChars[c] {
		b.WriteByte('\\')
	}
	b.WriteByte(c)
}
