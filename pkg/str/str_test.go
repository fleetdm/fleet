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

func TestParseUintList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []uint
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "42",
			expected: []uint{42},
		},
		{
			name:     "multiple values",
			input:    "1,2,3",
			expected: []uint{1, 2, 3},
		},
		{
			name:     "trims whitespace",
			input:    " 1 , 2 , 3 ",
			expected: []uint{1, 2, 3},
		},
		{
			name:     "skips non-numeric values",
			input:    "1,abc,2,,3",
			expected: []uint{1, 2, 3},
		},
		{
			name:     "skips negative values",
			input:    "1,-2,3",
			expected: []uint{1, 3},
		},
		{
			name:     "all invalid returns empty slice",
			input:    "a,b,c",
			expected: []uint{},
		},
		{
			name:     "zero is valid",
			input:    "0,1",
			expected: []uint{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseUintList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseStringList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "foo",
			expected: []string{"foo"},
		},
		{
			name:     "multiple values",
			input:    "foo,bar,baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "trims whitespace",
			input:    " foo , bar , baz ",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "drops empty values",
			input:    "foo,,bar, ,baz",
			expected: []string{"foo", "bar", "baz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStringList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
