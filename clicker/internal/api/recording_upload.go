package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// UploadTarget is the station a finished recording is sent to: its REST
// base and the credential the engine connected with. The station keeps
// recordings in its Record Store, where they sit next to the hold the
// session ran on.
type UploadTarget struct {
	BaseURL       string
	Authorization string // the Authorization header the engine connected with
	Client        *http.Client
}

// UploadResult is what the station recorded about the stored file.
type UploadResult struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Hold   string `json:"hold,omitempty"`
	Device string `json:"device,omitempty"`
	Event  string `json:"event,omitempty"`
}

// Fields is the result as the stop command reports it.
func (u UploadResult) Fields() map[string]interface{} {
	fields := map[string]interface{}{"url": u.URL, "sha256": u.SHA256, "size": u.Size}
	if u.Hold != "" {
		fields["hold"] = u.Hold
	}
	if u.Device != "" {
		fields["device"] = u.Device
	}
	if u.Event != "" {
		fields["event"] = u.Event
	}
	return fields
}

// StationBaseURL derives a station's REST base from the BiDi URL the
// engine connected to: wss becomes https, ws becomes http, and the
// /session path (with any session id after it) is dropped.
func StationBaseURL(connectURL string) (string, error) {
	u, err := url.Parse(connectURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%q is not a station URL", connectURL)
	}
	switch u.Scheme {
	case "wss":
		u.Scheme = "https"
	case "ws":
		u.Scheme = "http"
	case "http", "https":
	default:
		return "", fmt.Errorf("%q is not a station URL", connectURL)
	}
	if i := strings.Index(u.Path, "/session"); i >= 0 {
		u.Path = u.Path[:i]
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery, u.Fragment, u.User = "", "", nil
	return u.String(), nil
}

// ResolveUpload reads a stop command's upload option: nothing for false
// or absent, the station the engine is connected to for true, or the
// named station for a string. Either way the credential is the
// Authorization header the engine connected with.
func ResolveUpload(option interface{}, connectURL string, connectHeaders http.Header) (*UploadTarget, error) {
	var base string
	switch v := option.(type) {
	case nil:
		return nil, nil
	case bool:
		if !v {
			return nil, nil
		}
		if connectURL == "" {
			return nil, fmt.Errorf("upload needs a station: connect with --connect or VIBIUM_CONNECT_URL, or name the station's URL as the upload option")
		}
		b, err := StationBaseURL(connectURL)
		if err != nil {
			return nil, fmt.Errorf("upload: %w", err)
		}
		base = b
	case string:
		if v == "" {
			return nil, nil
		}
		b, err := StationBaseURL(v)
		if err != nil {
			return nil, fmt.Errorf("upload: %w", err)
		}
		base = b
	default:
		return nil, fmt.Errorf("upload must be true or a station URL")
	}
	auth := connectHeaders.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("upload needs the API key the engine connected with (VIBIUM_CONNECT_API_KEY, or --connect-header \"Authorization: Bearer …\")")
	}
	return &UploadTarget{BaseURL: base, Authorization: auth}, nil
}

// UploadName is the file name a recording is uploaded under: the path's,
// or the recording's name with .zip.
func UploadName(path, name string) string {
	if path != "" {
		return filepath.Base(path)
	}
	if name == "" {
		name = "record"
	}
	return name + ".zip"
}

// stationError is the text of a refusal: the error field of a JSON body
// when there is one, else the body itself.
func stationError(body []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return e.Error
	}
	return strings.TrimSpace(string(body))
}

// UploadRecording sends a recording zip to a station's Record Store: it
// asks the station for an upload authorization, signed on the agent's
// behalf, then sends the bytes with it. The station checks the hash as
// the bytes arrive.
func UploadRecording(ctx context.Context, t UploadTarget, zip []byte, name string) (*UploadResult, error) {
	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	sum := sha256.Sum256(zip)
	sha := hex.EncodeToString(sum[:])
	base := strings.TrimSuffix(t.BaseURL, "/")

	ask, _ := json.Marshal(map[string]interface{}{"sha256": sha, "size": len(zip), "type": "application/zip", "name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/records/upload-auth", bytes.NewReader(ask))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", t.Authorization)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the station at %s: %w", base, err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the station refused to authorize the upload (HTTP %d): %s", resp.StatusCode, stationError(body))
	}
	var grant struct {
		Authorization string `json:"authorization"`
		Upload        string `json:"upload"`
	}
	if err := json.Unmarshal(body, &grant); err != nil || grant.Authorization == "" {
		return nil, fmt.Errorf("the station's upload authorization was not understood")
	}
	if grant.Upload == "" {
		grant.Upload = base + "/upload"
	}

	put, err := http.NewRequestWithContext(ctx, http.MethodPut, grant.Upload, bytes.NewReader(zip))
	if err != nil {
		return nil, err
	}
	put.ContentLength = int64(len(zip))
	put.Header.Set("Authorization", grant.Authorization)
	put.Header.Set("Content-Type", "application/zip")
	resp, err = client.Do(put)
	if err != nil {
		return nil, fmt.Errorf("could not upload to %s: %w", grant.Upload, err)
	}
	body, _ = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("the station refused the upload (HTTP %d): %s", resp.StatusCode, stationError(body))
	}
	var out UploadResult
	if err := json.Unmarshal(body, &out); err != nil || out.URL == "" {
		return nil, fmt.Errorf("the station's upload answer was not understood")
	}
	if out.SHA256 == "" {
		out.SHA256 = sha
	}
	if out.Size == 0 {
		out.Size = int64(len(zip))
	}
	return &out, nil
}

// UploadSentence is the line the CLI and MCP tool add after a saved
// recording was uploaded.
func UploadSentence(u *UploadResult) string {
	msg := "Uploaded to " + u.URL
	var where []string
	if u.Device != "" {
		where = append(where, "device "+u.Device)
	}
	if u.Hold != "" {
		where = append(where, "hold "+u.Hold)
	}
	if len(where) > 0 {
		msg += " (" + strings.Join(where, ", ") + ")"
	}
	return msg
}
