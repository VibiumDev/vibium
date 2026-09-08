package api

import "testing"

const (
	willBegin = `{"method":"browsingContext.downloadWillBegin","params":{"context":"ctx1","navigation":"nav-1","url":"https://x.test/f.zip","suggestedFilename":"f.zip"}}`
	ended     = `{"method":"browsingContext.downloadEnd","params":{"context":"ctx1","navigation":"nav-1","status":"complete","filepath":"/tmp/dl/f.zip"}}`
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
