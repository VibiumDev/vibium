package speech

import "testing"

func TestSaysTheText(t *testing.T) {
	cases := []struct {
		text, heard string
		want        bool
	}{
		// What the models actually did wrong.
		{"Come here, Watson", "Come here.", false},
		{"help", "", false},
		{"Set a timer for ten minutes", "Set a timer.", false},
		{"What's the weather like in Chicago tomorrow?", "It looks sunny, with a high of seventy.", false},
		{"Tell me a joke.", "Why did the chicken cross the road? To get to the other side.", false},
		{"Come here, Watson", "Watson.", false},
		// Faithful readings, written the recognizer's way.
		{"Come here, Watson", "Come here, Watson.", true},
		{"help", "Help.", true},
		{"Set a timer for 10 minutes.", "Set a timer for ten minutes.", true},
		{"Remind me to call the dentist at three thirty.", "Remind me to call the dentist at 3:30.", true},
		{"Call me at 3:30 PM on Nov. 12th.", "Call me at 3:30 p.m. on November 12.", true},
		{"Dr. Smith lives at 221B Baker St.", "Dr. Smith lives at 221B Baker Street.", true},
		{"That costs four dollars and fifty cents, or twenty percent off.", "That costs $4.50, or 20% off.", true},
		{"It's currently seventy two degrees and sunny in Chicago, with a high of seventy eight expected tomorrow.", "It's currently 72 degrees and sunny in Chicago, with a high of 78 expected tomorrow.", true},
		{"Open soundcheck.vibium.dev in Safari.", "Open soundcheck.vibium.devm in Safari.", true},
		{"Two hundred and fifty six divided by eight is thirty two.", "256 divided by 8 is 32.", true},
		{"Hey Siri, what's two plus two?", "Hey Siri, what's 2 plus 2?", true},
		{"105", "One hundred and five.", true},
		{"...", "", true},
	}
	for _, c := range cases {
		if got := SaysTheText(c.text, c.heard); got != c.want {
			t.Errorf("SaysTheText(%q, %q) = %v, want %v", c.text, c.heard, got, c.want)
		}
	}
}
