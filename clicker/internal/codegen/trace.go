// Package codegen turns a recording's action stream into a runnable script.
package codegen

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const maxTraceBytes = 128 * 1024 * 1024
const maxTraceEvents = 100000

// StepKind classifies a timeline entry.
type StepKind int

const (
	StepAction StepKind = iota
	StepGroupStart
	StepGroupEnd
)

// Step is one recorded action or group boundary in timeline order.
type Step struct {
	Kind   StepKind
	Title  string
	Method string
	Params map[string]interface{}
	Error  string
}

// ReadTrace reads the "before"/"after" action stream from a version 8
// recording zip. Only .trace entries are read; screenshots, snapshots, and
// network data are ignored.
func ReadTrace(filename string) ([]Step, error) {
	info, err := os.Stat(filename)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("recording must be a readable regular zip file")
	}
	z, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("invalid recording zip")
	}
	defer z.Close()

	var streams []string
	for _, f := range z.File {
		if !f.FileInfo().IsDir() && strings.HasSuffix(f.Name, ".trace") {
			streams = append(streams, f.Name)
		}
	}
	if len(streams) == 0 {
		return nil, fmt.Errorf("zip contains no trace entries")
	}
	sort.Strings(streams)

	var steps []Step
	actionIndex := map[string]int{}
	groups := map[string]bool{}
	total := int64(0)
	for _, name := range streams {
		var f *zip.File
		for _, candidate := range z.File {
			if candidate.Name == name {
				f = candidate
				break
			}
		}
		if f.UncompressedSize64 > maxTraceBytes || total+int64(f.UncompressedSize64) > maxTraceBytes {
			return nil, fmt.Errorf("recording event data exceeds 128 MiB")
		}
		total += int64(f.UncompressedSize64)
		r, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("cannot read trace entry %s", name)
		}
		err = scanStream(r, name, &steps, actionIndex, groups)
		r.Close()
		if err != nil {
			return nil, err
		}
	}
	return steps, nil
}

func scanStream(r io.Reader, name string, steps *[]Step, actionIndex map[string]int, groups map[string]bool) error {
	scanner := bufio.NewScanner(io.LimitReader(r, maxTraceBytes+1))
	scanner.Buffer(make([]byte, 65536), 4*1024*1024)
	versionSeen := false
	for scanner.Scan() {
		if len(strings.TrimSpace(scanner.Text())) == 0 {
			continue
		}
		var e map[string]interface{}
		if json.Unmarshal(scanner.Bytes(), &e) != nil || e == nil {
			return fmt.Errorf("malformed trace event in %s", name)
		}
		switch e["type"] {
		case "context-options":
			if e["version"] != float64(8) {
				return fmt.Errorf("unsupported trace version; expected version 8")
			}
			versionSeen = true
		case "before":
			if len(*steps) >= maxTraceEvents {
				return fmt.Errorf("recording contains too many events")
			}
			callID := stringField(e, "callId")
			params, _ := e["params"].(map[string]interface{})
			if stringField(e, "method") == "tracingGroup" {
				groups[callID] = true
				*steps = append(*steps, Step{Kind: StepGroupStart, Title: stringField(e, "title")})
				continue
			}
			if stringField(e, "class") == "BiDi" {
				continue
			}
			actionIndex[callID] = len(*steps)
			*steps = append(*steps, Step{
				Kind:   StepAction,
				Title:  stringField(e, "title"),
				Method: stringField(e, "method"),
				Params: params,
			})
		case "after":
			callID := stringField(e, "callId")
			if groups[callID] {
				*steps = append(*steps, Step{Kind: StepGroupEnd})
				continue
			}
			if i, ok := actionIndex[callID]; ok {
				(*steps)[i].Error = errorMessage(e["error"])
			}
		}
	}
	if scanner.Err() != nil {
		return fmt.Errorf("unreadable or oversized trace event in %s", name)
	}
	if !versionSeen {
		return fmt.Errorf("trace lacks version metadata")
	}
	return nil
}

// errorMessage reads Playwright's after.error, which is {message} in Vibium
// recordings and may be a bare string in foreign traces.
func errorMessage(v interface{}) string {
	switch e := v.(type) {
	case string:
		return e
	case map[string]interface{}:
		return stringField(e, "message")
	}
	return ""
}

func stringField(m map[string]interface{}, k string) string {
	s, _ := m[k].(string)
	return s
}
