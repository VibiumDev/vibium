package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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

var version = "dev"

// Global flags
var (
	headless       bool
	verbose        bool
	jsonOutput     bool
	session        string
	engineName     string
	firefoxChannel string
	headlessSet    bool
	engineSet      bool
	channelSet     bool
)

// defaultEngine returns the browser engine to launch when --engine is not given.
func defaultEngine() string {
	if b := os.Getenv("VIBIUM_ENGINE"); b != "" {
		return b
	}
	return "chrome"
}

func main() {
	progName := filepath.Base(os.Args[0])

	rootCmd := &cobra.Command{
		Use:   progName,
		Short: "Browser automation for AI agents and humans",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			headlessSet = cmd.Flags().Changed("headless")
			engineSet = cmd.Flags().Changed("engine") || os.Getenv("VIBIUM_ENGINE") != ""
			channelSet = cmd.Flags().Changed("channel") || os.Getenv("VIBIUM_ENGINE_CHANNEL") != ""
			// Enable logging only if --verbose is used
			if verbose {
				log.Setup(log.LevelVerbose)
			}
			// Bridge the flag to the env var so the paths package and any
			// auto-started daemon child process resolve the same session.
			if session != "" {
				if err := os.Setenv("VIBIUM_SESSION", session); err != nil {
					return err
				}
			}
			if engineName != "chrome" && engineName != "firefox" {
				return fmt.Errorf("unsupported engine %q (supported: chrome, firefox)", engineName)
			}
			if firefoxChannel != "" && firefoxChannel != "release" && firefoxChannel != "beta" {
				return fmt.Errorf("unsupported channel %q (supported: release, beta)", firefoxChannel)
			}
			// VIBIUM_ENGINE_CHANNEL, VIBIUM_ENGINE_PATH, and VIBIUM_ENGINE_VERSION are
			// engine-neutral names but only Firefox implements them. Say so
			// rather than accepting the setting and ignoring it.
			if engineName != "firefox" {
				for _, unsupported := range []struct{ name, value string }{
					{"--channel/VIBIUM_ENGINE_CHANNEL", firefoxChannel},
					{"VIBIUM_ENGINE_PATH", os.Getenv("VIBIUM_ENGINE_PATH")},
					{"VIBIUM_ENGINE_VERSION", os.Getenv("VIBIUM_ENGINE_VERSION")},
				} {
					if unsupported.value != "" {
						return fmt.Errorf("%s is not supported for engine %q (firefox only)", unsupported.name, engineName)
					}
				}
			}
			// Bridge the flag to the env var, like --session: the paths
			// package resolves the channel from the environment at both
			// install and launch time, and a daemon child process spawned
			// later inherits it.
			if firefoxChannel != "" {
				if err := os.Setenv("VIBIUM_ENGINE_CHANNEL", firefoxChannel); err != nil {
					return err
				}
			}
			return paths.ValidateSessionName(paths.SessionName())
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add global flags for browser commands
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", false, "Hide browser window (visible by default)")
	rootCmd.PersistentFlags().StringVar(&engineName, "engine", defaultEngine(), "Browser engine to launch: chrome or firefox (env: VIBIUM_ENGINE)")
	rootCmd.PersistentFlags().StringVar(&firefoxChannel, "channel", os.Getenv("VIBIUM_ENGINE_CHANNEL"), "Engine release channel to install and run: release (default) or beta; Firefox only for now (env: VIBIUM_ENGINE_CHANNEL)")
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
	rootCmd.AddCommand(newCheckCmd())
	rootCmd.AddCommand(newUncheckCmd())
	rootCmd.AddCommand(newValueCmd())
	rootCmd.AddCommand(newAttrCmd())
	rootCmd.AddCommand(newA11yTreeCmd())
	rootCmd.AddCommand(newSleepCmd())
	rootCmd.AddCommand(newSkillCmd())
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
