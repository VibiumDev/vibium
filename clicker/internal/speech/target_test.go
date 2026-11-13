package speech

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A local target is two system programs. Here they are shell scripts: the
// player saves what it was given, and the microphone plays back a script.
func TestLocalTargetThroughSystemPrograms(t *testing.T) {
	dir := t.TempDir()
	played := filepath.Join(dir, "played.wav")
	heard := filepath.Join(dir, "mic.raw")
	room := Audio{SampleRate: Rate, Channels: 1, Samples: join(quiet(Rate, 600*time.Millisecond), tone(Rate, 400, -18, time.Second), quiet(Rate, 4*time.Second))}
	os.WriteFile(heard, room.Bytes(), 0o600)
	local := &LocalTarget{
		play:    []string{script(t, `cat > `+played)},
		capture: []string{script(t, `cat `+heard+`; sleep 5`)},
	}
	p := &fakeProvider{reply: Transcript{Text: "four"}}
	e := fakeEngine(t, p)

	res, err := e.Listen(context.Background(), local, ListenOptions{Silence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "four" || res.Target != "local" || res.StoppedBy != "silence" || *res.OnsetMs < 580 || *res.OnsetMs > 620 {
		t.Fatalf("%+v", res)
	}

	clip := Audio{SampleRate: 24000, Channels: 1, Samples: tone(24000, 300, -20, 300*time.Millisecond)}
	start, end, err := local.Play(context.Background(), clip)
	if err != nil || end < start {
		t.Fatalf("play: %v %v %v", start, end, err)
	}
	if got, err := os.ReadFile(played); err != nil || string(got) != string(clip.WAV()) {
		t.Fatalf("the player did not receive the clip as a WAV: %v", err)
	}
}

func TestLocalTargetWithoutPrograms(t *testing.T) {
	missing := &LocalTarget{play: []string{"/nonexistent/aplay"}, capture: []string{"/nonexistent/arecord"}, missing: "install alsa-utils"}
	if _, _, err := missing.Play(context.Background(), Audio{SampleRate: Rate, Channels: 1, Samples: make([]int16, 160)}); Code(err) != "no_audio" {
		t.Fatalf("play: %v", err)
	}
	if _, err := missing.Capture(context.Background()); Code(err) != "no_audio" {
		t.Fatalf("capture: %v", err)
	}
	none := &LocalTarget{missing: "not available here"}
	if _, err := none.Capture(context.Background()); Code(err) != "no_audio" {
		t.Fatalf("no capture program: %v", err)
	}
}

func TestFileTargetIsNotLive(t *testing.T) {
	f := FileTarget{Path: filepath.Join(t.TempDir(), "sub", "out.wav")}
	if f.Live() || !IsWAVPath("a/b.WAV") || IsWAVPath("notes.txt") || IsWAVPath("local") {
		t.Fatal("target kinds are confused")
	}
	if _, _, err := f.Play(context.Background(), Audio{SampleRate: Rate, Channels: 1, Samples: make([]int16, 160)}); err != nil {
		t.Fatalf("a missing directory should be created: %v", err)
	}
}
