// Package verifier runs an independent, bounded inference conversation. It has
// no access to the builder's messages, filesystem, shell, or browser transport.
package verifier

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const Method = "vibium:check.run"
const Timeout = 3 * time.Minute
const MaxActions = 24
const MaxOutputTokens = 4096
const MaxText = 16000
const MaxImage = 2 * 1024 * 1024
const MaxClaim = 4000

type Config struct {
	Role            string `json:"role,omitempty"` // empty means verifier; both roles share AI defaults
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	BaseURL         string `json:"baseURL"`
	APIKey          string `json:"apiKey,omitempty"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
}

func ConfigFromEnv() (Config, error) { return ConfigForRole("check") }

// ConfigForRole uses shared AI settings; the role selects instructions and tool policy.
func ConfigForRole(role string) (Config, error) { return ResolveConfig(role, Overrides{}) }

func (c Config) Prefix() string { return "VIBIUM_AI_" }
func (c Config) CredentialVariable() string {
	switch c.Provider {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "google":
		return "GOOGLE_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}
func (c Config) Endpoint() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	switch c.Provider {
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "google":
		return "https://generativelanguage.googleapis.com/v1beta"
	case "local":
		return "http://127.0.0.1:8080/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

// ConfigCheck describes configuration validity without exposing configured values.
type ConfigCheck struct {
	Variable string
	Error    string
}

// Checks is shared by normal verification and setup diagnostics. Keep messages
// value-free: even a malformed endpoint or model setting could contain a secret.
func (c Config) Checks() []ConfigCheck {
	prefix := c.Prefix()
	var checks []ConfigCheck
	check := func(variable string, valid bool, problem string) {
		if valid {
			problem = ""
		}
		checks = append(checks, ConfigCheck{Variable: variable, Error: problem})
	}
	validEffort := false
	switch c.ReasoningEffort {
	case "", "none", "minimal", "low", "medium", "high", "xhigh", "max":
		validEffort = true
	}
	if (c.Provider == "anthropic" || c.Provider == "google") && c.ReasoningEffort != "" {
		validEffort = false
	}
	check(prefix+"REASONING_EFFORT", validEffort, "invalid "+prefix+"REASONING_EFFORT")
	check(prefix+"PROVIDER", c.Provider == "openai" || c.Provider == "openai-compatible" || c.Provider == "local" || c.Provider == "anthropic" || c.Provider == "google", "set "+prefix+"PROVIDER to openai, anthropic, google, openai-compatible, or local")
	check(prefix+"MODEL", strings.TrimSpace(c.Model) != "", prefix+"MODEL is required")
	requiresKey := c.Provider == "openai" || c.Provider == "anthropic" || c.Provider == "google"
	check(c.CredentialVariable(), !requiresKey || strings.TrimSpace(c.APIKey) != "", c.CredentialVariable()+" is required")
	endpointProblem := ""
	if c.Provider == "openai-compatible" && c.BaseURL == "" {
		endpointProblem = prefix + "BASE_URL is required for openai-compatible"
	}
	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			endpointProblem = prefix + "BASE_URL must be an HTTP(S) URL without credentials, query, or fragment"
		}
	}
	check(prefix+"BASE_URL", endpointProblem == "", endpointProblem)
	return checks
}

func (c Config) Validate() error {
	for _, check := range c.Checks() {
		if check.Error != "" {
			return fmt.Errorf("%s", check.Error)
		}
	}
	return nil
}

type Request struct {
	Claim  string `json:"claim"`
	Record string `json:"record,omitempty"` // explicit archive on the runtime host
	Output string `json:"output,omitempty"` // optional new live recording on the runtime host
	// Configuration is transmitted only over the existing private daemon socket.
	// Never pass it to the recorder, tool executor, or provider messages.
	Config Config `json:"config"`
}

func (r Request) Validate() error {
	if r.Record != "" && r.Output != "" {
		return fmt.Errorf("input archive and live recording output cannot be combined")
	}
	if strings.TrimSpace(r.Claim) == "" || len(r.Claim) > MaxClaim {
		return fmt.Errorf("check requires a nonempty claim of at most %d bytes", MaxClaim)
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
	Check(context.Context, Request, ToolExecutor) (Result, error)
}

// Clip marks truncation explicitly so missing evidence is never presented as
// an exhaustive observation. The conversion also keeps JSON valid UTF-8.
func Clip(s string) string {
	if len(s) <= MaxText {
		return s
	}
	return strings.ToValidUTF8(s[:MaxText], "") + "\n[truncated; narrow the inspection]"
}

// RecordedResult exposes only the public result and concise recording metadata.
type RecordedResult interface {
	RecordingSummary() (status string, summary string, evidence []Evidence)
}

func (r Result) RecordingSummary() (string, string, []Evidence) {
	return r.Verdict(), r.Summary, r.Evidence
}

// ToolPolicy keeps Check's credential restriction while permitting Run to
// enter explicitly supplied test credentials through existing fill/type tools.
type ToolPolicy struct{ CredentialInput bool }

func (p ToolPolicy) AllowsCredentialInput(tool string) bool {
	return p.CredentialInput && (tool == "browser_fill" || tool == "browser_type" || tool == "browser_press")
}
