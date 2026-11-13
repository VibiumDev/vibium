package api

import (
	"encoding/json"
	"testing"
)

const (
	willBegin = `{"method":"browsingContext.downloadWillBegin","params":{"context":"ctx1","navigation":"nav-1","url":"https://x.test/f.zip","suggestedFilename":"f.zip"}}`
	ended     = `{"method":"browsingContext.downloadEnd","params":{"context":"ctx1","navigation":"nav-1","status":"complete","filepath":"/tmp/dl/f.zip"}}`

	// Chrome 156 shape for a link-click download: navigation is null and the
	// download has its own id (#604).
	willBegin156 = `{"method":"browsingContext.downloadWillBegin","params":{"context":"ctx1","download":"dl-1","navigation":null,"url":"https://x.test/f.zip","suggestedFilename":"f.zip","userContext":"default"},"type":"event"}`
	ended156     = `{"method":"browsingContext.downloadEnd","params":{"context":"ctx1","download":"dl-1","navigation":null,"status":"complete","filepath":"/tmp/dl/f.zip","userContext":"default"},"type":"event"}`
)

// Download completion is awaited in the binary (#446): the registry answers
// immediately once the download ended, however often it is asked.
func TestDownloadRegistryAnswersAfterEnd(t *testing.T) {
	dr := newDownloadRegistry()
	dr.Observe(willBegin)
	dr.Observe(ended)

	for i := 0; i < 2; i++ {
		result, ch, known := dr.Await("nav-1")
		if !known || ch != nil {
			t.Fatalf("a finished download must answer immediately (known=%v ch=%v)", known, ch)
		}
		if result.Status != "complete" || result.Filepath != "/tmp/dl/f.zip" {
			t.Fatalf("result = %+v", result)
		}
	}
}

// A waiter that arrives while the download is in flight is delivered to when
// downloadEnd comes in.
func TestDownloadRegistryDeliversToWaiters(t *testing.T) {
	dr := newDownloadRegistry()
	dr.Observe(willBegin)

	_, ch, known := dr.Await("nav-1")
	if !known || ch == nil {
		t.Fatalf("an in-flight download must hand out a wait channel (known=%v)", known)
	}

	dr.Observe(ended)
	select {
	case result := <-ch:
		if result.Status != "complete" || result.Filepath != "/tmp/dl/f.zip" {
			t.Fatalf("result = %+v", result)
		}
	default:
		t.Fatal("downloadEnd must deliver to the pending waiter")
	}
}

// The client can only learn a navigation id from the willBegin event the
// registry observed first, so an unseen id is a caller error, not a race.
func TestDownloadRegistryRejectsUnknownNavigation(t *testing.T) {
	dr := newDownloadRegistry()
	if _, _, known := dr.Await("nav-nope"); known {
		t.Fatal("an unseen navigation id must not be awaitable")
	}
	dr.Observe(`{"method":"browsingContext.downloadWillBegin","params":{"navigation":""}}`)
	if _, _, known := dr.Await(""); known {
		t.Fatal("an empty navigation id must not create state")
	}
}

// Chrome 156 nulls navigation on link-click downloads and mints a download
// id instead (#604): the registry answers to the download id.
func TestDownloadRegistryKeysByDownloadID(t *testing.T) {
	dr := newDownloadRegistry()
	dr.Observe(willBegin156)

	_, ch, known := dr.Await("dl-1")
	if !known || ch == nil {
		t.Fatalf("an in-flight download must be awaitable by its download id (known=%v)", known)
	}

	dr.Observe(ended156)
	select {
	case result := <-ch:
		if result.Status != "complete" || result.Filepath != "/tmp/dl/f.zip" {
			t.Fatalf("result = %+v", result)
		}
	default:
		t.Fatal("downloadEnd must deliver to the pending waiter")
	}
}

// Chrome 156 direct-navigation downloads carry both ids; a client may have
// read either off the event, so both must await the same download.
func TestDownloadRegistrySharesStateAcrossIDs(t *testing.T) {
	dr := newDownloadRegistry()
	dr.Observe(`{"method":"browsingContext.downloadWillBegin","params":{"context":"ctx1","download":"dl-1","navigation":"nav-1","url":"https://x.test/f.zip","suggestedFilename":"f.zip"}}`)
	dr.Observe(`{"method":"browsingContext.downloadEnd","params":{"context":"ctx1","download":"dl-1","navigation":"nav-1","status":"complete","filepath":"/tmp/dl/f.zip"}}`)

	for _, id := range []string{"dl-1", "nav-1"} {
		result, ch, known := dr.Await(id)
		if !known || ch != nil {
			t.Fatalf("id %q must answer immediately (known=%v ch=%v)", id, known, ch)
		}
		if result.Status != "complete" || result.Filepath != "/tmp/dl/f.zip" {
			t.Fatalf("id %q result = %+v", id, result)
		}
	}
}

// Shipped clients read params.navigation as their await handle, so the 156
// null is backfilled with the download id before the event is forwarded.
func TestNormalizeDownloadEventBackfillsNavigation(t *testing.T) {
	for _, msg := range []string{willBegin156, ended156} {
		out := normalizeDownloadEvent(msg)
		var evt struct {
			Type   string `json:"type"`
			Params struct {
				Navigation string `json:"navigation"`
			} `json:"params"`
		}
		if err := json.Unmarshal([]byte(out), &evt); err != nil {
			t.Fatalf("rewritten event is not JSON: %v", err)
		}
		if evt.Params.Navigation != "dl-1" {
			t.Fatalf("navigation = %q, want the download id", evt.Params.Navigation)
		}
		if evt.Type != "event" {
			t.Fatalf("sibling top-level fields must survive the rewrite, got type=%q", evt.Type)
		}
	}
}

// Events that already carry a navigation id, and non-download events, pass
// through byte-identical: nothing downstream should see a rewrite.
func TestNormalizeDownloadEventLeavesOthersAlone(t *testing.T) {
	for _, msg := range []string{
		willBegin,
		ended,
		`{"method":"browsingContext.downloadWillBegin","params":{"context":"ctx1","download":"dl-1","navigation":"nav-1","url":"https://x.test/f.zip"}}`,
		`{"method":"log.entryAdded","params":{"text":"browsingContext.downloadWillBegin mentioned in a log"}}`,
		`not json at all`,
	} {
		if out := normalizeDownloadEvent(msg); out != msg {
			t.Fatalf("message rewritten:\n in: %s\nout: %s", msg, out)
		}
	}
}

// A failed download reports its status instead of hanging or inventing a path.
func TestDownloadRegistryReportsFailure(t *testing.T) {
	dr := newDownloadRegistry()
	dr.Observe(willBegin)
	dr.Observe(`{"method":"browsingContext.downloadEnd","params":{"navigation":"nav-1","status":"canceled"}}`)

	result, ch, known := dr.Await("nav-1")
	if !known || ch != nil {
		t.Fatal("a finished download must answer immediately")
	}
	if result.Status != "canceled" || result.Filepath != "" {
		t.Fatalf("result = %+v", result)
	}
}
