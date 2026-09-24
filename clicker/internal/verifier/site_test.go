package verifier

import (
	"context"
	"testing"
)

type siteCall struct {
	name string
	args map[string]interface{}
}

type siteFakeExecutor struct {
	url    string
	urlErr error
	calls  []siteCall
}

func (f *siteFakeExecutor) Tools() []Tool { return nil }
func (f *siteFakeExecutor) Execute(_ context.Context, name string, args map[string]interface{}) (Observation, error) {
	f.calls = append(f.calls, siteCall{name, args})
	if name == "browser_get_url" {
		return Observation{Text: f.url}, f.urlErr
	}
	return Observation{Text: "ok"}, nil
}

func TestSiteResolvesRelativeNavigation(t *testing.T) {
	base, err := ParseSiteURL("http://site.example:3000/app/")
	if err != nil {
		t.Fatal(err)
	}
	inner := &siteFakeExecutor{}
	wrapped := WithSite(inner, base)
	original := map[string]interface{}{"url": "/cart"}
	if _, err := wrapped.Execute(context.Background(), "browser_navigate", original); err != nil {
		t.Fatal(err)
	}
	if got := inner.calls[0].args["url"]; got != "http://site.example:3000/cart" {
		t.Fatalf("relative target not resolved: %v", got)
	}
	if original["url"] != "/cart" {
		t.Fatal("caller's argument map was mutated")
	}
	for _, tc := range []struct{ name, url string }{
		{"absolute passes through", "https://payments.example/pay"},
		{"other origins are not locked out", "http://other.example/"},
	} {
		inner.calls = nil
		if _, err := wrapped.Execute(context.Background(), "browser_navigate", map[string]interface{}{"url": tc.url}); err != nil {
			t.Fatal(err)
		}
		if got := inner.calls[0].args["url"]; got != tc.url {
			t.Fatalf("%s: %v", tc.name, got)
		}
	}
	inner.calls = nil
	if _, err := wrapped.Execute(context.Background(), "browser_click", map[string]interface{}{"selector": "#go"}); err != nil {
		t.Fatal(err)
	}
	if inner.calls[0].args["selector"] != "#go" {
		t.Fatal("non-navigation tool arguments changed")
	}
}

func TestOpenSiteKeepsAPageAlreadyOnTheOrigin(t *testing.T) {
	for _, current := range []string{
		"https://shop.example/cart?step=2",
		`{"url":"https://shop.example/checkout"}`, // SDK executor observation shape
		"https://shop.example:443/",               // default port is the same origin
	} {
		base, _ := ParseSiteURL("https://shop.example")
		executor := &siteFakeExecutor{url: current}
		if err := OpenSite(context.Background(), executor, base); err != nil {
			t.Fatal(err)
		}
		for _, call := range executor.calls {
			if call.name == "browser_navigate" {
				t.Fatalf("navigated away from %s", current)
			}
		}
	}
}

func TestOpenSiteNavigatesFromElsewhere(t *testing.T) {
	for _, executor := range []*siteFakeExecutor{
		{url: "about:blank"},
		{url: "https://other.example/"},
		{urlErr: context.DeadlineExceeded}, // unreadable URL counts as elsewhere
	} {
		base, _ := ParseSiteURL("http://localhost:3000")
		if err := OpenSite(context.Background(), executor, base); err != nil {
			t.Fatal(err)
		}
		last := executor.calls[len(executor.calls)-1]
		if last.name != "browser_navigate" || last.args["url"] != "http://localhost:3000" {
			t.Fatalf("expected navigation to the site under test, got %+v", last)
		}
	}
}

func TestParseSiteURLRules(t *testing.T) {
	for _, valid := range []string{"http://localhost:3000", "https://staging.example/app"} {
		if _, err := ParseSiteURL(valid); err != nil {
			t.Fatalf("%s rejected: %v", valid, err)
		}
	}
	for _, invalid := range []string{"ftp://example.com", "/relative", "https://user:secret@example.com", "example.com"} {
		if _, err := ParseSiteURL(invalid); err == nil {
			t.Fatalf("%s accepted", invalid)
		}
	}
}

func TestCheckRequestSiteValidation(t *testing.T) {
	config := Config{Provider: "local", Model: "m"}
	if err := (Request{Claim: "c", Record: "trace.zip", BaseSite: "http://localhost:3000", Config: config}).Validate(); err == nil {
		t.Fatal("record combined with a site under test was accepted")
	}
	if err := (Request{Claim: "c", BaseSite: "not a url", Config: config}).Validate(); err == nil {
		t.Fatal("invalid site URL was accepted")
	}
	if err := (Request{Claim: "c", BaseSite: "http://localhost:3000", Config: config}).Validate(); err != nil {
		t.Fatal(err)
	}
}
