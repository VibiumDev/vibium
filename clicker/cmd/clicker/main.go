package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/paths"
)

// connectFromEnv reads VIBIUM_CONNECT_URL and VIBIUM_CONNECT_API_KEY from the environment.
// Returns the connect URL and any headers to send with the WebSocket connection.
func connectFromEnv() (string, http.Header) {
	url := os.Getenv("VIBIUM_CONNECT_URL")
	apiKey := os.Getenv("VIBIUM_CONNECT_API_KEY")

	var headers http.Header
	if apiKey != "" {
		headers = make(http.Header)
		headers.Set("Authorization", "Bearer "+apiKey)
	}

	return url, headers
}

// connectCapsFromEnv reads VIBIUM_CONNECT_CAPS, a JSON object of extra
// alwaysMatch capabilities for classic WebDriver endpoints (cloud grids all
// take their config this way — vendor-prefixed capability keys).
// Invalid JSON is fatal: sending a session request without the user's
// capabilities would silently run against the wrong browser or account.
func connectCapsFromEnv() map[string]interface{} {
	return parseConnectCaps(os.Getenv("VIBIUM_CONNECT_CAPS"))
}

// parseConnectCaps parses a JSON capabilities object, exiting with a clear
// message when the JSON is invalid. Empty input means no extra capabilities.
func parseConnectCaps(capsJSON string) map[string]interface{} {
	if capsJSON == "" {
		return nil
	}
	var caps map[string]interface{}
	if err := json.Unmarshal([]byte(capsJSON), &caps); err != nil {
		fmt.Fprintf(os.Stderr, "Error: connect capabilities are not a valid JSON object: %v\n", err)
		os.Exit(1)
	}
	return caps
}

var version = "dev"

// Global flags
var (
	headless      bool
	verbose       bool
	jsonOutput    bool
	session       string
	engineName    string
	engineChannel string
	headlessSet   bool
	engineSet     bool
	channelSet    bool
)

// defaultEngine returns the browser engine to launch when --engine is not given.
func defaultEngine() string {
	if b := os.Getenv("VIBIUM_ENGINE"); b != "" {
		return b
	}
	return "chrome"
}

// applyGlobalFlags computes the state the persistent flags control: the *Set
// booleans, the log level, the env-var bridges for --session and --channel,
// and the engine, channel and session validation.
//
// It runs from the root's PersistentPreRunE, and again from
// parseFlagsAllowNegative for the commands that set DisableFlagParsing. Those
// parse their flags inside Run, after PersistentPreRunE has already read them
// as unset, so without the second pass --session and --channel are accepted
// and silently ignored (#482).
func applyGlobalFlags(cmd *cobra.Command) error {
	headlessSet = cmd.Flags().Changed("headless")
	engineSet = cmd.Flags().Changed("engine") || os.Getenv("VIBIUM_ENGINE") != ""
	channelSet = cmd.Flags().Changed("channel") || os.Getenv("VIBIUM_ENGINE_CHANNEL") != ""
	// Enable logging only if --verbose is used
	if verbose {
		log.Setup(log.LevelVerbose)
	}
	// The installer's commentary is progress, not result. Under --json it
	// has to leave stdout, or `install --json` prints plain text on the
	// lines before the envelope and nothing can parse the stream.
	if jsonOutput {
		browser.Progress = os.Stderr
	}
	// Bridge the flag to the env var so the paths package and any
	// auto-started daemon child process resolve the same session.
	if session != "" {
		if err := os.Setenv("VIBIUM_SESSION", session); err != nil {
			return err
		}
	}
	if cmd.Name() == "pipe" {
		noBrowser, _ := cmd.Flags().GetBool("no-browser")
		if noBrowser {
			return paths.ValidateSessionName(paths.SessionName())
		}
	}
	if engineName != "chrome" && engineName != "firefox" {
		return fmt.Errorf("unsupported engine %q (supported: chrome, firefox)", engineName)
	}
	// Channels differ per engine: Mozilla's are release/beta, Chrome
	// for Testing's are stable/beta/dev/canary.
	if engineChannel != "" {
		valid := map[string][]string{
			"firefox": {"release", "beta"},
			"chrome":  {"stable", "beta", "dev", "canary"},
		}[engineName]
		if !slices.Contains(valid, engineChannel) {
			return fmt.Errorf("unsupported channel %q for %s (supported: %s)",
				engineChannel, engineName, strings.Join(valid, ", "))
		}
	}
	// VIBIUM_ENGINE_PATH is an engine-neutral name but only Firefox
	// implements it. Say so rather than accepting the setting and
	// ignoring it.
	if engineName != "firefox" && os.Getenv("VIBIUM_ENGINE_PATH") != "" {
		return fmt.Errorf("VIBIUM_ENGINE_PATH is not supported for engine %q (firefox only)", engineName)
	}
	// Bridge the flag to the env var, like --session: the paths
	// package resolves the channel from the environment at both
	// install and launch time, and a daemon child process spawned
	// later inherits it.
	if engineChannel != "" {
		if err := os.Setenv("VIBIUM_ENGINE_CHANNEL", engineChannel); err != nil {
			return err
		}
	}
	return paths.ValidateSessionName(paths.SessionName())
}

