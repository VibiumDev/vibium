package recording

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strings"
)

var (
	ErrMultiTrace = errors.New("multi-trace recordings not supported in this prototype")
	ErrNoTrace    = errors.New("parse error: no trace file found")
)

type Recording struct {
	Title     string
	StartMs   int64
	EndMs     int64
	Frames    []Frame
	Actions   []Action
	Boxes     map[string]Rect
	Resources map[string][]byte

	zip       *zip.ReadCloser
	resources map[string]*zip.File
	missing   map[string]struct{}
}

type Frame struct {
	Timestamp int64
	SHA1      string
	Width     int
	Height    int
}

type Action struct {
	CallID         string
	ParentID       string
	BeforeImageSHA string
	AfterImageSHA  string
	Title          string
	Method         string
	Params         map[string]any
	StartTime      int64
	EndTime        int64
	Depth          int
}

type Rect struct {
	X int
	Y int
	W int
	H int
}

type parseAction struct {
	action             Action
	beforeSnapshotName string
	afterSnapshotName  string
	ignored            bool
}

func Open(p string) (*Recording, error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, err
	}

	traceFile, resources, err := selectTrace(zr.File)
	if err != nil {
		zr.Close()
		return nil, err
	}

	rec := &Recording{
		Boxes:     map[string]Rect{},
		Resources: map[string][]byte{},
		zip:       zr,
		resources: resources,
		missing:   map[string]struct{}{},
	}

	if err := rec.parse(traceFile); err != nil {
		zr.Close()
		return nil, err
	}

	return rec, nil
}

func (r *Recording) Close() error {
	if r.zip == nil {
		return nil
	}
	return r.zip.Close()
}

func (r *Recording) FrameAt(t int64) (Frame, bool) {
	if len(r.Frames) == 0 {
		return Frame{}, false
	}
	idx := sort.Search(len(r.Frames), func(i int) bool {
		return r.Frames[i].Timestamp > t
	}) - 1
	if idx < 0 {
		return Frame{}, false
	}
	return r.Frames[idx], true
}

func (r *Recording) ActionAt(t int64) (Action, bool) {
	var found Action
	ok := false
	for _, action := range r.Actions {
		end := effectiveEnd(action)
		if t >= action.StartTime && t <= end {
			if !ok || action.StartTime >= found.StartTime {
				found = action
				ok = true
			}
		}
	}
	return found, ok
}

func (r *Recording) Resource(sha1 string) ([]byte, string, error) {
	if sha1 == "" {
		return nil, "", fmt.Errorf("empty resource key")
	}
	if b, ok := r.Resources[sha1]; ok {
		return b, contentType(sha1, b), nil
	}
	zf, ok := r.resources[sha1]
	if !ok {
		r.missing[sha1] = struct{}{}
		return nil, "", fmt.Errorf("resource %q missing from zip", sha1)
	}
	rc, err := zf.Open()
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}
	r.Resources[sha1] = b
	return b, contentType(sha1, b), nil
}

