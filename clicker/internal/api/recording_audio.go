package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/vibium/clicker/internal/speech"
)

// Audio in a recording: the clips say sent, one continuous track per
// target of what was heard, and listen's transcripts, all on the
// recording's timeline. The format is in docs/specs/say-listen-spec.md.
// It adds entries under audio/ and changes no existing one.

type recordAudio struct {
	tracks      []*audioTrack
	clips       []*audioClip
	transcripts []map[string]interface{}
}

type audioTrack struct {
	target  string
	startMs float64
	samples []int16
	gaps    []map[string]int
	anchors []map[string]float64
	clock   map[string]interface{}
}

type audioClip struct {
	id         string
	sha        string
	samples    []int16
	wav        []byte
	durationMs float64
	meta       map[string]interface{}
	plays      []map[string]interface{}
}

// audioGapMs is how far a chunk may start after the track's end before
// the difference is recorded as a gap and filled with silence.
const audioGapMs = 30.0

func round1(ms float64) float64 { return math.Round(ms*10) / 10 }

// TimelineMs places a wall-clock time on the recording's timeline.
func (t *Recorder) TimelineMs(at time.Time) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return float64(at.UnixMicro())/1000 - float64(t.monotonicBase)
}

// AddAudioClip stores the audio a say sent. The same audio is stored once.
// It returns the clip's id.
func (t *Recorder) AddAudioClip(clip speech.Audio, meta map[string]interface{}) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.recording {
		return ""
	}
	if t.audio == nil {
		t.audio = &recordAudio{}
	}
	mono := clip.Resample(speech.Rate)
	sha := mono.SHA256()
	for _, c := range t.audio.clips {
		if c.sha == sha {
			return c.id
		}
	}
	c := &audioClip{id: fmt.Sprintf("out-%04d", len(t.audio.clips)+1), sha: sha, samples: mono.Samples, wav: mono.WAV(), durationMs: mono.DurationMs(), meta: meta}
	t.audio.clips = append(t.audio.clips, c)
	return c.id
}

// NoteAudioPlay records one playing of a clip, on the recording's timeline.
func (t *Recorder) NoteAudioPlay(clipID, callID string, startMs, endMs float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.audio == nil {
		return
	}
	for _, c := range t.audio.clips {
		if c.id == clipID {
			c.plays = append(c.plays, map[string]interface{}{"callId": callID, "startMs": round1(startMs), "endMs": round1(endMs)})
		}
	}
}

func (t *Recorder) trackLocked(target string) *audioTrack {
	if t.audio == nil {
		t.audio = &recordAudio{}
	}
	for _, tr := range t.audio.tracks {
		if tr.target == target {
			return tr
		}
	}
	tr := &audioTrack{target: target, startMs: math.NaN()}
	t.audio.tracks = append(t.audio.tracks, tr)
	return tr
}

// AppendAudio adds captured 16 kHz mono audio to a target's track. startMs
// is when its first sample was heard, on the recording's timeline. Audio
// that arrives late for its place is dropped; a hole is filled with
// silence and noted.
func (t *Recorder) AppendAudio(target string, samples []int16, startMs float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.recording || len(samples) == 0 {
		return
	}
	tr := t.trackLocked(target)
	if math.IsNaN(tr.startMs) {
		tr.startMs = startMs
	}
	end := tr.startMs + float64(len(tr.samples))*1000/speech.Rate
	if gap := startMs - end; gap > audioGapMs {
		fill := int(gap * speech.Rate / 1000)
		tr.gaps = append(tr.gaps, map[string]int{"sample": len(tr.samples), "samples": fill})
		tr.samples = append(tr.samples, make([]int16, fill)...)
	} else if gap < -audioGapMs {
		skip := int(-gap * speech.Rate / 1000)
		if skip >= len(samples) {
			return
		}
		samples = samples[skip:]
	}
	// One anchor every ten seconds ties sample positions to the timeline.
	if n := len(tr.anchors); n == 0 || len(tr.samples)-int(tr.anchors[n-1]["sample"]) >= 10*speech.Rate {
		tr.anchors = append(tr.anchors, map[string]float64{"sample": float64(len(tr.samples)), "timeMs": round1(tr.startMs + float64(len(tr.samples))*1000/speech.Rate)})
	}
	tr.samples = append(tr.samples, samples...)
}

// SetAudioClock records how a track's times were brought onto the
// recording's timeline.
func (t *Recorder) SetAudioClock(target string, clock map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.recording {
		t.trackLocked(target).clock = clock
	}
}

// AddAudioTranscript records what one listen heard.
func (t *Recorder) AddAudioTranscript(entry map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.recording {
		return
	}
	if t.audio == nil {
		t.audio = &recordAudio{}
	}
	t.audio.transcripts = append(t.audio.transcripts, entry)
}

