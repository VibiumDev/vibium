package speech

import (
	"strings"
	"unicode"
)

// numberish are words a recognizer may write as digits or symbols, so
// "seventy two percent" and "72%" count as the same speech.
var numberish = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`zero one two three four five six seven eight nine ten eleven twelve thirteen fourteen
		fifteen sixteen seventeen eighteen nineteen twenty thirty forty fifty sixty seventy eighty ninety hundred thousand
		million billion first second third fourth fifth sixth seventh eighth ninth tenth eleventh twelfth twentieth
		thirtieth percent dollar dollars cent cents euro euros pound pounds degree degrees point oh`) {
		numberish[w] = true
	}
}

type spoken struct {
	word    string
	numeric bool
}

func spokenWords(s string) []spoken {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' })
	out := make([]spoken, 0, len(fields))
	for _, f := range fields {
		out = append(out, spoken{word: f, numeric: numberish[f] || strings.IndexFunc(f, unicode.IsDigit) >= 0})
	}
	return out
}

// sameWord matches a word and its abbreviation, as "st" and "street".
func sameWord(a, b string) bool {
	return a == b || (len(a) >= 2 && strings.HasPrefix(b, a)) || (len(b) >= 2 && strings.HasPrefix(a, b))
}

// endMatches reports whether the edge word of the text is at that edge of
// what was heard. A number may be heard as any number: recognizers write
// "three thirty" as "3:30".
func endMatches(want spoken, heard []spoken) bool {
	for _, h := range heard {
		if want.numeric && h.numeric || !want.numeric && sameWord(want.word, h.word) {
			return true
		}
	}
	return false
}

// SaysTheText reports whether a transcript of rendered audio is a reading
// of the text. Speech models sometimes render silence in place of words,
// most often the last ones, so both ends of the text must be heard and
// most of its letters must be there. Formatting is ignored: a recognizer
// writes numbers, money and abbreviations its own way.
func SaysTheText(text, heard string) bool {
	want, got := spokenWords(text), spokenWords(heard)
	if len(want) == 0 {
		return true
	}
	if len(got) == 0 {
		return false
	}
	edge := min(3, len(got))
	if !endMatches(want[0], got[:edge]) || !endMatches(want[len(want)-1], got[len(got)-edge:]) {
		return false
	}
	var a, b []rune
	for _, w := range want {
		if !w.numeric {
			a = append(a, []rune(w.word)...)
		}
	}
	for _, w := range got {
		if !w.numeric {
			b = append(b, []rune(w.word)...)
		}
	}
	if len(a) == 0 {
		return true
	}
	return float64(commonLetters(a, b)) >= 0.6*float64(len(a)) && len(b) <= 2*len(a)+20
}

// commonLetters is the length of the longest common subsequence.
func commonLetters(a, b []rune) int {
	prev := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else {
				cur[j] = max(prev[j], cur[j-1])
			}
		}
		prev = cur
	}
	return prev[len(b)]
}
