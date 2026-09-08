package api

import (
	"encoding/json"
	"sync"
)

// downloadRegistry tracks each download's completion by its navigation id,
// so clients await it over the wire (vibium:download.await) instead of each
// keeping a pending-downloads map per language (#446). Completed entries are
// kept for the session's lifetime: path() can be asked again after delivery,
// and one browser session's download count stays small.
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
			Navigation string `json:"navigation"`
			Status     string `json:"status"`
			Filepath   string `json:"filepath"`
		} `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &evt) != nil || evt.Params.Navigation == "" {
		return
	}

	switch evt.Method {
	case "browsingContext.downloadWillBegin":
		dr.mu.Lock()
		if _, ok := dr.states[evt.Params.Navigation]; !ok {
			dr.states[evt.Params.Navigation] = &downloadState{}
		}
		dr.mu.Unlock()

	case "browsingContext.downloadEnd":
		status := evt.Params.Status
		if status == "" {
			status = "complete"
		}
		result := downloadResult{Status: status, Filepath: evt.Params.Filepath}

		dr.mu.Lock()
		st, ok := dr.states[evt.Params.Navigation]
		if !ok {
			st = &downloadState{}
			dr.states[evt.Params.Navigation] = st
		}
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

// Await returns the finished result immediately, or a channel that delivers
// it when downloadEnd arrives. known is false when this navigation id was
// never seen; the id can only be wrong then, because the client learned it
// from the willBegin event this registry observed before it was forwarded.
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
