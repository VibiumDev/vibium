// Package watchtest plays a rig for tests: signaling like a Roadie's and a
// small H.264 stream behind it.
package watchtest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

//go:embed tiny.h264
var tiny []byte

// Width and Height are the fixture's picture size.
const Width, Height = 100, 60

// Frames returns the fixture as access units in Annex B form: 12 frames,
// a key frame with its parameter sets every 4.
func Frames() [][]byte {
	var frames [][]byte
	var frame []byte
	data := bytes.ReplaceAll(tiny, []byte{0, 0, 0, 1}, []byte{0, 0, 1})
	for _, nal := range bytes.Split(data, []byte{0, 0, 1}) {
		if len(nal) == 0 {
			continue
		}
		frame = append(append(frame, 0, 0, 0, 1), nal...)
		if kind := nal[0] & 0x1f; kind == 1 || kind == 5 {
			frames = append(frames, frame)
			frame = nil
		}
	}
	return frames
}

type message struct {
	Type      string                   `json:"type"`
	SDP       string                   `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit `json:"candidate,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Grant     json.RawMessage          `json:"grant,omitempty"`
}

// Rig answers a viewer's hello with an offer and sends Frames, 30 a
// second, until the signaling socket closes.
type Rig struct {
	// Refuse, when set, is the error the rig answers a hello with.
	Refuse string

	mu     sync.Mutex
	grants []string
	auth   []string
}

// Viewers reports, per viewer so far, the grant in its hello and the
// Authorization header it connected with.
func (r *Rig) Viewers() (grants, auth []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.grants...), append([]string(nil), r.auth...)
}

func (r *Rig) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ws, err := (&websocket.Upgrader{}).Upgrade(w, req, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	var hello message
	if err := ws.ReadJSON(&hello); err != nil || hello.Type != "hello" {
		return
	}
	r.mu.Lock()
	r.grants = append(r.grants, string(hello.Grant))
	r.auth = append(r.auth, req.Header.Get("Authorization"))
	r.mu.Unlock()
	if r.Refuse != "" {
		ws.WriteJSON(message{Type: "error", Error: r.Refuse})
		time.Sleep(200 * time.Millisecond)
		return
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return
	}
	defer pc.Close()
	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}, "video", "rig")
	if err != nil {
		return
	}
	if _, err := pc.AddTrack(track); err != nil {
		return
	}
	var wmu sync.Mutex
	send := func(m message) {
		wmu.Lock()
		defer wmu.Unlock()
		ws.WriteJSON(m)
	}
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			init := c.ToJSON()
			send(message{Type: "candidate", Candidate: &init})
		}
	})
	connected := make(chan struct{})
	var once sync.Once
	pc.OnConnectionStateChange(func(st webrtc.PeerConnectionState) {
		if st == webrtc.PeerConnectionStateConnected {
			once.Do(func() { close(connected) })
		}
	})
	offer, err := pc.CreateOffer(nil)
	if err == nil {
		err = pc.SetLocalDescription(offer)
	}
	if err != nil {
		return
	}
	send(message{Type: "offer", SDP: offer.SDP})

	gone := make(chan struct{})
	go func() {
		defer close(gone)
		for {
			var m message
			if err := ws.ReadJSON(&m); err != nil {
				return
			}
			switch m.Type {
			case "answer":
				pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: m.SDP})
			case "candidate":
				if m.Candidate != nil {
					pc.AddICECandidate(*m.Candidate)
				}
			}
		}
	}()
	select {
	case <-connected:
	case <-gone:
		return
	case <-time.After(15 * time.Second):
		return
	}
	frames := Frames()
	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()
	for i := 0; ; i++ {
		select {
		case <-gone:
			return
		case <-tick.C:
			track.WriteSample(media.Sample{Data: frames[i%len(frames)], Duration: time.Second / 30})
		}
	}
}

// Roadie serves a rig's stream where a Roadie has it.
func Roadie(t testing.TB) (*httptest.Server, *Rig) {
	rig := &Rig{}
	mux := http.NewServeMux()
	mux.Handle("/api/webrtc/signal", rig)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, rig
}
