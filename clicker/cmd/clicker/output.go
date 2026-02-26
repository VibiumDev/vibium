package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vibium/clicker/internal/mcp"
	"github.com/vibium/clicker/internal/process"
)

// jsonEnvelope is the output format for --json mode.
type jsonEnvelope struct {
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// printResult prints a tool call result, respecting --json mode.
// In JSON mode: {"ok":true,"result":"..."}
// In normal mode: just the text content.
func printResult(result *mcp.ToolsCallResult) {
	if result == nil {
		return
	}

	if jsonOutput {
		text := extractText(result)
		env := jsonEnvelope{OK: true, Result: text}
		printJSON(env)
		return
	}

	// Human-readable: just print the text content
	for _, c := range result.Content {
		if c.Type == "text" && c.Text != "" {
			fmt.Println(c.Text)
		}
	}
}

// printError prints an error, respecting --json mode.
// In JSON mode: {"ok":false,"error":"..."}
// In normal mode: prints to stderr and exits.
func printError(err error) {
	if jsonOutput {
		env := jsonEnvelope{OK: false, Error: err.Error()}
		printJSON(env)
		process.KillAll()
		os.Exit(1)
		return
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	process.KillAll()
	os.Exit(1)
}

// printJSON marshals and prints a value as a single JSON line.
func printJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// extractText returns the first text content from a result.
func extractText(result *mcp.ToolsCallResult) string {
	for _, c := range result.Content {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}
