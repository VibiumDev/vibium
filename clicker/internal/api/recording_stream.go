package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vibium/clicker/internal/watch"
)

// streamStartTimeout bounds joining the stream at recording start.
const streamStartTimeout = 30 * time.Second

// StreamVideo is a recording's video taken from the stream a station or a
// Roadie serves of the device, in place of an engine screencast. The remote
// end has no browser to write a file, and the stream is what a viewer sees.
type StreamVideo struct {
	rec    *watch.Recording
	reason string // why the stream could not be joined, when rec is nil
}

// OpenStreamVideo joins the video stream of the device a remote session
// drives, before the recording starts so the video covers all of it. It
// returns nil when there is nothing to join: a local browser, video off,
// video kept on the remote host, or a remote end that streams none.
//
// With video required, a stream that cannot be joined is an error. In auto
// mode the reason rides on the result and becomes videoUnavailable.
func OpenStreamVideo(opts RecordingStartOptions, connectURL string, headers http.Header) (*StreamVideo, error) {
	if connectURL == "" || opts.Video.Mode == VideoOff || opts.Video.RemoteKeep {
		return nil, nil
	}
	required := opts.Video.Mode == VideoRequired
	base, err := StationBaseURL(connectURL)
	if err != nil {
		if required {
			return nil, errors.New(RemoteVideoMessage)
		}
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), streamStartTimeout)
	defer cancel()
	rec, err := watch.Start(ctx, watch.Source{BaseURL: base, Header: headers}, "")
	switch {
	case err == nil:
		return &StreamVideo{rec: rec}, nil
	case errors.Is(err, watch.ErrNoStream):
		if required {
			return nil, errors.New(RemoteVideoMessage)
		}
		return nil, nil
	case required:
		return nil, fmt.Errorf("the device's video stream could not be recorded: %w", err)
	}
	return &StreamVideo{reason: "the device's video stream could not be recorded: " + err.Error()}, nil
}

// Abort drops a stream video that no recording took.
func (v *StreamVideo) Abort() {
	if v != nil && v.rec != nil {
		v.rec.Abort()
	}
}

// setStreamVideo makes a joined stream the recording's video track.
func (t *Recorder) setStreamVideo(v *StreamVideo, context string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	started := v.rec.StartedAt().UnixMilli()
	t.stream = v
	t.video = &VideoTrack{
		Context:   context,
		StartedAt: started,
		OffsetMs:  float64(started - t.startTime),
		MimeType:  "video/mp4",
	}
}

// finishStream ends a stream video and reports whether there was one.
func (t *Recorder) finishStream() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finishStreamLocked()
}

// finishStreamLocked ends a stream video and points the track at the file,
// which Stop moves into the zip like an engine's.
func (t *Recorder) finishStreamLocked() bool {
	if t.stream == nil {
		return false
	}
	v := t.stream
	t.stream = nil
	info, err := v.rec.Stop()
	if err != nil {
		t.video.Error = "the device's video stream: " + err.Error()
		return true
	}
	t.video.EnginePath = info.Path
	t.video.StartedAt = info.StartedAt.UnixMilli()
	t.video.OffsetMs = float64(t.video.StartedAt - t.startTime)
	t.video.DurationMs = info.DurationMs
	t.video.Width, t.video.Height = info.Width, info.Height
	t.video.Gaps = info.Gaps
	if info.Ended != "" {
		t.video.Error = fmt.Sprintf("%s %.1f s into the recording", info.Ended, float64(info.DurationMs)/1000)
	}
	return true
}
