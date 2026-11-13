package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func runPathsCmd(t *testing.T) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	newPathsCmd().Run(nil, nil)
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func setPathsGlobals(t *testing.T, engine string, jsonMode bool) {
	t.Helper()
	origEngine, origJSON := engineName, jsonOutput
	t.Cleanup(func() { engineName, jsonOutput = origEngine, origJSON })
	engineName, jsonOutput = engine, jsonMode
}

// paths printed with fmt.Printf unconditionally and never read --json or
// --engine (#392): scripts had to scrape the human text, and Firefox users
// were shown Chrome paths.
func TestPathsJSONEmitsEnvelope(t *testing.T) {
	t.Setenv("VIBIUM_CACHE_DIR", t.TempDir())
	setPathsGlobals(t, "chrome", true)

	var env struct {
		OK     bool `json:"ok"`
		Result struct {
			Engine   string `json:"engine"`
			CacheDir string `json:"cacheDir"`
			Firefox  string `json:"firefox"`
		} `json:"result"`
	}
	out := runPathsCmd(t)
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("--json output is not JSON: %v\n%s", err, out)
	}
	if !env.OK || env.Result.Engine != "chrome" || env.Result.CacheDir == "" {
		t.Fatalf("unexpected envelope: %s", out)
	}
	if env.Result.Firefox != "" {
		t.Fatalf("chrome engine should not report a firefox path: %s", out)
	}
}

func TestPathsFirefoxEngineReportsFirefox(t *testing.T) {
	t.Setenv("VIBIUM_CACHE_DIR", t.TempDir())
	setPathsGlobals(t, "firefox", false)

	out := runPathsCmd(t)
	if !strings.Contains(out, "Firefox:") {
		t.Fatalf("firefox engine output should name Firefox:\n%s", out)
	}
	if strings.Contains(out, "Chrome:") || strings.Contains(out, "Chromedriver:") {
		t.Fatalf("firefox engine output should not name Chrome:\n%s", out)
	}
}

func TestPathsFirefoxJSONUsesEnginePath(t *testing.T) {
	t.Setenv("VIBIUM_CACHE_DIR", t.TempDir())
	t.Setenv("VIBIUM_ENGINE_PATH", "/opt/firefox/firefox")
	setPathsGlobals(t, "firefox", true)

	var env struct {
		Result struct {
			Firefox string `json:"firefox"`
			Chrome  string `json:"chrome"`
		} `json:"result"`
	}
	out := runPathsCmd(t)
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("--json output is not JSON: %v\n%s", err, out)
	}
	if env.Result.Firefox != "/opt/firefox/firefox" || env.Result.Chrome != "" {
		t.Fatalf("unexpected result: %s", out)
	}
}
