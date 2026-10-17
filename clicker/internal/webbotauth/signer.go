// Package webbotauth signs outgoing requests with Web Bot Auth, the
// cooperative half of Cloudflare's Verified Bots program
// (draft-meunier-web-bot-auth-architecture, built on RFC 9421 HTTP Message
// Signatures over Ed25519). A signed request identifies the agent to sites
// that opt in to allow it; it is the opposite of fingerprint spoofing, and
// the only thing here is an honest signature over the request authority.
//
// This package does only the crypto and header construction. Installing the
// headers on live requests is the router's job (#28).
package webbotauth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"time"
)

// Tag is the literal Signature-Input tag the spec reserves for this purpose.
const Tag = "web-bot-auth"

// defaultValidity bounds how long a signature is accepted. The spec recommends
// at most 24h; a short window limits replay of a captured signature.
const defaultValidity = 5 * time.Minute

// Signer holds the agent's Ed25519 key and the optional directory URL that a
// verifier fetches its public key from.
type Signer struct {
	priv  ed25519.PrivateKey
	keyID string
	// directory is the https URL hosting the JWKS, echoed in the signed
	// Signature-Agent header. Empty means the header is omitted (and then not
	// signed), which the spec permits.
	directory string
	validity  time.Duration
}

// LoadSigner reads a PEM-encoded PKCS#8 Ed25519 private key from keyPath and
// returns a Signer. directory, when non-empty, must be the https URL that
// serves this key's /.well-known/http-message-signatures-directory.
func LoadSigner(keyPath, directory string) (*Signer, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read web-bot-auth key: %w", err)
	}
	priv, err := parseEd25519PKCS8(data)
	if err != nil {
		return nil, err
	}
	return &Signer{
		priv:      priv,
		keyID:     thumbprint(priv.Public().(ed25519.PublicKey)),
		directory: directory,
		validity:  defaultValidity,
	}, nil
}

// NewSigner builds a Signer from an in-memory key, for tests and callers that
// already hold one.
func NewSigner(priv ed25519.PrivateKey, directory string) *Signer {
	return &Signer{
		priv:      priv,
		keyID:     thumbprint(priv.Public().(ed25519.PublicKey)),
		directory: directory,
		validity:  defaultValidity,
	}
}

// KeyID is the base64url JWK SHA-256 thumbprint (RFC 8037) that identifies the
// public key; it is the keyid every signature carries.
func (s *Signer) KeyID() string { return s.keyID }

// Headers are the HTTP headers that carry one signature.
type Headers struct {
	// Signature-Input: the covered components and parameters.
	SignatureInput string
	// Signature: the base64 Ed25519 signature, as an RFC 9421 structured field.
	Signature string
	// SignatureAgent is the directory URL, or "" when none is configured. When
	// set it is included in the signature base, as the spec requires.
	SignatureAgent string
}

// Sign produces the Web Bot Auth headers for a request to authority (the
// request's host[:port], the RFC 9421 @authority component). created and nonce
// are injected so the result is deterministic in tests; callers in production
// pass the current time and fresh random bytes via SignNow.
func (s *Signer) Sign(authority string, created time.Time, nonce []byte) Headers {
	expires := created.Add(s.validity)
	agent := ""
	if s.directory != "" {
		agent = fmt.Sprintf("%q", s.directory)
	}

	base, params := signatureBase(authority, agent, sigParams{
		created: created.Unix(),
		expires: expires.Unix(),
		keyid:   s.keyID,
		nonce:   base64.StdEncoding.EncodeToString(nonce),
	})
	sig := ed25519.Sign(s.priv, []byte(base))

	h := Headers{
		SignatureInput: "sig1=" + params,
		Signature:      fmt.Sprintf("sig1=:%s:", base64.StdEncoding.EncodeToString(sig)),
	}
	if s.directory != "" {
		h.SignatureAgent = agent
	}
	return h
}

// SignNow signs authority with the current time and a fresh 64-byte nonce, the
// production entry point. The nonce source is crypto/rand via the ed25519
// package's own randomness expectations; a failure to read randomness is
// returned rather than silently producing a weak nonce.
func (s *Signer) SignNow(authority string, now time.Time, randRead func([]byte) (int, error)) (Headers, error) {
	nonce := make([]byte, 64)
	if _, err := randRead(nonce); err != nil {
		return Headers{}, fmt.Errorf("web-bot-auth nonce: %w", err)
	}
	return s.Sign(authority, now, nonce), nil
}

type sigParams struct {
	created, expires int64
	keyid, nonce     string
}

// signatureBase builds the RFC 9421 signature base and the @signature-params
// value. @authority is always covered; signature-agent is covered only when
// present, per the spec.
func signatureBase(authority, signatureAgent string, p sigParams) (base, params string) {
	covered := `"@authority"`
	lines := fmt.Sprintf("\"@authority\": %s\n", authority)
	if signatureAgent != "" {
		covered += ` "signature-agent"`
		lines += fmt.Sprintf("\"signature-agent\": %s\n", signatureAgent)
	}
	params = fmt.Sprintf(
		`(%s);created=%d;expires=%d;keyid=%q;alg="ed25519";nonce=%q;tag=%q`,
		covered, p.created, p.expires, p.keyid, p.nonce, Tag,
	)
	base = lines + fmt.Sprintf("\"@signature-params\": %s", params)
	return base, params
}

// thumbprint is the RFC 7638 / RFC 8037 base64url JWK SHA-256 thumbprint of an
// Ed25519 public key: SHA-256 over the canonical JSON with members crv, kty, x
// in lexicographic order and no whitespace.
func thumbprint(pub ed25519.PublicKey) string {
	jwk := fmt.Sprintf(`{"crv":"Ed25519","kty":"OKP","x":%q}`,
		base64.RawURLEncoding.EncodeToString(pub))
	sum := sha256.Sum256([]byte(jwk))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// parseEd25519PKCS8 decodes a PEM "PRIVATE KEY" block holding a PKCS#8
// Ed25519 key. Other key types are rejected with a clear message rather than
// a downstream panic.
func parseEd25519PKCS8(pemData []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("web-bot-auth key is not PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("web-bot-auth key is not a PKCS#8 key: %w", err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("web-bot-auth key is %T, want ed25519", key)
	}
	return priv, nil
}

// Directory renders the signed JWKS the agent publishes at
// /.well-known/http-message-signatures-directory. nbf/exp bound the directory
// itself; the spec has the directory response signed so it cannot be mirrored,
// which is left to whoever serves it.
func (s *Signer) Directory(now time.Time) ([]byte, error) {
	pub := s.priv.Public().(ed25519.PublicKey)
	doc := map[string]interface{}{
		"keys": []map[string]interface{}{{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(pub),
			"kid": s.keyID,
			"nbf": now.Unix(),
		}},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// publicKey is used by tests to verify a signature the way a site would.
func (s *Signer) publicKey() ed25519.PublicKey {
	return s.priv.Public().(ed25519.PublicKey)
}

// GenerateKeyPEM creates a new Ed25519 key and returns it PEM-encoded as a
// PKCS#8 "PRIVATE KEY" block, the form LoadSigner reads. randRead supplies the
// key material (crypto/rand.Reader in production).
func GenerateKeyPEM(randRead io.Reader) ([]byte, error) {
	_, priv, err := ed25519.GenerateKey(randRead)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}
