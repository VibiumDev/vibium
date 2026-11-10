package watch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vibium/clicker/internal/watch/watchtest"
)

func record(t *testing.T, src Source) Info {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rec, err := Start(ctx, src, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	time.Sleep(700 * time.Millisecond)
	info, err := rec.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != watchtest.Width || info.Height != watchtest.Height {
		t.Errorf("size %dx%d, want %dx%d", info.Width, info.Height, watchtest.Width, watchtest.Height)
	}
	// 0.7 s at 30 frames a second, give or take a slow machine.
	if info.Frames < 10 || info.Frames > 40 {
		t.Errorf("%d frames in 0.7 s", info.Frames)
	}
	if info.DurationMs < 600 || info.DurationMs > 1500 {
		t.Errorf("duration %d ms for 0.7 s", info.DurationMs)
	}
	if info.StartedAt.After(began) || began.Sub(info.StartedAt) > 2*time.Second {
		t.Errorf("started %s before Start returned", began.Sub(info.StartedAt))
	}
	if info.Ended != "" || info.Gaps != 0 {
		t.Errorf("ended %q, gaps %d", info.Ended, info.Gaps)
	}
	data, err := os.ReadFile(info.Path)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]byte{}
	boxes(t, data, "", got)
	if key, sps, pps := inspect(got["mdat"]); !key || sps == nil || pps == nil {
		t.Error("the file does not start with a key frame and its headers")
	}
	return info
}

func TestRecordsARoadiesStream(t *testing.T) {
	srv, rig := watchtest.Roadie(t)
	info := record(t, Source{BaseURL: srv.URL, Header: http.Header{"Authorization": {"Bearer local"}}})
	if grants, auth := rig.Viewers(); len(auth) != 1 || auth[0] != "Bearer local" || grants[0] != "" {
		t.Errorf("hello with grant %q and %q", grants, auth)
	}

	// Where ffprobe is installed, a second opinion on the file.
	t.Run("ffprobe", func(t *testing.T) {
		probe, err := exec.LookPath("ffprobe")
		if err != nil {
			t.Skip("no ffprobe")
		}
		out, err := exec.Command(probe, "-v", "error", "-count_frames", "-select_streams", "v:0",
			"-show_entries", "stream=codec_name,width,height,nb_read_frames", "-of", "csv=p=0", info.Path).CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		fields := strings.Split(strings.TrimSpace(string(out)), ",")
		if len(fields) != 4 || fields[0] != "h264" || fields[1] != "100" || fields[2] != "60" {
			t.Fatalf("ffprobe says %q", out)
		}
		if want := strconv.Itoa(info.Frames); fields[3] != want {
			t.Errorf("ffprobe decoded %s frames of %s", fields[3], want)
		}
	})
}

// station serves what find asks a station for, around a rig's stream.
func station(t *testing.T, rig http.Handler, devices string) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/station", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"Station","pubkey":"1f80"}`))
	})
	mux.HandleFunc("GET /v1/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			http.Error(w, `{"error":"no token"}`, http.StatusUnauthorized)
			return
		}
		w.Write([]byte(devices))
	})
	mux.HandleFunc("POST /v1/devices/rig/phone/hold", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Scopes []string `json:"scopes"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Scopes) != 1 || body.Scopes[0] != "view" {
			t.Errorf("asked for %v, want view alone", body.Scopes)
		}
		w.Write([]byte(`{"grant":{"id":"grant-1"},"signal":"/v1/devices/rig/phone/signal","ice":[{"urls":["stun:127.0.0.1:1"]}]}`))
	})
	mux.Handle("GET /v1/devices/rig/phone/signal", rig)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRecordsTheDeviceThisSessionHoldsOnAStation(t *testing.T) {
	rig := &watchtest.Rig{}
	srv := station(t, rig, `[
		{"id":"rig/other","hold":{"scopes":["view","bidi"]}},
		{"id":"rig/phone","hold":{"mine":true,"scopes":["view","bidi"]}},
		{"id":"rig/free","hold":null}]`)
	record(t, Source{BaseURL: srv.URL, Header: http.Header{"Authorization": {"Bearer key"}}})
	if grants, auth := rig.Viewers(); len(auth) != 1 || grants[0] != `{"id":"grant-1"}` || auth[0] != "Bearer key" {
		t.Errorf("hello with grant %q and %q", grants, auth)
	}
}

func TestFindOnAStationThatDoesNotMarkHolds(t *testing.T) {
	src := func(srv *httptest.Server) Source {
		return Source{BaseURL: srv.URL, Header: http.Header{"Authorization": {"Bearer key"}}}
	}
	// One driven device can only be this session's.
	one := station(t, http.NotFoundHandler(), `[{"id":"rig/phone","hold":{"scopes":["view","bidi"]}},{"id":"rig/watched","hold":{"scopes":["view"]}}]`)
	way, err := find(context.Background(), src(one))
	if err != nil || !way.station || !strings.HasSuffix(way.url, "/v1/devices/rig/phone/signal") || !strings.HasPrefix(way.url, "ws://") {
		t.Errorf("one driven device: %+v, %v", way, err)
	}
	// Two leave it open, and guessing would record someone else's screen.
	two := station(t, http.NotFoundHandler(), `[{"id":"rig/phone","hold":{"scopes":["view","bidi"]}},{"id":"rig/other","hold":{"scopes":["view","bidi"]}}]`)
	if _, err := find(context.Background(), src(two)); err == nil {
		t.Error("two driven devices: no error")
	}
	// A token the station refuses.
	if _, err := find(context.Background(), Source{BaseURL: one.URL}); err == nil {
		t.Error("no token: no error")
	}
}

func TestNoStreamAtARemoteBrowser(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	_, err := Start(context.Background(), Source{BaseURL: srv.URL}, t.TempDir())
	if !errors.Is(err, ErrNoStream) {
		t.Errorf("got %v, want ErrNoStream", err)
	}
	if _, err := Start(context.Background(), Source{BaseURL: "ws://example"}, t.TempDir()); !errors.Is(err, ErrNoStream) {
		t.Errorf("not http: got %v, want ErrNoStream", err)
	}
}

func TestARigThatRefusesIsAnError(t *testing.T) {
	srv, rig := watchtest.Roadie(t)
	rig.Refuse = "the encoder is busy"
	dir := t.TempDir()
	_, err := Start(context.Background(), Source{BaseURL: srv.URL}, dir)
	if err == nil || !strings.Contains(err.Error(), "the encoder is busy") {
		t.Errorf("got %v", err)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("%d files left behind", len(left))
	}
}
