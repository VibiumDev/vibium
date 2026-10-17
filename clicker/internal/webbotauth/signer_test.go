package webbotauth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The RFC 9421 Ed25519 test key ("test-key-ed25519"), reused by the web-bot-auth
// drafts. d is the base64url private seed.
const testKeyD = "n4Ni-HpISpVObnQMW0wOhCKROaIKqKtW_2ZYb2p9KcU"

// The keyid the architecture draft prints for that key.
const wantKeyID = "poqkLGiymh_W0uP6PZFw-dvez3QJT5SolqXBCW38r0U"

func testSigner(t *testing.T, directory string) *Signer {
	t.Helper()
	seed, err := base64.RawURLEncoding.DecodeString(testKeyD)
	if err != nil || len(seed) != ed25519.SeedSize {
		t.Fatalf("bad test seed: %v", err)
	}
	return NewSigner(ed25519.NewKeyFromSeed(seed), directory)
}

// keyid derivation must match the published test vector exactly, or a verifier
// looks up the wrong key and rejects every signature.
func TestKeyIDMatchesPublishedVector(t *testing.T) {
	if got := testSigner(t, "").KeyID(); got != wantKeyID {
		t.Fatalf("KeyID() = %q, want the published %q", got, wantKeyID)
	}
}

// A signature must verify against the public key the way a site's verifier
// checks it, and the signature base must cover exactly what Signature-Input
// advertises.
func TestSignVerifiesAndCoversAuthority(t *testing.T) {
	s := testSigner(t, "")
	created := time.Unix(1735689600, 0)
	h := s.Sign("example.com", created, make([]byte, 64))

	if !strings.Contains(h.SignatureInput, `tag="web-bot-auth"`) {
		t.Errorf("Signature-Input missing the web-bot-auth tag: %s", h.SignatureInput)
	}
	if !strings.Contains(h.SignatureInput, `keyid="`+wantKeyID+`"`) {
		t.Errorf("Signature-Input missing the keyid: %s", h.SignatureInput)
	}
	if h.SignatureAgent != "" {
		t.Errorf("no directory configured, want empty Signature-Agent, got %q", h.SignatureAgent)
	}

	// Rebuild the base from the advertised params and verify, as a site would.
	base := "\"@authority\": example.com\n\"@signature-params\": " + strings.TrimPrefix(h.SignatureInput, "sig1=")
	sig := decodeSig(t, h.Signature)
	if !ed25519.Verify(s.publicKey(), []byte(base), sig) {
		t.Fatal("signature did not verify against the public key")
	}
}

// With a directory configured, signature-agent is both emitted and folded into
// the signature base (the spec requires it be signed when present).
func TestSignCoversSignatureAgentWhenPresent(t *testing.T) {
	dir := "https://signer.example.com"
	s := testSigner(t, dir)
	h := s.Sign("example.com", time.Unix(1735689600, 0), make([]byte, 64))

	if h.SignatureAgent != `"`+dir+`"` {
		t.Fatalf("Signature-Agent = %q, want the quoted directory", h.SignatureAgent)
	}
	if !strings.Contains(h.SignatureInput, `"signature-agent"`) {
		t.Fatalf("signature-agent not in covered components: %s", h.SignatureInput)
	}
	base := "\"@authority\": example.com\n\"signature-agent\": " + h.SignatureAgent +
		"\n\"@signature-params\": " + strings.TrimPrefix(h.SignatureInput, "sig1=")
	if !ed25519.Verify(s.publicKey(), []byte(base), decodeSig(t, h.Signature)) {
		t.Fatal("signature with signature-agent did not verify")
	}
}

// created/expires bound the signature; expires must sit a fixed window past
// created so a captured signature cannot be replayed indefinitely.
func TestSignHasBoundedValidityWindow(t *testing.T) {
	s := testSigner(t, "")
	created := time.Unix(1735689600, 0)
	h := s.Sign("example.com", created, make([]byte, 64))
	if !strings.Contains(h.SignatureInput, "created=1735689600") {
		t.Errorf("missing created: %s", h.SignatureInput)
	}
	wantExpires := created.Add(defaultValidity).Unix()
	if !strings.Contains(h.SignatureInput, "expires="+strconv.FormatInt(wantExpires, 10)) {
		t.Errorf("expires not created+validity (%d): %s", wantExpires, h.SignatureInput)
	}
}

// A different authority produces a different signature: the per-request
// @authority is actually covered, not a constant.
func TestSignIsPerAuthority(t *testing.T) {
	s := testSigner(t, "")
	created := time.Unix(1735689600, 0)
	a := s.Sign("a.example", created, make([]byte, 64))
	b := s.Sign("b.example", created, make([]byte, 64))
	if a.Signature == b.Signature {
		t.Fatal("two different authorities produced the same signature")
	}
}

// The published directory carries the public key under the keyid clients use.
func TestDirectoryPublishesPublicKey(t *testing.T) {
	s := testSigner(t, "https://signer.example.com")
	raw, err := s.Directory(time.Unix(1735689600, 0))
	if err != nil {
		t.Fatalf("Directory: %v", err)
	}
	var doc struct {
		Keys []struct {
			Kty, Crv, X, Kid string
		}
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("directory is not valid JSON: %v", err)
	}
	if len(doc.Keys) != 1 || doc.Keys[0].Kid != wantKeyID || doc.Keys[0].Crv != "Ed25519" {
		t.Fatalf("directory key = %+v, want one Ed25519 key with kid %s", doc.Keys, wantKeyID)
	}
}

func decodeSig(t *testing.T, header string) []byte {
	t.Helper()
	inner := strings.TrimSuffix(strings.TrimPrefix(header, "sig1=:"), ":")
	sig, err := base64.StdEncoding.DecodeString(inner)
	if err != nil {
		t.Fatalf("signature not base64: %v", err)
	}
	return sig
}
