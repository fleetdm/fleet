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

func TestTruncateBytes(t *testing.T) {
	const marker = "... (truncated)"

	t.Run("under the limit passes through unchanged", func(t *testing.T) {
		require.Equal(t, "hello", TruncateBytes("hello", 64, marker))
	})

	t.Run("exactly at the limit passes through unchanged", func(t *testing.T) {
		s := strings.Repeat("x", 32)
		require.Equal(t, s, TruncateBytes(s, 32, marker))
	})

	t.Run("result including the marker stays within the limit", func(t *testing.T) {
		result := TruncateBytes(strings.Repeat("x", 1000), 64, marker)
		require.LessOrEqual(t, len(result), 64)
		require.True(t, strings.HasSuffix(result, marker))
	})

	t.Run("cut in the middle of a multi-byte rune stays valid UTF-8", func(t *testing.T) {
		// Place a 4-byte rune so the byte limit lands inside it.
		limit := 40
		s := strings.Repeat("a", limit-len(marker)-2) + "😀" + strings.Repeat("b", 100)
		result := TruncateBytes(s, limit, marker)
		require.True(t, utf8.ValidString(result), "result must be valid UTF-8")
		require.LessOrEqual(t, len(result), limit)
		require.True(t, strings.HasSuffix(result, marker))
	})

	t.Run("multi-byte text is capped by bytes, not characters", func(t *testing.T) {
		// The reason this helper exists: 100 two-byte runes is 200 bytes, which a 100-byte column cannot hold.
		result := TruncateBytes(strings.Repeat("é", 100), 100, marker)
		require.LessOrEqual(t, len(result), 100)
		require.True(t, utf8.ValidString(result))
	})

	t.Run("limit too small for the marker drops the marker", func(t *testing.T) {
		result := TruncateBytes(strings.Repeat("x", 100), 5, marker)
		require.Equal(t, "xxxxx", result)
	})

	t.Run("non-positive limit returns empty", func(t *testing.T) {
		require.Empty(t, TruncateBytes("hello", 0, marker))
		require.Empty(t, TruncateBytes("hello", -1, marker))
	})

	t.Run("empty marker truncates cleanly", func(t *testing.T) {
		require.Equal(t, "xxxxx", TruncateBytes(strings.Repeat("x", 100), 5, ""))
	})
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

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxRunes int
		expected string
	}{
		{
			name:     "exactly the limit is unchanged",
			input:    "hello",
			maxRunes: 5,
			expected: "hello",
		},
		{
			name:     "longer ASCII is cut to the limit",
			input:    "hello world",
			maxRunes: 5,
			expected: "hello",
		},
		{
			// The byte length exceeds the limit while the character count does not, which is what a
			// byte-based truncation would get wrong.
			name:     "counts characters, not bytes, so multi-byte text is not cut early",
			input:    "héllo",
			maxRunes: 5,
			expected: "héllo",
		},
		{
			name:     "cuts multi-byte text on a character boundary",
			input:    strings.Repeat("é", 10),
			maxRunes: 4,
			expected: strings.Repeat("é", 4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, TruncateRunes(tt.input, tt.maxRunes))
		})
	}
}
