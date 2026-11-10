package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/speech"
)

// A browser session carries audio through the page itself. A preload
// script gives every page a virtual microphone that say plays into, and
// taps what the page plays for listen. No browser flag, extension or audio
// device is involved, so it is the same in a visible browser, a headless
// one and a remote one. See docs/specs/say-listen-spec.md.

const browserAudioChannel = "vibium-audio"

// browserAudioScript runs in every document of the session, before the
// page's own scripts.
const browserAudioScript = `(channel) => {
  if (window.__vibiumAudio) return;
  const AC = window.AudioContext || window.webkitAudioContext;
  const A = window.__vibiumAudio = { micTracks: 0, capturing: false, spoke: false };
  if (!AC) return;
  const RATE = 16000;
  const now = () => performance.timeOrigin + performance.now();
  A.now = now;
  // The page's clock time at which a moment on a context's clock is heard.
  const clockAt = (ctx, t) => {
    let ts = null;
    try { ts = ctx.getOutputTimestamp(); } catch (e) {}
    if (ts && ts.performanceTime > 0 && ts.contextTime > 0) {
      return performance.timeOrigin + ts.performanceTime + (t - ts.contextTime) * 1000;
    }
    return now() + (t - ctx.currentTime) * 1000;
  };
  const connect = AudioNode.prototype.connect;
  const disconnect = AudioNode.prototype.disconnect;

  // ---- the virtual microphone -------------------------------------------
  let micCtx = null, micOut = null;
  const mic = () => {
    if (!micCtx) {
      micCtx = new AC();
      micCtx.__vibium = true;
      micOut = micCtx.createMediaStreamDestination();
      // A silent source keeps the track alive between clips.
      const quiet = micCtx.createGain();
      quiet.gain.value = 0;
      const osc = micCtx.createOscillator();
      connect.call(osc, quiet);
      connect.call(quiet, micOut);
      osc.start();
    }
    return micOut;
  };
  const md = navigator.mediaDevices;
  if (md && md.getUserMedia) {
    const real = md.getUserMedia.bind(md);
    md.getUserMedia = async function (constraints) {
      if (!constraints || !constraints.audio) return real(constraints);
      const out = mic();
      try { await micCtx.resume(); } catch (e) {}
      const track = out.stream.getAudioTracks()[0].clone();
      A.micTracks++;
      const stop = track.stop.bind(track);
      let stopped = false;
      track.stop = () => { if (!stopped) { stopped = true; A.micTracks--; } stop(); };
      if (constraints.video) {
        const stream = await real({ video: constraints.video });
        stream.addTrack(track);
        return stream;
      }
      return new MediaStream([track]);
    };
    if (md.enumerateDevices) {
      const list = md.enumerateDevices.bind(md);
      md.enumerateDevices = async () => (await list()).filter((d) => d.kind !== 'audioinput').concat([{
        deviceId: 'vibium', groupId: 'vibium', kind: 'audioinput', label: 'Vibium microphone',
        toJSON() { return { deviceId: this.deviceId, groupId: this.groupId, kind: this.kind, label: this.label }; },
      }]);
    }
  }
  // say plays 16-bit PCM into the microphone. It answers with when the
  // first and last sample played, as JSON, or "" when no microphone is open.
  A.say = async (b64, rate) => {
    if (!micCtx || A.micTracks <= 0) return '';
    if (micCtx.state !== 'running') {
      try { await Promise.race([micCtx.resume(), new Promise((r) => setTimeout(r, 1000))]); } catch (e) {}
      if (micCtx.state !== 'running') throw new Error('the page has not let audio start');
    }
    const bin = atob(b64);
    const n = bin.length >> 1;
    const buf = micCtx.createBuffer(1, n, rate);
    const data = buf.getChannelData(0);
    for (let i = 0; i < n; i++) {
      let v = bin.charCodeAt(2 * i) | (bin.charCodeAt(2 * i + 1) << 8);
      if (v >= 32768) v -= 65536;
      data[i] = v / 32768;
    }
    const src = micCtx.createBufferSource();
    src.buffer = buf;
    connect.call(src, micOut);
    const at = micCtx.currentTime + 0.05;
    const start = clockAt(micCtx, at);
    await new Promise((resolve) => { src.onended = resolve; src.start(at); });
    return JSON.stringify({ start, end: start + buf.duration * 1000 });
  };

  // ---- what the page plays ------------------------------------------------
  const send = (time, samples) => {
    let bin = '';
    for (let i = 0; i < samples.length; i++) {
      let v = Math.max(-1, Math.min(1, samples[i])) * 32767 | 0;
      if (v < 0) v += 65536;
      bin += String.fromCharCode(v & 255, v >> 8);
    }
    channel({ time, data: btoa(bin) });
  };
  const taps = new WeakMap();
  const tapFor = (ctx) => {
    let tap = taps.get(ctx);
    if (tap) return tap;
    tap = ctx.createScriptProcessor(4096, 2, 1);
    const mute = ctx.createGain();
    mute.gain.value = 0;
    connect.call(tap, mute);
    connect.call(mute, ctx.destination);
    const step = ctx.sampleRate / RATE;
    let out = [], outTime = 0, sum = 0, count = 0, pos = 0;
    tap.onaudioprocess = (e) => {
      if (!A.capturing) { out = []; sum = count = pos = 0; return; }
      const left = e.inputBuffer.getChannelData(0);
      const right = e.inputBuffer.numberOfChannels > 1 ? e.inputBuffer.getChannelData(1) : left;
      const base = clockAt(ctx, e.playbackTime);
      for (let i = 0; i < left.length; i++) {
        sum += (left[i] + right[i]) / 2;
        count++;
        pos++;
        if (pos >= step) {
          if (out.length === 0) outTime = base + (i / ctx.sampleRate) * 1000;
          out.push(sum / count);
          sum = count = 0;
          pos -= step;
          if (out.length >= RATE / 10) { send(outTime, out); out = []; }
        }
      }
    };
    taps.set(ctx, tap);
    return tap;
  };
  AudioNode.prototype.connect = function (to, output, input) {
    const result = connect.apply(this, arguments);
    try {
      if (to && to.context && !to.context.__vibium && to === to.context.destination && taps.get(to.context) !== this) {
        connect.call(this, tapFor(to.context), output || 0, 0);
      }
    } catch (e) {}
    return result;
  };
  AudioNode.prototype.disconnect = function (to) {
    try {
      if (to && to.context && to === to.context.destination && taps.get(to.context)) {
        disconnect.call(this, taps.get(to.context));
      }
    } catch (e) {}
    return disconnect.apply(this, arguments);
  };
  // Media elements: an <audio> or <video>, or one playing a WebRTC stream.
  let elementCtx = null;
  const tapped = new WeakSet();
  const play = HTMLMediaElement.prototype.play;
  HTMLMediaElement.prototype.play = function () {
    try {
      if (!tapped.has(this)) {
        const stream = this.captureStream ? this.captureStream() : (this.mozCaptureStream ? this.mozCaptureStream() : null);
        if (stream) {
          tapped.add(this);
          const hook = () => {
            if (!stream.getAudioTracks().length) return;
            if (!elementCtx) { elementCtx = new AC(); elementCtx.__vibium = true; }
            elementCtx.resume().catch(() => {});
            connect.call(elementCtx.createMediaStreamSource(stream), tapFor(elementCtx));
          };
          if (stream.getAudioTracks().length) hook(); else stream.addEventListener('addtrack', hook, { once: true });
        }
      }
    } catch (e) {}
    return play.apply(this, arguments);
  };
  if (window.speechSynthesis && window.speechSynthesis.speak) {
    const speak = window.speechSynthesis.speak.bind(window.speechSynthesis);
    window.speechSynthesis.speak = (u) => { A.spoke = true; return speak(u); };
  }
}`

