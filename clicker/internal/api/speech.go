package api

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/speech"
)

// Speech is say and listen for one session, or for none. The MCP server
// and the wire router each hold one and reach the session through their
// own Session, so both speak into the same targets the same way.
type Speech struct {
	// Getenv reads a speech setting. Nil reads this process's environment.
	Getenv func(string) string
	// Recorder returns the running recording, or nil when there is none.
	Recorder func() *Recorder

	// The engine lives as long as the Speech, so the capture window a say
	// opens is still there for the next listen.
	engine *speech.Engine

	mu      sync.Mutex
	owner   interface{} // what the session was bound for; a new owner is a new session
	session Session     // nil with no session open
	page    string      // the page the running call acts on; "" is every page
	roadie  *roadieAudio
	browser *browserAudio

	audioMu   sync.Mutex
	audioSink *roadieCapture // where roadie:audio.chunk events go
	pageSink  *browserCapture

	audioRec    *audioRecording
	lastSayCall string
}

func NewSpeech() *Speech { return &Speech{} }

func (s *Speech) speechEngine() *speech.Engine {
	if s.engine == nil {
		getenv := func(name string) string {
			if s.Getenv != nil {
				return s.Getenv(name)
			}
			return os.Getenv(name)
		}
		s.engine = speech.NewEngine(speech.Env{Getenv: getenv, LookPath: exec.LookPath, OS: runtime.GOOS})
	}
	return s.engine
}

func (s *Speech) recording() *Recorder {
	if s.Recorder == nil {
		return nil
	}
	return s.Recorder()
}

// Bind names the session say and listen act on. owner tells one session
// from the next: with a new owner the last session's audio is forgotten.
// A nil session means none is open.
func (s *Speech) Bind(owner interface{}, sess Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner == s.owner && (sess == nil) == (s.session == nil) {
		return
	}
	s.owner, s.session, s.roadie, s.browser = owner, sess, nil, nil
}

// Install gives every page of the bound session a virtual microphone and
// an audio tap, so say and listen work without setup. It reports whether
// the session took the script; a device on a Roadie does not run scripts.
func (s *Speech) Install(subscribe func() error) bool {
	sess := s.sess()
	if sess == nil || !installBrowserAudio(sess, subscribe) {
		return false
	}
	b := &browserAudio{s: s}
	b.tap = speech.NewTap(b.open)
	s.mu.Lock()
	s.browser = b
	s.mu.Unlock()
	return true
}

// Event routes an audio event from the session to the running capture. It
// reports whether the message was one, so audio stays out of the BiDi log
// and away from the client.
func (s *Speech) Event(msg string) bool {
	if s == nil {
		return false
	}
	return s.roadieAudioEvent(msg) || s.pageAudioEvent(msg)
}

func (s *Speech) sess() Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.session
}

func (s *Speech) currentPage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.page
}

// target resolves a target option. The default follows the session: with
// one open it is the session, and otherwise this machine. It never falls
// back from one to the other.
func (s *Speech) target(name string) (speech.Target, error) {
	s.mu.Lock()
	sess, roadie, browser := s.session, s.roadie, s.browser
	s.mu.Unlock()
	open := sess != nil
	if name == "" {
		name = "local"
		if open {
			name = "session"
		}
	}
	switch {
	case name == "local":
		return speech.NewLocalTarget(), nil
	case name == "session":
		if !open {
			return nil, &speech.Error{Code: "invalid_target", Message: "no session is open; use target local or a .wav path, or start a session first"}
		}
		if roadie != nil {
			return roadie, nil
		}
		if browser != nil {
			return browser, nil
		}
		if found, ok := s.roadieAudioFor(sess); ok {
			s.mu.Lock()
			s.roadie = found
			s.mu.Unlock()
			return found, nil
		}
		return nil, &speech.Error{Code: "no_audio", Message: "this session cannot carry audio; use --target local for this machine's speakers and microphone"}
	case speech.IsWAVPath(name):
		path, err := filepath.Abs(name)
		if err != nil {
			return nil, err
		}
		return speech.FileTarget{Path: path}, nil
	}
	return nil, &speech.Error{Code: "invalid_target", Message: fmt.Sprintf("target %q is not session, local, or a path ending in .wav", name)}
}

func speechArgs(args map[string]interface{}, allowed ...string) error {
	for key := range args {
		ok := key == "page"
		for _, a := range allowed {
			ok = ok || key == a
		}
		if !ok {
			return fmt.Errorf("unsupported argument %s", key)
		}
	}
	return nil
}

func argString(args map[string]interface{}, key string) (string, error) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return s, nil
}

