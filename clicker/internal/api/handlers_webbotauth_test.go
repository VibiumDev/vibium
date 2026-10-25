package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/vibium/clicker/internal/webbotauth"
)

// The RFC 9421 Ed25519 test key, so the router test shares the signer package's
// known vector without exporting internals.
const testKeyD = "n4Ni-HpISpVObnQMW0wOhCKROaIKqKtW_2ZYb2p9KcU"

func testRouterSigner(t *testing.T, directory string) *webbotauth.Signer {
	t.Helper()
	seed, err := base64.RawURLEncoding.DecodeString(testKeyD)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return webbotauth.NewSigner(ed25519.NewKeyFromSeed(seed), directory)
}

// signatureHeaders turns a request URL into the BiDi header entries the router
// appends on continueRequest. It must emit Signature and Signature-Input, and
// Signature-Agent only when a directory is configured.
func TestSignatureHeadersShape(t *testing.T) {
	r := &Router{}
	r.SetWebBotAuthSigner(testRouterSigner(t, ""))

	entries, ok := r.signatureHeaders("https://example.com/path?q=1")
	if !ok {
		t.Fatal("signatureHeaders returned ok=false for a valid URL")
	}
	names := headerNames(entries)
	if !names["Signature"] || !names["Signature-Input"] {
		t.Fatalf("missing core headers, got %v", names)
	}
	if names["Signature-Agent"] {
		t.Errorf("no directory configured, Signature-Agent should be absent: %v", names)
	}

	withDir := &Router{}
	withDir.SetWebBotAuthSigner(testRouterSigner(t, "https://signer.example.test"))
	entries, _ = withDir.signatureHeaders("https://example.com/")
	if !headerNames(entries)["Signature-Agent"] {
		t.Error("directory configured but Signature-Agent header absent")
	}
}

// A URL with no authority cannot be signed; the caller continues it unsigned
// rather than dropping it.
func TestSignatureHeadersRejectsAuthoritylessURL(t *testing.T) {
	r := &Router{}
	r.SetWebBotAuthSigner(testRouterSigner(t, ""))
	if _, ok := r.signatureHeaders("not a url"); ok {
		t.Error("expected ok=false for a URL with no host")
	}
}

// Each header entry is a BiDi network.Header: a name and a typed string value.
func TestBidiHeaderShape(t *testing.T) {
	h := bidiHeader("Signature", "sig1=:abc:")
	if h["name"] != "Signature" {
		t.Errorf("name = %v", h["name"])
	}
	val, ok := h["value"].(map[string]interface{})
	if !ok || val["type"] != "string" || val["value"] != "sig1=:abc:" {
		t.Errorf("value = %v, want a typed string field", h["value"])
	}
}

func headerNames(entries []map[string]interface{}) map[string]bool {
	out := map[string]bool{}
	for _, e := range entries {
		if n, ok := e["name"].(string); ok {
			out[n] = true
		}
	}
	return out
}