// browserAudio is the session target for a page in a browser.
type browserAudio struct {
	h      *Handlers
	client *bidi.Client
	tap    *speech.Tap
}

// installBrowserAudio adds the audio script to the session: every document
// from now on, and the pages already open. It reports whether the session
// took it; a device on a Roadie does not run scripts.
func (h *Handlers) installBrowserAudio(client *bidi.Client) bool {
	arg := []map[string]interface{}{{"type": "channel", "value": map[string]interface{}{"channel": browserAudioChannel}}}
	if _, err := client.SendCommandWithTimeout("script.addPreloadScript", map[string]interface{}{
		"functionDeclaration": browserAudioScript, "arguments": arg,
	}, 15*time.Second); err != nil {
		return false
	}
	subscribeEvents(client, []string{"script.message"})
	for _, ctx := range browserContexts(client, "") {
		client.SendCommandWithTimeout("script.callFunction", map[string]interface{}{
			"functionDeclaration": browserAudioScript, "arguments": arg,
			"target": map[string]interface{}{"context": ctx}, "awaitPromise": false,
		}, 10*time.Second)
	}
	return true
}

// browserContexts lists a page and its frames, or every page and frame
// when root is empty.
func browserContexts(client *bidi.Client, root string) []string {
	tree, err := client.GetTree()
	if err != nil {
		return nil
	}
	var out []string
	var walk func(nodes []bidi.BrowsingContextInfo, inside bool)
	walk = func(nodes []bidi.BrowsingContextInfo, inside bool) {
		for _, n := range nodes {
			in := inside || root == "" || n.Context == root
			if in {
				out = append(out, n.Context)
			}
			walk(n.Children, in)
		}
	}
	walk(tree.Contexts, false)
	return out
}

