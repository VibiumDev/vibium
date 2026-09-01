package api

import (
	"encoding/json"
	"fmt"

	"github.com/vibium/clicker/internal/log"
)

// Exposed functions (feature B of page.expose, #298): the page calls
// window[name](args), the client's host function runs, and the return value
// comes back to the page. Three wire pieces:
//
//   - vibium:page.exposeFunction {context, name} installs the binding
//   - vibium:expose.call {name, seq, args, context, realm} — event to the
//     client when the page calls the function
//   - vibium:expose.result {context, realm, seq, result | error} — command
//     from the client delivering the host function's outcome into the page
//
// The engine keeps no per-call state: the page parks each call's promise
// under a sequence number, and the client echoes that seq back with the
// result. The same channel machinery runs WebSocket monitoring
// (handlers_websocket.go).

// exposeChannelName is the BiDi channel every exposed-function binding posts
// through; one channel serves all names, multiplexed by the payload.
const exposeChannelName = "vibium-expose"

// exposeBindingDeclaration builds the preload source for one name. The name
// is embedded as a literal because addPreloadScript accepts only channel
// arguments. The stash is per document and dies with it, which is right: a
// pending page promise cannot survive navigation anyway.
func exposeBindingDeclaration(nameLit string) string {
	return fmt.Sprintf(`(channel) => {
	const store = window.__vibiumExpose = window.__vibiumExpose || { seq: 0, pending: {} };
	window[%s] = (...args) => new Promise((resolve, reject) => {
		const seq = ++store.seq;
		store.pending[seq] = { resolve: resolve, reject: reject };
		channel(JSON.stringify({ name: %s, seq: seq, args: args }));
	});
}`, nameLit, nameLit)
}

// exposeDeliverScript settles the parked promise for one call. The result
// travels as a JSON string so only primitives cross the argument boundary;
// a null errorMsg means success.
const exposeDeliverScript = `(seq, resultJson, errorMsg) => {
	const store = window.__vibiumExpose;
	const pending = store && store.pending[seq];
	if (!pending) return;
	delete store.pending[seq];
	if (errorMsg !== null) pending.reject(new Error(errorMsg));
	else pending.resolve(resultJson === null ? undefined : JSON.parse(resultJson));
}`

