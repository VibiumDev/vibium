package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// commandTimeout bounds one engine run.
const commandTimeout = 2 * time.Minute

// Command runs a speech program the user installed, one process per call.
// This is how local engines run, and it keeps their code and licenses out
// of Vibium. The contract is in docs/specs/say-listen-spec.md.
type Command struct {
	Line  string // split once into arguments; no shell runs it
	Label string // provider name in results
	adapt *adapter
	// direction is "say" or "listen" for a command resolved from
	// configuration: its line does one of the two.
	direction string
}

// adapter reads an engine's native output where it differs from the contract.
type adapter struct {
	outputSuffix string // appended to {output} by the engine itself
	transcript   func(raw []byte) (Transcript, error)
}

type preset struct {
	name, direction, program, line, install string
	level                                   string // system or local
	os                                      string // the one system it exists on; "" is any
	needs                                   string // "voice" or "model" when the engine has no default
	adapt                                   *adapter
}

const (
	levelSystem = "system"
	levelLocal  = "local"
)

// presets are built-in command lines: speech the operating system ships,
// and the recommended local engines. A preset is data: it holds no engine
// code. Within a level the first one found on PATH wins.
var presets = []preset{
	// macOS speaks with say(1). It reads the text from stdin, so text
	// that starts with a dash is not taken for an option.
	{name: "system", direction: "say", level: levelSystem, os: "darwin", program: "say",
		line: "say -v {voice} -o {output} --data-format=LEI16@22050"},
	{name: "piper", direction: "say", level: levelLocal, program: "piper", line: "piper -m {voice} -f {output} -- {text}", needs: "voice",
		install: "Install Piper (pip install piper-tts) and download a voice; the voice is the path to its .onnx file."},
	{name: "whisper.cpp", direction: "listen", level: levelLocal, program: "whisper-cli", needs: "model",
		line:    "whisper-cli -m {model} -f {input} -l {language} -ojf -of {output} -np",
		install: "Build whisper.cpp and download a ggml model; the model is the path to its .bin file.",
		adapt:   &adapter{outputSuffix: ".json", transcript: whisperCPPTranscript}},
}

func (c *Command) Name() string { return c.Label }

func (c *Command) CacheSettings(SynthesisRequest) map[string]string {
	return map[string]string{"command": c.Line}
}

func (c *Command) Synthesize(ctx context.Context, req SynthesisRequest) (Audio, error) {
	if c.direction == "listen" {
		return Audio{}, ErrUnsupported
	}
	dir, err := os.MkdirTemp("", "vibium-say-")
	if err != nil {
		return Audio{}, err
	}
	defer os.RemoveAll(dir)
	output := filepath.Join(dir, "out.wav")
	speed := ""
	if req.Speed > 0 {
		speed = strconv.FormatFloat(req.Speed, 'f', -1, 64)
	}
	argv, used, err := c.argv(map[string]string{"text": req.Text, "voice": req.Voice, "model": req.Model,
		"language": req.Language, "speed": speed, "output": output})
	if err != nil {
		return Audio{}, err
	}
	var stdin []byte
	if !used["text"] {
		stdin = []byte(req.Text)
	}
	stdout, err := c.run(ctx, "synthesize", argv, stdin)
	if err != nil {
		return Audio{}, err
	}
	raw := stdout
	if used["output"] {
		if raw, err = os.ReadFile(output); err != nil {
			return Audio{}, errf("engine_failed", "%s wrote no audio to its output file", argv[0])
		}
	}
	audio, err := ParseWAV(raw)
	if err != nil || len(audio.Samples) == 0 {
		return Audio{}, errf("engine_failed", "%s did not produce a PCM WAV", argv[0])
	}
	return audio, nil
}

func (c *Command) Transcribe(ctx context.Context, req TranscriptionRequest) (Transcript, error) {
	if c.direction == "say" {
		return Transcript{}, ErrUnsupported
	}
	dir, err := os.MkdirTemp("", "vibium-listen-")
	if err != nil {
		return Transcript{}, err
	}
	defer os.RemoveAll(dir)
	input, output := filepath.Join(dir, "in.wav"), filepath.Join(dir, "out")
	argv, used, err := c.argv(map[string]string{"input": input, "output": output, "model": req.Model, "language": req.Language})
	if err != nil {
		return Transcript{}, err
	}
	wav := req.Audio.WAV()
	var stdin []byte
	if used["input"] {
		if err := os.WriteFile(input, wav, 0o600); err != nil {
			return Transcript{}, err
		}
	} else {
		stdin = wav
	}
	stdout, err := c.run(ctx, "transcribe", argv, stdin)
	if err != nil {
		return Transcript{}, err
	}
	raw := stdout
	if used["output"] {
		suffix := ""
		if c.adapt != nil {
			suffix = c.adapt.outputSuffix
		}
		if raw, err = os.ReadFile(output + suffix); err != nil {
			return Transcript{}, errf("engine_failed", "%s wrote no transcript to its output file", argv[0])
		}
	}
	var t Transcript
	if c.adapt != nil && c.adapt.transcript != nil {
		t, err = c.adapt.transcript(raw)
	} else {
		t, err = contractTranscript(raw)
	}
	if err != nil {
		return Transcript{}, errf("engine_failed", "%s: %v", argv[0], err)
	}
	if t.Language == "" {
		t.Language = req.Language
	}
	return t, nil
}

