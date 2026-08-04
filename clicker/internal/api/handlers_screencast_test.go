package api

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestScreencastSupportErrorExplainsFirefoxOutputFailure(t *testing.T) {
	err := screencastSupportError(errors.New(
		`unknown error: NS_ERROR_FAILURE [nsIProperties.get]`,
	))
	if !strings.Contains(err.Error(), "could not resolve its screencast output directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScreencastOperationsAreSerialized(t *testing.T) {
	session := &BrowserSession{}
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		session.screencastMu.Lock()
		defer session.screencastMu.Unlock()
		close(firstEntered)
		<-releaseFirst
	}()
	<-firstEntered

	go func() {
		defer wg.Done()
		session.screencastMu.Lock()
		defer session.screencastMu.Unlock()
		close(secondEntered)
	}()

	select {
	case <-secondEntered:
		t.Fatal("concurrent screencast operation entered before the active operation completed")
	case <-time.After(25 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case <-secondEntered:
	case <-time.After(time.Second):
		t.Fatal("waiting screencast operation did not proceed")
	}
	wg.Wait()
}

func TestClosingSessionRejectsQueuedScreencastOperation(t *testing.T) {
	session := &BrowserSession{}
	if !session.beginScreencastOperation() {
		t.Fatal("first operation was unexpectedly rejected")
	}

	session.mu.Lock()
	session.closed = true
	session.mu.Unlock()
	session.endScreencastOperation()

	if session.beginScreencastOperation() {
		session.endScreencastOperation()
		t.Fatal("operation was accepted after session shutdown began")
	}
}
