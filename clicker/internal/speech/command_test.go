package speech

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func script(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake engines are shell scripts")
	}
	path := filepath.Join(t.TempDir(), "engine.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSplitArgs(t *testing.T) {
	got, err := splitArgs(`piper -m "my voice.onnx" --note 'a "b"' c\ d -- {text}`)
	want := []string{"piper", "-m", "my voice.onnx", "--note", `a "b"`, "c d", "--", "{text}"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := splitArgs(`say "unclosed`); err == nil {
		t.Fatal("an unclosed quote was accepted")
	}
}

func TestCommandSynthesizeThroughFileAndPipe(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "fixture.wav")
	os.WriteFile(fixture, Audio{SampleRate: 22050, Channels: 1, Samples: tone(22050, 300, -20, 500*time.Millisecond)}.WAV(), 0o600)
	log := filepath.Join(t.TempDir(), "log")

	// Text as an argument, WAV to a file. The text keeps its spaces and
	// quotes as one argument: no shell reads it.
	toFile := script(t, `printf '%s|%s|%s|%s' "$1" "$2" "$VIBIUM_SPEECH_PROTOCOL" "$VIBIUM_SPEECH_OP" > `+log+`; cp `+fixture+` "$3"`)
	c := &Command{Line: toFile + " {voice} {text} {output}", Label: "command"}
	text := `Say "hi"; rm -rf $HOME`
	a, err := c.Synthesize(context.Background(), SynthesisRequest{Text: text, Voice: "amy"})
	if err != nil || a.SampleRate != 22050 || len(a.Samples) != 11025 {
		t.Fatalf("%+v %v", a.SampleRate, err)
	}
	if got, _ := os.ReadFile(log); string(got) != "amy|"+text+"|1|synthesize" {
		t.Fatalf("the engine saw %q", got)
	}

	// Text on stdin, WAV on stdout.
	piped := script(t, `cat > `+log+`; cat `+fixture)
	c = &Command{Line: piped, Label: "command"}
	if a, err = c.Synthesize(context.Background(), SynthesisRequest{Text: "from stdin"}); err != nil || len(a.Samples) != 11025 {
		t.Fatalf("%v", err)
	}
	if got, _ := os.ReadFile(log); string(got) != "from stdin" {
		t.Fatalf("stdin carried %q", got)
	}
	if c.CacheSettings(SynthesisRequest{})["command"] != piped {
		t.Fatal("the command line is not part of the cache key")
	}
}

func TestCommandTranscribeContract(t *testing.T) {
	clip := Audio{SampleRate: Rate, Channels: 1, Samples: tone(Rate, 300, -20, time.Second)}
	size := filepath.Join(t.TempDir(), "size")

	json := script(t, `wc -c < "$1" | tr -d ' ' > `+size+`
echo '{"text": " Okay, ten minutes. ", "language": "en", "segments": [{"start": 0.31, "end": 2.5, "text": "Okay, ten minutes."}], "words": [{"start": 0.31, "end": 0.62, "word": "Okay,"}]}'`)
	c := &Command{Line: json + " {input} {language}", Label: "command"}
	tr, err := c.Transcribe(context.Background(), TranscriptionRequest{Audio: clip, Language: "en"})
	if err != nil || tr.Text != "Okay, ten minutes." || tr.Language != "en" {
		t.Fatalf("%+v %v", tr, err)
	}
	if len(tr.Segments) != 1 || tr.Segments[0].Start != 310*time.Millisecond || tr.Words[0].End != 620*time.Millisecond {
		t.Fatalf("times %+v %+v", tr.Segments, tr.Words)
	}
	if got, _ := os.ReadFile(size); strings.TrimSpace(string(got)) != "32044" {
		t.Fatalf("the engine got a %s byte file, want a 16 kHz mono WAV of one second", got)
	}

	plain := script(t, `cat > /dev/null; echo "  just   the text "`)
	c = &Command{Line: plain, Label: "command"}
	if tr, err = c.Transcribe(context.Background(), TranscriptionRequest{Audio: clip}); err != nil || tr.Text != "just the text" {
		t.Fatalf("plain output: %+v %v", tr, err)
	}

	noText := script(t, `cat > /dev/null; echo '{"segments": []}'`)
	c = &Command{Line: noText, Label: "command"}
	if _, err = c.Transcribe(context.Background(), TranscriptionRequest{Audio: clip}); Code(err) != "engine_failed" {
		t.Fatalf("JSON without text: %v", err)
	}
}

func TestCommandFailures(t *testing.T) {
	ctx := context.Background()
	missing := &Command{Line: "/nonexistent/engine {output}", Label: "command"}
	if _, err := missing.Synthesize(ctx, SynthesisRequest{Text: "hi"}); Code(err) != "engine_not_installed" {
		t.Fatalf("missing program: %v", err)
	}
	failing := &Command{Line: script(t, `echo "model file not found" >&2; exit 3`), Label: "command"}
	_, err := failing.Synthesize(ctx, SynthesisRequest{Text: "hi"})
	if Code(err) != "engine_failed" || !strings.Contains(err.Error(), "model file not found") || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("failing program: %v", err)
	}
	garbage := &Command{Line: script(t, `cat > /dev/null; echo "not a wav"`), Label: "command"}
	if _, err := garbage.Synthesize(ctx, SynthesisRequest{Text: "hi"}); Code(err) != "engine_failed" {
		t.Fatalf("garbage output: %v", err)
	}
}

func TestWhisperCPPOutput(t *testing.T) {
	raw := `{"result": {"language": "en"}, "transcription": [
	  {"offsets": {"from": 0, "to": 2600}, "text": " Come here, Watson.", "tokens": [
	    {"text": "[_BEG_]", "offsets": {"from": 0, "to": 0}},
	    {"text": " Come", "offsets": {"from": 120, "to": 400}},
	    {"text": " here", "offsets": {"from": 400, "to": 700}},
	    {"text": ",", "offsets": {"from": 700, "to": 760}},
	    {"text": " Wat", "offsets": {"from": 900, "to": 1200}},
	    {"text": "son", "offsets": {"from": 1200, "to": 1500}},
	    {"text": ".", "offsets": {"from": 1500, "to": 1560}}]},
	  {"offsets": {"from": 2600, "to": 3000}, "text": " ", "tokens": []}]}`
	tr, err := whisperCPPTranscript([]byte(raw))
	if err != nil || tr.Text != "Come here, Watson." || tr.Language != "en" || len(tr.Segments) != 1 {
		t.Fatalf("%+v %v", tr, err)
	}
	var got []string
	for _, w := range tr.Words {
		got = append(got, w.Word)
	}
	if !reflect.DeepEqual(got, []string{"Come", "here,", "Watson."}) || tr.Words[2].Start != 900*time.Millisecond || tr.Words[2].End != 1560*time.Millisecond {
		t.Fatalf("words %v %+v", got, tr.Words)
	}

	// The preset reads the file whisper-cli names itself.
	fake := script(t, `while [ $# -gt 0 ]; do case "$1" in -of) out="$2"; shift;; esac; shift; done
cat > "$out.json" <<'J'
`+raw+`
J`)
	c := &Command{Line: fake + " -m {model} -f {input} -l {language} -ojf -of {output} -np", Label: "whisper.cpp", adapt: presets[1].adapt}
	tr, err = c.Transcribe(context.Background(), TranscriptionRequest{Audio: Audio{SampleRate: Rate, Channels: 1, Samples: make([]int16, 1600)}, Model: "m.bin", Language: "en"})
	if err != nil || tr.Text != "Come here, Watson." {
		t.Fatalf("preset: %+v %v", tr, err)
	}
}
