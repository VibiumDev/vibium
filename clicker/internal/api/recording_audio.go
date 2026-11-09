package api

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/vibium/clicker/internal/log"
)

// audioDialTimeout bounds the connection and params handshake per stream.
const audioDialTimeout = 5 * time.Second

// AudioOptions is the recording's audio configuration. Source is the base
// URL of a roadie host; the engine has no audio to offer (the BiDi
// screencast audio track is unimplemented in every browser), so audio is
// off unless a source is named.
type AudioOptions struct {
	Source string
}

// AudioTrack is one direction of a roadie audio source attached to a
// recording: "output" is the target's speaker (/audio), "input" is the mixed
// mic stream (/audio-in/monitor).
type AudioTrack struct {
	Direction string
	StartedAt int64 // wall-clock ms at the params handshake
	// OffsetMs is the track's start relative to the recording's t0, aligning
	// the audio timeline with trace event timestamps.
	OffsetMs   float64
	SampleRate int
	Channels   int
	Format     string // "f32le" or "s16le"
	SpoolPath  string
	Bytes      int64
	DurationMs int64
	Error      string // stream failure; the zip still delivers, audio absent or partial
}

// audioCapture owns the live WebSocket subscriptions of a recording's audio
// source. Reader goroutines spool PCM to temp files; stop closes the
// connections and finalizes the tracks.
type audioCapture struct {
	mu     sync.Mutex
	tracks []*AudioTrack
	conns  []*websocket.Conn
	spools []*os.File
	wg     sync.WaitGroup
	closed bool
}

// StartRecordingAudio subscribes to both directions of the recording's audio
// source. Audio is always explicit — there is no auto mode — so any failure
// is returned and the recording must not start.
func StartRecordingAudio(recorder *Recorder, opts RecordingStartOptions) error {
	if opts.Audio.Source == "" {
		return nil
	}
	base, err := audioWSBase(opts.Audio.Source)
	if err != nil {
		return err
	}

	ac := &audioCapture{}
	for _, dir := range []struct{ direction, path string }{
		{"output", "/audio"},
		{"input", "/audio-in/monitor"},
	} {
		track, conn, spool, err := dialAudioStream(base, dir.direction, dir.path)
		if err != nil {
			ac.stop()
			ac.removeSpools()
			return err
		}
		ac.tracks = append(ac.tracks, track)
		ac.conns = append(ac.conns, conn)
		ac.spools = append(ac.spools, spool)
	}

	for i := range ac.tracks {
		ac.wg.Add(1)
		go ac.spoolStream(ac.tracks[i], ac.conns[i], ac.spools[i])
	}
	recorder.SetAudioCapture(ac)
	return nil
}

// StopRecordingAudio finalizes the audio subscriptions of a recording that
// is stopping. Failures live in the tracks rather than being returned — the
// zip still delivers, with audio absent or partial — because fail-fast
// applies only at start.
func StopRecordingAudio(recorder *Recorder) {
	if recorder == nil {
		return
	}
	ac := recorder.AudioCapture()
	if ac == nil {
		return
	}
	ac.stop()
	recorder.FinishAudio(ac.tracks)
}

// stop closes the connections and waits for the reader goroutines to drain.
func (c *audioCapture) stop() {
	c.mu.Lock()
	c.closed = true
	conns := c.conns
	c.mu.Unlock()
	for _, conn := range conns {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		conn.Close()
	}
	c.wg.Wait()
}

// removeSpools deletes the temp PCM files. Used for unwinding a failed start
// and for abandoned recordings; packaged recordings clean up in Stop.
func (c *audioCapture) removeSpools() {
	for _, f := range c.spools {
		if f != nil {
			os.Remove(f.Name())
		}
	}
	c.spools = nil
}

// spoolStream drains one audio WebSocket into its temp file.
func (c *audioCapture) spoolStream(track *AudioTrack, conn *websocket.Conn, spool *os.File) {
	defer c.wg.Done()
	defer spool.Close()
	for {
		typ, data, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if !c.closed && track.Error == "" {
				track.Error = "audio stream lost: " + err.Error()
			}
			c.mu.Unlock()
			return
		}
		if typ != websocket.BinaryMessage || len(data) == 0 {
			continue
		}
		n, err := spool.Write(data)
		c.mu.Lock()
		track.Bytes += int64(n)
		if err != nil && track.Error == "" {
			track.Error = "audio spool write failed: " + err.Error()
		}
		c.mu.Unlock()
		if err != nil {
			return
		}
	}
}

// dialAudioStream connects one roadie audio endpoint and completes the params
// handshake: the first text message carries sampleRate, channels, and format.
func dialAudioStream(base *url.URL, direction, path string) (*AudioTrack, *websocket.Conn, *os.File, error) {
	u := *base
	u.Path = strings.TrimRight(u.Path, "/") + path
	dialer := websocket.Dialer{HandshakeTimeout: audioDialTimeout}
	conn, resp, err := dialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 503 {
			return nil, nil, nil, fmt.Errorf("audio source has no %s stream (%s returned 503); is the roadie device capturing audio?", direction, u.String())
		}
		return nil, nil, nil, fmt.Errorf("audio source %s unreachable: %w", u.String(), err)
	}

	conn.SetReadDeadline(time.Now().Add(audioDialTimeout))
	typ, data, err := conn.ReadMessage()
	if err != nil || typ != websocket.TextMessage {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("audio source %s sent no params message", u.String())
	}
	conn.SetReadDeadline(time.Time{})

	var params struct {
		SampleRate int    `json:"sampleRate"`
		Channels   int    `json:"channels"`
		Format     string `json:"format"`
	}
	if err := json.Unmarshal(data, &params); err != nil || params.SampleRate <= 0 || params.Channels <= 0 {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("audio source %s sent unusable params %q", u.String(), string(data))
	}
	format, err := normalizeAudioFormat(params.Format)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("audio source %s: %w", u.String(), err)
	}

	spool, err := os.CreateTemp("", "vibium-audio-*.pcm")
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("audio spool: %w", err)
	}

	return &AudioTrack{
		Direction:  direction,
		StartedAt:  time.Now().UnixMilli(),
		SampleRate: params.SampleRate,
		Channels:   params.Channels,
		Format:     format,
		SpoolPath:  spool.Name(),
	}, conn, spool, nil
}

