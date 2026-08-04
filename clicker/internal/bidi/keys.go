package bidi

// ConvertToKeyChar maps string literal characters to their WebDriver BiDi key
// equivalents for keyboard input. For example, newline (\n) is a line break in
// strings but must be sent as the Return key (\uE006) to produce a key press.
func ConvertToKeyChar(r rune) rune {
	if r == '\n' {
		return '\uE006'
	}
	return r
}
