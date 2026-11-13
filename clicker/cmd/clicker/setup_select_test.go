package main

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func keys(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func TestReadKeyDecoding(t *testing.T) {
	cases := []struct {
		in   string
		want keyKind
	}{
		{"\x1b[A", keyUp},
		{"\x1b[B", keyDown},
		{"\x1b[C", keyRight},
		{"\x1b[D", keyLeft},
		{"\x1bOA", keyUp},
		{"\r", keyEnter},
		{"\n", keyEnter},
		{"\t", keyTab},
		{"\x1b", keyEsc},
		{"\x03", keyCtrlC},
		{"\x1b[H", keyNone}, // unknown sequence is ignored, not misread
	}
	for _, c := range cases {
		k, err := readKey(keys(c.in))
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if k.kind != c.want {
			t.Fatalf("%q: got kind %d, want %d", c.in, k.kind, c.want)
		}
	}
	k, err := readKey(keys("7"))
	if err != nil || k.kind != keyRune || k.r != '7' {
		t.Fatalf("digit: %+v %v", k, err)
	}
}

func testOptions() []selectOption {
	return []selectOption{{value: "one"}, {value: "two"}, {value: "three", hint: "a hint"}}
}

func TestRunSelectArrowsAndEnter(t *testing.T) {
	var out bytes.Buffer
	idx, err := runSelect(keys("\x1b[B\x1b[B\r"), &out, "Pick", testOptions(), 0, false)
	if err != nil || idx != 2 {
		t.Fatalf("got %d %v", idx, err)
	}
	if !strings.Contains(out.String(), "❯ 1) one") {
		t.Fatalf("missing highlight marker: %q", out.String())
	}
	if !strings.Contains(out.String(), "(a hint)") {
		t.Fatalf("missing hint: %q", out.String())
	}
	if strings.Contains(out.String(), "\x1b[38") {
		t.Fatalf("color escapes with color off: %q", out.String())
	}
}

func TestRunSelectWrapsAndVimKeys(t *testing.T) {
	var out bytes.Buffer
	// k from the top wraps to the last option.
	idx, err := runSelect(keys("k\r"), &out, "Pick", testOptions(), 0, false)
	if err != nil || idx != 2 {
		t.Fatalf("wrap up: got %d %v", idx, err)
	}
	idx, err = runSelect(keys("j\r"), &out, "Pick", testOptions(), 0, false)
	if err != nil || idx != 1 {
		t.Fatalf("j: got %d %v", idx, err)
	}
}

func TestRunSelectDigitJump(t *testing.T) {
	var out bytes.Buffer
	idx, err := runSelect(keys("3\r"), &out, "Pick", testOptions(), 0, false)
	if err != nil || idx != 2 {
		t.Fatalf("got %d %v", idx, err)
	}
	// A digit past the menu does not move the highlight.
	idx, err = runSelect(keys("9\r"), &out, "Pick", testOptions(), 1, false)
	if err != nil || idx != 1 {
		t.Fatalf("out of range: got %d %v", idx, err)
	}
}

func TestRunSelectEscKeepsDefault(t *testing.T) {
	var out bytes.Buffer
	idx, err := runSelect(keys("\x1b[B\x1b"), &out, "Pick", testOptions(), 1, false)
	if err != nil || idx != 1 {
		t.Fatalf("got %d %v", idx, err)
	}
}

func TestRunSelectCtrlCCancels(t *testing.T) {
	var out bytes.Buffer
	_, err := runSelect(keys("\x03"), &out, "Pick", testOptions(), 0, false)
	if !errors.Is(err, errSetupCancelled) {
		t.Fatalf("got %v", err)
	}
}

func TestRunToggle(t *testing.T) {
	var out bytes.Buffer
	yes, err := runToggle(keys("\x1b[C\r"), &out, "Sure?", true, false)
	if err != nil || yes {
		t.Fatalf("right arrow: got %v %v", yes, err)
	}
	yes, err = runToggle(keys("y\r"), &out, "Sure?", false, false)
	if err != nil || !yes {
		t.Fatalf("y: got %v %v", yes, err)
	}
	yes, err = runToggle(keys("\x1b"), &out, "Sure?", true, false)
	if err != nil || !yes {
		t.Fatalf("esc default: got %v %v", yes, err)
	}
	if _, err = runToggle(keys("\x03"), &out, "Sure?", true, false); !errors.Is(err, errSetupCancelled) {
		t.Fatalf("ctrl-c: got %v", err)
	}
}

func TestSelectOneFallsBackToTypedInput(t *testing.T) {
	// Buffer input is not a TTY, so selectOne must use the numbered list.
	ui := &setupUI{in: bytes.NewBufferString("2\n"), out: ioDiscard(), err: ioDiscard(), interactive: true}
	idx, err := ui.selectOne("Pick", testOptions(), 0)
	if err != nil || idx != 1 {
		t.Fatalf("got %d %v", idx, err)
	}
}

func TestSelectOneRepromptsOnBadInput(t *testing.T) {
	out := &bytes.Buffer{}
	ui := &setupUI{in: bytes.NewBufferString("9\ntwo\n"), out: out, err: ioDiscard(), interactive: true}
	idx, err := ui.selectOne("Pick", testOptions(), 0)
	if err != nil || idx != 1 {
		t.Fatalf("got %d %v", idx, err)
	}
	if !strings.Contains(out.String(), "unknown choice") {
		t.Fatalf("no complaint shown: %q", out.String())
	}

	ui = &setupUI{in: bytes.NewBufferString("9\n9\n9\n"), out: ioDiscard(), err: ioDiscard(), interactive: true}
	if _, err := ui.selectOne("Pick", testOptions(), 0); err == nil {
		t.Fatal("three bad answers must fail")
	}
}

func TestSetupAIRepromptsForEmptyModel(t *testing.T) {
	setupTestEnv(t)
	jsonOutput = false
	// openai-compatible has no model default, so the empty answer
	// re-prompts; the last empty line leaves the base URL unset.
	in := bytes.NewBufferString("5\n\nreal-model\n\n")
	ui := &setupUI{in: in, out: ioDiscard(), err: ioDiscard(), interactive: true}
	sec := setupAI(&cobra.Command{}, ui, false)
	if sec.Status != "done" {
		t.Fatalf("%+v", sec)
	}
	body, err := os.ReadFile(sec.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "export VIBIUM_AI_MODEL='real-model'") {
		t.Fatalf("model not written: %s", body)
	}
}

func TestSetupSkillsAsksWhichAgentWhenBothExist(t *testing.T) {
	home := setupTestEnv(t)
	jsonOutput = false
	for _, d := range []string{".grok", ".claude"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Choose "both", then confirm the install.
	in := bytes.NewBufferString("3\ny\n")
	ui := &setupUI{in: in, out: ioDiscard(), err: ioDiscard(), interactive: true}
	sec := setupSkills(&cobra.Command{}, ui, false)
	if sec.Status != "done" || sec.Agent != "claude,grok" {
		t.Fatalf("%+v", sec)
	}
	for _, d := range []string{".grok", ".claude"} {
		for _, name := range []string{"browser", "check"} {
			if _, err := os.Stat(filepath.Join(home, d, "skills", name, "SKILL.md")); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestSectionErrorsKeepCancelDistinct(t *testing.T) {
	if s := aiSectionError(errSetupCancelled); s.Status != "cancelled" {
		t.Fatalf("%+v", s)
	}
	if s := aiSectionError(errors.New("boom")); s.Status != "failed" || s.Message != "boom" {
		t.Fatalf("%+v", s)
	}
	if s := skillsSectionError(errSetupCancelled, "grok"); s.Status != "cancelled" || s.Agent != "grok" {
		t.Fatalf("%+v", s)
	}
}
