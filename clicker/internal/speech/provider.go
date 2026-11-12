package speech

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// SynthesisRequest is one phrase to render.
type SynthesisRequest struct {
	Text     string
	Model    string
	Voice    string
	Language string
	Speed    float64
}

// TranscriptionRequest is one stretch of 16 kHz mono audio to transcribe.
type TranscriptionRequest struct {
	Audio    Audio
	Model    string
	Language string
}

// Segment and Word times count from the start of the audio given.
type Segment struct {
	Start, End time.Duration
	Text       string
}

type Word struct {
	Start, End time.Duration
	Word       string
}

type Transcript struct {
	Text     string
	Language string
	Segments []Segment
	Words    []Word
}

// ErrUnsupported is returned by a provider that lacks one direction.
var ErrUnsupported = errors.New("the provider does not support this direction")

// Provider turns text into audio and audio into text. It sees nothing
// else: targets, detection, the cache and recordings are not its business.
type Provider interface {
	Name() string
	Synthesize(ctx context.Context, req SynthesisRequest) (Audio, error)
	Transcribe(ctx context.Context, req TranscriptionRequest) (Transcript, error)
	// CacheSettings names what else changes the audio, for the cache key.
	CacheSettings(req SynthesisRequest) map[string]string
}

// Error is a say or listen failure with a stable code.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func errf(code, format string, args ...interface{}) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Code is the stable code of a speech error, or "" for any other error.
func Code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// EnvNames are the variables say and listen read. The CLI sends its own
// values of these to the daemon, so each call uses the caller's settings.
var EnvNames = []string{
	"VIBIUM_SAY_PROVIDER", "VIBIUM_SAY_MODEL", "VIBIUM_SAY_VOICE", "VIBIUM_SAY_COMMAND",
	"VIBIUM_LISTEN_PROVIDER", "VIBIUM_LISTEN_MODEL", "VIBIUM_LISTEN_COMMAND",
	"OPENAI_API_KEY", "DEEPGRAM_API_KEY", "ELEVENLABS_API_KEY",
}

// Env is where configuration comes from. Tests replace its functions.
type Env struct {
	Getenv   func(string) string
	LookPath func(string) (string, error)
	OS       string // runtime.GOOS; decides which system speech exists
}

// SystemEnv reads the process environment and PATH.
func SystemEnv() Env { return Env{Getenv: os.Getenv, LookPath: exec.LookPath, OS: runtime.GOOS} }

func (e Env) get(name string) string { return strings.TrimSpace(e.Getenv(name)) }

// Choice is a resolved provider with the model and voice to use.
type Choice struct {
	Provider Provider
	Model    string
	Voice    string
	// HearModel is set for say when the same provider can also transcribe.
	// Say uses it to check a render before caching it.
	HearModel string
}

type hosted struct {
	name, key string
	make      func(key string) Provider
	sayModel  string
	sayVoice  string
	hearModel string
}

// hostedProviders are tried in this order when a key is set. Providers
// with no constructor are specified and not built yet.
var hostedProviders = []hosted{
	{name: "deepgram", key: "DEEPGRAM_API_KEY"},
	{name: "elevenlabs", key: "ELEVENLABS_API_KEY"},
	{name: "openai", key: "OPENAI_API_KEY", make: func(key string) Provider { return NewOpenAI(key) },
		sayModel: "gpt-4o-mini-tts", sayVoice: "cedar", hearModel: "gpt-transcribe"},
}

const setupHint = "Install a local engine and set %s, or set OPENAI_API_KEY for hosted speech. See `vibium %s --help`."

// Say resolves the text-to-speech provider. The arguments are the call's
// options; empty ones fall back to the environment, then to defaults.
func (e Env) Say(provider, model, voice string) (Choice, error) {
	return e.resolve("say", provider, model, voice)
}

// Listen resolves the speech-to-text provider.
func (e Env) Listen(provider, model string) (Choice, error) {
	return e.resolve("listen", provider, model, "")
}

func (e Env) resolve(direction, provider, model, voice string) (Choice, error) {
	prefix := "VIBIUM_" + strings.ToUpper(direction) + "_"
	envProvider := e.get(prefix + "PROVIDER")
	name := provider
	if name == "" {
		name = envProvider
	}
	// Environment model and voice belong to the environment's provider.
	if provider == "" || provider == envProvider {
		if model == "" {
			model = e.get(prefix + "MODEL")
		}
		if voice == "" && direction == "say" {
			voice = e.get(prefix + "VOICE")
		}
	}
	command := e.get(prefix + "COMMAND")

	// With nothing chosen, defaults go from what stays on the machine and
	// costs nothing to what does not: speech the operating system has
	// built in, then a local engine the user installed, then a hosted API.
	if name == "" && command != "" {
		name = "command"
	}
	for _, level := range []string{levelSystem, levelLocal} {
		for _, p := range presets {
			if name != "" || p.level != level || p.direction != direction || (p.os != "" && p.os != e.OS) {
				continue
			}
			if _, err := e.LookPath(p.program); err == nil {
				name = p.name
			}
		}
	}
	if name == "" {
		for _, h := range hostedProviders {
			if h.make != nil && e.get(h.key) != "" {
				name = h.name
				break
			}
		}
	}
	if name == "" {
		return Choice{}, errf("not_configured", "no %s provider is configured. "+setupHint, direction, prefix+"PROVIDER", direction)
	}

	for _, h := range hostedProviders {
		if h.name != name {
			continue
		}
		if h.make == nil {
			return Choice{}, errf("not_configured", "the %s provider is not available yet; use openai or a local engine", name)
		}
		key := e.get(h.key)
		if key == "" {
			return Choice{}, errf("no_api_key", "%s is not set; the %s provider needs it", h.key, name)
		}
		c := Choice{Provider: h.make(key), Model: model, Voice: voice}
		if c.Model == "" {
			if direction == "say" {
				c.Model = h.sayModel
			} else {
				c.Model = h.hearModel
			}
		}
		if c.Voice == "" && direction == "say" {
			c.Voice = h.sayVoice
		}
		if direction == "say" {
			c.HearModel = h.hearModel
		}
		return c, nil
	}

	if name == "command" {
		if command == "" {
			return Choice{}, errf("not_configured", "the command provider needs %sCOMMAND", prefix)
		}
		return Choice{Provider: &Command{Line: command, Label: "command", direction: direction}, Model: model, Voice: voice}, nil
	}
	known := false
	for _, p := range presets {
		if p.name != name {
			continue
		}
		known = true
		if p.direction != direction || (p.os != "" && p.os != e.OS) {
			continue
		}
		line := command
		if line == "" {
			if _, err := e.LookPath(p.program); err != nil {
				return Choice{}, errf("engine_not_installed", "%s is not on PATH. %s", p.program, p.install)
			}
			line = p.line
		}
		if p.needs == "voice" && voice == "" {
			return Choice{}, errf("not_configured", "%s needs a voice: set %sVOICE or pass --voice. %s", name, prefix, p.install)
		}
		if p.needs == "model" && model == "" {
			return Choice{}, errf("not_configured", "%s needs a model: set %sMODEL or pass --model. %s", name, prefix, p.install)
		}
		return Choice{Provider: &Command{Line: line, Label: name, adapt: p.adapt, direction: direction}, Model: model, Voice: voice}, nil
	}
	if name == "system" {
		return Choice{}, errf("not_configured", "this system has no built-in way to %s", direction)
	}
	if known {
		return Choice{}, errf("not_configured", "%s cannot %s", name, direction)
	}
	return Choice{}, errf("not_configured", "unknown %s provider %q", direction, name)
}
