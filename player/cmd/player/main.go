package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibium/player/internal/kitty"
	"github.com/vibium/player/internal/recording"
	"github.com/vibium/player/internal/tui"
)

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin, stdout, stderr *os.File) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: player path/to/record.zip")
		return 1
	}

	path := args[1]
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(stderr, "file not found: %s\n", path)
		return 1
	}

	rec, err := recording.Open(path)
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			fmt.Fprintf(stderr, "not a zip file: %s\n", path)
			return 1
		}
		fmt.Fprintln(stderr, err)
		if errors.Is(err, recording.ErrMultiTrace) || errors.Is(err, recording.ErrNoTrace) || strings.Contains(err.Error(), "parse error") {
			return 2
		}
		return 2
	}
	defer rec.Close()

	if !kitty.Supported(stdin, stdout) {
		fmt.Fprintln(stderr, "this terminal does not support the Kitty graphics protocol; tested terminals: kitty, wezterm, ghostty")
		return 3
	}

	model := tui.New(rec, stdout, kitty.CellPixels(stdin, stdout))
	program := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	for _, key := range rec.MissingResources() {
		fmt.Fprintf(stderr, "resource missing from zip: %s\n", key)
	}
	return 0
}
