package api

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/vibium/clicker/internal/urlmatch"
)

// routeRegistry tracks the route patterns registered for each browsing
// context and the one network intercept that serves them all. URL matching
// happens here, in the binary, so every client dispatches on the same
// dialect (urlmatch) instead of reimplementing glob semantics per language:
// before this, JS, Python, and Java each matched routes with their own glob,
// and the three had already drifted (#446).
type routeRegistry struct {
	mu       sync.RWMutex
	contexts map[string]*contextRoutes
}

type contextRoutes struct {
	interceptID string
	patterns    map[string]*routePattern
}

type routePattern struct {
	matcher *urlmatch.Matcher
	count   int
}

func newRouteRegistry() *routeRegistry {
	return &routeRegistry{contexts: map[string]*contextRoutes{}}
}

// add registers one use of pattern for the context. It returns the intercept
// id already serving the context; needIntercept reports that none exists yet
// and the caller must create one and record it with setIntercept.
func (rr *routeRegistry) add(context, pattern string) (interceptID string, needIntercept bool, err error) {
	matcher, err := urlmatch.Compile(pattern)
	if err != nil {
		return "", false, err
	}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	cr := rr.contexts[context]
	if cr == nil {
		cr = &contextRoutes{patterns: map[string]*routePattern{}}
		rr.contexts[context] = cr
	}
	p := cr.patterns[pattern]
	if p == nil {
		p = &routePattern{matcher: matcher}
		cr.patterns[pattern] = p
	}
	p.count++
	return cr.interceptID, cr.interceptID == "", nil
}

// setIntercept records the intercept id created for a context.
func (rr *routeRegistry) setIntercept(context, id string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if cr := rr.contexts[context]; cr != nil {
		cr.interceptID = id
	}
}

// remove drops one use of pattern for the context. When the context has no
// patterns left, it returns the intercept id to tear down and empty=true.
func (rr *routeRegistry) remove(context, pattern string) (interceptID string, empty bool) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	cr := rr.contexts[context]
	if cr == nil {
		return "", false
	}
	if p := cr.patterns[pattern]; p != nil {
		p.count--
		if p.count <= 0 {
			delete(cr.patterns, pattern)
		}
	}
	if len(cr.patterns) == 0 {
		delete(rr.contexts, context)
		return cr.interceptID, true
	}
	return cr.interceptID, false
}

// matched returns the registered patterns matching url for the context,
// sorted for deterministic delivery.
func (rr *routeRegistry) matched(context, url string) []string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	cr := rr.contexts[context]
	if cr == nil {
		return []string{}
	}
	out := []string{}
	for pattern, p := range cr.patterns {
		if p.matcher.Match(url) {
			out = append(out, pattern)
		}
	}
	sort.Strings(out)
	return out
}

// annotateBlockedRequest injects vibiumMatchedPatterns into a blocked
// network.beforeRequestSent event: the registered route patterns whose glob
// matches the request URL. Clients dispatch their route handlers on this
// list. The field is always present on blocked events, so an empty list
// means no registered route matched. Anything unparseable passes through
// untouched.
func (rr *routeRegistry) annotateBlockedRequest(msg string) string {
	// Cheap gate before JSON work: every network event names its method.
	if !strings.Contains(msg, `"network.beforeRequestSent"`) {
		return msg
	}
	var evt map[string]interface{}
	if err := json.Unmarshal([]byte(msg), &evt); err != nil || evt["method"] != "network.beforeRequestSent" {
		return msg
	}
	params, _ := evt["params"].(map[string]interface{})
	if params == nil {
		return msg
	}
	if blocked, _ := params["isBlocked"].(bool); !blocked {
		return msg
	}
	context, _ := params["context"].(string)
	request, _ := params["request"].(map[string]interface{})
	url, _ := request["url"].(string)
	if context == "" || url == "" {
		return msg
	}
	params["vibiumMatchedPatterns"] = rr.matched(context, url)
	out, err := json.Marshal(evt)
	if err != nil {
		return msg
	}
	return string(out)
}
