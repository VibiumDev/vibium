package speech

import (
	"os"
	"testing"
)

func envOf(vars map[string]string, onPath ...string) Env {
	return Env{Getenv: func(k string) string { return vars[k] }, LookPath: func(name string) (string, error) {
		for _, p := range onPath {
			if p == name {
				return "/usr/bin/" + name, nil
			}
		}
		return "", os.ErrNotExist
	}}
}

func TestOneOpenAIKeyConfiguresBothDirections(t *testing.T) {
	e := envOf(map[string]string{"OPENAI_API_KEY": "sk-test"})
	say, err := e.Say("", "", "")
	if err != nil || say.Provider.Name() != "openai" || say.Model != "gpt-4o-mini-tts" || say.Voice != "alloy" || say.HearModel != "gpt-transcribe" {
		t.Fatalf("say %+v %v", say, err)
	}
	hear, err := e.Listen("", "")
	if err != nil || hear.Provider.Name() != "openai" || hear.Model != "gpt-transcribe" || hear.HearModel != "" {
		t.Fatalf("listen %+v %v", hear, err)
	}
}

func TestResolutionOrder(t *testing.T) {
	// A flag beats the environment, which beats a key, which beats PATH.
	vars := map[string]string{"OPENAI_API_KEY": "sk-test", "VIBIUM_LISTEN_PROVIDER": "whisper.cpp", "VIBIUM_LISTEN_MODEL": "/models/tiny.bin"}
	e := envOf(vars, "whisper-cli", "piper")
	c, err := e.Listen("", "")
	if err != nil || c.Provider.Name() != "whisper.cpp" || c.Model != "/models/tiny.bin" {
		t.Fatalf("the environment's provider should win over a key: %+v %v", c, err)
	}
	c, err = e.Listen("openai", "")
	if err != nil || c.Provider.Name() != "openai" || c.Model != "gpt-transcribe" {
		t.Fatalf("a flag should win, and must not inherit the other provider's model: %+v %v", c, err)
	}
	c, err = e.Say("", "", "")
	if err != nil || c.Provider.Name() != "openai" {
		t.Fatalf("a key should win over an engine on PATH: %+v %v", c, err)
	}
	local := envOf(map[string]string{"VIBIUM_SAY_VOICE": "/voices/a.onnx"}, "piper")
	c, err = local.Say("", "", "")
	if err != nil || c.Provider.Name() != "piper" || c.Voice != "/voices/a.onnx" || c.HearModel != "" {
		t.Fatalf("with no key an engine on PATH should be used: %+v %v", c, err)
	}
}

func TestResolutionErrors(t *testing.T) {
	cases := []struct {
		name string
		env  Env
		call func(Env) error
		code string
	}{
		{"nothing set", envOf(nil), func(e Env) error { _, err := e.Say("", "", ""); return err }, "not_configured"},
		{"provider named without its key", envOf(nil), func(e Env) error { _, err := e.Listen("openai", ""); return err }, "no_api_key"},
		{"preset program missing", envOf(nil), func(e Env) error { _, err := e.Listen("whisper.cpp", "m.bin"); return err }, "engine_not_installed"},
		{"preset without a model", envOf(nil, "whisper-cli"), func(e Env) error { _, err := e.Listen("", ""); return err }, "not_configured"},
		{"preset without a voice", envOf(nil, "piper"), func(e Env) error { _, err := e.Say("", "", ""); return err }, "not_configured"},
		{"wrong direction", envOf(nil, "piper"), func(e Env) error { _, err := e.Listen("piper", "m"); return err }, "not_configured"},
		{"specified but not built", envOf(map[string]string{"DEEPGRAM_API_KEY": "k"}), func(e Env) error { _, err := e.Say("deepgram", "", ""); return err }, "not_configured"},
		{"unknown name", envOf(nil), func(e Env) error { _, err := e.Say("nope", "", ""); return err }, "not_configured"},
		{"command without a line", envOf(nil), func(e Env) error { _, err := e.Say("command", "", ""); return err }, "not_configured"},
	}
	for _, c := range cases {
		if got := Code(c.call(c.env)); got != c.code {
			t.Errorf("%s: got %q, want %q", c.name, got, c.code)
		}
	}
}

func TestCommandLineSelectsTheCommandProvider(t *testing.T) {
	e := envOf(map[string]string{"VIBIUM_SAY_COMMAND": "my-tts --out {output}"})
	c, err := e.Say("", "", "")
	if err != nil || c.Provider.Name() != "command" {
		t.Fatalf("%+v %v", c, err)
	}
	// A preset keeps its name and adapter when its command line is replaced.
	p := envOf(map[string]string{"VIBIUM_LISTEN_COMMAND": "python3 -m whisper_cli {input}", "VIBIUM_LISTEN_PROVIDER": "whisper.cpp", "VIBIUM_LISTEN_MODEL": "m"})
	c, err = p.Listen("", "")
	if err != nil || c.Provider.Name() != "whisper.cpp" || c.Provider.(*Command).Line != "python3 -m whisper_cli {input}" {
		t.Fatalf("%+v %v", c, err)
	}
}
