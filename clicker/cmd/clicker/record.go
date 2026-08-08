package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newRecordCmd() *cobra.Command {
	recordCmd := &cobra.Command{
		Use:   "record",
		Short: "Record browser sessions (screenshots and snapshots)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a recording",
		Example: `  vibium record start
  # Start recording with screenshots (default); video too when the
  # engine supports it (Firefox 154+)

  vibium record start --video -o record.zip
  # Require video — fails with an explanatory error on Chrome
  # Recording "record" started (video: on, ...), saving to record.zip

  vibium record start --video --video-size 1280x720 --video-fps 30
  # Explicit video dimensions (defaults follow the viewport)

  vibium record start --video=false
  # Record without video

  vibium record start --screenshots=false
  # Record without screenshots

  vibium record start --snapshots
  # Record with screenshots and HTML snapshots

  vibium record start --format png
  # Use PNG format instead of JPEG (larger files, lossless)

  vibium record start --quality 0.1
  # Lower JPEG quality for smaller recording files

  vibium record start --title "Login Flow"
  # Set a title shown in the trace viewer`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			screenshots, _ := cmd.Flags().GetBool("screenshots")
			snapshots, _ := cmd.Flags().GetBool("snapshots")
			bidi, _ := cmd.Flags().GetBool("bidi")
			name, _ := cmd.Flags().GetString("name")
			title, _ := cmd.Flags().GetString("title")
			sources, _ := cmd.Flags().GetBool("sources")
			format, _ := cmd.Flags().GetString("format")
			quality, _ := cmd.Flags().GetFloat64("quality")
			videoSize, _ := cmd.Flags().GetString("video-size")
			videoFPS, _ := cmd.Flags().GetInt("video-fps")
			output, _ := cmd.Flags().GetString("output")

			callArgs := map[string]interface{}{}
			if name != "" {
				callArgs["name"] = name
			}
			if title != "" {
				callArgs["title"] = title
			}
			callArgs["screenshots"] = screenshots
			if snapshots {
				callArgs["snapshots"] = true
			}
			if sources {
				callArgs["sources"] = true
			}
			if bidi {
				callArgs["bidi"] = true
			}
			if format != "jpeg" {
				callArgs["format"] = format
			}
			if quality != 0.5 {
				callArgs["quality"] = quality
			}
			if cmd.Flags().Changed("video") {
				video, _ := cmd.Flags().GetBool("video")
				callArgs["video"] = video
			}
			if videoSize != "" {
				w, h, err := parseVideoSize(videoSize)
				if err != nil {
					printError(err)
					return
				}
				callArgs["video_width"] = w
				callArgs["video_height"] = h
			}
			if videoFPS > 0 {
				callArgs["video_frame_rate"] = videoFPS
			}
			// The daemon is a separate long-lived process whose working
			// directory is not the user's, so resolve against this shell
			// before the path goes over the socket (#119) — including the
			// record.zip default, or it lands wherever the daemon started.
			if output == "" {
				output = "record.zip"
			}
			if abs, err := filepath.Abs(output); err == nil {
				output = abs
			}
			callArgs["path"] = output
			result, err := daemonCall("browser_record_start", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	startCmd.Flags().Bool("screenshots", true, "Capture screenshots after each action")
	startCmd.Flags().Bool("snapshots", false, "Capture HTML snapshots")
	startCmd.Flags().Bool("sources", false, "Include source information")
	startCmd.Flags().Bool("bidi", false, "Record raw BiDi commands in the recording")
	startCmd.Flags().String("name", "", "Name for the recording")
	startCmd.Flags().String("title", "", "Title shown in trace viewer (defaults to name)")
	startCmd.Flags().String("format", "jpeg", "Screenshot format: jpeg or png")
	startCmd.Flags().Float64("quality", 0.5, "JPEG quality 0.0-1.0 (ignored for png)")
	startCmd.Flags().Bool("video", false, "Require video (omit: video when the engine supports it; =false: off)")
	startCmd.Flags().String("video-size", "", "Video dimensions as WxH, e.g. 1280x720 (default: viewport)")
	startCmd.Flags().Int("video-fps", 0, "Video frame rate (engine default if omitted)")
	startCmd.Flags().StringP("output", "o", "", "Where the recording ZIP lands at stop (default: record.zip)")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop recording and save",
		Example: `  vibium record stop
  # Saved record.zip (23 steps, 14s video)

  vibium record stop -o my-recording.zip
  # Save to a custom path (overrides the path declared at start)`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			// The daemon is a separate long-lived process whose working directory
			// is not the user's, so resolve against this shell before the path goes
			// over the socket (#119).
			if output != "" {
				if abs, err := filepath.Abs(output); err == nil {
					output = abs
				}
			}

			callArgs := map[string]interface{}{}
			if output != "" {
				callArgs["path"] = output
			}
			result, err := daemonCall("browser_record_stop", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	stopCmd.Flags().StringP("output", "o", "", "Output file path (default: record.zip)")

	// Group subcommand (replaces start-group/stop-group)
	groupCmd := &cobra.Command{
		Use:   "group",
		Short: "Manage recording groups",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	groupStartCmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a named group in the recording",
		Example: `  vibium record group start "Login"
  # Groups nest actions in the trace viewer`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_record_start_group", map[string]interface{}{
				"name": args[0],
			})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	groupStopCmd := &cobra.Command{
		Use:     "stop",
		Short:   "End the current recording group",
		Example: `  vibium record group stop`,
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			result, err := daemonCall("browser_record_stop_group", map[string]interface{}{})
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}

	groupCmd.AddCommand(groupStartCmd)
	groupCmd.AddCommand(groupStopCmd)

	// Chunk subcommand (replaces start-chunk/stop-chunk)
	chunkCmd := &cobra.Command{
		Use:   "chunk",
		Short: "Manage recording chunks",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	chunkStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new chunk within the current recording",
		Example: `  vibium record chunk start
  # Start a new chunk (for splitting long recordings)

  vibium record chunk start --name "part2" --title "Checkout Flow"`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			title, _ := cmd.Flags().GetString("title")

			callArgs := map[string]interface{}{}
			if name != "" {
				callArgs["name"] = name
			}
			if title != "" {
				callArgs["title"] = title
			}
			result, err := daemonCall("browser_record_start_chunk", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	chunkStartCmd.Flags().String("name", "", "Name for the chunk")
	chunkStartCmd.Flags().String("title", "", "Title shown in trace viewer")

	chunkStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Package current chunk into a ZIP file (recording stays active)",
		Example: `  vibium record chunk stop
  # Save chunk to chunk.zip

  vibium record chunk stop -o part1.zip`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			// The daemon is a separate long-lived process whose working directory
			// is not the user's, so resolve against this shell before the path goes
			// over the socket (#119).
			if output != "" {
				if abs, err := filepath.Abs(output); err == nil {
					output = abs
				}
			}

			callArgs := map[string]interface{}{}
			if output != "" {
				callArgs["path"] = output
			}
			result, err := daemonCall("browser_record_stop_chunk", callArgs)
			if err != nil {
				printError(err)
				return
			}
			printResult(result)
		},
	}
	chunkStopCmd.Flags().StringP("output", "o", "", "Output file path (default: chunk.zip)")

	chunkCmd.AddCommand(chunkStartCmd)
	chunkCmd.AddCommand(chunkStopCmd)

	recordCmd.AddCommand(startCmd)
	recordCmd.AddCommand(stopCmd)
	recordCmd.AddCommand(groupCmd)
	recordCmd.AddCommand(chunkCmd)
	return recordCmd
}

// parseVideoSize parses a WxH flag value like "1280x720".
func parseVideoSize(size string) (int, int, error) {
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid --video-size %q: expected WxH, e.g. 1280x720", size)
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("invalid --video-size %q: expected WxH, e.g. 1280x720", size)
	}
	return w, h, nil
}
