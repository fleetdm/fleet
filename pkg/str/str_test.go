package str

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name        string
		testString  string
		delimiter   string
		removeEmpty bool
		expected    []string
	}{
		{
			name:        "basic comma split",
			testString:  "a,b,c",
			delimiter:   ",",
			removeEmpty: false,
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "trims whitespace",
			testString:  " a , b , c ",
			delimiter:   ",",
			removeEmpty: false,
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "keeps empty parts when removeEmpty is false",
			testString:  "a,,b,,c",
			delimiter:   ",",
			removeEmpty: false,
			expected:    []string{"a", "", "b", "", "c"},
		},
		{
			name:        "removes empty parts when removeEmpty is true",
			testString:  "a,,b,,c",
			delimiter:   ",",
			removeEmpty: true,
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "removes whitespace-only parts when removeEmpty is true",
			testString:  "a, ,b, ,c",
			delimiter:   ",",
			removeEmpty: true,
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "empty string",
			testString:  "",
			delimiter:   ",",
			removeEmpty: true,
			expected:    []string{},
		},
		{
			name:        "empty string without removeEmpty",
			testString:  "",
			delimiter:   ",",
			removeEmpty: false,
			expected:    []string{""},
		},
		{
			name:        "no delimiter found",
			testString:  "abc",
			delimiter:   ",",
			removeEmpty: false,
			expected:    []string{"abc"},
		},
		{
			name:        "multi-char delimiter",
			testString:  "a::b::c",
			delimiter:   "::",
			removeEmpty: false,
			expected:    []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitAndTrim(tt.testString, tt.delimiter, tt.removeEmpty)
			assert.Equal(t, tt.expected, result)
		})
	}
}
