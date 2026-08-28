package api

import (
	"strings"
	"testing"
)

// One-shot captures are matched and delivered in the binary (#446).
func TestCaptureRegistry(t *testing.T) {
	cr := newCaptureRegistry()

	id, ch, err := cr.register("network.responseCompleted", "ctx1", "**/api/**")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("wrong kind, context, and URL do not deliver", func(t *testing.T) {
		cr.offer(`{"method":"network.beforeRequestSent","params":{"context":"ctx1","request":{"url":"https://x.test/api/a"}}}`)
		cr.offer(`{"method":"network.responseCompleted","params":{"context":"ctx2","request":{"url":"https://x.test/api/a"}}}`)
		cr.offer(`{"method":"network.responseCompleted","params":{"context":"ctx1","request":{"url":"https://x.test/other"}}}`)
		select {
		case <-ch:
			t.Fatal("nothing should have been delivered")
		default:
		}
	})

	t.Run("matching event delivers params once", func(t *testing.T) {
		cr.offer(`{"method":"network.responseCompleted","params":{"context":"ctx1","request":{"url":"https://x.test/api/a"},"response":{"status":200}}}`)
		params := <-ch
		if !strings.Contains(string(params), `"status":200`) {
			t.Fatalf("params = %s", params)
		}
		// One-shot: a second matching event has no listener left.
		cr.offer(`{"method":"network.responseCompleted","params":{"context":"ctx1","request":{"url":"https://x.test/api/b"}}}`)
		select {
		case <-ch:
			t.Fatal("capture must be one-shot")
		default:
		}
	})

	t.Run("blocked requests do not satisfy request captures", func(t *testing.T) {
		_, ch2, _ := cr.register("network.beforeRequestSent", "ctx1", "**")
		cr.offer(`{"method":"network.beforeRequestSent","params":{"context":"ctx1","isBlocked":true,"request":{"url":"https://x.test/a"}}}`)
		select {
		case <-ch2:
			t.Fatal("blocked request belongs to routes, not captures")
		default:
		}
		cr.offer(`{"method":"network.beforeRequestSent","params":{"context":"ctx1","request":{"url":"https://x.test/a"}}}`)
		<-ch2
	})

	t.Run("cancel removes a pending capture", func(t *testing.T) {
		id2, ch3, _ := cr.register("network.responseCompleted", "ctx1", "**")
		cr.cancel(id2)
		cr.offer(`{"method":"network.responseCompleted","params":{"context":"ctx1","request":{"url":"https://x.test/a"}}}`)
		select {
		case <-ch3:
			t.Fatal("cancelled capture must not deliver")
		default:
		}
	})

	_ = id
}

// vibium:page.captureEvent kinds: each waits for its own event shape, scoped
// to one browsing context (#446).
func TestCaptureEventKinds(t *testing.T) {
	deliver := func(t *testing.T, kind, context string, miss, hit []string) string {
		t.Helper()
		cr := newCaptureRegistry()
		capture, err := captureEventKind(kind, context)
		if err != nil {
			t.Fatal(err)
		}
		_, ch := cr.add(capture)
		for _, msg := range miss {
			cr.offer(msg)
		}
		select {
		case params := <-ch:
			t.Fatalf("non-matching event delivered: %s", params)
		default:
		}
		for _, msg := range hit {
			cr.offer(msg)
		}
		select {
		case params := <-ch:
			return string(params)
		default:
			t.Fatal("matching event was not delivered")
			return ""
		}
	}

	t.Run("navigation matches load, fragment, and history for its context", func(t *testing.T) {
		got := deliver(t, "navigation", "ctx1",
			[]string{
				`{"method":"browsingContext.load","params":{"context":"ctx2","url":"https://x.test/other"}}`,
				`{"method":"browsingContext.navigationStarted","params":{"context":"ctx1","url":"https://x.test/a"}}`,
				`{"method":"browsingContext.load","params":{"context":"ctx1"}}`,
			},
			[]string{`{"method":"browsingContext.fragmentNavigated","params":{"context":"ctx1","url":"https://x.test/a#frag"}}`},
		)
		if !strings.Contains(got, "#frag") {
			t.Fatalf("params = %s", got)
		}
	})

	t.Run("dialog matches userPromptOpened for its context", func(t *testing.T) {
		got := deliver(t, "dialog", "ctx1",
			[]string{`{"method":"browsingContext.userPromptOpened","params":{"context":"ctx2","type":"alert"}}`},
			[]string{`{"method":"browsingContext.userPromptOpened","params":{"context":"ctx1","type":"confirm","message":"go?"}}`},
		)
		if !strings.Contains(got, "go?") {
			t.Fatalf("params = %s", got)
		}
	})

	t.Run("console and error split log.entryAdded by type", func(t *testing.T) {
		consoleEntry := `{"method":"log.entryAdded","params":{"type":"console","method":"warn","text":"careful","source":{"context":"ctx1"}}}`
		errorEntry := `{"method":"log.entryAdded","params":{"type":"javascript","text":"boom","source":{"context":"ctx1"}}}`
		got := deliver(t, "console", "ctx1", []string{errorEntry}, []string{consoleEntry})
		if !strings.Contains(got, "careful") {
			t.Fatalf("params = %s", got)
		}
		got = deliver(t, "error", "ctx1", []string{consoleEntry}, []string{errorEntry})
		if !strings.Contains(got, "boom") {
			t.Fatalf("params = %s", got)
		}
	})

	t.Run("unknown kind errors", func(t *testing.T) {
		if _, err := captureEventKind("websocket", "ctx1"); err == nil {
			t.Fatal("unknown kind must error")
		}
	})
}

// A pending dialog capture claims its context's prompts, so the router's
// auto-dismiss default leaves the captured dialog open.
func TestDialogCaptureClaimsPrompt(t *testing.T) {
	cr := newCaptureRegistry()
	if cr.wantsPrompt("ctx1") {
		t.Fatal("no capture, no claim")
	}

	capture, err := captureEventKind("dialog", "ctx1")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := cr.add(capture)

	if !cr.wantsPrompt("ctx1") {
		t.Fatal("a pending dialog capture must claim its context's prompts")
	}
	if cr.wantsPrompt("ctx2") {
		t.Fatal("the claim is scoped to the capture's context")
	}

	cr.cancel(id)
	if cr.wantsPrompt("ctx1") {
		t.Fatal("a cancelled capture must release its claim")
	}

	// Non-dialog captures never claim prompts.
	nav, _ := captureEventKind("navigation", "ctx1")
	cr.add(nav)
	if cr.wantsPrompt("ctx1") {
		t.Fatal("only dialog captures claim prompts")
	}
}