// argv splits the command line and fills its placeholders. It reports
// which placeholders the line used.
func (c *Command) argv(values map[string]string) ([]string, map[string]bool, error) {
	parts, err := splitArgs(c.Line)
	if err != nil || len(parts) == 0 {
		return nil, nil, errf("not_configured", "the %s command line is empty or has an unclosed quote", c.Label)
	}
	used := map[string]bool{}
	out := parts[:0]
	for _, part := range parts {
		// A placeholder that stands alone and has no value is dropped,
		// along with the option before it: "-v {voice}" with no voice.
		if name := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}"); len(name)+2 == len(part) {
			if value, known := values[name]; known && value == "" && name != "text" {
				if n := len(out); n > 1 && strings.HasPrefix(out[n-1], "-") && out[n-1] != "--" {
					out = out[:n-1]
				}
				continue
			}
		}
		for name, value := range values {
			token := "{" + name + "}"
			if strings.Contains(part, token) {
				used[name] = true
				part = strings.ReplaceAll(part, token, value)
			}
		}
		out = append(out, part)
	}
	return out, used, nil
}

func (c *Command) run(ctx context.Context, op string, argv []string, stdin []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "VIBIUM_SPEECH_PROTOCOL=1", "VIBIUM_SPEECH_OP="+op)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), nil
	}
	if errors.Is(err, exec.ErrNotFound) || os.IsNotExist(err) {
		return nil, errf("engine_not_installed", "%s is not installed or not on PATH", argv[0])
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errf("engine_failed", "%s did not finish within %s", argv[0], commandTimeout)
	}
	tail := strings.TrimSpace(stderr.String())
	if len(tail) > 2048 {
		tail = tail[len(tail)-2048:]
	}
	return nil, errf("engine_failed", "%s failed (%v): %s", argv[0], err, tail)
}

// splitArgs splits a command line the way a shell would, with quotes and
// backslashes, and nothing else a shell does.
func splitArgs(line string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inArg := false
	var quote rune
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else if r == '\\' && quote == '"' && i+1 < len(runes) {
				i++
				cur.WriteRune(runes[i])
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inArg = r, true
		case r == '\\' && i+1 < len(runes):
			i++
			cur.WriteRune(runes[i])
			inArg = true
		case r == ' ' || r == '\t' || r == '\n':
			if inArg {
				out = append(out, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}
	if inArg {
		out = append(out, cur.String())
	}
	return out, nil
}

// contractTranscript reads the command contract's result: JSON with a
// required text, or any other output taken as the text alone.
func contractTranscript(raw []byte) (Transcript, error) {
	var j struct {
		Text     *string `json:"text"`
		Language string  `json:"language"`
		Segments []struct {
			Start, End float64
			Text       string
		} `json:"segments"`
		Words []struct {
			Start, End float64
			Word       string
		} `json:"words"`
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' || json.Unmarshal(trimmed, &j) != nil {
		return Transcript{Text: strings.Join(strings.Fields(string(trimmed)), " ")}, nil
	}
	if j.Text == nil {
		return Transcript{}, fmt.Errorf("the transcript JSON has no text field")
	}
	t := Transcript{Text: strings.TrimSpace(*j.Text), Language: j.Language}
	for _, s := range j.Segments {
		t.Segments = append(t.Segments, Segment{Start: seconds(s.Start), End: seconds(s.End), Text: strings.TrimSpace(s.Text)})
	}
	for _, w := range j.Words {
		t.Words = append(t.Words, Word{Start: seconds(w.Start), End: seconds(w.End), Word: strings.TrimSpace(w.Word)})
	}
	return t, nil
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

// whisperCPPTranscript reads whisper-cli's full JSON: segments with
// millisecond offsets, each with the tokens it was decoded from.
func whisperCPPTranscript(raw []byte) (Transcript, error) {
	type offsets struct {
		From, To int64
	}
	var j struct {
		Result struct {
			Language string `json:"language"`
		} `json:"result"`
		Transcription []struct {
			Offsets offsets `json:"offsets"`
			Text    string  `json:"text"`
			Tokens  []struct {
				Text    string  `json:"text"`
				Offsets offsets `json:"offsets"`
			} `json:"tokens"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(raw, &j); err != nil {
		return Transcript{}, fmt.Errorf("unreadable whisper.cpp output")
	}
	ms := func(v int64) time.Duration { return time.Duration(v) * time.Millisecond }
	t := Transcript{Language: j.Result.Language}
	var texts []string
	for _, seg := range j.Transcription {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		texts = append(texts, text)
		t.Segments = append(t.Segments, Segment{Start: ms(seg.Offsets.From), End: ms(seg.Offsets.To), Text: text})
		open := false
		for _, tok := range seg.Tokens {
			if strings.HasPrefix(tok.Text, "[_") || tok.Text == "" {
				continue
			}
			if strings.HasPrefix(tok.Text, " ") || !open {
				t.Words = append(t.Words, Word{Start: ms(tok.Offsets.From), End: ms(tok.Offsets.To), Word: strings.TrimSpace(tok.Text)})
				open = true
				continue
			}
			last := &t.Words[len(t.Words)-1]
			last.Word += tok.Text
			last.End = ms(tok.Offsets.To)
		}
	}
	t.Text = strings.Join(texts, " ")
	return t, nil
}
