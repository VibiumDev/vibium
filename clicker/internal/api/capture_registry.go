package api

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/vibium/clicker/internal/urlmatch"
)

// captureRegistry holds the one-shot request/response captures for a
// session. The engine owns both halves that used to live in every client:
// matching the URL pattern (urlmatch, one dialect) and waiting for the first
// matching event (vibium:page.captureRequest/captureResponse block until it
// arrives or the timeout passes). Clients just await the command (#446).
type captureRegistry struct {
	mu      sync.Mutex
	nextID  int
	pending map[int]*pendingCapture
}

type pendingCapture struct {
	method  string // the BiDi event method this capture waits for
	context string
	matcher *urlmatch.Matcher
	ch      chan json.RawMessage
}

func newCaptureRegistry() *captureRegistry {
	return &captureRegistry{pending: map[int]*pendingCapture{}}
}

// register adds a one-shot capture and returns its id and delivery channel.
func (cr *captureRegistry) register(eventMethod, context, pattern string) (int, chan json.RawMessage, error) {
	matcher, err := urlmatch.Compile(pattern)
	if err != nil {
		return 0, nil, err
	}
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.nextID++
	id := cr.nextID
	// Buffered: offer must never block the event-forwarding loop.
	ch := make(chan json.RawMessage, 1)
	cr.pending[id] = &pendingCapture{method: eventMethod, context: context, matcher: matcher, ch: ch}
	return id, ch, nil
}

// cancel removes a capture that timed out or whose session is closing.
func (cr *captureRegistry) cancel(id int) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	delete(cr.pending, id)
}

// offer inspects one browser event and delivers its params to every pending
// capture it satisfies. The event always continues to the client afterwards:
// captures observe traffic, they do not consume it.
func (cr *captureRegistry) offer(msg string) {
	// Cheap gate before JSON work.
	if !strings.Contains(msg, `"network.beforeRequestSent"`) && !strings.Contains(msg, `"network.responseCompleted"`) {
		return
	}
	var evt struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &evt) != nil || evt.Params == nil {
		return
	}
	var params struct {
		Context   string `json:"context"`
		IsBlocked bool   `json:"isBlocked"`
		Request   struct {
			URL string `json:"url"`
		} `json:"request"`
	}
	if json.Unmarshal(evt.Params, &params) != nil || params.Request.URL == "" {
		return
	}
	// Blocked requests belong to route interception; captures see the same
	// traffic the request listeners do.
	if evt.Method == "network.beforeRequestSent" && params.IsBlocked {
		return
	}

	cr.mu.Lock()
	defer cr.mu.Unlock()
	for id, p := range cr.pending {
		if p.method != evt.Method || p.context != params.Context || !p.matcher.Match(params.Request.URL) {
			continue
		}
		select {
		case p.ch <- evt.Params:
		default:
		}
		delete(cr.pending, id)
	}
}
