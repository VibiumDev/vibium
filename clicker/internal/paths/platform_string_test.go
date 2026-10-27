package paths

import "testing"

// aarch64 Linux must resolve to linux-arm64 so it downloads its own Chrome for
// Testing build; it used to map to linux64 (x86), which aborted with "exec
// format error" on a Pi (#638, #675).
func TestPlatformStringFor(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "arm64", "linux-arm64"},
		{"linux", "amd64", "linux64"},
		{"darwin", "arm64", "mac-arm64"},
		{"darwin", "amd64", "mac-x64"},
		{"windows", "amd64", "win64"},
		{"windows", "arm64", "win64"},
	}
	for _, c := range cases {
		if got := platformStringFor(c.goos, c.goarch); got != c.want {
			t.Errorf("platformStringFor(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}
