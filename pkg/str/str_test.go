package str

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTruncateErrorResponse(t *testing.T) {
	t.Run("short string passes through unchanged", func(t *testing.T) {
		require.Equal(t, "hello", TruncateErrorResponse("hello"))
	})

	t.Run("exactly at limit passes through unchanged", func(t *testing.T) {
		s := strings.Repeat("x", MaxErrorResponseBytes)
		result := TruncateErrorResponse(s)
		require.Equal(t, s, result)
		require.False(t, strings.HasSuffix(result, " [truncated]"))
	})

	t.Run("one byte over limit is truncated", func(t *testing.T) {
		s := strings.Repeat("x", MaxErrorResponseBytes+1)
		result := TruncateErrorResponse(s)
		require.True(t, strings.HasSuffix(result, " [truncated]"))
		require.LessOrEqual(t, len(result), MaxErrorResponseBytes+len(" [truncated]"))
	})

	t.Run("result is always valid UTF-8", func(t *testing.T) {
		// Build a string that is over the limit and ends with a partial multi-byte rune
		// at the cut point. U+1F600 (😀) encodes as 4 bytes; place it straddling the limit.
		prefix := strings.Repeat("a", MaxErrorResponseBytes-1)
		s := prefix + "😀" + strings.Repeat("b", 100)
		result := TruncateErrorResponse(s)
		assert.True(t, utf8.ValidString(result), "result must be valid UTF-8")
		assert.True(t, strings.HasSuffix(result, " [truncated]"))
	})

	t.Run("empty string passes through unchanged", func(t *testing.T) {
		require.Empty(t, TruncateErrorResponse(""))
	})
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
