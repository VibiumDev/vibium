// Package watch records the video a Vibium Station or a Roadie streams of
// the device a session drives. It joins as a WebRTC viewer and writes the
// H.264 it receives into an MP4, without decoding it. The recording's
// sound can go into the same file.
package watch

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"

	"github.com/vibium/clicker/internal/aac"
)

// ErrNoStream says the remote end is neither a station nor a Roadie, so
// there is no stream to record.
var ErrNoStream = errors.New("the remote end streams no video")

const (
	lookupTimeout = 8 * time.Second
	firstFrame    = 20 * time.Second
	keyEvery      = 2 * time.Second
	pingEvery     = 20 * time.Second
)

// Source is the remote end a session came in by.
type Source struct {
	BaseURL string      // the station's or the Roadie's address, http or https
	Header  http.Header // what the session connected with
}

// Info describes a finished recording.
type Info struct {
	Path       string
	StartedAt  time.Time // when the first frame was on screen, by this machine's clock
	DurationMs int64
	Width      int
	Height     int
	Frames     int
	Bytes      int64
	Gaps       int    // times the stream lost packets and waited for a key frame
	Ended      string // why the stream stopped before the recording did, if it did
	Sound      bool   // the file has an audio track
}

// SoundRate is the sample rate of the sound a file takes.
const SoundRate = aac.Rate

// Sound gives the sound that goes with a stretch of video: one channel of
// samples at SoundRate, the first one at start. No samples means no sound.
type Sound func(start time.Time, length time.Duration) []int16

// Recording is a stream being written to a file.
type Recording struct {
	pc   *webrtc.PeerConnection
	ws   *websocket.Conn
	wmu  sync.Mutex
	stop chan struct{}
	key  chan struct{}

	started chan struct{}
	failed  chan struct{}
	once    sync.Once

	amu     sync.Mutex
	arrived map[uint32]time.Time

	mu      sync.Mutex
	file    *mp4File
	closed  bool
	waitKey bool
	seen    bool
	lastRTP uint32
	lastAt  time.Time
	rtp     int64
	rtp0    int64
	first   time.Time
	gaps    int
	ended   string
}

type signal struct {
	Type      string                   `json:"type"`
	SDP       string                   `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit `json:"candidate,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Grant     json.RawMessage          `json:"grant,omitempty"`
}

// Start joins the stream and returns once the first key frame is in the
// file. dir is where the file is made; empty means the temporary directory.
func Start(ctx context.Context, src Source, dir string) (*Recording, error) {
	way, err := find(ctx, src)
	if err != nil {
		return nil, err
	}
	dialer := websocket.Dialer{HandshakeTimeout: lookupTimeout}
	ws, resp, err := dialer.DialContext(ctx, way.url, way.header)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
			if !way.station && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed) {
				return nil, ErrNoStream
			}
			return nil, fmt.Errorf("the video stream refused the connection: %s", resp.Status)
		}
		if !way.station {
			return nil, ErrNoStream
		}
		return nil, fmt.Errorf("the video stream did not answer: %w", err)
	}

	f, err := os.CreateTemp(dir, "vibium-stream-*.mp4")
	if err != nil {
		ws.Close()
		return nil, err
	}
	f.Close()
	file, err := createMP4(f.Name())
	if err != nil {
		ws.Close()
		os.Remove(f.Name())
		return nil, err
	}

	r := &Recording{
		ws: ws, file: file, waitKey: true,
		stop: make(chan struct{}), key: make(chan struct{}, 1),
		started: make(chan struct{}), failed: make(chan struct{}),
		arrived: map[uint32]time.Time{},
	}
	if err := r.connect(way); err != nil {
		r.Abort()
		return nil, err
	}

	select {
	case <-r.started:
		return r, nil
	case <-r.failed:
		why := r.why()
		r.Abort()
		return nil, errors.New(why)
	case <-time.After(firstFrame):
		r.Abort()
		return nil, fmt.Errorf("no video arrived within %s", firstFrame)
	case <-ctx.Done():
		r.Abort()
		return nil, ctx.Err()
	}
}

