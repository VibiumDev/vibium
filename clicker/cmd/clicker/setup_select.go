package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/term"
)

// errSetupCancelled reports that the user cancelled the wizard (Ctrl-C in a
// raw-mode prompt). runSetup turns it into a quiet non-zero exit.
var errSetupCancelled = errors.New("setup cancelled")

type keyKind int

const (
	keyNone keyKind = iota
	keyRune
	keyUp
	keyDown
	keyLeft
	keyRight
	keyEnter
	keyTab
	keyEsc
	keyCtrlC
)

type keyPress struct {
	kind keyKind
	r    byte
}

// readKey decodes one key press. Arrow keys arrive as ESC [ A..D in one
// burst, so a lone ESC is only treated as the escape key when nothing is
// buffered behind it; unknown sequences decode to keyNone and are ignored.
func readKey(br *bufio.Reader) (keyPress, error) {
	c, err := br.ReadByte()
	if err != nil {
		return keyPress{}, err
	}
	switch c {
	case 0x03:
		return keyPress{kind: keyCtrlC}, nil
	case '\r', '\n':
		return keyPress{kind: keyEnter}, nil
	case '\t':
		return keyPress{kind: keyTab}, nil
	case 0x1b:
		if br.Buffered() == 0 {
			return keyPress{kind: keyEsc}, nil
		}
		c, err = br.ReadByte()
		if err != nil || (c != '[' && c != 'O') {
			return keyPress{kind: keyEsc}, nil
		}
		c, err = br.ReadByte()
		if err != nil {
			return keyPress{kind: keyNone}, nil
		}
		switch c {
		case 'A':
			return keyPress{kind: keyUp}, nil
		case 'B':
			return keyPress{kind: keyDown}, nil
		case 'C':
			return keyPress{kind: keyRight}, nil
		case 'D':
			return keyPress{kind: keyLeft}, nil
		}
		return keyPress{kind: keyNone}, nil
	}
	return keyPress{kind: keyRune, r: c}, nil
}

type selectOption struct {
	value string
	hint  string
}

// runSelect drives a highlighted option list: up/down or j/k move, digits
// jump, Enter accepts, Esc accepts the default, Ctrl-C cancels. It draws in
// raw mode, so lines end in \r\n, and it erases its output before returning;
// the caller prints the collapsed "Label: value" line afterwards.
func runSelect(br *bufio.Reader, out io.Writer, label string, options []selectOption, defIdx int, color bool) (int, error) {
	idx := defIdx
	hint := fmt.Sprintf("(↑/↓ or 1-%d, Enter selects)", len(options))
	fmt.Fprintf(out, "%s  %s\r\n", maybePaint(color, brandAccent, label), maybePaint(color, brandMuted, hint))
	drawn := false
	draw := func() {
		if drawn {
			fmt.Fprintf(out, "\x1b[%dA", len(options))
		}
		drawn = true
		for i, o := range options {
			row := fmt.Sprintf("%d) %s", i+1, o.value)
			if i == idx {
				row = maybePaint(color, brandAccent, "❯ "+row)
			} else {
				row = "  " + row
			}
			if o.hint != "" {
				row += "  " + maybePaint(color, brandMuted, "("+o.hint+")")
			}
			fmt.Fprintf(out, "\r\x1b[2K  %s\r\n", row)
		}
	}
	erase := func() {
		fmt.Fprintf(out, "\x1b[%dA\r\x1b[J", len(options)+1)
	}
	draw()
	for {
		k, err := readKey(br)
		if err != nil {
			erase()
			return 0, err
		}
		switch k.kind {
		case keyUp:
			idx = (idx + len(options) - 1) % len(options)
		case keyDown:
			idx = (idx + 1) % len(options)
		case keyRune:
			switch {
			case k.r == 'k':
				idx = (idx + len(options) - 1) % len(options)
			case k.r == 'j':
				idx = (idx + 1) % len(options)
			case k.r >= '1' && k.r < byte('1'+len(options)):
				idx = int(k.r - '1')
			}
		case keyEnter:
			erase()
			return idx, nil
		case keyEsc:
			erase()
			return defIdx, nil
		case keyCtrlC:
			erase()
			return 0, errSetupCancelled
		}
		draw()
	}
}

