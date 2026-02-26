package proxy

import (
	"encoding/json"
	"fmt"
)

// wsMonitorPreloadScript is injected via script.addPreloadScript to wrap
// window.WebSocket and observe all WS connections, messages, and close events.
// It receives a BiDi channel function as its argument.
const wsMonitorPreloadScript = `(channel) => {
	const OrigWS = window.WebSocket;
	let nextId = 1;

	window.WebSocket = function(url, protocols) {
		const id = nextId++;
		const urlStr = typeof url === 'string' ? url : url.toString();
		const realWS = protocols !== undefined ? new OrigWS(url, protocols) : new OrigWS(url);

		channel(JSON.stringify({ type: 'created', id: id, url: urlStr }));

		realWS.addEventListener('open', () => {
			channel(JSON.stringify({ type: 'open', id: id }));
		});
		realWS.addEventListener('message', (e) => {
			channel(JSON.stringify({ type: 'message', id: id, data: typeof e.data === 'string' ? e.data : '[binary]', direction: 'received' }));
		});
		realWS.addEventListener('close', (e) => {
			channel(JSON.stringify({ type: 'close', id: id, code: e.code, reason: e.reason }));
		});
		realWS.addEventListener('error', () => {
			channel(JSON.stringify({ type: 'error', id: id }));
		});

		const origSend = realWS.send.bind(realWS);
		realWS.send = function(data) {
			channel(JSON.stringify({ type: 'message', id: id, data: typeof data === 'string' ? data : '[binary]', direction: 'sent' }));
			return origSend(data);
		};

		return realWS;
	};

	window.WebSocket.CONNECTING = 0;
	window.WebSocket.OPEN = 1;
	window.WebSocket.CLOSING = 2;
	window.WebSocket.CLOSED = 3;
	window.WebSocket.prototype = OrigWS.prototype;
}`

const wsChannelName = "vibium-ws"

// handlePageOnWebSocket handles vibium:page.onWebSocket — installs the WebSocket
// monitoring preload script and subscribes to script.message events.
func (r *Router) handlePageOnWebSocket(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Subscribe to script.message events (once per session)
	session.mu.Lock()
	needSubscribe := !session.wsSubscribed
	session.mu.Unlock()

	if needSubscribe {
		resp, err := r.sendInternalCommand(session, "session.subscribe", map[string]interface{}{
			"events": []string{"script.message"},
		})
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		if bidiErr := checkBidiError(resp); bidiErr != nil {
			r.sendError(session, cmd.ID, bidiErr)
			return
		}
		session.mu.Lock()
		session.wsSubscribed = true
		session.mu.Unlock()
	}

	// Install preload script (once per session) — applies to all future navigations
	session.mu.Lock()
	needPreload := session.wsPreloadScriptID == ""
	session.mu.Unlock()

	if needPreload {
		resp, err := r.sendInternalCommand(session, "script.addPreloadScript", map[string]interface{}{
			"functionDeclaration": wsMonitorPreloadScript,
			"arguments": []map[string]interface{}{
				{
					"type": "channel",
					"value": map[string]interface{}{
						"channel": wsChannelName,
					},
				},
			},
			"contexts": []interface{}{context},
		})
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		if bidiErr := checkBidiError(resp); bidiErr != nil {
			r.sendError(session, cmd.ID, bidiErr)
			return
		}

		var result struct {
			Result struct {
				Script string `json:"script"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addPreloadScript response: %w", err))
			return
		}

		session.mu.Lock()
		session.wsPreloadScriptID = result.Result.Script
		session.mu.Unlock()
	}

	// Also inject on the current page (preload only fires on future navigations)
	_, err = r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": wsMonitorPreloadScript,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{
				"type": "channel",
				"value": map[string]interface{}{
					"channel": wsChannelName,
				},
			},
		},
		"awaitPromise": false,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// isWsChannelEvent checks if a browser message is a script.message from the WS channel.
// If so, it translates the event and sends it to the client, returning true.
func (r *Router) isWsChannelEvent(session *BrowserSession, msg string) bool {
	session.mu.Lock()
	subscribed := session.wsSubscribed
	session.mu.Unlock()

	if !subscribed {
		return false
	}

	var event struct {
		Method string `json:"method"`
		Params struct {
			Channel string          `json:"channel"`
			Data    json.RawMessage `json:"data"`
			Source  struct {
				Context string `json:"context"`
			} `json:"source"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(msg), &event); err != nil {
		return false
	}

	if event.Method != "script.message" || event.Params.Channel != wsChannelName {
		return false
	}

	// Parse the data — it's a BiDi remote value wrapping our JSON string
	var dataValue struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(event.Params.Data, &dataValue); err != nil {
		return false
	}

	if dataValue.Type != "string" {
		return false
	}

	// Parse our WS event JSON
	var wsEvent map[string]interface{}
	if err := json.Unmarshal([]byte(dataValue.Value), &wsEvent); err != nil {
		return false
	}

	context := event.Params.Source.Context
	r.translateWsEvent(session, wsEvent, context)
	return true
}

// translateWsEvent converts a raw WS monitor event into a vibium:ws.* event and sends to client.
func (r *Router) translateWsEvent(session *BrowserSession, wsEvent map[string]interface{}, context string) {
	eventType, _ := wsEvent["type"].(string)

	var method string
	params := map[string]interface{}{
		"context": context,
	}

	// Copy the id field
	if id, ok := wsEvent["id"].(float64); ok {
		params["id"] = int(id)
	}

	switch eventType {
	case "created":
		method = "vibium:ws.created"
		params["url"], _ = wsEvent["url"].(string)
	case "open":
		method = "vibium:ws.open"
	case "message":
		method = "vibium:ws.message"
		params["data"], _ = wsEvent["data"].(string)
		params["direction"], _ = wsEvent["direction"].(string)
	case "close":
		method = "vibium:ws.closed"
		if code, ok := wsEvent["code"].(float64); ok {
			params["code"] = int(code)
		}
		params["reason"], _ = wsEvent["reason"].(string)
	case "error":
		method = "vibium:ws.error"
	default:
		return
	}

	eventMsg := map[string]interface{}{
		"method": method,
		"params": params,
	}
	data, _ := json.Marshal(eventMsg)
	session.Client.Send(string(data))
}
