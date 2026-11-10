package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/daemon"
	"github.com/vibium/clicker/internal/speech"
)

// speechCall sends say or listen to the daemon, starting it when needed.
// Unlike other commands it does not ask for a browser first: these two
// work without one.
func speechCall(tool string, args map[string]interface{}, timeout time.Duration) (*agent.ToolsCallResult, error) {
	// The daemon may have started long ago. Each call carries this
	// process's speech settings, as run and check carry their model's.
	env := map[string]string{}
	for _, name := range speech.EnvNames {
		env[name] = os.Getenv(name)
	}
	result, err := daemon.Speech(tool, args, env, timeout)
	if err == nil || !isConnectionError(err) {
		return result, err
	}
	daemon.CleanStale()
	if err := autoStartDaemon(); err != nil {
		return nil, fmt.Errorf("auto-start daemon: %w", err)
	}
	return daemon.Speech(tool, args, env, timeout)
}

// speechTargetArg makes a file target absolute, since the daemon runs in
// another directory.
func speechTargetArg(target string) (string, error) {
	if speech.IsWAVPath(target) {
		return filepath.Abs(target)
	}
	return target, nil
}

func seconds(ms float64) string { return fmt.Sprintf("%.1f s", ms/1000) }

func newSayCmd() *cobra.Command {
	var target, provider, model, voice, language, cacheDir string
	var speed float64
	var noCache bool
	cmd := &cobra.Command{
		Use:   `say "<text>"`,
		Short: "Speak text into the session, this machine's speakers, or a WAV file",
		Long: "Turn text into speech and play it into a target.\n" +
			"With a session open the target is the session. With none open it is this machine's speakers.\n" +
			"Rendered audio is cached, so the same text plays the same audio on every run.",
		Example: `  vibium say "Come here, Watson"
  # Said "Come here, Watson" (1.4 s, openai gpt-4o-mini-tts alloy)

  vibium say "Come here, Watson" --target watson.wav
  # Wrote watson.wav (1.4 s, openai gpt-4o-mini-tts alloy, cached)

  echo "What time is it?" | vibium say - --target local
  # Reads the text from stdin.

  vibium say -- "-5 degrees"
  # Text that starts with a dash goes after --.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			text := args[0]
			if text == "-" {
				raw, err := io.ReadAll(os.Stdin)
				if err != nil {
					printError(err)
					return
				}
				text = strings.TrimSpace(string(raw))
			}
			targetArg, err := speechTargetArg(target)
			if err != nil {
				printError(err)
				return
			}
			params := map[string]interface{}{"text": text}
			for key, value := range map[string]string{"target": targetArg, "provider": provider, "model": model, "voice": voice, "language": language} {
				if value != "" {
					params[key] = value
				}
			}
			if cacheDir != "" {
				abs, err := filepath.Abs(cacheDir)
				if err != nil {
					printError(err)
					return
				}
				params["cacheDir"] = abs
			}
			if cmd.Flags().Changed("speed") {
				params["speed"] = speed
			}
			if noCache {
				params["cache"] = false
			}
			result, err := speechCall("vibium_say", params, 3*time.Minute)
			if err != nil {
				printError(err)
				return
			}
			var res speech.SayResult
			if err := json.Unmarshal([]byte(extractText(result)), &res); err != nil {
				printError(err)
				return
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: res})
				return
			}
			detail := seconds(res.DurationMs) + ", " + strings.TrimSpace(res.Provider+" "+res.Model+" "+res.Voice)
			if res.Cached {
				detail += ", cached"
			}
			if res.Path != "" {
				fmt.Printf("Wrote %s (%s)\n", target, detail)
				return
			}
			fmt.Printf("Said %q (%s)\n", res.Text, detail)
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "session, local, or a path ending in .wav (default: the session if one is open, else local)")
	cmd.Flags().StringVar(&provider, "provider", "", "Text-to-speech provider or preset (default: from the environment)")
	cmd.Flags().StringVar(&model, "model", "", "Provider's model id")
	cmd.Flags().StringVar(&voice, "voice", "", "Provider's voice id")
	cmd.Flags().StringVar(&language, "language", "", "BCP 47 tag, for providers that take one")
	cmd.Flags().Float64Var(&speed, "speed", 1, "Speaking rate multiplier, where the provider supports it")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Render again and replace the cached audio")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Where rendered audio is kept")
	return cmd
}

func newListenCmd() *cobra.Command {
	var target, provider, model, language, output string
	var silence, timeout, duration int
	var allowSilence bool
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Hear the session, this machine's microphone, or a WAV file, and print what was said",
		Long: "Capture audio from a target and return a transcript.\n" +
			"With a session open the target is the session. With none open it is this machine's microphone.\n" +
			"After a say, listening starts where the say ended, and the result carries the reply latency.",
		Example: `  vibium listen
  # Come here, Watson.

  vibium listen --timeout 5000 --json
  # {"ok":true,"result":{"text":"Come here, Watson.","speech":true,"onsetMs":640,"speechEndMs":2010,"latencyMs":640,"stoppedBy":"silence",...}}

  vibium listen --target reply.wav --silence 0
  # Transcribes the whole file.

  vibium listen --silence 0 --duration 5000 -o five-seconds.wav
  # Captures exactly five seconds and saves them.`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			targetArg, err := speechTargetArg(target)
			if err != nil {
				printError(err)
				return
			}
			params := map[string]interface{}{}
			for key, value := range map[string]string{"target": targetArg, "provider": provider, "model": model, "language": language} {
				if value != "" {
					params[key] = value
				}
			}
			wait := speech.DefaultTimeout + speech.DefaultDuration
			for key, value := range map[string]int{"silence": silence, "timeout": timeout, "duration": duration} {
				if cmd.Flags().Changed(key) {
					if value < 0 {
						printError(fmt.Errorf("--%s must not be negative", key))
						return
					}
					params[key] = value
				}
			}
			if cmd.Flags().Changed("timeout") || cmd.Flags().Changed("duration") {
				wait = time.Duration(max(timeout, int(speech.DefaultTimeout/time.Millisecond))+max(duration, int(speech.DefaultDuration/time.Millisecond))) * time.Millisecond
			}
			if allowSilence {
				params["allowSilence"] = true
			}
			if output != "" {
				abs, err := filepath.Abs(output)
				if err != nil {
					printError(err)
					return
				}
				params["path"] = abs
			}
			result, err := speechCall("vibium_listen", params, wait+time.Minute)
			if err != nil {
				printError(err)
				return
			}
			var res speech.ListenResult
			if err := json.Unmarshal([]byte(extractText(result)), &res); err != nil {
				printError(err)
				return
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: res})
				return
			}
			fmt.Println(res.Text)
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "session, local, or a path ending in .wav (default: the session if one is open, else local)")
	cmd.Flags().StringVar(&provider, "provider", "", "Speech-to-text provider or preset (default: from the environment)")
	cmd.Flags().StringVar(&model, "model", "", "Provider's model id")
	cmd.Flags().StringVar(&language, "language", "", "BCP 47 tag (default en)")
	cmd.Flags().IntVar(&silence, "silence", 1000, "Milliseconds of trailing silence that end the reply; 0 turns endpointing off")
	cmd.Flags().IntVar(&timeout, "timeout", 10000, "Milliseconds to wait for speech to start")
	cmd.Flags().IntVar(&duration, "duration", 30000, "Maximum milliseconds of capture")
	cmd.Flags().BoolVar(&allowSilence, "allow-silence", false, "Return an empty transcript instead of failing when nothing is heard")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Also save the captured audio as a WAV file")
	return cmd
}
