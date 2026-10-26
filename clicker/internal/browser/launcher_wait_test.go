package browser

import (
	"net"
	"sync"
	"testing"
	"time"
)

// A chromedriver that accepts the TCP connection but never answers must not
// hang waitForChromedriver past its deadline: the probe used the default HTTP
// client with no timeout, so one wedged Get blocked forever (#725). The
// listener here accepts and holds connections without ever writing a response.
func TestWaitForChromedriverBoundedOnWedgedEndpoint(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var conns []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			c.Close()
		}
	})

	done := make(chan error, 1)
	go func() {
		done <- waitForChromedriver("http://"+ln.Addr().String(), 500*time.Millisecond)
	}()

	// Generous bound: the 500ms deadline plus at most one full 2s probe.
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected a timeout error from a wedged endpoint")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("waitForChromedriver hung past its deadline on a wedged endpoint")
	}
}
