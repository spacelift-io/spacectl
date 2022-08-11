package stack

import (
	"strings"
	"testing"
)

func TestTrimmedValue(t *testing.T) {
	limit := 80
	type testRow struct {
		input    string
		expected string
	}

	testTable := []testRow{
		{input: "foo", expected: "foo"},
		{input: "abc\rdef\nghi\r\njkl", expected: "abc def ghi jkl"},
		{input: "a\r\r\nb\n\r\rc", expected: "a  b   c"},
		{input: strings.Repeat("a", limit), expected: strings.Repeat("a", limit)},
		{input: strings.Repeat("a", limit+1), expected: strings.Repeat("a", limit-3) + "..."},
	}

	for _, testCase := range testTable {
		o := listEnvElementOutput{
			Value: &testCase.input,
		}

		result := o.trimmedValue()

		if result != testCase.expected {
			t.Errorf("trimmed value of %s should be %s but was %s", testCase.input, testCase.expected, result)
		}
	}
}
