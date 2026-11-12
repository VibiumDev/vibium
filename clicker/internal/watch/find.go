package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/pion/webrtc/v4"
)

// way is how to reach a stream's signaling.
type way struct {
	station bool
	url     string
	header  http.Header
	grant   json.RawMessage
	ice     []webrtc.ICEServer
}

// find works out where the stream of the session's device is. A station
// names its devices and hands out a grant for the one this caller holds; a
// Roadie has one stream at a fixed path. Anything else has none.
func find(ctx context.Context, src Source) (*way, error) {
	base := strings.TrimSuffix(src.BaseURL, "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return nil, ErrNoStream
	}
	ws := "ws" + strings.TrimPrefix(base, "http")
	header := http.Header{}
	if auth := src.Header.Get("Authorization"); auth != "" {
		header.Set("Authorization", auth)
	}
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	client := &http.Client{}

	var about struct {
		Pubkey string `json:"pubkey"`
	}
	if status, err := call(ctx, client, header, "GET", base+"/v1/station", nil, &about); err != nil || status != http.StatusOK || about.Pubkey == "" {
		// Not a station. A Roadie answers its signaling path or nothing does.
		return &way{url: ws + "/api/webrtc/signal", header: header}, nil
	}

	var devices []struct {
		ID   string `json:"id"`
		Hold *struct {
			Mine   bool     `json:"mine"`
			Scopes []string `json:"scopes"`
		} `json:"hold"`
	}
	if status, err := call(ctx, client, header, "GET", base+"/v1/devices", nil, &devices); err != nil {
		return nil, fmt.Errorf("the station's devices: %w", err)
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("the station's devices: HTTP %d", status)
	}
	var mine, driven []string
	for _, d := range devices {
		if d.Hold == nil {
			continue
		}
		if d.Hold.Mine {
			mine = append(mine, d.ID)
		}
		if slices.Contains(d.Hold.Scopes, "bidi") {
			driven = append(driven, d.ID)
		}
	}
	// A station from before holds were marked says only which devices are
	// driven; one driven device can only be this session's.
	if len(mine) == 0 {
		mine = driven
	}
	if len(mine) != 1 {
		return nil, errors.New("the station does not say which device this session drives")
	}

	var hold struct {
		Grant  json.RawMessage    `json:"grant"`
		Signal string             `json:"signal"`
		ICE    []webrtc.ICEServer `json:"ice"`
		Error  string             `json:"error"`
	}
	status, err := call(ctx, client, header, "POST", base+"/v1/devices/"+mine[0]+"/hold", []byte(`{"scopes":["view"]}`), &hold)
	if err != nil {
		return nil, fmt.Errorf("a grant to watch %s: %w", mine[0], err)
	}
	if status != http.StatusOK || hold.Signal == "" {
		if hold.Error != "" {
			return nil, fmt.Errorf("a grant to watch %s: %s", mine[0], hold.Error)
		}
		return nil, fmt.Errorf("a grant to watch %s: HTTP %d", mine[0], status)
	}
	return &way{station: true, url: ws + hold.Signal, header: header, grant: hold.Grant, ice: hold.ICE}, nil
}

func call(ctx context.Context, client *http.Client, header http.Header, method, url string, body []byte, into any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header = header.Clone()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	json.Unmarshal(data, into)
	return resp.StatusCode, nil
}
