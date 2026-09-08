package browser

import "testing"

// With no override, the release channel installs the baked known-good
// version, offline (#469). Firefox 155 broke every fresh install on its
// release day (#464) because this used to resolve Mozilla's current
// release.
func TestFirefoxReleaseChannelResolvesToBakedPin(t *testing.T) {
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	v, err := resolveFirefoxVersion("release")
	if err != nil {
		t.Fatalf("resolveFirefoxVersion(release) error = %v", err)
	}
	if v != pinnedFirefoxVersion {
		t.Errorf("resolveFirefoxVersion(release) = %q, want the baked %q", v, pinnedFirefoxVersion)
	}
}

// A pinned version must not consult the network: resolution returns the pin
// as-is, for any channel (#326).
func TestResolveFirefoxVersionHonorsPin(t *testing.T) {
	t.Setenv("VIBIUM_ENGINE_VERSION", "153.0.4")
	for _, channel := range []string{"release", "beta"} {
		v, err := resolveFirefoxVersion(channel)
		if err != nil {
			t.Fatalf("resolveFirefoxVersion(%s) error = %v", channel, err)
		}
		if v != "153.0.4" {
			t.Errorf("resolveFirefoxVersion(%s) = %q, want the pinned 153.0.4", channel, v)
		}
	}
}
