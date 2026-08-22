package browser

import "testing"

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
