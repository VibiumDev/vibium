package api

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
	s        *Speech
	recorder *Recorder
	target   speech.Target
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
func (s *Speech) beginAudioRecording(t speech.Target) *audioRecording {
	recorder := s.recording()
	if recorder == nil || !recorder.IsRecording() {
		return nil
	}
	if s.audioRec != nil && s.audioRec.recorder == recorder && s.audioRec.target.Key() == t.Key() {
		return s.audioRec
	}
	rec := &audioRecording{s: s, recorder: recorder, target: t}
	if !t.Live() {
		return rec
	}
	now, rtt := t.Now(), 0.0
	if s, ok := t.(clockSyncer); ok {
		now, rtt = s.Sync()
	}
	at := recorder.TimelineMs(time.Now())
	rec.offset = now - at
	source := t.Name()
	switch t.(type) {
	case *roadieAudio:
		source = "roadie"
	case *browserAudio:
		source = "page"
	}
	recorder.SetAudioClock(t.Name(), map[string]interface{}{
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
	s.StopAudioRecording()
	rec.capture, rec.done = capture, make(chan struct{})
	s.audioRec = rec
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

// StopAudioRecording ends the continuous track. Called before the
// recording is built into its zip.
func (s *Speech) StopAudioRecording() {
	if s == nil {
		return
	}
	rec := s.audioRec
	s.audioRec = nil
	if rec == nil || rec.capture == nil {
		return
	}
	rec.once.Do(rec.capture.Close)
	select {
	case <-rec.done:
	case <-time.After(2 * time.Second):
	}
}

// said records the clip a say sent and when it played. It returns what
// the step's after event carries.
func (r *audioRecording) said(res *speech.SayResult, callID string) map[string]interface{} {
	if r == nil {
		return nil
	}
	meta := map[string]interface{}{"target": res.Target, "text": res.Text, "provider": res.Provider, "model": res.Model, "voice": res.Voice}
	if res.File != "" {
		meta["source"] = "file"
	}
	clip := r.recorder.AddAudioClip(res.Clip, meta)
	out := resultMap(res)
	out["clip"] = clip
	if r.target.Live() {
		start, end := round1(res.PlayStart-r.offset), round1(res.PlayEnd-r.offset)
		r.recorder.NoteAudioPlay(clip, callID, start, end)
		out["playStartMs"], out["playEndMs"] = start, end
	}
	r.s.lastSayCall = callID
	return out
}

// heard records what a listen heard. Its times move from the capture's
// own start onto the recording's timeline. It returns what the step's
// after event carries.
func (r *audioRecording) heard(res *speech.ListenResult, callID string) map[string]interface{} {
	if r == nil {
		return nil
	}
	out := resultMap(res)
	if !r.target.Live() {
		return out
	}
	lastSay := r.s.lastSayCall
	base := res.CaptureStart - r.offset
	shift := func(ms float64) float64 { return round1(base + ms) }
	track := "in-" + r.target.Name()
	entry := map[string]interface{}{
		"callId": callID, "track": track, "startMs": shift(0), "endMs": shift(res.DurationMs),
		"text": res.Text, "transcribed": res.Transcribed, "provider": res.Provider, "model": res.Model, "stoppedBy": res.StoppedBy,
	}
	out["track"], out["audioStartMs"], out["audioEndMs"] = track, shift(0), shift(res.DurationMs)
	if res.OnsetMs != nil {
		entry["onsetMs"], entry["speechEndMs"] = shift(*res.OnsetMs), shift(*res.SpeechEndMs)
		out["onsetMs"], out["speechEndMs"] = entry["onsetMs"], entry["speechEndMs"]
	}
	if res.LatencyMs != nil {
		entry["latencyMs"], entry["latencyFrom"] = *res.LatencyMs, lastSay
		out["latencyFrom"] = lastSay
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
	r.recorder.AddAudioTranscript(entry)
	return out
}

func resultMap(v interface{}) map[string]interface{} {
	raw, _ := json.Marshal(v)
	out := map[string]interface{}{}
	json.Unmarshal(raw, &out)
	return out
}
