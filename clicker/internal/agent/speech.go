package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vibium/clicker/internal/api"
)

// speechUnit is say and listen for these handlers, bound to the session
// open now. It is created on first use and lives as long as the handlers,
// so the capture window a say opens is still there for the next listen.
func (h *Handlers) speechUnit() *api.Speech {
	if h.speech == nil {
		h.speech = api.NewSpeech()
		h.speech.Getenv = h.speechGetenv
		h.speech.Recorder = func() *api.Recorder { return h.recorder }
	}
	h.sessionMu.Lock()
	client := h.client
	h.sessionMu.Unlock()
	if client == nil {
		h.speech.Bind(nil, nil)
	} else {
		h.speech.Bind(client, api.NewAgentSession(client))
	}
	return h.speech
}

// speechGetenv reads a speech setting: the CLI caller's value when the
// call came from the CLI, and this process's environment otherwise.
func (h *Handlers) speechGetenv(name string) string {
	if h.speechEnv != nil {
		if v, ok := h.speechEnv[name]; ok {
			return v
		}
	}
	return os.Getenv(name)
}

// SpeechCLI runs say or listen for the CLI with the caller's settings and
// credentials, as CheckCLI takes the caller's model configuration. The MCP
// server never reaches this: it has no way to pass a key.
func (h *Handlers) SpeechCLI(tool string, args map[string]interface{}, env map[string]string) (*ToolsCallResult, error) {
	if tool != "vibium_say" && tool != "vibium_listen" {
		return nil, fmt.Errorf("%s is not a speech tool", tool)
	}
	h.speechEnv = env
	defer func() { h.speechEnv = nil }()
	return h.Call(tool, args)
}

func speechResult(v interface{}) (*ToolsCallResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &ToolsCallResult{Content: []Content{{Type: "text", Text: string(data)}}}, nil
}

func (h *Handlers) sayMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	res, recorded, err := h.speechUnit().Say(args, h.currentContext(), h.speechCallID)
	if err != nil {
		return nil, err
	}
	if recorded != nil {
		h.speechResult = recorded
	}
	return speechResult(res)
}

func (h *Handlers) listenMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	res, recorded, err := h.speechUnit().Listen(args, h.currentContext(), h.speechCallID)
	if err != nil {
		return nil, err
	}
	if recorded != nil {
		h.speechResult = recorded
	}
	return speechResult(res)
}