func (r *Recording) connect(way *way) error {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return err
	}
	reg := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, reg); err != nil {
		return err
	}
	// pion logs to stdout by default, which is the protocol channel when
	// Vibium runs as an MCP server or a pipe.
	quiet := webrtc.SettingEngine{LoggerFactory: &logging.DefaultLoggerFactory{
		Writer: io.Discard, DefaultLogLevel: logging.LogLevelDisabled, ScopeLevels: map[string]logging.LogLevel{},
	}}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(reg), webrtc.WithSettingEngine(quiet))
	pc, err := api.NewPeerConnection(webrtc.Configuration{ICEServers: way.ice})
	if err != nil {
		return err
	}
	r.pc = pc

	pc.OnTrack(func(tr *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if tr.Kind() != webrtc.RTPCodecTypeVideo {
			for {
				if _, _, err := tr.ReadRTP(); err != nil {
					return
				}
			}
		}
		if !strings.EqualFold(tr.Codec().MimeType, webrtc.MimeTypeH264) {
			r.fail("the stream's video is " + tr.Codec().MimeType + ", and only H.264 can be recorded")
			return
		}
		go r.askForKeys(uint32(tr.SSRC()))
		r.read(tr)
	})
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			init := c.ToJSON()
			r.send(signal{Type: "candidate", Candidate: &init})
		}
	})
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			r.fail("the video stream ended")
		}
	})

	if err := r.send(signal{Type: "hello", Grant: way.grant}); err != nil {
		return fmt.Errorf("the video stream did not answer: %w", err)
	}
	go r.signals()
	go r.ping()
	return nil
}

func (r *Recording) send(m signal) error {
	r.wmu.Lock()
	defer r.wmu.Unlock()
	r.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return r.ws.WriteJSON(m)
}

// signals answers the offer and trades candidates until the socket ends.
// The rig ends the stream when its signaling socket closes.
func (r *Recording) signals() {
	for {
		var m signal
		if err := r.ws.ReadJSON(&m); err != nil {
			var syntax *json.SyntaxError
			if errors.As(err, &syntax) {
				continue
			}
			r.fail("the video stream ended")
			return
		}
		switch m.Type {
		case "offer":
			if err := r.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: m.SDP}); err != nil {
				r.fail("the stream's offer was not usable: " + err.Error())
				return
			}
			ans, err := r.pc.CreateAnswer(nil)
			if err == nil {
				err = r.pc.SetLocalDescription(ans)
			}
			if err != nil {
				r.fail("the stream's offer could not be answered: " + err.Error())
				return
			}
			r.send(signal{Type: "answer", SDP: ans.SDP})
		case "candidate":
			if m.Candidate != nil {
				r.pc.AddICECandidate(*m.Candidate)
			}
		case "error":
			r.fail("the device refused the video stream: " + m.Error)
		}
	}
}

func (r *Recording) ping() {
	t := time.NewTicker(pingEvery)
	defer t.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-t.C:
			r.wmu.Lock()
			r.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			r.wmu.Unlock()
		}
	}
}

// askForKeys asks the sender for a key frame at the start, on a schedule
// and after a loss. A key frame is where a player can seek to, and where a
// stream that lost packets is whole again.
func (r *Recording) askForKeys(ssrc uint32) {
	t := time.NewTicker(keyEvery)
	defer t.Stop()
	for {
		r.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}})
		select {
		case <-r.stop:
			return
		case <-t.C:
		case <-r.key:
		}
	}
}

func (r *Recording) read(tr *webrtc.TrackRemote) {
	frames := samplebuilder.New(512, &codecs.H264Packet{IsAVC: true}, clock)
	for {
		pkt, _, err := tr.ReadRTP()
		if err != nil {
			return
		}
		r.amu.Lock()
		if _, ok := r.arrived[pkt.Timestamp]; !ok {
			if len(r.arrived) > 4096 {
				clear(r.arrived)
			}
			r.arrived[pkt.Timestamp] = time.Now()
		}
		r.amu.Unlock()
		frames.Push(pkt)
		for s := frames.Pop(); s != nil; s = frames.Pop() {
			r.frame(s)
		}
	}
}

