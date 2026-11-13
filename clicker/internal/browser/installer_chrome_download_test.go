package browser

import "testing"

// The Chrome download URL comes from a fetched JSON document and the binary is
// then extracted and executed, so only HTTPS URLs on the Chrome for Testing
// host may be fetched (#security hardening). Firefox verifies SHA256SUMS; Chrome
// publishes none, so this host pin is its integrity control.
func TestValidateChromeDownloadURL(t *testing.T) {
	ok := []string{
		"https://storage.googleapis.com/chrome-for-testing-public/156.0.8078.73/linux64/chrome-linux64.zip",
		"https://storage.googleapis.com/chrome-for-testing-public/156.0.8078.73/mac-arm64/chromedriver-mac-arm64.zip",
	}
	for _, u := range ok {
		if err := validateChromeDownloadURL(u); err != nil {
			t.Errorf("validateChromeDownloadURL(%q) = %v, want nil", u, err)
		}
	}

	bad := []string{
		"http://storage.googleapis.com/chrome-for-testing-public/x.zip",    // not https
		"https://evil.example.com/chrome-for-testing-public/x.zip",         // wrong host
		"https://storage.googleapis.com.evil.com/chrome-for-testing/x.zip", // lookalike host
		"ftp://storage.googleapis.com/x.zip",                               // wrong scheme
		"not a url",                                                        // no scheme/host
	}
	for _, u := range bad {
		if err := validateChromeDownloadURL(u); err == nil {
			t.Errorf("validateChromeDownloadURL(%q) = nil, want error", u)
		}
	}
}