// handlePageExposeFunction handles vibium:page.exposeFunction — installs the
// promise-returning stub for one name, session-wide and into the already-open
// document, exactly like feature A's source injection (handlePageExpose).
// Repeated exposure of a name replaces its previous binding either way, so
// the two forms of page.expose stay interchangeable per name.
func (r *Router) handlePageExposeFunction(session *BrowserSession, cmd bidiCommand) {
	name, _ := cmd.Params["name"].(string)
	if name == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("name is required"))
		return
	}

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if err := r.ensureScriptMessageSubscription(session); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	nameLit, err := json.Marshal(name)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("invalid name: %w", err))
		return
	}
	declaration := exposeBindingDeclaration(string(nameLit))
	channelArg := []map[string]interface{}{
		{
			"type": "channel",
			"value": map[string]interface{}{
				"channel": exposeChannelName,
			},
		},
	}

	// Replace any previous script for this name rather than stacking another.
	session.mu.Lock()
	previous := session.exposedPreloadIDs[name]
	session.mu.Unlock()
	if previous != "" {
		if _, err := r.sendInternalCommand(session, "script.removePreloadScript", map[string]interface{}{
			"script": previous,
		}); err != nil {
			log.Debug("failed to remove previous exposed function", "name", name, "error", err)
		}
	}

	resp, err := r.sendInternalCommand(session, "script.addPreloadScript", map[string]interface{}{
		"functionDeclaration": declaration,
		"arguments":           channelArg,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var added struct {
		Result struct {
			Script string `json:"script"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &added); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addPreloadScript response: %w", err))
		return
	}
	session.mu.Lock()
	session.exposedPreloadIDs[name] = added.Result.Script
	session.mu.Unlock()

	// The preload only runs for documents loaded from now on, so bind in the
	// document that is already open too.
	if _, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": declaration,
		"target":              map[string]interface{}{"context": context},
		"arguments":           channelArg,
		"awaitPromise":        false,
	}); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"exposed": true})
}

// ensureScriptMessageSubscription subscribes the session to script.message
// once; WebSocket monitoring and exposed functions share the subscription.
func (r *Router) ensureScriptMessageSubscription(session *BrowserSession) error {
	session.mu.Lock()
	subscribed := session.wsSubscribed
	session.mu.Unlock()
	if subscribed {
		return nil
	}

	resp, err := r.sendInternalCommand(session, "session.subscribe", map[string]interface{}{
		"events": []string{"script.message"},
	})
	if err != nil {
		return err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}
	session.mu.Lock()
	session.wsSubscribed = true
	session.mu.Unlock()
	return nil
}

// parseExposeCall extracts a vibium:expose.call event's params from a raw
// browser message, or nil when the message is not an expose channel message.
func parseExposeCall(msg string) map[string]interface{} {
	var event struct {
		Method string `json:"method"`
		Params struct {
			Channel string          `json:"channel"`
			Data    json.RawMessage `json:"data"`
			Source  struct {
				Context string `json:"context"`
				Realm   string `json:"realm"`
			} `json:"source"`
		} `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &event) != nil {
		return nil
	}
	if event.Method != "script.message" || event.Params.Channel != exposeChannelName {
		return nil
	}

	// The channel payload is a BiDi remote value wrapping our JSON string.
	var dataValue struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if json.Unmarshal(event.Params.Data, &dataValue) != nil || dataValue.Type != "string" {
		return nil
	}

	var call struct {
		Name string        `json:"name"`
		Seq  float64       `json:"seq"`
		Args []interface{} `json:"args"`
	}
	if json.Unmarshal([]byte(dataValue.Value), &call) != nil || call.Name == "" {
		return nil
	}

	args := call.Args
	if args == nil {
		args = []interface{}{}
	}
	return map[string]interface{}{
		"name":    call.Name,
		"seq":     call.Seq,
		"args":    args,
		"context": event.Params.Source.Context,
		"realm":   event.Params.Source.Realm,
	}
}

// isExposeChannelEvent intercepts a script.message on the expose channel and
// forwards it to the client as a vibium:expose.call event, returning true.
func (r *Router) isExposeChannelEvent(session *BrowserSession, msg string) bool {
	params := parseExposeCall(msg)
	if params == nil {
		return false
	}
	eventMsg := map[string]interface{}{
		"method": "vibium:expose.call",
		"params": params,
	}
	data, _ := json.Marshal(eventMsg)
	session.Client.Send(string(data))
	return true
}

// exposeDeliverArgs translates a vibium:expose.result command's params into
// the primitive arguments exposeDeliverScript takes: the seq, the result as a
// JSON string or null, and the error message or null.
func exposeDeliverArgs(params map[string]interface{}) ([]map[string]interface{}, error) {
	seq, ok := params["seq"].(float64)
	if !ok {
		return nil, fmt.Errorf("expose.result requires a numeric seq")
	}

	resultArg := map[string]interface{}{"type": "null"}
	errorArg := map[string]interface{}{"type": "null"}
	if errMsg, ok := params["error"].(string); ok && errMsg != "" {
		errorArg = map[string]interface{}{"type": "string", "value": errMsg}
	} else if result, ok := params["result"]; ok && result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("result is not serializable: %w", err)
		}
		resultArg = map[string]interface{}{"type": "string", "value": string(data)}
	}

	return []map[string]interface{}{
		{"type": "number", "value": seq},
		resultArg,
		errorArg,
	}, nil
}

// handleExposeResult handles vibium:expose.result — delivers a host
// function's outcome into the page, settling the promise parked under seq.
// The calling realm from the expose.call event is the preferred target: with
// only a context, a navigation between call and result could deliver into
// the wrong document (the deliver script then finds no pending seq and does
// nothing, which is the correct outcome for a dead call).
func (r *Router) handleExposeResult(session *BrowserSession, cmd bidiCommand) {
	args, err := exposeDeliverArgs(cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	target := map[string]interface{}{}
	if realm, _ := cmd.Params["realm"].(string); realm != "" {
		target["realm"] = realm
	} else if context, _ := cmd.Params["context"].(string); context != "" {
		target["context"] = context
	} else {
		r.sendError(session, cmd.ID, fmt.Errorf("expose.result requires a realm or context"))
		return
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": exposeDeliverScript,
		"target":              target,
		"arguments":           args,
		"awaitPromise":        false,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}
