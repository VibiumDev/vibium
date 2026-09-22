// Package envfile loads ~/.config/vibium/ai.env into empty process variables.
// It is not a shell: command substitution is ignored, and nonempty environment
// values still win.
package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/vibium/clicker/internal/paths"
)

const DisableVar = "VIBIUM_LOAD_AI_ENV"

var allowed = map[string]bool{
	"VIBIUM_AI_PROVIDER":         true,
	"VIBIUM_AI_MODEL":            true,
	"VIBIUM_AI_BASE_URL":         true,
	"VIBIUM_AI_REASONING_EFFORT": true,
	"OPENAI_API_KEY":             true,
	"ANTHROPIC_API_KEY":          true,
	"GOOGLE_API_KEY":             true,
	"GEMINI_API_KEY":             true,
	"XAI_API_KEY":                true,
	"LINEAR_API_KEY":             true,
	"VIBIUM_LINEAR_API_KEY":      true,
	"VIBIUM_LINEAR_TEAM":         true,
	"VIBIUM_LINEAR_TEAM_ID":      true,
	"VIBIUM_LINEAR_ENDPOINT":     true,
}

// LoadAIEnv applies unset AI settings from the config-dir ai.env file.
// Missing file is a no-op. Nonempty process values are left alone.
func LoadAIEnv() error {
	if aiEnvDisabled() {
		return nil
	}
	dir, err := paths.GetConfigDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(dir, "ai.env")
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if ok, _ := eligibleMode(info.Mode()); !ok {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	vars := Parse(string(raw))
	for name, value := range vars {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return nil
}

const linearDisableVar = "VIBIUM_LOAD_LINEAR_ENV"

// LoadLinearEnv applies unset Linear settings from config-dir linear.env.
func LoadLinearEnv() error {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(linearDisableVar))) {
	case "0", "false", "no", "off":
		return nil
	}
	dir, err := paths.GetConfigDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(dir, "linear.env")
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if ok, _ := eligibleMode(info.Mode()); !ok {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	vars := Parse(string(raw))
	for name, value := range vars {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return nil
}

// Disabled reports whether auto-load is opted out via VIBIUM_LOAD_AI_ENV.
func Disabled() bool {
	return aiEnvDisabled()
}

// Eligible reports whether LoadAIEnv reads the file at path. When it does
// not, reason says why as a sentence fragment. A missing or unstattable
// file is ineligible with an empty reason; readiness has its own message
// for that case.
func Eligible(path string) (ok bool, reason string) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, ""
	}
	return eligibleMode(info.Mode())
}

func eligibleMode(mode os.FileMode) (ok bool, reason string) {
	if !mode.IsRegular() {
		return false, "it is not a regular file"
	}
	if !ownerOnly(mode) {
		return false, fmt.Sprintf("its permissions are %04o, not owner-only", mode.Perm())
	}
	return true, ""
}

func aiEnvDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(DisableVar))) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

// Parse returns allowed assignments. Invalid or unsafe lines are skipped.
// Values are literals: no $HOME or command substitution.
func Parse(content string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(line[len("export "):])
		}
		name, value, ok := splitAssignment(line)
		if !ok || !allowed[name] {
			continue
		}
		if unsafeValue(value) {
			continue
		}
		value = unquote(value)
		if unsafeValue(value) {
			continue
		}
		out[name] = value
	}
	return out
}

func ownerOnly(mode os.FileMode) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	return mode.Perm()&0o077 == 0
}

func splitAssignment(line string) (name, value string, ok bool) {
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", false
	}
	name = strings.TrimSpace(line[:eq])
	value = strings.TrimSpace(line[eq+1:])
	if name == "" || !isEnvName(name) {
		return "", "", false
	}
	return name, value, true
}

func isEnvName(name string) bool {
	for i, r := range name {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func unsafeValue(value string) bool {
	return strings.Contains(value, "$(") || strings.Contains(value, "`")
}

func unquote(value string) string {
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			return strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`)
		}
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
	}
	return value
}