// runToggle drives a one-line yes/no choice: left/right or Tab switch, y/n
// jump, Enter accepts, Esc accepts the default, Ctrl-C cancels. Like
// runSelect it erases itself; the caller prints the collapsed line.
func runToggle(br *bufio.Reader, out io.Writer, label string, defYes bool, color bool) (bool, error) {
	yes := defYes
	draw := func() {
		y, n := "Yes", "No"
		if yes {
			y = maybePaint(color, brandAccent, "❯ Yes")
		} else {
			n = maybePaint(color, brandAccent, "❯ No")
		}
		fmt.Fprintf(out, "\r\x1b[2K%s   %s   %s", maybePaint(color, brandAccent, label), y, n)
	}
	erase := func() { fmt.Fprint(out, "\r\x1b[2K") }
	draw()
	for {
		k, err := readKey(br)
		if err != nil {
			erase()
			return false, err
		}
		switch k.kind {
		case keyLeft, keyRight, keyTab:
			yes = !yes
		case keyRune:
			switch k.r {
			case 'y', 'Y':
				yes = true
			case 'n', 'N':
				yes = false
			}
		case keyEnter:
			erase()
			return yes, nil
		case keyEsc:
			erase()
			return defYes, nil
		case keyCtrlC:
			erase()
			return false, errSetupCancelled
		}
		draw()
	}
}

// rawFile returns the terminal file when raw-mode prompts can run: a real
// TTY on stdin and a terminal that understands escape sequences.
func (ui *setupUI) rawFile() (*os.File, bool) {
	if !ui.interactive {
		return nil, false
	}
	f, ok := ui.in.(*os.File)
	if !ok || os.Getenv("TERM") == "dumb" || !term.IsTerminal(int(f.Fd())) {
		return nil, false
	}
	return f, true
}

// withRaw runs fn with the terminal in raw mode, restoring it on every exit
// path. The shared ui.reader keeps any bytes a paste already buffered.
func (ui *setupUI) withRaw(f *os.File, fn func(br *bufio.Reader) error) error {
	state, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(f.Fd()), state)
	if ui.reader == nil {
		ui.reader = bufio.NewReader(ui.in)
	}
	return fn(ui.reader)
}

// selectOne asks the user to pick one option. On a capable terminal it runs
// the arrow-key list; otherwise it prints the numbered list and reads a
// typed choice (number or value), re-asking on bad input.
func (ui *setupUI) selectOne(label string, options []selectOption, defIdx int) (int, error) {
	if f, ok := ui.rawFile(); ok {
		idx := 0
		err := ui.withRaw(f, func(br *bufio.Reader) (err error) {
			idx, err = runSelect(br, ui.out, label, options, defIdx, ui.useColor())
			return err
		})
		if err == nil {
			ui.println("%s: %s", maybePaint(ui.useColor(), brandAccent, label), options[idx].value)
			return idx, nil
		}
		if errors.Is(err, errSetupCancelled) {
			return 0, err
		}
		// Raw mode failed underneath us; fall through to the typed list.
	}
	for i, o := range options {
		num := maybePaint(ui.useColor(), brandAccent, fmt.Sprintf("%d)", i+1))
		row := fmt.Sprintf("  %s %s", num, o.value)
		if o.hint != "" {
			row += "  " + maybePaint(ui.useColor(), brandMuted, "("+o.hint+")")
		}
		ui.println("%s", row)
	}
	var lastErr error
	for tries := 0; tries < 3; tries++ {
		choice, err := ui.prompt(label, fmt.Sprintf("%d", defIdx+1))
		if err != nil {
			return 0, err
		}
		idx, err := parseSelectChoice(choice, options)
		if err == nil {
			return idx, nil
		}
		lastErr = err
		ui.println("%s", err.Error())
	}
	return 0, lastErr
}

func parseSelectChoice(choice string, options []selectOption) (int, error) {
	choice = strings.TrimSpace(strings.ToLower(choice))
	if len(choice) == 1 && choice[0] >= '1' && choice[0] < byte('1'+len(options)) {
		return int(choice[0] - '1'), nil
	}
	values := make([]string, len(options))
	for i, o := range options {
		values[i] = o.value
		if choice == strings.ToLower(o.value) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown choice %q; choose %s", choice, strings.Join(values, ", "))
}

// guardSecret restores the terminal and exits cleanly if Ctrl-C arrives
// while a no-echo read is blocked; without it the process would die with
// echo still off.
func guardSecret(ui *setupUI, f *os.File) func() {
	state, err := term.GetState(int(f.Fd()))
	if err != nil {
		return func() {}
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		if _, ok := <-c; !ok {
			return
		}
		_ = term.Restore(int(f.Fd()), state)
		fmt.Fprintln(ui.err, "\nSetup cancelled.")
		os.Exit(1)
	}()
	return func() {
		signal.Stop(c)
		close(c)
	}
}
