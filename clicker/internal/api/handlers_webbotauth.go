package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

// installSignatureIntercept registers one global beforeRequestSent intercept so
// the router can attach Web Bot Auth headers to every outgoing request (#28).
// Omitting contexts and urlPatterns covers every context, present and future.
// A failure here is logged, not fatal: the browser still works, requests just
// go unsigned.
func (r *Router) installSignatureIntercept(session *BrowserSession) {
	if _, err := r.sendInternalCommand(session, "network.addIntercept", map[string]interface{}{
		"phases": []string{"beforeRequestSent"},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[router] web-bot-auth: could not install request intercept; requests will not be signed: %v\n", err)
	}
}

// beforeRequestSentEvent is the slice of the event the signer needs.
type beforeRequestSentEvent struct {
	Params struct {
		Request struct {
			Request string            `json:"request"`
			URL     string            `json:"url"`
			Headers []json.RawMessage `json:"headers"`
		} `json:"request"`
	} `json:"params"`
}

// signAndContinue signs one blocked request and continues it with the signature
// headers added. It FAILS OPEN: any problem parsing, signing, or sending
// continues the request unsigned rather than leaving it blocked, which would
// hang the page (the #397 failure mode). Runs on its own goroutine so the read
// loop keeps draining while continueRequest round-trips.
func (r *Router) signAndContinue(session *BrowserSession, raw string) {
	var evt beforeRequestSentEvent
	if json.Unmarshal([]byte(raw), &evt) != nil || evt.Params.Request.Request == "" {
		return // not a shape we can act on; the normal path already handled it
	}
	reqID := evt.Params.Request.Request

	headers, ok := r.signatureHeaders(evt.Params.Request.URL)
	if !ok {
		r.continueUnsigned(session, reqID)
		return
	}

	// continueRequest replaces the request headers when headers is provided, so
	// the originals must be carried alongside the signature headers or the
	// request loses them.
	merged := append([]json.RawMessage{}, evt.Params.Request.Headers...)
	for _, h := range headers {
		b, err := json.Marshal(h)
		if err != nil {
			r.continueUnsigned(session, reqID)
			return
		}
		merged = append(merged, b)
	}

	if _, err := r.sendInternalCommand(session, "network.continueRequest", map[string]interface{}{
		"request": reqID,
		"headers": merged,
	}); err != nil {
		// The signed continue failed; say so and try once unsigned so the
		// page still loads.
		fmt.Fprintf(os.Stderr, "[router] web-bot-auth: signed continue failed, retrying unsigned: %v\n", err)
		r.continueUnsigned(session, reqID)
	}
}

func (r *Router) continueUnsigned(session *BrowserSession, reqID string) {
	if _, err := r.sendInternalCommand(session, "network.continueRequest", map[string]interface{}{"request": reqID}); err != nil {
		// A request that cannot be continued at all stays blocked and hangs
		// its fetch; this line is the only trace of why.
		fmt.Fprintf(os.Stderr, "[router] web-bot-auth: request %s could not be continued: %v\n", reqID, err)
	}
}

// signatureHeaders builds the BiDi header entries for one request URL, or
// ok=false when the URL has no authority to sign or signing fails.
func (r *Router) signatureHeaders(rawURL string) ([]map[string]interface{}, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil, false
	}
	h, err := r.signer.SignNow(u.Host, time.Now(), rand.Read)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[router] web-bot-auth: signing failed, continuing unsigned: %v\n", err)
		return nil, false
	}
	entries := []map[string]interface{}{
		bidiHeader("Signature-Input", h.SignatureInput),
		bidiHeader("Signature", h.Signature),
	}
	if h.SignatureAgent != "" {
		entries = append(entries, bidiHeader("Signature-Agent", h.SignatureAgent))
	}
	return entries, true
}

func bidiHeader(name, value string) map[string]interface{} {
	return map[string]interface{}{
		"name":  name,
		"value": map[string]interface{}{"type": "string", "value": value},
	}
}