func (r *Recording) MissingResources() []string {
	out := make([]string, 0, len(r.missing))
	for key := range r.missing {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func selectTrace(files []*zip.File) (*zip.File, map[string]*zip.File, error) {
	var traces []*zip.File
	resources := map[string]*zip.File{}

	for _, f := range files {
		if strings.HasPrefix(f.Name, "resources/") && !strings.HasSuffix(f.Name, "/") {
			resources[strings.TrimPrefix(f.Name, "resources/")] = f
			continue
		}
		base := path.Base(f.Name)
		if isTraceName(base) {
			traces = append(traces, f)
		}
	}

	if len(traces) == 0 {
		return nil, nil, ErrNoTrace
	}
	if len(traces) > 1 {
		return nil, nil, ErrMultiTrace
	}
	return traces[0], resources, nil
}

func isTraceName(name string) bool {
	if name == "trace.trace" {
		return true
	}
	if !strings.HasSuffix(name, "-trace.trace") {
		return false
	}
	prefix := strings.TrimSuffix(name, "-trace.trace")
	if prefix == "" {
		return false
	}
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (r *Recording) parse(traceFile *zip.File) error {
	rc, err := traceFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	actionsByID := map[string]*parseAction{}
	var order []string
	snapshotImages := map[string]string{}
	var haveTime bool

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 256*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &head); err != nil {
			return fmt.Errorf("parse error on line %d: %w", lineNo, err)
		}

		switch head.Type {
		case "context-options":
			var ev struct {
				Title         string `json:"title"`
				MonotonicTime int64  `json:"monotonicTime"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				return fmt.Errorf("parse error on line %d: %w", lineNo, err)
			}
			r.Title = ev.Title
			haveTime = r.includeTime(ev.MonotonicTime, haveTime)
		case "screencast-frame":
			var ev struct {
				SHA1      string `json:"sha1"`
				Width     int    `json:"width"`
				Height    int    `json:"height"`
				Timestamp int64  `json:"timestamp"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				return fmt.Errorf("parse error on line %d: %w", lineNo, err)
			}
			if ev.SHA1 != "" {
				r.Frames = append(r.Frames, Frame{
					Timestamp: ev.Timestamp,
					SHA1:      ev.SHA1,
					Width:     ev.Width,
					Height:    ev.Height,
				})
			}
			haveTime = r.includeTime(ev.Timestamp, haveTime)
		case "frame-snapshot":
			var ev struct {
				Snapshot struct {
					SnapshotName      string `json:"snapshotName"`
					Timestamp         int64  `json:"timestamp"`
					ResourceOverrides []struct {
						SHA1 string `json:"sha1"`
					} `json:"resourceOverrides"`
				} `json:"snapshot"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				return fmt.Errorf("parse error on line %d: %w", lineNo, err)
			}
			if ev.Snapshot.SnapshotName != "" {
				if sha := r.firstImageOverride(ev.Snapshot.ResourceOverrides); sha != "" {
					snapshotImages[ev.Snapshot.SnapshotName] = sha
				}
			}
			haveTime = r.includeTime(ev.Snapshot.Timestamp, haveTime)
		case "before":
			var ev struct {
				CallID         string         `json:"callId"`
				ParentID       string         `json:"parentId"`
				BeforeSnapshot string         `json:"beforeSnapshot"`
				Title          string         `json:"title"`
				Class          string         `json:"class"`
				Method         string         `json:"method"`
				Params         map[string]any `json:"params"`
				StartTime      int64          `json:"startTime"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				return fmt.Errorf("parse error on line %d: %w", lineNo, err)
			}
			if ev.CallID == "" {
				continue
			}
			pa := &parseAction{
				action: Action{
					CallID:    ev.CallID,
					ParentID:  ev.ParentID,
					Title:     ev.Title,
					Method:    ev.Method,
					Params:    ev.Params,
					StartTime: ev.StartTime,
				},
				beforeSnapshotName: ev.BeforeSnapshot,
				ignored:            ev.Class == "BiDi",
			}
			if pa.action.Params == nil {
				pa.action.Params = map[string]any{}
			}
			actionsByID[ev.CallID] = pa
			if !pa.ignored {
				order = append(order, ev.CallID)
			}
			haveTime = r.includeTime(ev.StartTime, haveTime)
		case "after":
			var ev struct {
				CallID        string `json:"callId"`
				AfterSnapshot string `json:"afterSnapshot"`
				EndTime       int64  `json:"endTime"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				return fmt.Errorf("parse error on line %d: %w", lineNo, err)
			}
			if pa := actionsByID[ev.CallID]; pa != nil {
				pa.action.EndTime = ev.EndTime
				pa.afterSnapshotName = ev.AfterSnapshot
			}
			haveTime = r.includeTime(ev.EndTime, haveTime)
		case "input":
			var ev struct {
				CallID string `json:"callId"`
				Box    struct {
					X      float64 `json:"x"`
					Y      float64 `json:"y"`
					Width  float64 `json:"width"`
					Height float64 `json:"height"`
				} `json:"box"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				return fmt.Errorf("parse error on line %d: %w", lineNo, err)
			}
			if ev.CallID != "" {
				r.Boxes[ev.CallID] = Rect{
					X: int(math.Round(ev.Box.X)),
					Y: int(math.Round(ev.Box.Y)),
					W: int(math.Round(ev.Box.Width)),
					H: int(math.Round(ev.Box.Height)),
				}
			}
		case "event":
			var ev struct {
				Time int64 `json:"time"`
			}
			if err := json.Unmarshal(line, &ev); err == nil {
				haveTime = r.includeTime(ev.Time, haveTime)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("parse error reading trace: %w", err)
	}

	sort.SliceStable(r.Frames, func(i, j int) bool {
		return r.Frames[i].Timestamp < r.Frames[j].Timestamp
	})

	for _, id := range order {
		pa := actionsByID[id]
		if pa == nil || pa.ignored {
			continue
		}
		if sha := snapshotImages[pa.beforeSnapshotName]; sha != "" {
			pa.action.BeforeImageSHA = sha
		}
		if sha := snapshotImages[pa.afterSnapshotName]; sha != "" {
			pa.action.AfterImageSHA = sha
		}
		r.Actions = append(r.Actions, pa.action)
	}

	r.computeDepths()
	sort.SliceStable(r.Actions, func(i, j int) bool {
		return r.Actions[i].StartTime < r.Actions[j].StartTime
	})

	if !haveTime && (len(r.Frames) > 0 || len(r.Actions) > 0) {
		r.StartMs = earliestTime(r.Frames, r.Actions)
		r.EndMs = latestTime(r.Frames, r.Actions)
	}
	if r.EndMs < r.StartMs {
		r.EndMs = r.StartMs
	}
	return nil
}

func (r *Recording) includeTime(t int64, have bool) bool {
	if t == 0 && have {
		return true
	}
	if !have || t < r.StartMs {
		r.StartMs = t
	}
	if !have || t > r.EndMs {
		r.EndMs = t
	}
	return true
}

func (r *Recording) firstImageOverride(overrides []struct {
	SHA1 string `json:"sha1"`
}) string {
	var fallback string
	for _, o := range overrides {
		if o.SHA1 == "" {
			continue
		}
		if fallback == "" {
			fallback = o.SHA1
		}
		if isImageKey(o.SHA1) {
			return o.SHA1
		}
		if f := r.resources[o.SHA1]; f != nil && looksLikeImageFile(f) {
			return o.SHA1
		}
	}
	return fallback
}

func (r *Recording) computeDepths() {
	index := make(map[string]int, len(r.Actions))
	for i := range r.Actions {
		index[r.Actions[i].CallID] = i
	}

	var depthOf func(string, map[string]bool) int
	depthOf = func(id string, seen map[string]bool) int {
		i, ok := index[id]
		if !ok || seen[id] {
			return 0
		}
		seen[id] = true
		parent := r.Actions[i].ParentID
		if parent == "" {
			return 0
		}
		if _, ok := index[parent]; !ok {
			return 0
		}
		return depthOf(parent, seen) + 1
	}

	for i := range r.Actions {
		r.Actions[i].Depth = depthOf(r.Actions[i].CallID, map[string]bool{})
	}
}

func earliestTime(frames []Frame, actions []Action) int64 {
	var min int64
	have := false
	for _, f := range frames {
		if !have || f.Timestamp < min {
			min = f.Timestamp
			have = true
		}
	}
	for _, a := range actions {
		if !have || a.StartTime < min {
			min = a.StartTime
			have = true
		}
	}
	return min
}

func latestTime(frames []Frame, actions []Action) int64 {
	var max int64
	for _, f := range frames {
		if f.Timestamp > max {
			max = f.Timestamp
		}
	}
	for _, a := range actions {
		if end := effectiveEnd(a); end > max {
			max = end
		}
	}
	return max
}

func effectiveEnd(action Action) int64 {
	if action.EndTime > 0 {
		return action.EndTime
	}
	return action.StartTime
}

func contentType(name string, b []byte) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case len(b) >= 2 && b[0] == 0xff && b[1] == 0xd8:
		return "image/jpeg"
	case len(b) >= 4 && b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4e && b[3] == 0x47:
		return "image/png"
	default:
		return ""
	}
}

func isImageKey(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png")
}

func looksLikeImageFile(f *zip.File) bool {
	rc, err := f.Open()
	if err != nil {
		return false
	}
	defer rc.Close()
	var head [4]byte
	n, _ := io.ReadFull(rc, head[:])
	return contentType(f.Name, head[:n]) != ""
}
