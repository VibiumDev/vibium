package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sha256("abc"), a fixed test vector.
const abcSHA256 = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

const sampleSums = "87e9fcaecaef101d7a0910ba8960d4b72b5b4ae91fd4e08069088d3c06f9a792  linux-aarch64/en-US/firefox-153.0.4.tar.xz\n" +
	abcSHA256 + "  mac/en-US/Firefox 153.0.4.dmg\n" +
	"da8897a6a618e73878e6022a2bece76af509c304c73ae5c53dc523d35cb7bae6  linux-x86_64/en-US/firefox-153.0.4.tar.xz\n"

func writeArchive(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// The SHA256SUMS paths use unescaped spaces (mac DMG), so the lookup key must
// be the raw relative path, not the URL form.
func TestFirefoxArchiveRelPathMatchesSums(t *testing.T) {
	cases := []struct {
		goos, goarch, version, want string
	}{
		{"darwin", "arm64", "153.0.4", "mac/en-US/Firefox 153.0.4.dmg"},
		{"linux", "amd64", "153.0.4", "linux-x86_64/en-US/firefox-153.0.4.tar.xz"},
		{"linux", "arm64", "153.0.4", "linux-aarch64/en-US/firefox-153.0.4.tar.xz"},
	}
	for _, tc := range cases {
		if got := firefoxArchiveRelPath(tc.goos, tc.goarch, tc.version); got != tc.want {
			t.Errorf("firefoxArchiveRelPath(%s, %s, %s) = %q, want %q",
				tc.goos, tc.goarch, tc.version, got, tc.want)
		}
	}
}

func TestFindFirefoxChecksum(t *testing.T) {
	if got := findFirefoxChecksum(sampleSums, "mac/en-US/Firefox 153.0.4.dmg"); got != abcSHA256 {
		t.Errorf("findFirefoxChecksum = %q, want %q", got, abcSHA256)
	}
	if got := findFirefoxChecksum(sampleSums, "mac/en-US/Firefox 999.0.dmg"); got != "" {
		t.Errorf("findFirefoxChecksum for absent path = %q, want empty", got)
	}
}

func TestVerifyArchiveAgainstSums(t *testing.T) {
	good := writeArchive(t, "abc")
	if err := verifyArchiveAgainstSums(good, sampleSums, "mac/en-US/Firefox 153.0.4.dmg"); err != nil {
		t.Errorf("matching archive rejected: %v", err)
	}

	tampered := writeArchive(t, "abc-tampered")
	err := verifyArchiveAgainstSums(tampered, sampleSums, "mac/en-US/Firefox 153.0.4.dmg")
	if err == nil || !strings.Contains(err.Error(), "SHA256 mismatch") {
		t.Errorf("tampered archive: got %v, want SHA256 mismatch error", err)
	}

	err = verifyArchiveAgainstSums(good, sampleSums, "mac/en-US/Firefox 999.0.dmg")
	if err == nil || !strings.Contains(err.Error(), "no SHA256SUMS entry") {
		t.Errorf("absent entry: got %v, want no-entry error", err)
	}
}
