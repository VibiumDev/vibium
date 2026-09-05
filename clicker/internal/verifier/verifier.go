// Package verifier runs an independent, bounded inference conversation. It has
// no access to the builder's messages, filesystem, shell, or browser transport.
package verifier

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const Method = "vibium:verify.run"
const Timeout = 3 * time.Minute
const MaxActions = 24
const MaxText = 16000
const MaxImage = 2 * 1024 * 1024
const MaxClaim = 4000

type Config struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	BaseURL         string `json:"baseURL"`
	APIKey          string `json:"apiKey,omitempty"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
}

func ConfigFromEnv() (Config, error) {
	c := Config{Provider: os.Getenv("VIBIUM_VERIFIER_PROVIDER"), Model: os.Getenv("VIBIUM_VERIFIER_MODEL"), BaseURL: os.Getenv("VIBIUM_VERIFIER_BASE_URL"), APIKey: os.Getenv("OPENAI_API_KEY"), ReasoningEffort: os.Getenv("VIBIUM_VERIFIER_REASONING_EFFORT")}
	return c, c.Validate()
}

func (c Config) Validate() error {
	switch c.ReasoningEffort {
	case "", "none", "minimal", "low", "medium", "high", "xhigh", "max":
	default:
		return fmt.Errorf("invalid VIBIUM_VERIFIER_REASONING_EFFORT")
	}
	if c.Provider != "openai" && c.Provider != "openai-compatible" {
		return fmt.Errorf("set VIBIUM_VERIFIER_PROVIDER to openai or openai-compatible")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("VIBIUM_VERIFIER_MODEL is required")
	}
	if c.Provider == "openai" && c.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	if c.Provider == "openai-compatible" && c.BaseURL == "" {
		return fmt.Errorf("VIBIUM_VERIFIER_BASE_URL is required for openai-compatible")
	}
	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("VIBIUM_VERIFIER_BASE_URL must be an HTTP(S) URL without credentials, query, or fragment")
		}
	}
	return nil
}

type Request struct {
	Claim  string `json:"claim"`
	Record string `json:"record,omitempty"` // explicit archive on the runtime host
	// Configuration is transmitted only over the existing private daemon socket.
	// Never pass it to the recorder, tool executor, or provider messages.
	Config Config `json:"config"`
}

func (r Request) Validate() error {
	if strings.TrimSpace(r.Claim) == "" || len(r.Claim) > MaxClaim {
		return fmt.Errorf("verify requires a nonempty claim of at most %d bytes", MaxClaim)
	}
	return r.Config.Validate()
}

type Evidence struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}
type Result struct {
	Status   string     `json:"status"`
	Claim    string     `json:"claim"`
	Summary  string     `json:"summary"`
	Evidence []Evidence `json:"evidence"`
}

func (r Result) Verdict() string {
	switch r.Status {
	case "passed":
		return "PASS"
	case "failed":
		return "FAIL"
	default:
		return "INCONCLUSIVE"
	}
}
func (r Result) Validate() error {
	if r.Status != "passed" && r.Status != "failed" && r.Status != "inconclusive" {
		return fmt.Errorf("invalid verifier status")
	}
	if strings.TrimSpace(r.Summary) == "" || len(r.Summary) > 2000 || len(r.Evidence) > 12 {
		return fmt.Errorf("invalid verifier summary or evidence")
	}
	if r.Status != "inconclusive" && len(r.Evidence) == 0 {
		return fmt.Errorf("verifier verdict requires evidence")
	}
	for _, e := range r.Evidence {
		if e.Type != "observation" || strings.TrimSpace(e.Summary) == "" || len(e.Summary) > 2000 {
			return fmt.Errorf("invalid verifier evidence")
		}
	}
	return nil
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}
type Observation struct {
	Text  string
	Image string // base64 PNG, only when explicitly requested
	MIME  string // image/png by default; archives may contain JPEG screenshots
}
type ToolExecutor interface {
	Tools() []Tool
	Execute(context.Context, string, map[string]interface{}) (Observation, error)
}
type Verifier interface {
	Verify(context.Context, Request, ToolExecutor) (Result, error)
}

// Clip marks truncation explicitly so missing evidence is never presented as
// an exhaustive observation. The conversion also keeps JSON valid UTF-8.
func Clip(s string) string {
	if len(s) <= MaxText {
		return s
	}
	return strings.ToValidUTF8(s[:MaxText], "") + "\n[truncated; narrow the inspection]"
}