// normalizeAudioFormat maps a source's format label onto a WAV-expressible
// sample format. Roadie labels its /audio stream "f32-planar" but delivers
// interleaved samples, so the label only selects the sample width here.
func normalizeAudioFormat(label string) (string, error) {
	switch {
	case strings.Contains(label, "f32"):
		return "f32le", nil
	case strings.Contains(label, "s16"):
		return "s16le", nil
	default:
		return "", fmt.Errorf("unsupported audio format %q", label)
	}
}

// audioWSBase turns the audio source URL into a WebSocket base URL.
func audioWSBase(source string) (*url.URL, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("audio source: %w", err)
	}
	switch u.Scheme {
	case "http", "ws":
		u.Scheme = "ws"
	case "https", "wss":
		u.Scheme = "wss"
	default:
		return nil, fmt.Errorf("audio source must be an http(s) or ws(s) URL, got %q", source)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("audio source %q has no host", source)
	}
	return u, nil
}

// bytesPerSample returns the sample width of a normalized format.
func bytesPerSample(format string) int {
	if format == "f32le" {
		return 4
	}
	return 2
}

// audioDurationMs computes a track's duration from its byte count.
func audioDurationMs(track *AudioTrack) int64 {
	bytesPerSecond := int64(track.SampleRate * track.Channels * bytesPerSample(track.Format))
	if bytesPerSecond == 0 {
		return 0
	}
	return track.Bytes * 1000 / bytesPerSecond
}

// wavHeader renders the 44-byte RIFF header for a track's PCM data: format
// 3 (IEEE float) for f32le, format 1 (integer PCM) for s16le.
func wavHeader(track *AudioTrack, dataLen int64) []byte {
	bps := bytesPerSample(track.Format)
	var audioFormat uint16 = 1
	if track.Format == "f32le" {
		audioFormat = 3
	}
	h := make([]byte, 44)
	copy(h[0:4], "RIFF")
	binary.LittleEndian.PutUint32(h[4:8], uint32(36+dataLen))
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16)
	binary.LittleEndian.PutUint16(h[20:22], audioFormat)
	binary.LittleEndian.PutUint16(h[22:24], uint16(track.Channels))
	binary.LittleEndian.PutUint32(h[24:28], uint32(track.SampleRate))
	binary.LittleEndian.PutUint32(h[28:32], uint32(track.SampleRate*track.Channels*bps))
	binary.LittleEndian.PutUint16(h[32:34], uint16(track.Channels*bps))
	binary.LittleEndian.PutUint16(h[34:36], uint16(bps*8))
	copy(h[36:40], "data")
	binary.LittleEndian.PutUint32(h[40:44], uint32(dataLen))
	return h
}

// writeAudioEntriesLocked writes audio/<direction>.wav and audio/index.json.
// Must be called with t.mu held and t.audio non-empty.
func (t *Recorder) writeAudioEntriesLocked(zw *zip.Writer, now time.Time) error {
	entries := make([]interface{}, 0, len(t.audio))
	for _, track := range t.audio {
		entry := map[string]interface{}{
			"direction":  track.Direction,
			"startedAt":  track.StartedAt,
			"offsetMs":   track.OffsetMs,
			"sampleRate": track.SampleRate,
			"channels":   track.Channels,
			"mimeType":   "audio/wav",
			"durationMs": track.DurationMs,
		}
		if track.Error != "" {
			entry["error"] = track.Error
		}
		if track.Bytes > 0 && track.SpoolPath != "" {
			name := "audio/" + track.Direction + ".wav"
			if err := writeWavEntry(zw, now, name, track); err != nil {
				if track.Error == "" {
					entry["error"] = "audio spool unreadable: " + err.Error()
				}
			} else {
				entry["file"] = name
			}
		}
		entries = append(entries, entry)
	}

	iw, err := zw.CreateHeader(&zip.FileHeader{
		Name:     "audio/index.json",
		Method:   zip.Deflate,
		Modified: now,
	})
	if err != nil {
		return fmt.Errorf("failed to create audio index: %w", err)
	}
	index, err := json.Marshal(map[string]interface{}{
		"version": 1,
		"audio":   entries,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal audio index: %w", err)
	}
	iw.Write(index)
	return nil
}

// writeWavEntry streams one spool file into the zip behind a WAV header.
func writeWavEntry(zw *zip.Writer, now time.Time, name string, track *AudioTrack) error {
	f, err := os.Open(track.SpoolPath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: now,
	})
	if err != nil {
		return err
	}
	if _, err := w.Write(wavHeader(track, info.Size())); err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// removeAudioSpoolsLocked deletes the audio temp files. Must be called with
// t.mu held.
func (t *Recorder) removeAudioSpoolsLocked() {
	for _, track := range t.audio {
		if track.SpoolPath != "" {
			if err := os.Remove(track.SpoolPath); err != nil && !os.IsNotExist(err) {
				log.Debug("failed to delete audio spool", "path", track.SpoolPath, "error", err)
			}
			track.SpoolPath = ""
		}
	}
}
