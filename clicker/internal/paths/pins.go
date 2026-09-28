package paths

import (
	_ "embed"
	"encoding/json"
)

// browsers.json holds the known-good browser versions the default channels
// install and launch. It is a data file on purpose: the version-bump
// workflow edits it with jq, and CI's browser cache keys hash only this
// file, so a pin bump fetches a fresh browser while unrelated installer
// changes do not (#580). The pins stay compiled into the binary; a release
// launches the browsers it was tested with. VIBIUM_ENGINE_VERSION remains
// the runtime override.
//
//go:embed browsers.json
var browsersJSON []byte

// PinnedChromeVersion is the known-good Chrome for Testing version the
// stable channel installs and launches (#470, #579). CI tests exactly this
// version; the version-bump workflow opens a tested PR when Google ships a
// new Stable.
var PinnedChromeVersion string

// PinnedFirefoxVersion is the known-good Firefox version the release
// channel installs (#469). CI tests exactly this version; the version-bump
// workflow opens a tested PR when Mozilla ships a new release.
var PinnedFirefoxVersion string

func init() {
	var pins struct {
		Chrome  string `json:"chrome"`
		Firefox string `json:"firefox"`
	}
	if err := json.Unmarshal(browsersJSON, &pins); err != nil {
		panic("browsers.json is not valid JSON: " + err.Error())
	}
	if pins.Chrome == "" || pins.Firefox == "" {
		panic("browsers.json must pin both chrome and firefox")
	}
	PinnedChromeVersion = pins.Chrome
	PinnedFirefoxVersion = pins.Firefox
}
