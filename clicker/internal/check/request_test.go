package check

import (
	"testing"

	"github.com/vibium/clicker/internal/ai"
)

func TestCheckRequestSiteValidation(t *testing.T) {
	config := ai.Config{Provider: "local", Model: "m"}
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
