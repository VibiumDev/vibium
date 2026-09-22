package main

import "testing"

func TestParseReportFlag(t *testing.T) {
	cases := []struct {
		in, provider, issue string
		err                 bool
	}{
		{"", "", "", false},
		{"linear", "linear", "", false},
		{"linear:ENG-12", "linear", "ENG-12", false},
		{"linear:RAV-1957", "linear", "RAV-1957", false},
		{"linear:ENG", "", "", true},
		{"jira", "", "", true},
	}
	for _, c := range cases {
		p, issue, err := parseReportFlag(c.in)
		if c.err {
			if err == nil {
				t.Fatalf("%q: want error", c.in)
			}
			continue
		}
		if err != nil || p != c.provider || issue != c.issue {
			t.Fatalf("%q: got %q %q %v", c.in, p, issue, err)
		}
	}
}
