package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/process"
)

var version = "dev"

// Global flags
var (
	headless   bool
	waitOpen   int
	waitClose  int
	verbose    bool
	jsonOutput bool
	oneshot    bool
)

func main() {
	// Setup signal handler to cleanup on Ctrl+C
	process.SetupSignalHandler()

	progName := filepath.Base(os.Args[0])

	rootCmd := &cobra.Command{
		Use:   progName,
		Short: "Browser automation for AI agents and humans",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Enable logging only if --verbose is used
			if verbose {
				log.Setup(log.LevelVerbose)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add global flags for browser commands
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", false, "Hide browser window (visible by default)")
	rootCmd.PersistentFlags().IntVar(&waitOpen, "wait-open", 0, "Seconds to wait after navigation for page to load")
	rootCmd.PersistentFlags().IntVar(&waitClose, "wait-close", 0, "Seconds to keep browser open before closing")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&oneshot, "oneshot", false, "One-shot mode (no daemon, launch+execute+teardown)")

	// VIBIUM_ONESHOT=1 env var forces oneshot mode (for tests)
	if os.Getenv("VIBIUM_ONESHOT") == "1" {
		oneshot = true
	}

	// Register all commands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newPathsCmd())
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
	rootCmd.AddCommand(newCheckActionableCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newDaemonCmd())
	rootCmd.AddCommand(newTextCmd())
	rootCmd.AddCommand(newURLCmd())
	rootCmd.AddCommand(newTitleCmd())
	rootCmd.AddCommand(newHTMLCmd())
	rootCmd.AddCommand(newFindAllCmd())
	rootCmd.AddCommand(newWaitCmd())
	rootCmd.AddCommand(newHoverCmd())
	rootCmd.AddCommand(newSelectCmd())
	rootCmd.AddCommand(newScrollCmd())
	rootCmd.AddCommand(newKeysCmd())
	rootCmd.AddCommand(newTabNewCmd())
	rootCmd.AddCommand(newTabsCmd())
	rootCmd.AddCommand(newTabSwitchCmd())
	rootCmd.AddCommand(newTabCloseCmd())
	rootCmd.AddCommand(newBackCmd())
	rootCmd.AddCommand(newForwardCmd())
	rootCmd.AddCommand(newReloadCmd())
	rootCmd.AddCommand(newQuitCmd())
	rootCmd.AddCommand(newFillCmd())
	rootCmd.AddCommand(newPressCmd())
	rootCmd.AddCommand(newCheckCmd())
	rootCmd.AddCommand(newUncheckCmd())
	rootCmd.AddCommand(newScrollIntoViewCmd())
	rootCmd.AddCommand(newValueCmd())
	rootCmd.AddCommand(newAttrCmd())
	rootCmd.AddCommand(newIsVisibleCmd())
	rootCmd.AddCommand(newA11yTreeCmd())
	rootCmd.AddCommand(newWaitForURLCmd())
	rootCmd.AddCommand(newWaitForLoadCmd())
	rootCmd.AddCommand(newSleepCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newMapCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newPDFCmd())
	rootCmd.AddCommand(newHighlightCmd())
	rootCmd.AddCommand(newDblClickCmd())
	rootCmd.AddCommand(newFocusCmd())
	rootCmd.AddCommand(newCountCmd())
	rootCmd.AddCommand(newIsEnabledCmd())
	rootCmd.AddCommand(newIsCheckedCmd())
	rootCmd.AddCommand(newWaitForTextCmd())
	rootCmd.AddCommand(newWaitForFnCmd())
	rootCmd.AddCommand(newDialogCmd())
	rootCmd.AddCommand(newCookiesCmd())
	rootCmd.AddCommand(newMouseMoveCmd())
	rootCmd.AddCommand(newMouseDownCmd())
	rootCmd.AddCommand(newMouseUpCmd())
	rootCmd.AddCommand(newMouseClickCmd())
	rootCmd.AddCommand(newDragCmd())
	rootCmd.AddCommand(newSetViewportCmd())
	rootCmd.AddCommand(newViewportCmd())
	rootCmd.AddCommand(newWindowCmd())
	rootCmd.AddCommand(newSetWindowCmd())
	rootCmd.AddCommand(newEmulateMediaCmd())
	rootCmd.AddCommand(newSetGeolocationCmd())
	rootCmd.AddCommand(newSetContentCmd())
	rootCmd.AddCommand(newFramesCmd())
	rootCmd.AddCommand(newFrameCmd())
	rootCmd.AddCommand(newUploadCmd())
	rootCmd.AddCommand(newTraceCmd())
	rootCmd.AddCommand(newStorageStateCmd())
	rootCmd.AddCommand(newRestoreStorageCmd())
	rootCmd.AddCommand(newDownloadCmd())

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(progName + " v{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