// frame writes one access unit, from a key frame on, and skips to the next
// key frame after a loss so the file never holds a picture that cannot be
// decoded.
func (r *Recording) frame(s *media.Sample) {
	r.amu.Lock()
	at, ok := r.arrived[s.PacketTimestamp]
	delete(r.arrived, s.PacketTimestamp)
	r.amu.Unlock()
	if !ok {
		at = time.Now()
	}
	key, sps, pps := inspect(s.Data)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	if r.seen {
		d := int64(int32(s.PacketTimestamp - r.lastRTP))
		if d <= 0 || d > 10*clock {
			d = max(ticks(at.Sub(r.lastAt)), 1)
		}
		r.rtp += d
	}
	r.seen, r.lastRTP, r.lastAt = true, s.PacketTimestamp, at

	if s.PrevDroppedPackets > 0 && !r.waitKey {
		r.waitKey = true
		r.gaps++
		select {
		case r.key <- struct{}{}:
		default:
		}
	}
	if key && r.file.sps == nil && sps != nil && pps != nil {
		r.file.sps, r.file.pps = bytes.Clone(sps), bytes.Clone(pps)
	}
	if (r.waitKey && !key) || r.file.sps == nil {
		return
	}
	r.waitKey = false
	start := r.first.IsZero()
	if start {
		r.first, r.rtp0 = at, r.rtp
	}
	r.file.add(s.Data, frame{key: key, rtp: r.rtp - r.rtp0, wall: ticks(at.Sub(r.first))})
	if start {
		close(r.started)
	}
}

// inspect reads an access unit of length-prefixed NAL units.
func inspect(data []byte) (key bool, sps, pps []byte) {
	for len(data) > 4 {
		n := int(binary.BigEndian.Uint32(data))
		if n <= 0 || n > len(data)-4 {
			break
		}
		nal := data[4 : 4+n]
		switch nal[0] & 0x1f {
		case 5:
			key = true
		case 7:
			sps = nal
		case 8:
			pps = nal
		}
		data = data[4+n:]
	}
	return key, sps, pps
}

func ticks(d time.Duration) int64 { return int64(d) * clock / int64(time.Second) }

func (r *Recording) fail(why string) {
	r.mu.Lock()
	if r.ended == "" && !r.closed {
		r.ended = why
	}
	r.mu.Unlock()
	r.once.Do(func() { close(r.failed) })
}

func (r *Recording) why() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ended == "" {
		return "the video stream ended"
	}
	return r.ended
}

// hangUp ends the stream. Closing can wait on a peer that is gone, so it
// gets a moment and no more.
func (r *Recording) hangUp() {
	close(r.stop)
	done := make(chan struct{})
	go func() {
		if r.pc != nil {
			r.pc.Close()
		}
		r.ws.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// Stop ends the recording and finishes the file. sound, when given, is
// asked for the audio track once the video's span is known.
func (r *Recording) Stop(sound Sound) (Info, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return Info{}, errors.New("the recording is already stopped")
	}
	r.closed = true
	end := time.Now()
	file, first, gaps, ended := r.file, r.first, r.gaps, r.ended
	r.mu.Unlock()
	r.hangUp()

	at := func(t int64) time.Duration { return time.Duration(t) * time.Second / clock }
	var audio func(shift, length int64) []int16
	if sound != nil {
		audio = func(shift, length int64) []int16 { return sound(first.Add(at(shift)), at(length)) }
	}
	got, err := file.finish(ticks(end.Sub(first)), audio)
	if err != nil {
		os.Remove(file.path)
		return Info{}, err
	}
	return Info{
		Path:       file.path,
		StartedAt:  first.Add(at(got.Shift)),
		DurationMs: got.Duration * 1000 / clock,
		Width:      got.Width,
		Height:     got.Height,
		Frames:     got.Frames,
		Bytes:      got.Bytes,
		Gaps:       gaps,
		Ended:      ended,
		Sound:      got.Sound,
	}, nil
}

// StartedAt is when the first frame arrived.
func (r *Recording) StartedAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.first
}

// Abort ends the recording and removes the file.
func (r *Recording) Abort() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()
	r.hangUp()
	r.file.discard()
}
