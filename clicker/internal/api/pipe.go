package api

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// PipeClientConn implements ClientTransport over stdin/stdout pipes.
//
// Writes are decoupled from the callers through an unbounded queue drained
// by a single writer goroutine that flushes with no lock held. Send never
// blocks and never drops: a blocking Send would stall the browser-to-client
// routing goroutine (freezing command responses and event delivery for the
// whole session), and a dropped response or event permanently hangs the
// client command or paused request waiting on it.
type PipeClientConn struct {
	writer *bufio.Writer
	mu     sync.Mutex // guards queue, closed
	cond   *sync.Cond
	queue  []string
	closed bool
	done   chan struct{}

	flushStart int64 // unix nanos of in-progress flush, 0 = idle (guarded by mu)
}

// NewPipeClientConn creates a PipeClientConn that writes protocol messages to w.
func NewPipeClientConn(w io.Writer) *PipeClientConn {
	c := &PipeClientConn{
		writer: bufio.NewWriter(w),
		done:   make(chan struct{}),
	}
	c.cond = sync.NewCond(&c.mu)
	go c.writeLoop()
	go c.watchStalledFlush()
	return c
}

// writeLoop drains the queue and writes to the pipe. Flushing happens with
// the lock released so a stalled reader never blocks Send callers.
func (c *PipeClientConn) writeLoop() {
	defer close(c.done)
	for {
		c.mu.Lock()
		for len(c.queue) == 0 && !c.closed {
			c.cond.Wait()
		}
		if len(c.queue) == 0 && c.closed {
			c.mu.Unlock()
			return
		}
		batch := c.queue
		c.queue = nil
		c.flushStart = time.Now().UnixNano()
		c.mu.Unlock()

		var err error
		for _, msg := range batch {
			if _, err = c.writer.WriteString(msg); err != nil {
				break
			}
			if err = c.writer.WriteByte('\n'); err != nil {
				break
			}
		}
		if err == nil {
			err = c.writer.Flush()
		}

		c.mu.Lock()
		c.flushStart = 0
		if err != nil {
			// The pipe is gone (client exited) — stop accepting messages so
			// the queue can't grow unboundedly against a dead reader.
			c.closed = true
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
	}
}

// watchStalledFlush periodically reports a flush that is currently blocked —
// the signature of a client that stopped reading its end of the pipe.
func (c *PipeClientConn) watchStalledFlush() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			c.mu.Lock()
			start := c.flushStart
			depth := len(c.queue)
			c.mu.Unlock()
			if start != 0 {
				if blocked := time.Since(time.Unix(0, start)); blocked > 5*time.Second {
					fmt.Fprintf(os.Stderr, "[pipe] write blocked for %.0fs (%d messages queued) — client not reading\n",
						blocked.Seconds(), depth)
				}
			}
		}
	}
}

// ID returns a fixed client ID (pipe mode supports exactly one client).
func (c *PipeClientConn) ID() uint64 { return 1 }

// Send queues a JSON message for writing to the pipe. It never blocks and
// never drops messages.
func (c *PipeClientConn) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("pipe closed")
	}
	c.queue = append(c.queue, msg)
	c.cond.Signal()
	return nil
}

// Close marks the pipe as closed and waits for queued messages to be written.
func (c *PipeClientConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.cond.Signal()
	c.mu.Unlock()
	<-c.done
	return nil
}
