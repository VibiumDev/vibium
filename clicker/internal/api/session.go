package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vibium/clicker/internal/bidi"
)

// Session abstracts BiDi communication so that both the proxy (WebSocket router)
// and the MCP server (direct bidi.Client) can share the same browser-automation
// logic. All shared standalone functions (Navigate, Click, etc.) accept a Session.
type Session interface {
	// SendBidiCommand sends a BiDi command and returns the full response JSON.
	// The response format matches the proxy's sendInternalCommand output:
	//   {"result": { ... }}   (success)
	//   {"type":"error", "error":"...", "message":"..."}  (error)
	SendBidiCommand(method string, params map[string]interface{}) (json.RawMessage, error)

	// SendBidiCommandWithTimeout is like SendBidiCommand but with a custom timeout.
	SendBidiCommandWithTimeout(method string, params map[string]interface{}, timeout time.Duration) (json.RawMessage, error)

	// GetContextID returns a browsing context ID. If the session tracks a
	// "current" context it returns that; otherwise it fetches the first context
	// from browsingContext.getTree.
	GetContextID() (string, error)

	// SetLastElementBox stores the bounding box of the last resolved element for recording.
	SetLastElementBox(box *BoxInfo)

	// NavTracker returns the session's navigation tracker, or nil if it has
	// none. Used to avoid issuing a screenshot into an in-flight navigation.
	NavTracker() *NavigationTracker
}

// ---------------------------------------------------------------------------
// APISession — adapts Router + BrowserSession to Session.
// ---------------------------------------------------------------------------

// APISession wraps a Router and BrowserSession pair so that shared
// standalone functions can call sendInternalCommand through the Session
// interface.
type APISession struct {
	Router  *Router
	Session *BrowserSession
	Context string // optional explicit context override
}

// NewAPISession creates a APISession.
func NewAPISession(r *Router, s *BrowserSession, context string) *APISession {
	return &APISession{Router: r, Session: s, Context: context}
}

func (p *APISession) SendBidiCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	return p.Router.sendInternalCommand(p.Session, method, params)
}

func (p *APISession) SendBidiCommandWithTimeout(method string, params map[string]interface{}, timeout time.Duration) (json.RawMessage, error) {
	return p.Router.sendInternalCommandWithTimeout(p.Session, method, params, timeout)
}

func (p *APISession) GetContextID() (string, error) {
	if p.Context != "" {
		return p.Context, nil
	}
	return p.Router.getContext(p.Session)
}

func (p *APISession) SetLastElementBox(box *BoxInfo) {
	p.Session.SetLastElementBox(box)
}

// ---------------------------------------------------------------------------
// AgentSession — adapts *bidi.Client to Session.
// ---------------------------------------------------------------------------

// AgentSession wraps a bidi.Client so that shared standalone functions can send
// BiDi commands through the Session interface. The bidi.Client already handles
// error responses as Go errors, so checkBidiError on wrapped responses is a
// safe no-op.
type AgentSession struct {
	Client   *bidi.Client
	Context  string             // optional explicit context override (active tab)
	OnBoxSet func(box *BoxInfo) // optional callback when element box is set
	Prompts  *PromptTracker     // optional; when set, prompt-blocked commands fail fast
	// optional; when set, captures wait out an in-flight navigation (#289)
	Navigations *NavigationTracker
}

// NewAgentSession creates an AgentSession.
func NewAgentSession(client *bidi.Client) *AgentSession {
	return &AgentSession{Client: client}
}

func (m *AgentSession) SendBidiCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	if err := checkPromptBlocked(m.Prompts, method, params); err != nil {
		return nil, err
	}
	msg, err := m.Client.SendCommand(method, params)
	if err != nil {
		return nil, err
	}
	// Wrap msg.Result as {"result": <msg.Result>} to match the proxy response
	// format that parseScriptResult, checkBidiError, etc. expect.
	wrapped, err := json.Marshal(map[string]json.RawMessage{"result": msg.Result})
	if err != nil {
		return nil, fmt.Errorf("failed to wrap bidi result: %w", err)
	}
	return wrapped, nil
}

func (m *AgentSession) SendBidiCommandWithTimeout(method string, params map[string]interface{}, timeout time.Duration) (json.RawMessage, error) {
	if err := checkPromptBlocked(m.Prompts, method, params); err != nil {
		return nil, err
	}
	msg, err := m.Client.SendCommandWithTimeout(method, params, timeout)
	if err != nil {
		return nil, err
	}
	wrapped, err := json.Marshal(map[string]json.RawMessage{"result": msg.Result})
	if err != nil {
		return nil, fmt.Errorf("failed to wrap bidi result: %w", err)
	}
	return wrapped, nil
}

func (p *APISession) NavTracker() *NavigationTracker {
	if p.Session == nil {
		return nil
	}
	return p.Session.navigations
}

func (m *AgentSession) NavTracker() *NavigationTracker { return m.Navigations }

func (m *AgentSession) SetLastElementBox(box *BoxInfo) {
	if m.OnBoxSet != nil {
		m.OnBoxSet(box)
	}
}

func (m *AgentSession) GetContextID() (string, error) {
	if m.Context != "" {
		return m.Context, nil
	}
	tree, err := m.Client.GetTree()
	if err != nil {
		return "", fmt.Errorf("failed to get browsing context: %w", err)
	}
	if len(tree.Contexts) == 0 {
		return "", fmt.Errorf("no browsing contexts available")
	}
	return tree.Contexts[0].Context, nil
}
