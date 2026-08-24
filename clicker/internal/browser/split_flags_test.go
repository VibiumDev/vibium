package browser

import (
	"reflect"
	"testing"
)

// Flag values containing spaces must survive as one token when quoted;
// strings.Fields cut them at the space, so Chrome received the quoted tail
// as a separate flag (#306).
func TestSplitFlagString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   \t ", nil},
		{"plain flags", "--no-sandbox --disable-gpu", []string{"--no-sandbox", "--disable-gpu"}},
		{"extra whitespace skipped", "  --a   --b  ", []string{"--a", "--b"}},
		{
			"double-quoted value with space",
			`--user-data-dir=/tmp/v --profile-directory="Profile 1"`,
			[]string{"--user-data-dir=/tmp/v", "--profile-directory=Profile 1"},
		},
		{
			"single-quoted value with space",
			`--profile-directory='Profile 1'`,
			[]string{"--profile-directory=Profile 1"},
		},
		{
			"fully quoted token",
			`"--profile-directory=Profile 1"`,
			[]string{"--profile-directory=Profile 1"},
		},
		{
			"quote character inside the other quote kind",
			`--flag="it's here"`,
			[]string{"--flag=it's here"},
		},
		{
			"unterminated quote runs to the end",
			`--flag="Profile 1`,
			[]string{"--flag=Profile 1"},
		},
	}
	for _, tc := range cases {
		if got := splitFlagString(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: splitFlagString(%q) = %#v, want %#v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestCustomChromeArgsHonorsQuoting(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", `--user-data-dir=/tmp/v --profile-directory="Profile 1"`)
	want := []string{"--user-data-dir=/tmp/v", "--profile-directory=Profile 1"}
	if got := customChromeArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("customChromeArgs() = %#v, want %#v", got, want)
	}
}
