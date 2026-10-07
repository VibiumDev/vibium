package api

import (
	"encoding/json"
	"strings"
	"sync"
)

// downloadRegistry tracks each download's completion, so clients await it
// over the wire (vibium:download.await) instead of each keeping a
// pending-downloads map per language (#446). Chrome 156 gives downloads
// their own id and nulls navigation on link-click downloads (#604), so each
// download is indexed under both its download id and its navigation id,
// whichever are present; Await answers to either handle. Completed entries
// are kept for the session's lifetime: path() can be asked again after
// delivery, and one browser session's download count stays small.
type downloadRegistry struct {
	mu     sync.Mutex
	states map[string]*downloadState
}

type downloadResult struct {
	Status   string
	Filepath string
}

type downloadState struct {
	done    bool
	result  downloadResult
	waiters []chan downloadResult
}

func newDownloadRegistry() *downloadRegistry {
	return &downloadRegistry{states: map[string]*downloadState{}}
}

// Observe records download progress from one raw browser event; anything
// that is not a download event is ignored.
func (dr *downloadRegistry) Observe(msg string) {
	var evt struct {
		Method string `json:"method"`
		Params struct {
			Download   string `json:"download"`
			Navigation string `json:"navigation"`
			Status     string `json:"status"`
			Filepath   string `json:"filepath"`
		} `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &evt) != nil {
		return
	}
	ids := make([]string, 0, 2)
	for _, id := range []string{evt.Params.Download, evt.Params.Navigation} {
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}

	switch evt.Method {
	case "browsingContext.downloadWillBegin":
		dr.mu.Lock()
		dr.stateFor(ids)
		dr.mu.Unlock()

	case "browsingContext.downloadEnd":
		status := evt.Params.Status
		if status == "" {
			status = "complete"
		}
		result := downloadResult{Status: status, Filepath: evt.Params.Filepath}

		dr.mu.Lock()
		st := dr.stateFor(ids)
		st.done = true
		st.result = result
		waiters := st.waiters
		st.waiters = nil
		dr.mu.Unlock()

		// Buffered channels: delivery cannot block the event loop.
		for _, ch := range waiters {
			ch <- result
		}
	}
}

// stateFor returns the one state shared by this download's ids, creating it
// if no id is known yet and indexing it under any ids seen for the first
// time. Callers hold dr.mu.
func (dr *downloadRegistry) stateFor(ids []string) *downloadState {
	var st *downloadState
	for _, id := range ids {
		if existing, ok := dr.states[id]; ok {
			st = existing
			break
		}
	}
	if st == nil {
		st = &downloadState{}
	}
	for _, id := range ids {
		dr.states[id] = st
	}
	return st
}

// normalizeDownloadEvent backfills a null or missing navigation field on
// download events with the download id before the event is forwarded.
// Chrome 156 nulls navigation on link-click downloads but mints a download
// id (#604); shipped clients pass params.navigation as their await handle,
// so the backfill keeps them working without a client release. Non-download
// events, and events already carrying both ids, pass through untouched.
func normalizeDownloadEvent(msg string) string {
	// Cheap reject before parsing: nearly every event is not a download.
	if !strings.Contains(msg, "browsingContext.download") {
		return msg
	}
	var evt struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &evt) != nil || evt.Params == nil {
		return msg
	}
	if evt.Method != "browsingContext.downloadWillBegin" && evt.Method != "browsingContext.downloadEnd" {
		return msg
	}
	download, _ := evt.Params["download"].(string)
	if navigation, _ := evt.Params["navigation"].(string); navigation != "" || download == "" {
		return msg
	}

	// Rewrite just the params object inside the original message so sibling
	// top-level fields (type, extensions) survive untouched.
	var full map[string]json.RawMessage
	if json.Unmarshal([]byte(msg), &full) != nil {
		return msg
	}
	evt.Params["navigation"] = download
	params, err := json.Marshal(evt.Params)
	if err != nil {
		return msg
	}
	full["params"] = params
	rewritten, err := json.Marshal(full)
	if err != nil {
		return msg
	}
	return string(rewritten)
}

// Await returns the finished result immediately, or a channel that delivers
// it when downloadEnd arrives. The handle is the download id or navigation
// id the client read off the willBegin event. known is false when the handle
// was never seen; it can only be wrong then, because the registry observed
// that event before it was forwarded.
func (dr *downloadRegistry) Await(navigation string) (result downloadResult, ch chan downloadResult, known bool) {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	st, ok := dr.states[navigation]
	if !ok {
		return downloadResult{}, nil, false
	}
	if st.done {
		return st.result, nil, true
	}
	ch = make(chan downloadResult, 1)
	st.waiters = append(st.waiters, ch)
	return downloadResult{}, ch, true
}
