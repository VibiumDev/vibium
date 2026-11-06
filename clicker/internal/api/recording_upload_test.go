package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStationBaseURL(t *testing.T) {
	cases := map[string]string{
		"wss://station.example/session":          "https://station.example",
		"ws://127.0.0.1:18443/session":           "http://127.0.0.1:18443",
		"wss://station.example/session/abc123":   "https://station.example",
		"wss://host/station/session?x=1":         "https://host/station",
		"https://station.example":                "https://station.example",
		"wss://host/v1/devices/rig_1/phone/bidi": "https://host/v1/devices/rig_1/phone/bidi",
	}
	for in, want := range cases {
		got, err := StationBaseURL(in)
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "not a url", "ftp://host/session", "/session"} {
		if _, err := StationBaseURL(bad); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}

func TestResolveUpload(t *testing.T) {
	auth := http.Header{"Authorization": {"Bearer stk_test"}}
	if tg, err := ResolveUpload(nil, "wss://s/session", auth); tg != nil || err != nil {
		t.Fatalf("absent: %v %v", tg, err)
	}
	if tg, err := ResolveUpload(false, "wss://s/session", auth); tg != nil || err != nil {
		t.Fatalf("false: %v %v", tg, err)
	}
	if tg, err := ResolveUpload("", "wss://s/session", auth); tg != nil || err != nil {
		t.Fatalf("empty string: %v %v", tg, err)
	}
	if _, err := ResolveUpload(true, "", auth); err == nil || !strings.Contains(err.Error(), "needs a station") {
		t.Fatalf("true without a connection: %v", err)
	}
	if _, err := ResolveUpload(true, "wss://s/session", nil); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("true without a credential: %v", err)
	}
	tg, err := ResolveUpload(true, "wss://s/session", auth)
	if err != nil || tg.BaseURL != "https://s" || tg.Authorization != "Bearer stk_test" {
		t.Fatalf("true: %+v %v", tg, err)
	}
	tg, err = ResolveUpload("https://other.example/", "", auth)
	if err != nil || tg.BaseURL != "https://other.example" {
		t.Fatalf("named station: %+v %v", tg, err)
	}
	if _, err := ResolveUpload(42, "wss://s/session", auth); err == nil {
		t.Fatal("a number is not an upload option")
	}
}

// fakeStation stands in for a station's Record Store: the authorization
// call and the upload, checking what the client sends.
type fakeStation struct {
	t         *testing.T
	bearer    string
	refuse    int    // status to refuse the authorization with, 0 for none
	refusePut int    // status to refuse the upload with
	auth      string // the authorization it hands out
	sha       string // what the client declared
	gotPut    []byte
	putAuth   string
	putType   string
	server    *httptest.Server
}

func newFakeStation(t *testing.T) *fakeStation {
	f := &fakeStation{t: t, bearer: "Bearer stk_test", auth: "Nostr ZmFrZQ=="}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/records/upload-auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != f.bearer {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "sign in, or send Authorization: Bearer <token>"})
			return
		}
		var req struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
			Type   string `json:"type"`
			Name   string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Type != "application/zip" || req.Name == "" || req.Size <= 0 || len(req.SHA256) != 64 {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "sha256, size, type and name are needed"})
			return
		}
		if f.refuse != 0 {
			w.WriteHeader(f.refuse)
			json.NewEncoder(w).Encode(map[string]string{"error": "a file is at most 4 bytes here"})
			return
		}
		f.sha = req.SHA256
		json.NewEncoder(w).Encode(map[string]string{"authorization": f.auth, "upload": f.server.URL + "/upload", "expires": "2026-11-06T00:00:00Z"})
	})
	mux.HandleFunc("PUT /upload", func(w http.ResponseWriter, r *http.Request) {
		f.putAuth = r.Header.Get("Authorization")
		f.putType = r.Header.Get("Content-Type")
		f.gotPut, _ = io.ReadAll(r.Body)
		if f.putAuth != f.auth {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "send Authorization: Nostr <base64 event>"})
			return
		}
		sum := sha256.Sum256(f.gotPut)
		if hex.EncodeToString(sum[:]) != f.sha {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "the body does not hash to the authorized file"})
			return
		}
		if f.refusePut != 0 {
			w.WriteHeader(f.refusePut)
			json.NewEncoder(w).Encode(map[string]string{"error": "the store is full"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"url": f.server.URL + "/" + f.sha + ".zip", "sha256": f.sha, "size": len(f.gotPut),
			"type": "application/zip", "uploaded": 1, "name": "r.zip", "uploader": "token:ci", "hold": "hld_1", "device": "rig_1/phone", "event": "ev1"})
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func TestUploadRecording(t *testing.T) {
	f := newFakeStation(t)
	zip := []byte("PK\x03\x04 a recording")
	target := UploadTarget{BaseURL: f.server.URL, Authorization: "Bearer stk_test"}
	res, err := UploadRecording(context.Background(), target, zip, "r.zip")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(zip)
	if res.SHA256 != hex.EncodeToString(sum[:]) || res.Size != int64(len(zip)) || res.Hold != "hld_1" || res.Device != "rig_1/phone" || res.Event != "ev1" || !strings.HasSuffix(res.URL, ".zip") {
		t.Fatalf("result: %+v", res)
	}
	if string(f.gotPut) != string(zip) || f.putAuth != f.auth || f.putType != "application/zip" {
		t.Fatalf("the upload carried %d bytes, auth %q, type %q", len(f.gotPut), f.putAuth, f.putType)
	}
	fields := res.Fields()
	if fields["url"] != res.URL || fields["hold"] != "hld_1" {
		t.Fatalf("fields: %v", fields)
	}
	if s := UploadSentence(res); !strings.Contains(s, res.URL) || !strings.Contains(s, "hold hld_1") {
		t.Fatalf("sentence: %s", s)
	}

	// The wrong credential is refused at the authorization, with the
	// station's reason in the error.
	_, err = UploadRecording(context.Background(), UploadTarget{BaseURL: f.server.URL, Authorization: "Bearer stk_wrong"}, zip, "r.zip")
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Bearer") {
		t.Fatalf("wrong credential: %v", err)
	}
	f.refuse = 413
	_, err = UploadRecording(context.Background(), target, zip, "r.zip")
	if err == nil || !strings.Contains(err.Error(), "413") || !strings.Contains(err.Error(), "at most 4 bytes") {
		t.Fatalf("refused authorization: %v", err)
	}
	f.refuse, f.refusePut = 0, 507
	_, err = UploadRecording(context.Background(), target, zip, "r.zip")
	if err == nil || !strings.Contains(err.Error(), "507") || !strings.Contains(err.Error(), "full") {
		t.Fatalf("refused upload: %v", err)
	}
	// A station that is not there.
	_, err = UploadRecording(context.Background(), UploadTarget{BaseURL: "http://127.0.0.1:1", Authorization: "Bearer x"}, zip, "r.zip")
	if err == nil || !strings.Contains(err.Error(), "could not reach") {
		t.Fatalf("unreachable: %v", err)
	}
}

func TestUploadName(t *testing.T) {
	if UploadName("/tmp/out/run.zip", "login") != "run.zip" || UploadName("", "login") != "login.zip" || UploadName("", "") != "record.zip" {
		t.Fatal("upload names")
	}
}
