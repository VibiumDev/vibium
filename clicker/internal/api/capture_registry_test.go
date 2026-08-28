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
