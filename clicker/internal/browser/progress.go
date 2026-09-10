package browser

import (
	"fmt"
	"io"
	"os"
)

// Progress receives the installer's running commentary: "already installed",
// "Downloading ...", the percentage lines. It is progress, not a result, so
// the CLI points it at stderr under --json; otherwise the JSON envelope
// shares stdout with plain text and neither can be parsed.
//
// Nil means "whatever os.Stdout is at the time". The lookup must stay
// per-call: pipe and mcp reassign os.Stdout to os.Stderr inside their own
// Run, long after this package is initialized, and a writer captured at init
// would pin progress to the real stdout and corrupt the protocol stream pipe
// writes there.
var Progress io.Writer

func progressOut() io.Writer {
	if Progress != nil {
		return Progress
	}
	return os.Stdout
}

func progressf(format string, a ...interface{}) {
	fmt.Fprintf(progressOut(), format, a...)
}
