package bidi

// NormalizeKeyChar converts characters to their WebDriver BiDi equivalents.
// For example, newline (\n) must be sent as carriage return (\r) to produce Enter.
func NormalizeKeyChar(r rune) rune {
	if r == '\n' {
		return '\r'
	}
	return r
}
