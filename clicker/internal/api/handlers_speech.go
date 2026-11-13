package api

import (
	"encoding/json"
	"errors"

	"github.com/vibium/clicker/internal/speech"
)

const (
	speechSayMethod    = "vibium:speech.say"
	speechListenMethod = "vibium:speech.listen"
)

func isSpeechMethod(method string) bool {
	return method == speechSayMethod || method == speechListenMethod
}

// speechOutcome is what a say or listen step's after event carries.
type speechOutcome struct {
	result map[string]interface{}
	err    error
}

// attachSpeech binds the session's say and listen to its browser and gives
// every page a virtual microphone and an audio tap.
func (r *Router) attachSpeech(session *BrowserSession) {
	session.speech.Recorder = func() *Recorder {
		session.mu.Lock()
		defer session.mu.Unlock()
		return session.recorder
	}
	session.speech.Bind(session, NewAPISession(r, session, ""))
	session.speech.Install(func() error { return r.ensureScriptMessageSubscription(session) })
}

// speechArgsOf is a command's params without the keys the router owns.
func speechArgsOf(params map[string]interface{}) map[string]interface{} {
	args := make(map[string]interface{}, len(params))
	for k, v := range params {
		if k != "context" && k != "_recordCallId" {
			args[k] = v
		}
	}
	return args
}

// runSpeech runs one say or listen. The result is nil on an error, and
// never a typed nil.
func runSpeech(s *Speech, method string, args map[string]interface{}, page, callID string) (interface{}, map[string]interface{}, error) {
	if method == speechSayMethod {
		res, recorded, err := s.Say(args, page, callID)
		if err != nil {
			return nil, nil, err
		}
		return res, recorded, nil
	}
	res, recorded, err := s.Listen(args, page, callID)
	if err != nil {
		return nil, nil, err
	}
	return res, recorded, nil
}

// handleSpeech serves vibium:speech.say and vibium:speech.listen. A
// session target acts on the page named, or on the current one.
func (r *Router) handleSpeech(session *BrowserSession, cmd bidiCommand) {
	callID, _ := cmd.Params["_recordCallId"].(string)
	// A session with no page to name, such as a device, has no context.
	page, _ := r.resolveContext(session, cmd.Params)
	result, recorded, err := runSpeech(session.speech, cmd.Method, speechArgsOf(cmd.Params), page, callID)
	if callID != "" {
		session.mu.Lock()
		session.speechOutcomes[callID] = speechOutcome{result: recorded, err: err}
		session.mu.Unlock()
	}
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, result)
}

// takeSpeechOutcome returns what a speech step left for its after event.
func (s *BrowserSession) takeSpeechOutcome(callID string) (speechOutcome, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.speechOutcomes[callID]
	delete(s.speechOutcomes, callID)
	return o, ok
}

// SetSessionless lets say and listen run with no session open, on this
// machine's speakers and microphone or a WAV file. The no-browser pipe
// turns it on.
func (r *Router) SetSessionless(on bool) { r.sessionless = on }

// handleSessionlessSpeech serves say and listen for a client that has no
// session. It reports whether the message was one of them.
func (r *Router) handleSessionlessSpeech(client ClientTransport, raw string) bool {
	if !r.sessionless {
		return false
	}
	var cmd bidiCommand
	if json.Unmarshal([]byte(raw), &cmd) != nil || !isSpeechMethod(cmd.Method) {
		return false
	}
	r.speechMu.Lock()
	if r.speech == nil {
		r.speech = NewSpeech()
	}
	s := r.speech
	r.speechMu.Unlock()
	go func() {
		result, _, err := runSpeech(s, cmd.Method, speechArgsOf(cmd.Params), "", "")
		response := bidiResponse{ID: cmd.ID, Type: "success", Result: result}
		if err != nil {
			response = bidiResponse{ID: cmd.ID, Type: "error", Error: errorCode(err), Message: errorMessage(err)}
		}
		data, _ := json.Marshal(response)
		client.Send(string(data))
	}()
	return true
}

// errorMessage is an error's text for the wire. A speech error's code
// travels in the error field, so its message goes without it.
func errorMessage(err error) string {
	var se *speech.Error
	if errors.As(err, &se) {
		return se.Message
	}
	return err.Error()
}
