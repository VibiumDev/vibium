package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/speech"
)

// audioRecording puts what say sends and listen hears into the running
// recording. For a session it also keeps one continuous track of what the
// device plays, from the first say or listen to the end of the recording.
type audioRecording struct {
	h      *Handlers
	target speech.Target
	// offset turns a time on the target's clock into the recording's
	// timeline: timeline = target time - offset.
	offset float64

	capture speech.Capture
	done    chan struct{}
	once    sync.Once
}

// clockSyncer is a target that can say how good its clock reading is.
type clockSyncer interface {
	Sync() (now float64, rttMs float64)
}

// beginAudioRecording returns the recording's audio for this target,
// starting its continuous track when the target is the session. It
// returns nil when nothing is being recorded; its methods accept that.
func (h *Handlers) beginAudioRecording(t speech.Target) *audioRecording {
	if h.recorder == nil || !h.recorder.IsRecording() {
		return nil
	}
	if h.audioRec != nil && h.audioRec.target.Key() == t.Key() {
		return h.audioRec
	}
	rec := &audioRecording{h: h, target: t}
	if !t.Live() {
		return rec
	}
	now, rtt := t.Now(), 0.0
	if s, ok := t.(clockSyncer); ok {
		now, rtt = s.Sync()
	}
	at := h.recorder.TimelineMs(time.Now())
	rec.offset = now - at
	source := t.Name()
	if _, ok := t.(*roadieAudio); ok {
		source = "roadie"
	}
	h.recorder.SetAudioClock(t.Name(), map[string]interface{}{
		"source": source,
		"sync":   []map[string]float64{{"atMs": round1(at), "offsetMs": round1(rec.offset), "rttMs": round1(rtt)}},
	})
	if t.Name() != "session" {
		// The local microphone is recorded only while a listen runs.
		return rec
	}
	capture, err := t.Capture(context.Background())
	if err != nil {
		return rec
	}
	h.stopAudioRecording()
	rec.capture, rec.done = capture, make(chan struct{})
	h.audioRec = rec
	recorder := h.recorder
	go func() {
		defer close(rec.done)
		for {
			chunk, err := capture.Next(context.Background())
			if err != nil {
				return
			}
			recorder.AppendAudio(t.Name(), chunk.Samples, chunk.Time-rec.offset)
		}
	}()
	return rec
}

// stopAudioRecording ends the continuous track. Called before the
// recording is built into its zip.
func (h *Handlers) stopAudioRecording() {
	rec := h.audioRec
	h.audioRec = nil
	if rec == nil || rec.capture == nil {
		return
	}
	rec.once.Do(rec.capture.Close)
	select {
	case <-rec.done:
	case <-time.After(2 * time.Second):
	}
}

func round1(ms float64) float64 { return float64(int64(ms*10+0.5)) / 10 }

// said records the clip a say sent and when it played.
func (r *audioRecording) said(res *speech.SayResult) {
	if r == nil {
		return
	}
	h := r.h
	meta := map[string]interface{}{"target": res.Target, "text": res.Text, "provider": res.Provider, "model": res.Model, "voice": res.Voice}
	if res.File != "" {
		meta["source"] = "file"
	}
	clip := h.recorder.AddAudioClip(res.Clip, meta)
	out := resultMap(res)
	out["clip"] = clip
	if r.target.Live() {
		start, end := round1(res.PlayStart-r.offset), round1(res.PlayEnd-r.offset)
		h.recorder.NoteAudioPlay(clip, h.speechCallID, start, end)
		out["playStartMs"], out["playEndMs"] = start, end
	}
	h.speechResult = out
	h.lastSayCall = h.speechCallID
}

// heard records what a listen heard. Its times move from the capture's
// own start onto the recording's timeline.
func (r *audioRecording) heard(res *speech.ListenResult) {
	if r == nil {
		return
	}
	h := r.h
	out := resultMap(res)
	if !r.target.Live() {
		h.speechResult = out
		return
	}
	base := res.CaptureStart - r.offset
	shift := func(ms float64) float64 { return round1(base + ms) }
	track := "in-" + r.target.Name()
	entry := map[string]interface{}{
		"callId": h.speechCallID, "track": track, "startMs": shift(0), "endMs": shift(res.DurationMs),
		"text": res.Text, "transcribed": res.Transcribed, "provider": res.Provider, "model": res.Model, "stoppedBy": res.StoppedBy,
	}
	out["track"], out["audioStartMs"], out["audioEndMs"] = track, shift(0), shift(res.DurationMs)
	if res.OnsetMs != nil {
		entry["onsetMs"], entry["speechEndMs"] = shift(*res.OnsetMs), shift(*res.SpeechEndMs)
		out["onsetMs"], out["speechEndMs"] = entry["onsetMs"], entry["speechEndMs"]
	}
	if res.LatencyMs != nil {
		entry["latencyMs"], entry["latencyFrom"] = *res.LatencyMs, h.lastSayCall
		out["latencyFrom"] = h.lastSayCall
	}
	segments := []map[string]interface{}{}
	for _, s := range res.Segments {
		segments = append(segments, map[string]interface{}{"startMs": shift(s.StartMs), "endMs": shift(s.EndMs), "text": s.Text})
	}
	words := []map[string]interface{}{}
	for _, w := range res.Words {
		words = append(words, map[string]interface{}{"startMs": shift(w.StartMs), "endMs": shift(w.EndMs), "word": w.Word})
	}
	entry["segments"], entry["words"] = segments, words
	out["segments"] = segments
	if len(words) > 0 {
		out["words"] = words
	}
	h.recorder.AddAudioTranscript(entry)
	h.speechResult = out
}

func resultMap(v interface{}) map[string]interface{} {
	raw, _ := json.Marshal(v)
	out := map[string]interface{}{}
	json.Unmarshal(raw, &out)
	return out
}
