package httpapi

import "testing"

func TestTrimCountsCharactersNotBytes(t *testing.T) {
	cases := []struct {
		name  string
		value string
		max   int
		want  string
	}{
		{"ascii", "  hello  ", 10, "hello"},
		{"truncates ascii", "abcdef", 3, "abc"},
		{"keeps emoji whole", "🙂🙂🙂", 2, "🙂🙂"},
		{"keeps accents whole", "éééé", 2, "éé"},
		{"empty", "   ", 5, ""},
	}

	for _, testCase := range cases {
		if got := trim(testCase.value, testCase.max); got != testCase.want {
			t.Errorf("%s: trim(%q, %d) = %q, want %q", testCase.name, testCase.value, testCase.max, got, testCase.want)
		}
	}
}
