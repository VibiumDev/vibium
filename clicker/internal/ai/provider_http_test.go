package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type providerReply struct {
	status     int
	retryAfter string
	body       string
}

func TestPostRetriesTransientProviderErrors(t *testing.T) {
	const quota = `{"error":{"code":"insufficient_quota"}}`
	const limited = `{"error":{"code":"rate_limit_exceeded"}}`
	half, full := retryBackoff/2, retryBackoff
	for _, row := range []struct {
		name      string
		replies   []providerReply
		cancelled bool
		requests  int
		waits     [][2]time.Duration
		wantErr   string
	}{
		{name: "rate limit then success", replies: []providerReply{{status: 429, body: limited}, {status: 200}}, requests: 2, waits: [][2]time.Duration{{half, full}}},
		{name: "two server errors then success", replies: []providerReply{{status: 503}, {status: 529}, {status: 200}}, requests: 3, waits: [][2]time.Duration{{half, full}, {full, 2 * full}}},
		{name: "request timeout then success", replies: []providerReply{{status: 408}, {status: 200}}, requests: 2, waits: [][2]time.Duration{{half, full}}},
		{name: "retry-after sets the wait", replies: []providerReply{{status: 429, retryAfter: "7"}, {status: 200}}, requests: 2, waits: [][2]time.Duration{{7 * time.Second, 7 * time.Second}}},
		{name: "unparseable retry-after backs off", replies: []providerReply{{status: 429, retryAfter: "soon"}, {status: 200}}, requests: 2, waits: [][2]time.Duration{{half, full}}},
		{name: "gives up after the last attempt", replies: []providerReply{{status: 500}, {status: 500}, {status: 500}, {status: 200}}, requests: 3, waits: [][2]time.Duration{{half, full}, {full, 2 * full}}, wantErr: "verifier provider returned HTTP 500 after 3 attempts"},
		{name: "retry-after past the cap fails fast", replies: []providerReply{{status: 429, retryAfter: "3600", body: limited}, {status: 200}}, requests: 1, wantErr: "verifier provider returned HTTP 429 (rate_limit_exceeded)"},
		{name: "cancelled wait reports the provider error", replies: []providerReply{{status: 503}, {status: 200}}, cancelled: true, requests: 1, waits: [][2]time.Duration{{half, full}}, wantErr: "verifier provider returned HTTP 503"},
		{name: "exhausted quota is not retried", replies: []providerReply{{status: 429, body: quota}, {status: 200}}, requests: 1, wantErr: "verifier provider returned HTTP 429 (insufficient_quota)"},
		{name: "auth failure is not retried", replies: []providerReply{{status: 401}, {status: 200}}, requests: 1, wantErr: "verifier provider returned HTTP 401"},
		{name: "bad request is not retried", replies: []providerReply{{status: 400}, {status: 200}}, requests: 1, wantErr: "verifier provider returned HTTP 400"},
	} {
		t.Run(row.name, func(t *testing.T) {
			var bodies []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				bodies = append(bodies, string(body))
				reply := row.replies[len(bodies)-1]
				if reply.retryAfter != "" {
					w.Header().Set("Retry-After", reply.retryAfter)
				}
				w.WriteHeader(reply.status)
				if reply.status == 200 {
					reply.body = `{"ok":true}`
				}
				fmt.Fprint(w, reply.body)
			}))
			defer server.Close()
			var waits []time.Duration
			model := &Model{sleep: func(_ context.Context, wait time.Duration) error {
				waits = append(waits, wait)
				if row.cancelled {
					return context.Canceled
				}
				return nil
			}}
			data, _, err := model.post(context.Background(), server.URL, map[string]string{"turn": "one"}, nil)
			if row.wantErr == "" && (err != nil || string(data) != `{"ok":true}`) {
				t.Fatalf("want success, got %q, %v", data, err)
			}
			if row.wantErr != "" && (err == nil || err.Error() != row.wantErr) {
				t.Fatalf("want error %q, got %v", row.wantErr, err)
			}
			if len(bodies) != row.requests {
				t.Fatalf("want %d requests, got %d", row.requests, len(bodies))
			}
			for _, body := range bodies {
				if !strings.Contains(body, `"turn":"one"`) {
					t.Fatalf("a retry did not resend the payload: %q", body)
				}
			}
			if len(waits) != len(row.waits) {
				t.Fatalf("want %d waits, got %v", len(row.waits), waits)
			}
			for i, bounds := range row.waits {
				if waits[i] < bounds[0] || waits[i] > bounds[1] {
					t.Fatalf("wait %d = %s, want %s to %s", i, waits[i], bounds[0], bounds[1])
				}
			}
		})
	}
}
