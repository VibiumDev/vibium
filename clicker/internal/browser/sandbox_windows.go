//go:build windows

package browser

// checkSandboxable is a no-op on Windows: platformChromeArgs already passes
// --no-sandbox there unconditionally, because the Chrome for Testing sandbox
// cannot read its own executable under %LOCALAPPDATA% ACLs (27ff6cc).
func checkSandboxable() error { return nil }
