package main

import (
	"fmt"
	"os"
	"time"

	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/process"
)

// doWaitOpen waits for page to load if --wait-open is set.
func doWaitOpen() {
	if waitOpen > 0 {
		fmt.Printf("Waiting %d seconds for page to load...\n", waitOpen)
		time.Sleep(time.Duration(waitOpen) * time.Second)
	}
}

// waitAndClose handles the --wait-close flag before closing the browser.
func waitAndClose(launchResult *browser.LaunchResult) {
	if waitClose > 0 {
		fmt.Printf("\nKeeping browser open for %d seconds...\n", waitClose)
		time.Sleep(time.Duration(waitClose) * time.Second)
	}
	launchResult.Close()
}

// fatalExit kills all tracked browser processes and exits with code 1.
// Use instead of os.Exit(1) in oneshot commands to ensure Chrome cleanup.
func fatalExit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	process.KillAll()
	os.Exit(1)
}

// printCheck prints an actionability check result with a checkmark or X.
func printCheck(name string, passed bool) {
	if passed {
		fmt.Printf("✓ %s: true\n", name)
	} else {
		fmt.Printf("✗ %s: false\n", name)
	}
}
