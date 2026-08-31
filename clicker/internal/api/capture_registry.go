package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/vibium/clicker/internal/urlmatch"
)

// captureRegistry holds the one-shot event captures for a session. The engine
// owns both halves that used to live in every client: deciding whether an
// event satisfies a capture and waiting for the first one that does
// (vibium:page.captureRequest/captureResponse/captureEvent block until it
// arrives or the timeout passes). Clients just await the command (#446).
type captureRegistry struct {
	mu      sync.Mutex
	nextID  int
	pending map[int]*pendingCapture
}

type pendingCapture struct {
	match func(method string, params json.RawMessage) bool
	// promptContext is set on dialog captures: a prompt opening in this
	// context belongs to the capturer, so the router's auto-dismiss default
	// must leave it alone.
	promptContext string
	ch            chan json.RawMessage
}

func newCaptureRegistry() *captureRegistry {
	return &captureRegistry{pending: map[int]*pendingCapture{}}
}

// add registers a one-shot capture and returns its id and delivery channel.
func (cr *captureRegistry) add(p *pendingCapture) (int, chan json.RawMessage) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.nextID++
	id := cr.nextID
	// Buffered: offer must never block the event-forwarding loop.
	p.ch = make(chan json.RawMessage, 1)
	cr.pending[id] = p
	return id, p.ch
}

// register adds a one-shot network capture and returns its id and delivery
// channel.
func (cr *captureRegistry) register(eventMethod, context, pattern string) (int, chan json.RawMessage, error) {
	matcher, err := urlmatch.Compile(pattern)
	if err != nil {
		return 0, nil, err
	}
	id, ch := cr.add(&pendingCapture{match: networkMatch(eventMethod, context, matcher)})
	return id, ch, nil
}

// networkMatch matches one network event method for a context, with the URL
// run through the session's one glob dialect. Blocked requests belong to
// route interception; captures see the same traffic the request listeners do.
func networkMatch(eventMethod, context string, matcher *urlmatch.Matcher) func(string, json.RawMessage) bool {
	return func(method string, raw json.RawMessage) bool {
		if method != eventMethod {
			return false
		}
		var params struct {
			Context   string `json:"context"`
			IsBlocked bool   `json:"isBlocked"`
			Request   struct {
				URL string `json:"url"`
			} `json:"request"`
		}
		if json.Unmarshal(raw, &params) != nil || params.Request.URL == "" {
			return false
		}
		if method == "network.beforeRequestSent" && params.IsBlocked {
			return false
		}
		return params.Context == context && matcher.Match(params.Request.URL)
	}
}

// captureEventKind builds the pending capture for a vibium:page.captureEvent
// kind. Kinds mirror the client capture surface: capture.navigation,
// capture.download, capture.dialog, and capture.event(name).
func captureEventKind(kind, context string) (*pendingCapture, error) {
	methodIn := func(method string, methods ...string) bool {
		for _, m := range methods {
			if method == m {
				return true
			}
		}
		return false
	}

	switch kind {
	case "navigation":
		return &pendingCapture{match: func(method string, raw json.RawMessage) bool {
			if !methodIn(method, "browsingContext.load", "browsingContext.fragmentNavigated", "browsingContext.historyUpdated") {
				return false
			}
			var params struct {
				Context string `json:"context"`
				URL     string `json:"url"`
			}
			return json.Unmarshal(raw, &params) == nil && params.Context == context && params.URL != ""
		}}, nil
	case "download":
		return &pendingCapture{match: func(method string, raw json.RawMessage) bool {
			if method != "browsingContext.downloadWillBegin" {
				return false
			}
			var params struct {
				Context string `json:"context"`
			}
			return json.Unmarshal(raw, &params) == nil && params.Context == context
		}}, nil
	case "dialog":
		return &pendingCapture{
			promptContext: context,
			match: func(method string, raw json.RawMessage) bool {
				if method != "browsingContext.userPromptOpened" {
					return false
				}
				var params struct {
					Context string `json:"context"`
				}
				return json.Unmarshal(raw, &params) == nil && params.Context == context
			},
		}, nil
	case "console", "error":
		wantType := "console"
		if kind == "error" {
			wantType = "javascript"
		}
		return &pendingCapture{match: func(method string, raw json.RawMessage) bool {
			if method != "log.entryAdded" {
				return false
			}
			var params struct {
				Type   string `json:"type"`
				Source struct {
					Context string `json:"context"`
				} `json:"source"`
			}
			return json.Unmarshal(raw, &params) == nil && params.Type == wantType && params.Source.Context == context
		}}, nil
	default:
		return nil, fmt.Errorf("unknown capture kind %q (want navigation, download, dialog, console, or error)", kind)
	}
}

// cancel removes a capture that timed out or whose session is closing.
func (cr *captureRegistry) cancel(id int) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	delete(cr.pending, id)
}

// wantsPrompt reports whether a pending dialog capture claims prompts opening
// in the given context. The router's auto-dismiss default defers to it.
func (cr *captureRegistry) wantsPrompt(context string) bool {
	if cr == nil {
		return false
	}
	cr.mu.Lock()
	defer cr.mu.Unlock()
	for _, p := range cr.pending {
		if p.promptContext == context {
			return true
		}
	}
	return false
}

// offer inspects one browser event and delivers its params to every pending
// capture it satisfies. The event always continues to the client afterwards:
// captures observe traffic, they do not consume it.
func (cr *captureRegistry) offer(msg string) {
	// Cheap gate before JSON work: events have a method, responses do not.
	if !strings.Contains(msg, `"method"`) {
		return
	}
	cr.mu.Lock()
	idle := len(cr.pending) == 0
	cr.mu.Unlock()
	if idle {
		return
	}

	var evt struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &evt) != nil || evt.Method == "" || evt.Params == nil {
		return
	}

	cr.mu.Lock()
	defer cr.mu.Unlock()
	for id, p := range cr.pending {
		if !p.match(evt.Method, evt.Params) {
			continue
		}
		select {
		case p.ch <- evt.Params:
		default:
		}
		delete(cr.pending, id)
	}
}
