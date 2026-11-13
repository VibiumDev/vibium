package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type setupUI struct {
	in          io.Reader
	reader      *bufio.Reader
	out         io.Writer
	err         io.Writer
	interactive bool
}

func newSetupUI(cmd *cobra.Command, interactive bool) *setupUI {
	return &setupUI{
		in:          cmd.InOrStdin(),
		out:         cmd.OutOrStdout(),
		err:         cmd.ErrOrStderr(),
		interactive: interactive,
	}
}

func inputIsTTY(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (ui *setupUI) useColor() bool {
	return writerColor(ui.out)
}

func (ui *setupUI) banner() {
	if jsonOutput {
		return
	}
	fmt.Fprint(ui.out, setupBanner(ui.useColor()))
}

func (ui *setupUI) heading(title string) {
	ui.println("%s", maybePaint(ui.useColor(), brandAccent, "▸ "+title))
}

func (ui *setupUI) println(format string, args ...any) {
	if jsonOutput {
		return
	}
	fmt.Fprintf(ui.out, format+"\n", args...)
}

func (ui *setupUI) ok(format string, args ...any) {
	ui.println("%s", maybePaint(ui.useColor(), brandOK, fmt.Sprintf(format, args...)))
}

func (ui *setupUI) skip(format string, args ...any) {
	ui.println("%s", maybePaint(ui.useColor(), brandMuted, fmt.Sprintf(format, args...)))
}

func (ui *setupUI) prompt(label, def string) (string, error) {
	if !ui.interactive {
		return def, nil
	}
	label = maybePaint(ui.useColor(), brandAccent, label)
	if def != "" {
		fmt.Fprintf(ui.out, "%s [%s]: ", label, maybePaint(ui.useColor(), brandMuted, def))
	} else {
		fmt.Fprintf(ui.out, "%s: ", label)
	}
	line, err := ui.readLine()
	if err != nil {
		return "", err
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}

func (ui *setupUI) promptSecret(label string) (string, error) {
	if !ui.interactive {
		return "", nil
	}
	fmt.Fprintf(ui.out, "%s: ", maybePaint(ui.useColor(), brandAccent, label))
	// A buffered paste already delivered the answer; echo is moot then.
	pasted := ui.reader != nil && ui.reader.Buffered() > 0
	if f, ok := ui.in.(*os.File); ok && !pasted && term.IsTerminal(int(f.Fd())) {
		stop := guardSecret(ui, f)
		b, err := term.ReadPassword(int(f.Fd()))
		stop()
		fmt.Fprintln(ui.out)
		return string(b), err
	}
	line, err := ui.readLine()
	fmt.Fprintln(ui.out)
	return line, err
}

func (ui *setupUI) confirm(label string, defYes bool) (bool, error) {
	if !ui.interactive {
		return defYes, nil
	}
	if f, ok := ui.rawFile(); ok {
		yes := defYes
		err := ui.withRaw(f, func(br *bufio.Reader) (err error) {
			yes, err = runToggle(br, ui.out, label, defYes, ui.useColor())
			return err
		})
		if err == nil {
			ans := "No"
			if yes {
				ans = "Yes"
			}
			ui.println("%s %s", maybePaint(ui.useColor(), brandAccent, label), ans)
			return yes, nil
		}
		if errors.Is(err, errSetupCancelled) {
			return false, err
		}
		// Raw mode failed underneath us; fall through to the typed prompt.
	}
	hint := "Y/n"
	if !defYes {
		hint = "y/N"
	}
	ans, err := ui.prompt(label+" ["+hint+"]", "")
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans == "" {
		return defYes, nil
	}
	return ans == "y" || ans == "yes", nil
}

// readLine shares one buffered reader across prompts so input arriving in a
// single read (a multi-line paste, piped answers) survives to later prompts.
func (ui *setupUI) readLine() (string, error) {
	if ui.reader == nil {
		ui.reader = bufio.NewReader(ui.in)
	}
	s, err := ui.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}