func (b *browserAudio) Name() string             { return "session" }
func (b *browserAudio) Key() string              { return "session" }
func (b *browserAudio) Live() bool               { return true }
func (b *browserAudio) EchoGuard() time.Duration { return 0 }

// call runs a function in one context and returns its string result.
func (b *browserAudio) call(ctx string, fn string, timeout time.Duration, args ...interface{}) (string, error) {
	serialized := []map[string]interface{}{}
	for _, a := range args {
		switch v := a.(type) {
		case string:
			serialized = append(serialized, map[string]interface{}{"type": "string", "value": v})
		case int:
			serialized = append(serialized, map[string]interface{}{"type": "number", "value": v})
		case bool:
			serialized = append(serialized, map[string]interface{}{"type": "boolean", "value": v})
		}
	}
	resp, err := b.client.SendCommandWithTimeout("script.callFunction", map[string]interface{}{
		"functionDeclaration": fn, "arguments": serialized, "awaitPromise": true, "resultOwnership": "none",
		"target": map[string]interface{}{"context": ctx},
	}, timeout)
	if err != nil {
		return "", err
	}
	res, err := bidi.ParseScriptResult(resp.Result)
	if err != nil {
		return "", err
	}
	var value struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}
	raw, _ := json.Marshal(res.Result)
	json.Unmarshal(raw, &value)
	s, _ := value.Value.(string)
	return s, nil
}

func (b *browserAudio) page() string { return b.h.currentContext() }

// Now is the present on the page's clock: milliseconds since the epoch, as
// performance.timeOrigin + performance.now() gives them.
func (b *browserAudio) Now() float64 {
	now, _ := b.Sync()
	return now
}

func (b *browserAudio) Sync() (float64, float64) {
	contexts := browserContexts(b.client, b.page())
	if len(contexts) == 0 {
		return 0, 0
	}
	best, bestRTT := 0.0, -1.0
	var bestAt time.Time
	for i := 0; i < 5; i++ {
		sent := time.Now()
		s, err := b.call(contexts[0], `() => String(performance.timeOrigin + performance.now())`, 5*time.Second)
		got := time.Now()
		var now float64
		if err != nil || json.Unmarshal([]byte(s), &now) != nil {
			continue
		}
		if rtt := float64(got.Sub(sent)) / float64(time.Millisecond); bestRTT < 0 || rtt < bestRTT {
			best, bestRTT, bestAt = now, rtt, sent.Add(got.Sub(sent)/2)
		}
	}
	if bestRTT < 0 {
		return 0, 0
	}
	return best + float64(time.Since(bestAt))/float64(time.Millisecond), bestRTT
}

func (b *browserAudio) Play(_ context.Context, a speech.Audio) (float64, float64, error) {
	clip := a.Resample(speech.Rate)
	if clip.Duration() > roadieMaxClip {
		return 0, 0, speechErr("no_audio", "a clip for a page is at most 60 seconds")
	}
	data := base64.StdEncoding.EncodeToString(clip.Bytes())
	for _, ctx := range browserContexts(b.client, b.page()) {
		s, err := b.call(ctx, `(data, rate) => window.__vibiumAudio && window.__vibiumAudio.say ? window.__vibiumAudio.say(data, rate) : ''`,
			clip.Duration()+30*time.Second, data, speech.Rate)
		if err != nil {
			return 0, 0, speechErr("no_audio", "the page did not take the audio: "+err.Error())
		}
		var res struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		}
		if s != "" && json.Unmarshal([]byte(s), &res) == nil && res.End > res.Start {
			return res.Start, res.End, nil
		}
	}
	return 0, 0, speechErr("no_audio", "no page has the microphone open: the page must ask for the microphone before say can speak into it. Use --target local for this machine's speakers")
}

// capturing turns the page's audio tap on or off in every frame.
func (b *browserAudio) capturing(on bool) {
	for _, ctx := range browserContexts(b.client, b.page()) {
		b.call(ctx, `(on) => { if (window.__vibiumAudio) window.__vibiumAudio.capturing = on; return '' }`, 5*time.Second, on)
	}
}

func (b *browserAudio) Capture(ctx context.Context) (speech.Capture, error) {
	// A page that navigated has a fresh script with its tap off.
	b.capturing(true)
	return b.tap.Capture(ctx)
}

func (b *browserAudio) open(context.Context) (speech.Capture, error) {
	now, _ := b.Sync()
	if now == 0 {
		return nil, speechErr("no_audio", "the page did not answer")
	}
	c := &browserCapture{b: b, chunks: make(chan speech.Chunk, 256), done: make(chan struct{}), next: now, offset: now - localMs()}
	b.h.setPageSink(c)
	b.capturing(true)
	return c, nil
}

