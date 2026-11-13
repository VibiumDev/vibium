package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/speech"
)

// readySpeechPhrase is what the check says and expects to hear.
const readySpeechPhrase = "The quick brown fox jumps over the lazy dog."

// checkSpeechSetup tests say and listen as the commands would run them:
// the say provider renders a phrase into a WAV file, and the listen
// provider transcribes that file. No sound is played or captured.
func checkSpeechSetup(ctx context.Context, env speech.Env) setupResult {
	result := setupResult{Ready: true, Scope: "speech", Checks: []setupCheck{}, Notes: []string{
		"Speech readiness does not test this machine's speakers and microphone, a browser page, or a device.",
	}}
	fail := func(name string, err error) {
		result.Ready = false
		result.Checks = append(result.Checks, setupCheck{name, "failed", speechMessage(err), speechFix(err)})
	}

	dir, err := os.MkdirTemp("", "vibium-ready-speech-")
	if err != nil {
		fail("say", err)
		return result
	}
	defer os.RemoveAll(dir)
	engine := speech.NewEngine(env)
	defer engine.Close()
	target := speech.FileTarget{Path: filepath.Join(dir, "phrase.wav")}

	said := false
	if choice, err := env.Say("", "", ""); err != nil {
		fail("say", err)
	} else {
		started := time.Now()
		// A cache of its own, so the provider renders now and the user's
		// cache is left alone.
		res, err := engine.Say(ctx, target, speech.SayOptions{Text: readySpeechPhrase, CacheDir: filepath.Join(dir, "cache")})
		if err != nil {
			fail("say", err)
		} else {
			said = true
			result.Checks = append(result.Checks, setupCheck{Name: "say", Status: "passed", Message: fmt.Sprintf("%s rendered %.1f s of speech in %.1f s.",
				speechChoice(choice), res.DurationMs/1000, time.Since(started).Seconds())})
		}
	}

	choice, err := env.Listen("", "")
	switch {
	case err != nil:
		fail("listen", err)
	case !said:
		result.Checks = append(result.Checks, setupCheck{"listen", "skipped", speechChoice(choice) + " was not tested: say rendered nothing for it to hear.", ""})
	default:
		started := time.Now()
		res, err := engine.Listen(ctx, target, speech.ListenOptions{Silence: 0})
		switch {
		case err != nil:
			fail("listen", err)
		case !speech.SaysTheText(readySpeechPhrase, res.Text):
			result.Ready = false
			result.Checks = append(result.Checks, setupCheck{"listen", "failed",
				fmt.Sprintf("%s heard %q for %q.", speechChoice(choice), res.Text, readySpeechPhrase),
				"Listen to what say rendered with `vibium say \"" + readySpeechPhrase + "\" --target local`. If it sounds right, try another listen model."})
		default:
			result.Checks = append(result.Checks, setupCheck{Name: "listen", Status: "passed", Message: fmt.Sprintf("%s heard %q in %.1f s.",
				speechChoice(choice), res.Text, time.Since(started).Seconds())})
		}
	}

	result.Summary = "Fix the failed checks and rerun vibium ready speech."
	if result.Ready {
		result.Summary = "Say and listen passed: a phrase went through both providers."
	}
	return result
}

// speechChoice names a provider with its model and voice, as say and
// listen print them.
func speechChoice(c speech.Choice) string {
	return strings.Join(strings.Fields(c.Provider.Name()+" "+c.Model+" "+c.Voice), " ")
}

// speechMessage is a failure as the report shows it: the code, then the
// message, which says what to do.
func speechMessage(err error) string {
	var e *speech.Error
	if errors.As(err, &e) {
		return e.Code + ": " + e.Message
	}
	return err.Error()
}

// speechFix adds what a failure's own message does not say.
func speechFix(err error) string {
	switch speech.Code(err) {
	case "no_api_key":
		return "Set the key in ~/.config/vibium/ai.env (loaded automatically) or export it; keep its value out of chat and logs."
	case "provider_error":
		return "Check the API key, the model's name and the account's quota."
	}
	return ""
}
