package daemon

import (
	"bufio"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/vibium/clicker/internal/paths"
)

// fakeDaemon listens on the session socket and serves one connection with the
// given script. Returns after the connection is handled.
func fakeDaemon(t *testing.T, serve func(conn net.Conn)) {
	t.Helper()

	socketPath, err := paths.GetSocketPath()
	if err != nil {
		t.Fatalf("get socket path: %v", err)
	}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		serve(conn)
	}()
}

// shrinkTimeouts makes the package deadlines hermetic-test sized and restores
// them afterwards.
func shrinkTimeouts(t *testing.T, read, grace time.Duration) {
	t.Helper()
	origRead, origGrace := readTimeout, launchGrace
	readTimeout, launchGrace = read, grace
	t.Cleanup(func() { readTimeout, launchGrace = origRead, origGrace })
}

func setupSocketDir(t *testing.T) {
	t.Helper()
	// Keep the socket path short: sun_path caps around 104 bytes on macOS.
	dir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", dir)
	t.Setenv("VIBIUM_SESSION", "")
	if len(filepath.Join(dir, "vibium.sock")) > 100 {
		t.Skip("temp dir too long for a unix socket path")
	}
}

const okResponse = `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]}}`

func readRequest(t *testing.T, conn net.Conn) {
	t.Helper()
	if _, err := bufio.NewReader(conn).ReadBytes('\n'); err != nil {
		t.Errorf("read request: %v", err)
	}
}

// A response that arrives after readTimeout but within the extended deadline
// succeeds when the daemon announced a launch first. On a client without the
// notification protocol this fails twice over: the notification line is
// mistaken for the response, and the deadline is never extended.
func TestLaunchNotificationExtendsReadDeadline(t *testing.T) {
	setupSocketDir(t)
	shrinkTimeouts(t, 200*time.Millisecond, 2*time.Second)

	fakeDaemon(t, func(conn net.Conn) {
		readRequest(t, conn)
		fmt.Fprintf(conn, "{\"jsonrpc\":\"2.0\",\"method\":%q}\n", launchingBrowserMethod)
		time.Sleep(400 * time.Millisecond) // past readTimeout, inside the grace
		fmt.Fprintln(conn, okResponse)
	})

	result, err := Call("browser_title", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// Without a launch notification the plain deadline still applies, so a daemon
// that goes silent is reported at readTimeout, not readTimeout+grace.
func TestNoNotificationKeepsPlainDeadline(t *testing.T) {
	setupSocketDir(t)
	shrinkTimeouts(t, 200*time.Millisecond, 10*time.Second)

	done := make(chan struct{})
	fakeDaemon(t, func(conn net.Conn) {
		readRequest(t, conn)
		<-done // never respond
	})
	defer close(done)

	start := time.Now()
	_, err := Call("browser_title", nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("timed out after %v; the deadline was extended without a notification", elapsed)
	}
}

// A silent daemon that did announce a launch is still bounded: the extension
// is the launch grace, not forever.
func TestExtendedDeadlineStillBounds(t *testing.T) {
	setupSocketDir(t)
	shrinkTimeouts(t, 100*time.Millisecond, 300*time.Millisecond)

	done := make(chan struct{})
	fakeDaemon(t, func(conn net.Conn) {
		readRequest(t, conn)
		fmt.Fprintf(conn, "{\"jsonrpc\":\"2.0\",\"method\":%q}\n", launchingBrowserMethod)
		<-done // never respond
	})
	defer close(done)

	start := time.Now()
	_, err := Call("browser_title", nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed >= 3*time.Second {
		t.Fatalf("timed out after %v; extension should be bounded by the grace", elapsed)
	}
}

// Unknown notifications are skipped without extending the deadline, so a
// future daemon cannot silently disable the wedge bound on old clients.
func TestUnknownNotificationSkippedWithoutExtension(t *testing.T) {
	setupSocketDir(t)
	shrinkTimeouts(t, 300*time.Millisecond, 10*time.Second)

	fakeDaemon(t, func(conn net.Conn) {
		readRequest(t, conn)
		fmt.Fprintln(conn, `{"jsonrpc":"2.0","method":"daemon/somethingElse"}`)
		fmt.Fprintln(conn, okResponse)
	})

	result, err := Call("browser_title", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
