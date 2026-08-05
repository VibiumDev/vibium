package daemon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/paths"
)

// Daemon manages a long-lived browser session accessible via a Unix socket.
type Daemon struct {
	listener     net.Listener
	handlers     *agent.Handlers
	mu           sync.Mutex     // serializes handler access
	wg           sync.WaitGroup // tracks in-flight handler goroutines
	version      string
	startTime    time.Time
	lastActivity time.Time
	idleTimeout  time.Duration
	socketPath   string
	shutdownOnce sync.Once
	done         chan struct{} // signals shutdown started
	shutdownDone chan struct{} // closed when shutdown is fully complete
}

// Options configures a new Daemon.
type Options struct {
	Version        string
	ScreenshotDir  string
	Engine         string // Browser to launch: "chrome" (default) or "firefox"
	Headless       bool
	IdleTimeout    time.Duration
	ConnectURL     string                 // Remote BiDi WebSocket URL (empty = local browser)
	ConnectHeaders http.Header            // Headers for remote WebSocket connection
	ConnectCaps    map[string]interface{} // Extra alwaysMatch capabilities for classic endpoints
}

// New creates a new Daemon instance.
func New(opts Options) *Daemon {
	return &Daemon{
		handlers:     agent.NewHandlers(opts.ScreenshotDir, opts.Engine, opts.Headless, opts.ConnectURL, opts.ConnectHeaders, opts.ConnectCaps),
		version:      opts.Version,
		idleTimeout:  opts.IdleTimeout,
		startTime:    time.Now(),
		lastActivity: time.Now(),
		done:         make(chan struct{}),
		shutdownDone: make(chan struct{}),
	}
}

// Run starts the daemon, listening for connections until the context is cancelled.
func (d *Daemon) Run(ctx context.Context) error {
	socketPath, err := paths.GetSocketPath()
	if err != nil {
		return fmt.Errorf("get socket path: %w", err)
	}
	d.socketPath = socketPath

	// Ensure parent directory exists
	dir, err := paths.GetDaemonDir()
	if err != nil {
		return fmt.Errorf("get daemon dir: %w", err)
	}
	if err := ensureDaemonDir(dir); err != nil {
		return err
	}

	// Remove stale socket file
	os.Remove(socketPath)

	listener, err := listen(socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	d.listener = listener
	if err := secureSocketPath(socketPath); err != nil {
		listener.Close()
		return err
	}

	// Write PID file
	if err := WritePID(); err != nil {
		listener.Close()
		return fmt.Errorf("write PID: %w", err)
	}

	log.Debug("daemon started", "socket", socketPath, "pid", os.Getpid())

	// Start idle timeout watcher if configured
	if d.idleTimeout > 0 {
		go d.watchIdle(ctx)
	}

	// Accept connections until context cancelled
	go func() {
		<-ctx.Done()
		d.Shutdown()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-d.done:
				<-d.shutdownDone // Wait for full shutdown (browser cleanup)
				return nil
			default:
				log.Debug("accept error", "error", err)
				continue
			}
		}
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.handleConnection(conn)
		}()
	}
}

// Shutdown performs a clean daemon shutdown.
func (d *Daemon) Shutdown() {
	d.shutdownOnce.Do(func() {
		log.Debug("daemon shutting down")
		defer close(d.shutdownDone)
		close(d.done)

		// Stop accepting new connections
		if d.listener != nil {
			d.listener.Close()
		}

		// Wait for in-flight handlers to finish (with timeout)
		waitDone := make(chan struct{})
		go func() { d.wg.Wait(); close(waitDone) }()
		select {
		case <-waitDone:
		case <-time.After(10 * time.Second):
			log.Debug("timed out waiting for in-flight handlers to finish")
		}

		// Now safe to close browser session
		d.mu.Lock()
		d.handlers.Close()
		d.mu.Unlock()

		// Clean up socket file
		if d.socketPath != "" {
			os.Remove(d.socketPath)
		}

		// Remove PID file
		RemovePID()
	})
}

// touchActivity updates the last activity timestamp.
func (d *Daemon) touchActivity() {
	d.mu.Lock()
	d.lastActivity = time.Now()
	d.mu.Unlock()
}

// watchIdle monitors for idle timeout and triggers shutdown.
func (d *Daemon) watchIdle(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.mu.Lock()
			idle := time.Since(d.lastActivity)
			d.mu.Unlock()

			if idle >= d.idleTimeout {
				log.Debug("idle timeout reached, shutting down", "idle", idle, "timeout", d.idleTimeout)
				d.Shutdown()
				return
			}
		case <-d.done:
			return
		case <-ctx.Done():
			return
		}
	}
}