// mixAudioLocked adds up every track and every playing of a clip over a
// stretch of the timeline, at speech.Rate. It returns nil when none of
// them reaches into the stretch.
func (t *Recorder) mixAudioLocked(fromMs float64, length time.Duration) []int16 {
	if t.audio == nil {
		return nil
	}
	out := make([]int16, int(length.Seconds()*speech.Rate))
	found := false
	add := func(samples []int16, atMs float64) {
		if len(samples) == 0 {
			return
		}
		at := int(math.Round((atMs - fromMs) * speech.Rate / 1000))
		lo, hi := max(at, 0), min(at+len(samples), len(out))
		for i := lo; i < hi; i++ {
			out[i] = int16(max(-32768, min(32767, int32(out[i])+int32(samples[i-at]))))
			found = true
		}
	}
	for _, tr := range t.audio.tracks {
		add(tr.samples, tr.startMs)
	}
	for _, c := range t.audio.clips {
		for _, play := range c.plays {
			if at, ok := play["startMs"].(float64); ok {
				add(c.samples, at)
			}
		}
	}
	if !found {
		return nil
	}
	return out
}

// AudioSummary is what a stop reports about the recording's audio.
type AudioSummary struct {
	Tracks      int     `json:"tracks"`
	Clips       int     `json:"clips"`
	Transcripts int     `json:"transcripts"`
	Seconds     float64 `json:"seconds"`
}

// AudioSummary is the recording's audio so far, or nil when it has none.
func (t *Recorder) AudioSummary() *AudioSummary {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.audioSummaryLocked()
}

func (t *Recorder) audioSummaryLocked() *AudioSummary {
	if t.audio == nil {
		return nil
	}
	s := &AudioSummary{Tracks: len(t.audio.tracks), Clips: len(t.audio.clips), Transcripts: len(t.audio.transcripts)}
	for _, tr := range t.audio.tracks {
		s.Seconds += float64(len(tr.samples)) / speech.Rate
	}
	s.Seconds = math.Round(s.Seconds*10) / 10
	return s
}

// writeAudioEntriesLocked writes audio/index.json, one WAV per track and
// one per clip. Must be called with t.mu held.
func (t *Recorder) writeAudioEntriesLocked(zw *zip.Writer, now time.Time) error {
	if t.audio == nil {
		return nil
	}
	write := func(name string, data []byte, method uint16) error {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: now})
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	format := map[string]interface{}{"sampleRate": speech.Rate, "channels": 1, "encoding": "pcm_s16le"}
	with := func(m map[string]interface{}) map[string]interface{} {
		for k, v := range format {
			m[k] = v
		}
		return m
	}
	tracks := []map[string]interface{}{}
	for _, tr := range t.audio.tracks {
		if len(tr.samples) == 0 {
			continue
		}
		id := "in-" + tr.target
		file := "audio/" + id + ".wav"
		audio := speech.Audio{SampleRate: speech.Rate, Channels: 1, Samples: tr.samples}
		if err := write(file, audio.WAV(), zip.Deflate); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
		marks := []map[string]float64{}
		for _, s := range speech.SpeechStretches(tr.samples, time.Second) {
			marks = append(marks, map[string]float64{
				"startMs": round1(tr.startMs + float64(s.Start)/float64(time.Millisecond)),
				"endMs":   round1(tr.startMs + float64(s.End)/float64(time.Millisecond)),
			})
		}
		clock := map[string]interface{}{}
		for k, v := range tr.clock {
			clock[k] = v
		}
		clock["anchors"] = tr.anchors
		gaps := tr.gaps
		if gaps == nil {
			gaps = []map[string]int{}
		}
		tracks = append(tracks, with(map[string]interface{}{
			"id": id, "target": tr.target, "file": file, "startMs": round1(tr.startMs), "durationMs": audio.DurationMs(),
			"clock": clock, "gaps": gaps, "speech": marks,
		}))
	}
	clips := []map[string]interface{}{}
	for _, c := range t.audio.clips {
		file := "audio/" + c.id + ".wav"
		if err := write(file, c.wav, zip.Deflate); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
		entry := with(map[string]interface{}{"id": c.id, "file": file, "sha256": c.sha, "durationMs": c.durationMs, "plays": c.plays})
		for k, v := range c.meta {
			entry[k] = v
		}
		clips = append(clips, entry)
	}
	transcripts := t.audio.transcripts
	if transcripts == nil {
		transcripts = []map[string]interface{}{}
	}
	manifest, err := json.MarshalIndent(map[string]interface{}{"version": 1, "tracks": tracks, "clips": clips, "transcripts": transcripts}, "", "  ")
	if err != nil {
		return err
	}
	return write("audio/index.json", t.redactAudioManifestLocked(manifest), zip.Deflate)
}

// redactAudioManifestLocked removes known secrets from the manifest's
// text, as trace events have them removed.
func (t *Recorder) redactAudioManifestLocked(manifest []byte) []byte {
	var value interface{}
	if json.Unmarshal(manifest, &value) != nil {
		return manifest
	}
	out, err := json.MarshalIndent(t.redactKnown(value), "", "  ")
	if err != nil {
		return manifest
	}
	return out
}
