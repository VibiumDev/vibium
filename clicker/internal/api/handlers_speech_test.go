package api

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vibium/clicker/internal/speech"
)

func speechTone(hz float64, d time.Duration) []int16 {
	out := make([]int16, int(d.Seconds()*speech.Rate))
	for i := range out {
		out[i] = int16(6000 * math.Sin(2*math.Pi*hz*float64(i)/speech.Rate))
	}
	return out
}

// audioDevice is a session that speaks roadie:audio and runs no scripts,
// as a phone on a Roadie does. Its clock starts at 5000 ms. A clip plays
// at once; the device then waits 640 ms and plays a tone for a second.
type audioDevice struct {
	mu     sync.Mutex
	log    []string
	played int
	ws     *websocket.Conn
	wmu    sync.Mutex
}

func (d *audioDevice) send(v interface{}) {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	d.ws.WriteJSON(v)
}

func (d *audioDevice) url(t *testing.T) string {
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		d.ws = ws
		for {
			var cmd struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
				Params struct {
					Data string `json:"data"`
				} `json:"params"`
			}
			if ws.ReadJSON(&cmd) != nil {
				return
			}
			d.mu.Lock()
			d.log = append(d.log, cmd.Method)
			d.mu.Unlock()
			ok := func(result interface{}) {
				d.send(map[string]interface{}{"type": "success", "id": cmd.ID, "result": result})
			}
			switch cmd.Method {
			case "session.status":
				ok(map[string]interface{}{"ready": false, "message": "already has session"})
			case "script.addPreloadScript":
				d.send(map[string]interface{}{"type": "error", "id": cmd.ID, "error": "unknown command", "message": "a device runs no scripts"})
			case "browsingContext.getTree":
				ok(map[string]interface{}{"contexts": []map[string]interface{}{{"context": "device", "url": "", "children": []interface{}{}}}})
			case "roadie:audio.clock":
				ok(map[string]float64{"now": 5000})
			case "roadie:audio.capture.start":
				ok(map[string]interface{}{"capture": "c1", "sampleRate": 16000, "startTime": 5000})
			case "roadie:audio.play":
				pcm, _ := base64.StdEncoding.DecodeString(cmd.Params.Data)
				d.mu.Lock()
				d.played = len(pcm)
				d.mu.Unlock()
				ms := float64(len(pcm)/2) * 1000 / 16000
				ok(map[string]float64{"startTime": 5040, "endTime": 5040 + ms})
				heard := make([]int16, int((40+ms+640)*16))
				heard = append(heard, speechTone(400, time.Second)...)
				heard = append(heard, make([]int16, 3*16000)...)
				for i := 0; i*1600 < len(heard); i++ {
					piece := speech.Audio{SampleRate: 16000, Channels: 1, Samples: heard[i*1600 : min((i+1)*1600, len(heard))]}
					d.send(map[string]interface{}{"type": "event", "method": "roadie:audio.chunk", "params": map[string]interface{}{
						"capture": "c1", "time": 5000 + float64(i)*100, "data": base64.StdEncoding.EncodeToString(piece.Bytes()),
					}})
				}
			default:
				ok(map[string]interface{}{})
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

type wireReply struct {
	Type    string                 `json:"type"`
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Result  map[string]interface{} `json:"result"`
}

// wireCall sends one command as a client would and waits for its answer.
func wireCall(t *testing.T, router *Router, client *recordingClient, id int, method string, params map[string]interface{}) wireReply {
	t.Helper()
	raw, _ := json.Marshal(map[string]interface{}{"id": id, "method": method, "params": params})
	router.OnClientMessage(client, string(raw))
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		client.mu.Lock()
		sent := append([]string(nil), client.sent...)
		client.mu.Unlock()
		for _, msg := range sent {
			var head struct {
				ID int `json:"id"`
			}
			if json.Unmarshal([]byte(msg), &head) == nil && head.ID == id {
				var reply wireReply
				json.Unmarshal([]byte(msg), &reply)
				return reply
			}
		}
	}
	t.Fatalf("no answer to %s (id %d)", method, id)
	return wireReply{}
}

func deviceRouter(t *testing.T) (*Router, *recordingClient, *audioDevice) {
	t.Helper()
	d := &audioDevice{}
	router := NewRouter("chrome", true, d.url(t), nil, nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)
	router.OnClientConnect(client)
	return router, client, d
}

func writeClip(t *testing.T, path string) {
	t.Helper()
	clip := speech.Audio{SampleRate: 16000, Channels: 1, Samples: speechTone(300, 500*time.Millisecond)}
	if err := os.WriteFile(path, clip.WAV(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSayAndListenOverTheWire(t *testing.T) {
	router, client, d := deviceRouter(t)
	clip := filepath.Join(t.TempDir(), "q.wav")
	writeClip(t, clip)

	said := wireCall(t, router, client, 11, speechSayMethod, map[string]interface{}{"file": clip})
	if said.Type != "success" || said.Result["target"] != "session" || said.Result["durationMs"] != 500.0 {
		t.Fatalf("say: %+v", said)
	}
	d.mu.Lock()
	played, log := d.played, strings.Join(d.log, " ")
	d.mu.Unlock()
	if played != 16000 || !strings.Contains(log, "roadie:audio.capture.start roadie:audio.play") {
		t.Fatalf("the device got %d bytes; its commands: %s", played, log)
	}

	heard := wireCall(t, router, client, 12, speechListenMethod, map[string]interface{}{"transcribe": false, "context": "device"})
	if heard.Type != "success" || heard.Result["speech"] != true || heard.Result["stoppedBy"] != "silence" || heard.Result["target"] != "session" {
		t.Fatalf("listen: %+v", heard)
	}
	if latency, _ := heard.Result["latencyMs"].(float64); latency < 600 || latency > 700 {
		t.Errorf("the reply began %v ms after the say ended, want about 640", heard.Result["latencyMs"])
	}

	// A failure keeps its code, and the message goes without it.
	failed := wireCall(t, router, client, 13, speechSayMethod, map[string]interface{}{"text": "hello", "provider": "nobody"})
	if failed.Type != "error" || failed.Error != "not_configured" || strings.HasPrefix(failed.Message, "not_configured") || !strings.Contains(failed.Message, "nobody") {
		t.Errorf("an unknown provider: %+v", failed)
	}
	bad := wireCall(t, router, client, 14, speechListenMethod, map[string]interface{}{"loud": true})
	if bad.Type != "error" || bad.Error != "error" || !strings.Contains(bad.Message, "unsupported argument loud") {
		t.Errorf("an unknown option: %+v", bad)
	}

	// The device's audio goes to the capture, never to the client.
	client.mu.Lock()
	defer client.mu.Unlock()
	for _, msg := range client.sent {
		if strings.Contains(msg, "roadie:audio.chunk") {
			t.Fatal("an audio chunk reached the client")
		}
	}
}

func TestSpeechStepsAreRecordedOverTheWire(t *testing.T) {
	router, client, _ := deviceRouter(t)
	sessionVal, _ := router.sessions.Load(client.ID())
	session := sessionVal.(*BrowserSession)
	recorder := NewRecorder()
	recorder.Start(RecordingStartOptions{}, nil)
	session.mu.Lock()
	session.recorder = recorder
	session.mu.Unlock()

	clip := filepath.Join(t.TempDir(), "q.wav")
	writeClip(t, clip)
	if r := wireCall(t, router, client, 21, speechSayMethod, map[string]interface{}{"file": clip}); r.Type != "success" {
		t.Fatalf("say: %+v", r)
	}
	if r := wireCall(t, router, client, 22, speechListenMethod, map[string]interface{}{"transcribe": false}); r.Type != "success" {
		t.Fatalf("listen: %+v", r)
	}
	if r := wireCall(t, router, client, 23, speechSayMethod, map[string]interface{}{"text": "hello", "provider": "nobody"}); r.Type != "error" {
		t.Fatalf("a say that fails: %+v", r)
	}
	// An after event is written when its step ends, just after the answer.
	time.Sleep(200 * time.Millisecond)
	session.speech.StopAudioRecording()
	raw, err := recorder.Stop()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, zf := range zr.File {
		rc, _ := zf.Open()
		files[zf.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	for _, name := range []string{"trace.trace", "audio/index.json", "audio/in-session.wav", "audio/out-0001.wav"} {
		if len(files[name]) == 0 {
			t.Fatalf("the zip has no %s; it has %d entries", name, len(files))
		}
	}

	var calls []string
	results := map[string]map[string]interface{}{}
	failures := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(files["trace.trace"])), "\n") {
		var ev map[string]interface{}
		json.Unmarshal([]byte(line), &ev)
		id, _ := ev["callId"].(string)
		method, _ := ev["method"].(string)
		switch {
		case ev["type"] == "before" && isSpeechMethod(method):
			calls = append(calls, id)
			if ev["class"] != "Speech" {
				t.Errorf("%s is in class %v", method, ev["class"])
			}
			if params, _ := ev["params"].(map[string]interface{}); params["_recordCallId"] != nil {
				t.Errorf("the step's params carry the router's own key: %v", params)
			}
		case ev["type"] == "after":
			if r, ok := ev["result"].(map[string]interface{}); ok {
				results[id] = r
			}
			if e, ok := ev["error"].(map[string]interface{}); ok {
				failures[id], _ = e["message"].(string)
			}
		}
	}
	if len(calls) != 3 {
		t.Fatalf("%d speech steps in the trace, want 3:\n%s", len(calls), files["trace.trace"])
	}
	say, listen, failed := calls[0], calls[1], calls[2]
	if results[say]["clip"] != "out-0001" || results[listen]["track"] != "in-session" || results[listen]["latencyFrom"] != say {
		t.Errorf("step results: say %v, listen %v", results[say], results[listen])
	}
	if !strings.HasPrefix(failures[failed], "not_configured: ") || results[failed] != nil {
		t.Errorf("the failed step: error %q, result %v", failures[failed], results[failed])
	}
}

func TestSayAndListenWithNoSession(t *testing.T) {
	dir := t.TempDir()
	clip, out := filepath.Join(dir, "q.wav"), filepath.Join(dir, "out.wav")
	writeClip(t, clip)

	router := NewRouter("chrome", true, "", nil, nil)
	client := &recordingClient{}
	// A client with no session is turned away, as before.
	if r := wireCall(t, router, client, 1, speechSayMethod, map[string]interface{}{"file": clip, "target": out}); r.Type != "error" || r.Error != "invalid session id" {
		t.Fatalf("with no session and no leave: %+v", r)
	}

	router.SetSessionless(true)
	said := wireCall(t, router, client, 2, speechSayMethod, map[string]interface{}{"file": clip, "target": out})
	if said.Type != "success" || said.Result["target"] != out || said.Result["path"] != out || said.Result["durationMs"] != 500.0 {
		t.Fatalf("say into a file: %+v", said)
	}
	heard := wireCall(t, router, client, 3, speechListenMethod, map[string]interface{}{"target": out, "silence": 0, "transcribe": false})
	if heard.Type != "success" || heard.Result["speech"] != true || heard.Result["target"] != out {
		t.Fatalf("listen to that file: %+v", heard)
	}
	// There is no session to speak into.
	none := wireCall(t, router, client, 4, speechSayMethod, map[string]interface{}{"file": clip, "target": "session"})
	if none.Type != "error" || none.Error != "invalid_target" || !strings.Contains(none.Message, "no session is open") {
		t.Errorf("target session with none open: %+v", none)
	}
	// Other commands still need a session.
	if r := wireCall(t, router, client, 5, "vibium:page.title", map[string]interface{}{}); r.Type != "error" || r.Error != "invalid session id" {
		t.Errorf("a page command with no session: %+v", r)
	}
}