// argMs reads a duration in milliseconds. A missing value is -1.
func argMs(args map[string]interface{}, key string) (time.Duration, error) {
	v, ok := args[key]
	if !ok || v == nil {
		return -1, nil
	}
	n, ok := v.(float64)
	if !ok || n < 0 {
		return 0, fmt.Errorf("%s must be a number of milliseconds, zero or more", key)
	}
	return time.Duration(n * float64(time.Millisecond)), nil
}

func argBool(args map[string]interface{}, key string) (value, set bool, err error) {
	v, ok := args[key]
	if !ok || v == nil {
		return false, false, nil
	}
	b, isBool := v.(bool)
	if !isBool {
		return false, false, fmt.Errorf("%s must be true or false", key)
	}
	return b, true, nil
}

// Say speaks args' text, or plays its file, into the target. page is the
// page a session target acts on, and callID the recorded step the call
// belongs to. With a recording running it also returns what that step's
// after event carries.
func (s *Speech) Say(args map[string]interface{}, page, callID string) (*speech.SayResult, map[string]interface{}, error) {
	if err := speechArgs(args, "text", "file", "target", "provider", "model", "voice", "language", "speed", "cache", "cacheDir"); err != nil {
		return nil, nil, err
	}
	var o speech.SayOptions
	var target string
	for key, dst := range map[string]*string{"text": &o.Text, "file": &o.File, "target": &target, "provider": &o.Provider, "model": &o.Model,
		"voice": &o.Voice, "language": &o.Language, "cacheDir": &o.CacheDir} {
		v, err := argString(args, key)
		if err != nil {
			return nil, nil, err
		}
		*dst = v
	}
	if o.File != "" {
		abs, err := filepath.Abs(o.File)
		if err != nil {
			return nil, nil, err
		}
		o.File = abs
	}
	if v, ok := args["speed"]; ok && v != nil {
		n, isNumber := v.(float64)
		if !isNumber || n <= 0 {
			return nil, nil, fmt.Errorf("speed must be a positive number")
		}
		o.Speed = n
	}
	if use, set, err := argBool(args, "cache"); err != nil {
		return nil, nil, err
	} else if set {
		o.NoCache = !use
	}
	t, err := s.target(target)
	if err != nil {
		return nil, nil, err
	}
	s.setPage(page)
	rec := s.beginAudioRecording(t)
	res, err := s.speechEngine().Say(context.Background(), t, o)
	if err != nil {
		return nil, nil, err
	}
	return res, rec.said(res, callID), nil
}

// Listen captures from the target and transcribes what it heard. page and
// callID are as for Say.
func (s *Speech) Listen(args map[string]interface{}, page, callID string) (*speech.ListenResult, map[string]interface{}, error) {
	if err := speechArgs(args, "target", "provider", "model", "language", "silence", "timeout", "duration", "allowSilence", "path", "transcribe"); err != nil {
		return nil, nil, err
	}
	var o speech.ListenOptions
	var target string
	for key, dst := range map[string]*string{"target": &target, "provider": &o.Provider, "model": &o.Model, "language": &o.Language, "path": &o.Path} {
		v, err := argString(args, key)
		if err != nil {
			return nil, nil, err
		}
		*dst = v
	}
	var err error
	if o.Silence, err = argMs(args, "silence"); err != nil {
		return nil, nil, err
	}
	if o.Timeout, err = argMs(args, "timeout"); err != nil {
		return nil, nil, err
	}
	if o.Duration, err = argMs(args, "duration"); err != nil {
		return nil, nil, err
	}
	if allow, set, err := argBool(args, "allowSilence"); err != nil {
		return nil, nil, err
	} else if set {
		o.AllowSilence = allow
	}
	if transcribe, set, err := argBool(args, "transcribe"); err != nil {
		return nil, nil, err
	} else if set {
		o.NoTranscribe = !transcribe
	}
	if o.Path != "" {
		if !speech.IsWAVPath(o.Path) {
			return nil, nil, fmt.Errorf("path must end in .wav")
		}
		if o.Path, err = filepath.Abs(o.Path); err != nil {
			return nil, nil, err
		}
	}
	t, err := s.target(target)
	if err != nil {
		return nil, nil, err
	}
	s.setPage(page)
	rec := s.beginAudioRecording(t)
	res, err := s.speechEngine().Listen(context.Background(), t, o)
	if err != nil {
		return nil, nil, err
	}
	return res, rec.heard(res, callID), nil
}

func (s *Speech) setPage(page string) {
	s.mu.Lock()
	s.page = page
	s.mu.Unlock()
}
