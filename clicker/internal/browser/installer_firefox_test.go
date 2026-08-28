package browser

import "testing"

// The per-OS/arch URL shape is exactly what breaks silently when Mozilla
// changes their archive layout, so it is pinned here (#320). The mac DMG
// name contains a space, which must arrive percent-encoded.
func TestFirefoxDownloadURL(t *testing.T) {
	cases := []struct {
		name    string
		goos    string
		goarch  string
		version string
		want    string
	}{
		{
			name: "mac dmg with escaped space", goos: "darwin", goarch: "arm64", version: "153.0.4",
			want: "https://ftp.mozilla.org/pub/firefox/releases/153.0.4/mac/en-US/Firefox%20153.0.4.dmg",
		},
		{
			name: "linux x86_64 tarball", goos: "linux", goarch: "amd64", version: "153.0.4",
			want: "https://ftp.mozilla.org/pub/firefox/releases/153.0.4/linux-x86_64/en-US/firefox-153.0.4.tar.xz",
		},
		{
			name: "linux arm64 tarball", goos: "linux", goarch: "arm64", version: "153.0.4",
			want: "https://ftp.mozilla.org/pub/firefox/releases/153.0.4/linux-aarch64/en-US/firefox-153.0.4.tar.xz",
		},
		{
			name: "beta lives under the same releases tree", goos: "linux", goarch: "amd64", version: "154.0b6",
			want: "https://ftp.mozilla.org/pub/firefox/releases/154.0b6/linux-x86_64/en-US/firefox-154.0b6.tar.xz",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := firefoxDownloadURLFor(tc.goos, tc.goarch, tc.version); got != tc.want {
				t.Errorf("firefoxDownloadURLFor(%s, %s, %s) = %q, want %q",
					tc.goos, tc.goarch, tc.version, got, tc.want)
			}
		})
	}
}
