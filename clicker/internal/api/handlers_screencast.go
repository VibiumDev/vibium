package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// handleScreencastStart handles vibium:screencast.start — native browser
// video recording via browsingContext.startScreencast. The browser encodes
// the video itself and writes it to a file it chooses; stop moves it where
// the client asked.
// Options: context, mimeType, width, height, frameRate, audio.
func (r *Router) handleScreencastStart(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	active := session.screencastID != ""
	session.mu.Unlock()
	if active {
		r.sendError(session, cmd.ID, fmt.Errorf("screencast is already running"))
		return
	}

	context, _ := cmd.Params["context"].(string)
	if context == "" {
		session.mu.Lock()
		context = session.lastContext
		session.mu.Unlock()
	}
	if context == "" {
		var err error
		context, err = r.getContext(session)
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
	}

	params := map[string]interface{}{"context": context}
	if v, ok := cmd.Params["mimeType"].(string); ok && v != "" {
		params["mimeType"] = v
	}
	video := map[string]interface{}{}
	for _, k := range []string{"width", "height", "frameRate"} {
		if v, ok := cmd.Params[k].(float64); ok {
			video[k] = int(v)
		}
	}
	if len(video) > 0 {
		params["video"] = video
	}
	if v, ok := cmd.Params["audio"].(bool); ok && v {
		params["audio"] = true
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.startScreencast", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, screencastSupportError(bidiErr))
		return
	}

	var result struct {
		Result struct {
			Screencast string `json:"screencast"`
			Path       string `json:"path"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || result.Result.Screencast == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("unexpected startScreencast response"))
		return
	}

	session.mu.Lock()
	session.screencastID = result.Result.Screencast
	session.screencastPath = result.Result.Path
	session.mu.Unlock()

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleScreencastStop handles vibium:screencast.stop — finalizes the
// recording and delivers the file.
// Options: path (move the video there; otherwise return it base64-inline).
func (r *Router) handleScreencastStop(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	id := session.screencastID
	startPath := session.screencastPath
	session.screencastID = ""
	session.screencastPath = ""
	session.mu.Unlock()

	if id == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("screencast is not started"))
		return
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.stopScreencast", map[string]interface{}{
		"screencast": id,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var result struct {
		Result struct {
			Path  string `json:"path"`
			Error string `json:"error"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("unexpected stopScreencast response"))
		return
	}
	if result.Result.Error != "" {
		r.sendError(session, cmd.ID, fmt.Errorf("screencast write failed: %s", result.Result.Error))
		return
	}

	videoPath := result.Result.Path
	if videoPath == "" {
		videoPath = startPath
	}

	// The spec leaves cleanup of the browser-written file to us.
	if outPath, ok := cmd.Params["path"].(string); ok && outPath != "" {
		if err := moveFile(videoPath, outPath); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to save screencast: %w", err))
			return
		}
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"path": outPath})
		return
	}

	data, err := os.ReadFile(videoPath)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to read screencast: %w", err))
		return
	}
	os.Remove(videoPath)
	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"data": base64.StdEncoding.EncodeToString(data),
	})
}

// screencastSupportError turns the browser's "unknown command" on
// startScreencast into something the user can act on. Only that case is
// rewritten: a browser that has the command but refuses an option (Firefox
// 154 answers `unsupported operation: The audio track is not supported`)
// already names the problem, and replacing its message would hide it.
func screencastSupportError(err error) error {
	if strings.Contains(err.Error(), "unknown command") {
		return fmt.Errorf("screen recording is not supported by this browser yet " +
			"(Chrome: not implemented; Firefox: requires 154+). " +
			"Launch with browser \"firefox\" to record video, " +
			"or use recording.start() for a trace with screenshots")
	}
	return err
}

// moveFile renames src to dst, falling back to copy+remove across
// filesystems (the browser writes to its own temp dir, which can be a
// different mount than the destination).
func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
