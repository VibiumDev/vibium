package main

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/speech"
)

// speechTestEnv has shell scripts stand in for the two engines. The say
// engine writes a tone and the listen engine answers with heard.
func speechTestEnv(t *testing.T, heard string, extra map[string]string) speech.Env {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake engines are shell scripts")
	}
	dir := t.TempDir()
	samples := make([]int16, 2*16000)
	for i := 8000; i < 24000; i++ {
		samples[i] = int16(6000 * math.Sin(2*math.Pi*300*float64(i)/16000))
	}
	voice := filepath.Join(dir, "voice.wav")
	if err := os.WriteFile(voice, speech.Audio{SampleRate: 16000, Channels: 1, Samples: samples}.WAV(), 0o600); err != nil {
		t.Fatal(err)
	}
	script := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	vars := map[string]string{
		"VIBIUM_SAY_COMMAND":    script("tts.sh", `cat > `+dir+`/said.txt; cp `+voice+` "$1"`) + " {output}",
		"VIBIUM_LISTEN_COMMAND": script("stt.sh", `echo '{"text": "`+heard+`"}'`) + " {input}",
	}
	for k, v := range extra {
		vars[k] = v
	}
	t.Cleanup(func() {
		if said, err := os.ReadFile(filepath.Join(dir, "said.txt")); err == nil && string(said) != readySpeechPhrase {
			t.Errorf("the say engine was given %q", said)
		}
	})
	return speech.Env{Getenv: func(name string) string { return vars[name] }, LookPath: exec.LookPath, OS: "linux"}
}

func speechCheck(t *testing.T, result setupResult, name string) setupCheck {
	t.Helper()
	for _, c := range result.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %s check in %+v", name, result.Checks)
	return setupCheck{}
}

func TestReadySpeechSendsOnePhraseThroughBothProviders(t *testing.T) {
	t.Setenv("VIBIUM_CACHE_DIR", t.TempDir())
	result := checkSpeechSetup(context.Background(), speechTestEnv(t, "the quick brown fox jumps over the lazy dog", nil))
	if !result.Ready || result.Scope != "speech" || len(result.Checks) != 2 {
		t.Fatalf("%+v", result)
	}
	if c := speechCheck(t, result, "say"); c.Status != "passed" || !strings.HasPrefix(c.Message, "command rendered 1.1 s of speech in ") {
		t.Errorf("say: %+v", c)
	}
	if c := speechCheck(t, result, "listen"); c.Status != "passed" || !strings.HasPrefix(c.Message, `command heard "the quick brown fox jumps over the lazy dog" in `) {
		t.Errorf("listen: %+v", c)
	}
	// The user's cache is not the place for a test phrase.
	if left, _ := filepath.Glob(filepath.Join(os.Getenv("VIBIUM_CACHE_DIR"), "*")); len(left) != 0 {
		t.Errorf("the check wrote to the cache: %v", left)
	}
}

func TestReadySpeechFailsWhenListenHearsSomethingElse(t *testing.T) {
	result := checkSpeechSetup(context.Background(), speechTestEnv(t, "thank you for watching", nil))
	if result.Ready {
		t.Fatalf("%+v", result)
	}
	if c := speechCheck(t, result, "say"); c.Status != "passed" {
		t.Errorf("say: %+v", c)
	}
	c := speechCheck(t, result, "listen")
	if c.Status != "failed" || !strings.Contains(c.Message, `heard "thank you for watching" for "The quick brown fox`) || c.Fix == "" {
		t.Errorf("listen: %+v", c)
	}
}

func TestReadySpeechReportsEachDirectionOnItsOwn(t *testing.T) {
	nothing := speech.Env{Getenv: func(string) string { return "" }, LookPath: func(string) (string, error) { return "", exec.ErrNotFound }, OS: "linux"}
	result := checkSpeechSetup(context.Background(), nothing)
	if result.Ready || result.Summary != "Fix the failed checks and rerun vibium ready speech." {
		t.Fatalf("%+v", result)
	}
	for _, name := range []string{"say", "listen"} {
		if c := speechCheck(t, result, name); c.Status != "failed" || !strings.HasPrefix(c.Message, "not_configured: no "+name+" provider is configured.") {
			t.Errorf("%s: %+v", name, c)
		}
	}

	// A say that fails leaves listen with nothing to hear, and says so.
	broken := speechTestEnv(t, "unused", map[string]string{"VIBIUM_SAY_COMMAND": "/no/such/engine {output}"})
	result = checkSpeechSetup(context.Background(), broken)
	if c := speechCheck(t, result, "say"); c.Status != "failed" || result.Ready {
		t.Errorf("say: %+v", c)
	}
	if c := speechCheck(t, result, "listen"); c.Status != "skipped" || !strings.HasPrefix(c.Message, "command was not tested") {
		t.Errorf("listen: %+v", c)
	}

	keyless := speech.Env{Getenv: func(name string) string {
		return map[string]string{"VIBIUM_SAY_PROVIDER": "openai", "VIBIUM_LISTEN_PROVIDER": "openai"}[name]
	}, LookPath: exec.LookPath, OS: "linux"}
	result = checkSpeechSetup(context.Background(), keyless)
	if c := speechCheck(t, result, "say"); c.Status != "failed" || !strings.HasPrefix(c.Message, "no_api_key: OPENAI_API_KEY is not set") || !strings.Contains(c.Fix, "ai.env") {
		t.Errorf("say: %+v", c)
	}
}