func main() {
	progName := filepath.Base(os.Args[0])

	rootCmd := &cobra.Command{
		Use: progName + " [command | \"<prompt>\"]",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && !strings.ContainsAny(args[0], " \t\r\n") {
				return fmt.Errorf("unknown command %q; for a one-word goal use %s run %q", args[0], cmd.Name(), args[0])
			}
			return cobra.NoArgs(cmd, args)
		},
		Example: `  vibium "open example.com and find its contact page"
  # Equivalent to vibium run "open example.com and find its contact page".
  vibium run "stop"
  # One-word prompts need the explicit run command.`,
		Short: "Browser automation for AI agents and humans",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if isReadyCommand(cmd) {
				return nil
			}
			return applyGlobalFlags(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add global flags for browser commands
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", false, "Hide browser window (visible by default)")
	rootCmd.PersistentFlags().StringVar(&engineName, "engine", defaultEngine(), "Browser engine to launch: chrome or firefox (env: VIBIUM_ENGINE)")
	rootCmd.PersistentFlags().StringVar(&engineChannel, "channel", os.Getenv("VIBIUM_ENGINE_CHANNEL"), "Engine release channel to install and run (env: VIBIUM_ENGINE_CHANNEL); firefox: release (default) or beta; chrome: stable (default), beta, dev, or canary")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&session, "session", "", "Named daemon session for isolated concurrent use (env: VIBIUM_SESSION)")

	// Cobra's built-in completion command emits a bare `compdef` call, which
	// fails when the script is sourced before compinit has run (#201).
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(newCompletionCmd(rootCmd))

	// Register all commands
	rootCmd.AddCommand(newCommandsCmd(rootCmd))
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newPathsCmd())
	rootCmd.AddCommand(newIsInstalledCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newLaunchTestCmd())
	rootCmd.AddCommand(newWSTestCmd())
	rootCmd.AddCommand(newBiDiTestCmd())
	rootCmd.AddCommand(newNavigateCmd())
	rootCmd.AddCommand(newCheckCmd())
	runCmd := newRunCmd()
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(newReadyCmd())
	rootCmd.AddCommand(newScreenshotCmd())
	rootCmd.AddCommand(newEvalCmd())
	rootCmd.AddCommand(newFindCmd())
	rootCmd.AddCommand(newClickCmd())
	rootCmd.AddCommand(newTypeCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newPipeCmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newDaemonCmd())
	rootCmd.AddCommand(newTextCmd())
	rootCmd.AddCommand(newURLCmd())
	rootCmd.AddCommand(newTitleCmd())
	rootCmd.AddCommand(newHTMLCmd())
	rootCmd.AddCommand(newWaitCmd())
	rootCmd.AddCommand(newHoverCmd())
	rootCmd.AddCommand(newSelectCmd())
	rootCmd.AddCommand(newScrollCmd())
	rootCmd.AddCommand(newKeysCmd())
	rootCmd.AddCommand(newPagesCmd())
	rootCmd.AddCommand(newBackCmd())
	rootCmd.AddCommand(newForwardCmd())
	rootCmd.AddCommand(newReloadCmd())
	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newFillCmd())
	rootCmd.AddCommand(newPressCmd())
	rootCmd.AddCommand(newSetCmd())
	rootCmd.AddCommand(newUnsetCmd())
	rootCmd.AddCommand(newValueCmd())
	rootCmd.AddCommand(newAttrCmd())
	rootCmd.AddCommand(newA11yTreeCmd())
	rootCmd.AddCommand(newSleepCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newMapCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newPDFCmd())
	rootCmd.AddCommand(newHighlightCmd())
	rootCmd.AddCommand(newDblClickCmd())
	rootCmd.AddCommand(newFocusCmd())
	rootCmd.AddCommand(newCountCmd())
	rootCmd.AddCommand(newDialogCmd())
	rootCmd.AddCommand(newCookiesCmd())
	rootCmd.AddCommand(newDragCmd())
	rootCmd.AddCommand(newViewportCmd())
	rootCmd.AddCommand(newWindowCmd())
	rootCmd.AddCommand(newFramesCmd())
	rootCmd.AddCommand(newFrameCmd())
	rootCmd.AddCommand(newUploadCmd())
	rootCmd.AddCommand(newRecordCmd())
	rootCmd.AddCommand(newDownloadCmd())

	// Subcommand groups
	rootCmd.AddCommand(newIsCmd())
	rootCmd.AddCommand(newPageCmd())
	rootCmd.AddCommand(newMouseCmd())
	rootCmd.AddCommand(newStorageCmd())

	// Renamed commands
	rootCmd.AddCommand(newGeolocationCmd())
	rootCmd.AddCommand(newContentCmd())
	rootCmd.AddCommand(newMediaCmd())

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(progName + " v{{.Version}}\n")

	rootCmd.SetArgs(promptArgs(rootCmd, runCmd, os.Args[1:]))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
