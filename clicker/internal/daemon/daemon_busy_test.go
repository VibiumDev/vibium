package daemon

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// A long run or check holds d.mu for its whole duration (#855).
func busyDaemon(t *testing.T) *Daemon {
	t.Helper()
	withTempCacheDir(t)
	d := New(Options{Version: "test"})
	d.mu.Lock()
	return d
}

func TestStatusAnswersWhileARequestHoldsTheLock(t *testing.T) {
	d := busyDaemon(t)
	client, server := net.Pipe()
	defer client.Close()
	go d.handleConnection(server)

	client.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := client.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"daemon/status"}` + "\n")); err != nil {
		t.Fatalf("status request was not read: %v", err)
	}
	line, err := bufio.NewReader(client).ReadString('\n')
	if err != nil || !strings.Contains(line, `"version":"test"`) {
		t.Fatalf("no status while a request was running: %q, %v", line, err)
	}
}

func TestShutdownFinishesWhileARequestHoldsTheLock(t *testing.T) {
	d := busyDaemon(t)
	d.drainTimeout = 10 * time.Millisecond
	d.wg.Add(1)
	done := make(chan struct{})
	go func() {
		d.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown waited for the running request to release the lock")
	}
}

func TestARunningRequestIsNotIdle(t *testing.T) {
	d := New(Options{Version: "test"})
	d.lastActivity = time.Now().Add(-time.Hour)
	if idle := d.idleFor(); idle < time.Hour {
		t.Fatalf("idle = %s, want at least an hour", idle)
	}
	d.trackRequest(1)
	d.lastActivity = time.Now().Add(-time.Hour)
	if idle := d.idleFor(); idle != 0 {
		t.Fatalf("idle = %s during a request, want 0", idle)
	}
	d.trackRequest(-1)
	if idle := d.idleFor(); idle > time.Minute {
		t.Fatalf("idle = %s right after a request ended", idle)
	}
}
