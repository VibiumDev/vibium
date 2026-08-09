package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RemoteVideoMessage explains why video capture cannot deliver over
// --connect — the browser writes the video on the remote host — and names
// the opt-in for hosts the caller controls.
const RemoteVideoMessage = "screen recording is not supported on remote browser connections: the browser writes the video on the remote host. Use a local browser, record without video, or — for a remote host you control — video: {remote: 'keep'} to record and leave the file there"

// stopScreencastTimeout bounds the engine's video finalization at stop.
const stopScreencastTimeout = 15 * time.Second

// StartRecordingVideo attaches an engine screencast to a recording that is
// starting. The recording's video binds to the browsing context active now
// and does not follow focus.
//
// With video required (video: true or explicit dimensions), any failure is
// returned and the recording must not start. In auto mode (video omitted),
// failures are recorded as videoUnavailable on the recorder and the
// recording proceeds without video.
func StartRecordingVideo(s Session, recorder *Recorder, opts RecordingStartOptions, remote bool, viewport map[string]interface{}) error {
	if opts.Video.Mode == VideoOff {
		return nil
	}
	fail := func(err error) error {
		if opts.Video.Mode == VideoRequired {
			return err
		}
		recorder.SetVideoUnavailable(err.Error())
		return nil
	}

	// remote: 'keep' opts out of the refusal: the engine records on the
	// remote host and the file stays there, reported via remotePath.
	if remote && !opts.Video.RemoteKeep {
		return fail(errors.New(RemoteVideoMessage))
	}

	context, err := s.GetContextID()
	if err != nil {
		return fail(err)
	}

	params := map[string]interface{}{"context": context}
	video := map[string]interface{}{}
	if opts.Video.Width > 0 {
		video["width"] = opts.Video.Width
	}
	if opts.Video.Height > 0 {
		video["height"] = opts.Video.Height
	}
	if opts.Video.FrameRate > 0 {
		video["frameRate"] = opts.Video.FrameRate
	}
	if len(video) > 0 {
		params["video"] = video
	}

	resp, err := s.SendBidiCommand("browsingContext.startScreencast", params)
	if err != nil {
		return fail(videoSupportError(err))
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return fail(videoSupportError(bidiErr))
	}

	var result struct {
		Result struct {
			Screencast string `json:"screencast"`
			Path       string `json:"path"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || result.Result.Screencast == "" {
		return fail(fmt.Errorf("unexpected startScreencast response"))
	}

	width, height := opts.Video.Width, opts.Video.Height
	if width == 0 {
		width, _ = viewport["width"].(int)
	}
	if height == 0 {
		height, _ = viewport["height"].(int)
	}
	recorder.SetVideoTrack(&VideoTrack{
		Context:    context,
		ID:         result.Result.Screencast,
		EnginePath: result.Result.Path,
		Remote:     remote,
		StartedAt:  time.Now().UnixMilli(),
		Width:      width,
		Height:     height,
	})
	return nil
}

// StopRecordingVideo finalizes the engine screencast of a recording that is
// stopping. Failures are recorded into the video track rather than returned
// — the zip still delivers, with the video absent or partial — because
// fail-fast applies only at start.
func StopRecordingVideo(s Session, recorder *Recorder) {
	track := recorder.ActiveVideo()
	if track == nil || track.ID == "" {
		return
	}

	resp, err := s.SendBidiCommandWithTimeout("browsingContext.stopScreencast", map[string]interface{}{
		"screencast": track.ID,
	}, stopScreencastTimeout)
	if err != nil {
		recorder.FinishVideo("", err.Error())
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		recorder.FinishVideo("", bidiErr.Error())
		return
	}

	var result struct {
		Result struct {
			Path  string `json:"path"`
			Error string `json:"error"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		recorder.FinishVideo("", "unexpected stopScreencast response")
		return
	}
	if result.Result.Error != "" {
		recorder.FinishVideo("", "screencast write failed: "+result.Result.Error)
		return
	}
	recorder.FinishVideo(result.Result.Path, "")
}

// FinalizeRecordingOnClose delivers an active recording when the browser
// session ends without recording.stop(): the recording packages and lands at
// its declared path as if stop() had been called. Recordings without a
// declared path are lost on close; the engine's video temp file is deleted
// either way.
func FinalizeRecordingOnClose(s Session, recorder *Recorder) {
	if recorder == nil {
		return
	}
	StopRecordingVideo(s, recorder)
	if recorder.IsRecording() && recorder.Options().Path != "" {
		if zipData, err := recorder.Stop(); err == nil {
			WriteRecordToFile(zipData, recorder.Options().Path)
			return
		}
	}
	recorder.RemoveEngineFile()
}

// RecordingSavedSentence is the one-line stop result shown by the CLI and
// MCP surfaces: "Saved record.zip (23 steps, 14s video)".
func RecordingSavedSentence(path string, s RecordingSummary) string {
	msg := fmt.Sprintf("Saved %s (%d steps", path, s.Steps)
	if len(s.Videos) > 0 && s.Videos[0].Error == "" && s.Videos[0].DurationMs > 0 {
		msg += fmt.Sprintf(", %ds video", (s.Videos[0].DurationMs+500)/1000)
	}
	msg += ")"
	if s.VideoUnavailable != "" {
		msg += " — video unavailable: " + s.VideoUnavailable
	} else if len(s.Videos) > 0 && s.Videos[0].Error != "" {
		msg += " — video error: " + s.Videos[0].Error
	} else if len(s.Videos) > 0 && s.Videos[0].RemotePath != "" {
		msg += " — video on remote host: " + s.Videos[0].RemotePath
	}
	return msg
}

// videoSupportError turns the browser's "unknown command" on startScreencast
// into something the user can act on. Only that case is rewritten: a browser
// that has the command but refuses an option already names the problem, and
// replacing its message would hide it.
func videoSupportError(err error) error {
	if strings.Contains(err.Error(), "unknown command") {
		return fmt.Errorf("video recording is not supported by this browser yet " +
			"(Chrome: not implemented; Firefox: requires 154+). " +
			"Install it with `vibium install --engine firefox` and launch with browser \"firefox\", " +
			"or record without video for a trace with screenshots")
	}
	if strings.Contains(err.Error(), "NS_ERROR_FAILURE") &&
		strings.Contains(err.Error(), "nsIProperties.get") {
		return fmt.Errorf("Firefox could not resolve its screencast output directory")
	}
	return err
}