func localMs() float64 { return float64(time.Now().UnixMicro()) / 1000 }

// browserCapture turns the page's chunks into one continuous stream. A
// page sends audio only while one of its audio contexts runs, so the time
// between chunks is filled with silence on the page's clock.
type browserCapture struct {
	b       *browserAudio
	chunks  chan speech.Chunk
	done    chan struct{}
	once    sync.Once
	next    float64 // page-clock time of the next sample owed
	offset  float64 // page clock minus this machine's clock
	pending *speech.Chunk
}

// browserQuiet is how long the page may send nothing before the stream is
// carried on with silence.
const browserQuiet = 250 * time.Millisecond

func (c *browserCapture) deliver(chunk speech.Chunk) {
	select {
	case c.chunks <- chunk:
	case <-c.done:
	default:
	}
}

func (c *browserCapture) Next(ctx context.Context) (speech.Chunk, error) {
	if c.pending != nil {
		chunk := *c.pending
		c.pending = nil
		c.next = chunk.Time + float64(len(chunk.Samples))*1000/speech.Rate
		return chunk, nil
	}
	for {
		select {
		case chunk := <-c.chunks:
			// Audio for a time already covered, by silence or by another
			// frame, gives up the part that overlaps.
			if late := c.next - chunk.Time; late > roadieChunkGap {
				skip := int(late * speech.Rate / 1000)
				if skip >= len(chunk.Samples) {
					continue
				}
				chunk.Samples, chunk.Time = chunk.Samples[skip:], c.next
			} else if chunk.Time-c.next > roadieChunkGap {
				gap := speech.Chunk{Samples: make([]int16, int((chunk.Time-c.next)*speech.Rate/1000)), Time: c.next}
				c.next = chunk.Time
				c.pending = &chunk // read on the next call
				return gap, nil
			}
			c.next = chunk.Time + float64(len(chunk.Samples))*1000/speech.Rate
			return chunk, nil
		case <-time.After(browserQuiet):
			// Nothing is playing. Carry the stream up to a moment ago.
			until := localMs() + c.offset - float64(browserQuiet/time.Millisecond)
			if until-c.next < 100 {
				continue
			}
			silence := speech.Chunk{Samples: make([]int16, int((until-c.next)*speech.Rate/1000)), Time: c.next}
			c.next = until
			return silence, nil
		case <-c.done:
			return speech.Chunk{}, io.EOF
		case <-ctx.Done():
			return speech.Chunk{}, ctx.Err()
		}
	}
}

func (c *browserCapture) Close() {
	c.once.Do(func() {
		close(c.done)
		c.b.h.clearPageSink(c)
		go c.b.capturing(false)
	})
}

func (h *Handlers) setPageSink(c *browserCapture) {
	h.audioMu.Lock()
	h.pageSink = c
	h.audioMu.Unlock()
}

func (h *Handlers) clearPageSink(c *browserCapture) {
	h.audioMu.Lock()
	if h.pageSink == c {
		h.pageSink = nil
	}
	h.audioMu.Unlock()
}

// pageAudioEvent routes a chunk a page sent over the audio channel to the
// running capture. It reports whether the event was one.
func (h *Handlers) pageAudioEvent(msg string) bool {
	if !strings.Contains(msg, `"`+browserAudioChannel+`"`) {
		return false
	}
	var ev struct {
		Method string `json:"method"`
		Params struct {
			Channel string `json:"channel"`
			Data    struct {
				Value [][]json.RawMessage `json:"value"`
			} `json:"data"`
		} `json:"params"`
	}
	if json.Unmarshal([]byte(msg), &ev) != nil || ev.Method != "script.message" || ev.Params.Channel != browserAudioChannel {
		return false
	}
	// The data is a BiDi object: [["time", {number}], ["data", {string}]].
	var at float64
	var data string
	for _, pair := range ev.Params.Data.Value {
		if len(pair) != 2 {
			continue
		}
		var key string
		var value struct {
			Value interface{} `json:"value"`
		}
		json.Unmarshal(pair[0], &key)
		json.Unmarshal(pair[1], &value)
		switch key {
		case "time":
			at, _ = value.Value.(float64)
		case "data":
			data, _ = value.Value.(string)
		}
	}
	h.audioMu.Lock()
	sink := h.pageSink
	h.audioMu.Unlock()
	if sink == nil || data == "" {
		return true
	}
	pcm, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return true
	}
	sink.deliver(speech.Chunk{Samples: speech.FromBytes(pcm, speech.Rate, 1).Samples, Time: at})
	return true
}
