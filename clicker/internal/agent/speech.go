package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/vibium/clicker/internal/speech"
)

// speechEngine is created on first use. It lives as long as the handlers,
// so the capture window a say opens is still there for the next listen.
func (h *Handlers) speechEngine() *speech.Engine {
	if h.speech == nil {
		h.speech = speech.NewEngine(speech.Env{Getenv: h.speechGetenv, LookPath: exec.LookPath})
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

// speechTarget resolves a target option. The default follows the session:
// with one open it is the session, and otherwise this machine. It never
// falls back from one to the other.
func (h *Handlers) speechTarget(name string) (speech.Target, error) {
	h.sessionMu.Lock()
	client := h.client
	h.sessionMu.Unlock()
	open := client != nil
	if name == "" {
		name = "local"
		if open {
			name = "session"
		}
	}
	switch {
	case name == "local":
		return speech.NewLocalTarget(), nil
	case name == "session":
		if !open {
			return nil, &speech.Error{Code: "invalid_target", Message: "no session is open; use target local or a .wav path, or start a session first"}
		}
		if target, ok := h.roadieAudioFor(client); ok {
			return target, nil
		}
		return nil, &speech.Error{Code: "no_audio", Message: "this session cannot carry audio; use --target local for this machine's speakers and microphone"}
	case speech.IsWAVPath(name):
		path, err := filepath.Abs(name)
		if err != nil {
			return nil, err
		}
		return speech.FileTarget{Path: path}, nil
	}
	return nil, &speech.Error{Code: "invalid_target", Message: fmt.Sprintf("target %q is not session, local, or a path ending in .wav", name)}
}

func speechArgs(args map[string]interface{}, allowed ...string) error {
	for key := range args {
		ok := key == "page"
		for _, a := range allowed {
			ok = ok || key == a
		}
		if !ok {
			return fmt.Errorf("unsupported argument %s", key)
		}
	}
	return nil
}

func argString(args map[string]interface{}, key string) (string, error) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return s, nil
}

// argMs reads a duration in milliseconds. A missing value is -1.
func argMs(args map[string]interface{}, key string) (time.Duration, error) {
	v, ok := args[key]
	if !ok || v == nil {
		return -1, nil
	}
	n, ok := v.(float64)
	if !ok || n < 0 {
		return 0, fmt.Errorf("%s must be a number of milliseconds, zero or more", key)
	}
	return time.Duration(n * float64(time.Millisecond)), nil
}

func speechResult(v interface{}) (*ToolsCallResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &ToolsCallResult{Content: []Content{{Type: "text", Text: string(data)}}}, nil
}

func (h *Handlers) sayMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := speechArgs(args, "text", "file", "target", "provider", "model", "voice", "language", "speed", "cache", "cacheDir"); err != nil {
		return nil, err
	}
	var o speech.SayOptions
	var target string
	for key, dst := range map[string]*string{"text": &o.Text, "file": &o.File, "target": &target, "provider": &o.Provider, "model": &o.Model,
		"voice": &o.Voice, "language": &o.Language, "cacheDir": &o.CacheDir} {
		v, err := argString(args, key)
		if err != nil {
			return nil, err
		}
		*dst = v
	}
	if o.File != "" {
		abs, err := filepath.Abs(o.File)
		if err != nil {
			return nil, err
		}
		o.File = abs
	}
	if v, ok := args["speed"]; ok && v != nil {
		n, isNumber := v.(float64)
		if !isNumber || n <= 0 {
			return nil, fmt.Errorf("speed must be a positive number")
		}
		o.Speed = n
	}
	if v, ok := args["cache"]; ok && v != nil {
		use, isBool := v.(bool)
		if !isBool {
			return nil, fmt.Errorf("cache must be true or false")
		}
		o.NoCache = !use
	}
	t, err := h.speechTarget(target)
	if err != nil {
		return nil, err
	}
	res, err := h.speechEngine().Say(context.Background(), t, o)
	if err != nil {
		return nil, err
	}
	return speechResult(res)
}

func (h *Handlers) listenMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := speechArgs(args, "target", "provider", "model", "language", "silence", "timeout", "duration", "allowSilence", "path", "transcribe"); err != nil {
		return nil, err
	}
	var o speech.ListenOptions
	var target string
	for key, dst := range map[string]*string{"target": &target, "provider": &o.Provider, "model": &o.Model, "language": &o.Language, "path": &o.Path} {
		v, err := argString(args, key)
		if err != nil {
			return nil, err
		}
		*dst = v
	}
	var err error
	if o.Silence, err = argMs(args, "silence"); err != nil {
		return nil, err
	}
	if o.Timeout, err = argMs(args, "timeout"); err != nil {
		return nil, err
	}
	if o.Duration, err = argMs(args, "duration"); err != nil {
		return nil, err
	}
	if v, ok := args["allowSilence"]; ok && v != nil {
		allow, isBool := v.(bool)
		if !isBool {
			return nil, fmt.Errorf("allowSilence must be true or false")
		}
		o.AllowSilence = allow
	}
	if v, ok := args["transcribe"]; ok && v != nil {
		transcribe, isBool := v.(bool)
		if !isBool {
			return nil, fmt.Errorf("transcribe must be true or false")
		}
		o.NoTranscribe = !transcribe
	}
	if o.Path != "" {
		if !speech.IsWAVPath(o.Path) {
			return nil, fmt.Errorf("path must end in .wav")
		}
		if o.Path, err = filepath.Abs(o.Path); err != nil {
			return nil, err
		}
	}
	t, err := h.speechTarget(target)
	if err != nil {
		return nil, err
	}
	res, err := h.speechEngine().Listen(context.Background(), t, o)
	if err != nil {
		return nil, err
	}
	return speechResult(res)
}
