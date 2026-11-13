package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/paths"
)

type readyBrowser struct {
	Engine    string `json:"engine"`
	Channel   string `json:"channel"`
	Installed bool   `json:"installed"`
	Selected  bool   `json:"selected"`
	Path      string `json:"path,omitempty"`
	Driver    string `json:"driver,omitempty"`
}

var readyChannels = map[string][]string{
	"chrome":  {"stable", "beta", "dev", "canary"},
	"firefox": {"release", "beta"},
}

func readyBrowserSelection(cmd *cobra.Command, args []string) (string, string, error) {
	engine, channel := engineName, engineChannel
	if cmd.Name() == "browser" && len(args) == 1 {
		if cmd.Flags().Changed("engine") && engine != args[0] {
			return "", "", fmt.Errorf("browser argument conflicts with --engine")
		}
		if engine != args[0] && !cmd.Flags().Changed("channel") {
			channel = ""
		}
		engine = args[0]
	}
	if readyChannels[engine] == nil {
		return "", "", fmt.Errorf("supported browsers are chrome and firefox")
	}
	if channel == "" {
		channel = readyChannels[engine][0]
	}
	if !slices.Contains(readyChannels[engine], channel) {
		return "", "", fmt.Errorf("unsupported channel for selected browser")
	}
	if engine != "firefox" && os.Getenv("VIBIUM_ENGINE_PATH") != "" {
		return "", "", fmt.Errorf("VIBIUM_ENGINE_PATH is supported only for Firefox")
	}
	return engine, channel, nil
}

// The existing path resolver reads the channel from the environment. Readiness
// runs serially in its own CLI process and restores this setting after use.
func withReadyChannel(channel string, fn func()) {
	previous, exists := os.LookupEnv("VIBIUM_ENGINE_CHANNEL")
	defer func() {
		if exists {
			_ = os.Setenv("VIBIUM_ENGINE_CHANNEL", previous)
		} else {
			_ = os.Unsetenv("VIBIUM_ENGINE_CHANNEL")
		}
	}()
	_ = os.Setenv("VIBIUM_ENGINE_CHANNEL", channel)
	fn()
}

func executableInstalled(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && (runtime.GOOS == "windows" || info.Mode().Perm()&0111 != 0)
}

func discoverReadyBrowsers(engine, channel string) []readyBrowser {
	var found []readyBrowser
	for _, candidate := range []string{"chrome", "firefox"} {
		channels := readyChannels[candidate]
		if candidate == "firefox" && os.Getenv("VIBIUM_ENGINE_PATH") != "" {
			channels = []string{"release"}
			if engine == "firefox" {
				channels = []string{channel}
			}
		}
		for _, ch := range channels {
			b := readyBrowser{Engine: candidate, Channel: ch, Selected: candidate == engine && ch == channel}
			withReadyChannel(ch, func() {
				if candidate == "chrome" {
					b.Path, _ = paths.GetChromeExecutable()
					b.Driver, _ = paths.GetChromedriverPath()
					b.Installed = executableInstalled(b.Path) && executableInstalled(b.Driver)
				} else {
					b.Path, _ = paths.GetFirefoxExecutableForChannel(ch)
					b.Installed = executableInstalled(b.Path)
				}
			})
			found = append(found, b)
		}
	}
	return found
}

func checkBrowserSetup(cmd *cobra.Command, args []string) setupResult {
	result := setupResult{Ready: false, Checks: []setupCheck{}, Notes: []string{"Browser readiness checks executable files only. It does not launch a browser or driver, test BiDi connectivity, or establish that browser automation will succeed."}}
	engine, channel, err := readyBrowserSelection(cmd, args)
	if err != nil {
		result.Checks = append(result.Checks, setupCheck{"browser.configuration", "failed", err.Error(), "Choose chrome or firefox and a supported --channel; use vibium ready browser --help."})
		return result
	}
	result.Browsers = discoverReadyBrowsers(engine, channel)
	installed := false
	for _, b := range result.Browsers {
		if b.Selected {
			installed = b.Installed
		}
	}
	if !installed {
		result.Checks = append(result.Checks,
			setupCheck{"browser.installation", "failed", engine + " (" + channel + ") is missing or not executable.", "Run vibium install --engine " + engine + " --channel " + channel + "; for a custom Firefox, check VIBIUM_ENGINE_PATH and executable permissions."},
			setupCheck{"browser.connection", "skipped", "Browser launch and BiDi connectivity are not tested by readiness.", ""})
		return result
	}
	result.Checks = append(result.Checks, setupCheck{"browser.installation", "passed", engine + " (" + channel + ") and its required executable files are installed.", ""})
	// Readiness must remain a file inspection. Even a headless browser launch
	// can trigger expensive or stalled OS/GPU initialization.
	result.Checks = append(result.Checks, setupCheck{"browser.connection", "skipped", "Browser launch and BiDi connectivity are not tested by readiness.", ""})
	result.Ready = true
	return result
}
