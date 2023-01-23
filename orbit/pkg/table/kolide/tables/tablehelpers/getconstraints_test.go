package tablehelpers

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetConstraints(t *testing.T) {
	t.Parallel()

	mockQC := MockQueryContext(map[string][]string{
		"empty_array":      {},
		"blank":            {""},
		"single":           {"a"},
		"double":           {"a", "b"},
		"duplicates":       {"a", "a", "b", "b"},
		"duplicate_blanks": {"a", "a", "", ""},
	})

	var tests = []struct {
		name     string
		expected []string
		opts     []GetConstraintOpts
	}{
		// Basic queries
		{
			name:     "single",
			expected: []string{"a"},
		},
		{
			name:     "does_not_exist",
			expected: []string(nil),
		},
		{
			name:     "empty_array",
			expected: []string(nil),
		},
		{
			name:     "blank",
			expected: []string{""},
		},
		{
			name:     "double",
			expected: []string{"a", "b"},
		},
		{
			name:     "duplicates",
			expected: []string{"a", "b"},
		},
		{
			name:     "duplicate_blanks",
			expected: []string{"", "a"},
		},

		// defaults
		{
			name:     "does_not_exist_with_defaults",
			expected: []string{"a", "b"},
			opts:     []GetConstraintOpts{WithDefaults("a", "b")},
		},
		{
			name:     "does_not_exist_with_defaults_empty_string",
			expected: []string{""},
			opts:     []GetConstraintOpts{WithDefaults("")},
		},

		{
			name:     "empty_array",
			expected: []string{"a", "b"},
			opts:     []GetConstraintOpts{WithDefaults("a", "b")},
		},
		{
			name:     "blank",
			expected: []string{""},
			opts:     []GetConstraintOpts{WithDefaults("a", "b")},
		},
		{
			name:     "single",
			expected: []string{"a"},
			opts:     []GetConstraintOpts{WithDefaults("a", "b")},
		},

		// default plus allowed characters
		{

			name:     "double",
			expected: []string{"a"},
			opts:     []GetConstraintOpts{WithDefaults("a", "b"), WithAllowedCharacters("a")},
		},
		{
			// allowed zeros everything, no default is returned.
			name:     "double",
			expected: []string{},
			opts:     []GetConstraintOpts{WithDefaults("a", "b"), WithAllowedCharacters("z")},
		},
		{
			// no matches, so defaults applies, even if it doesn't match allowed
			name:     "does_not_exist_with_defaults",
			expected: []string{"a", "b"},
			opts:     []GetConstraintOpts{WithDefaults("a", "b"), WithAllowedCharacters("z")},
		},

		// allowed values
		{

			name:     "double",
			expected: []string{"a"},
			opts:     []GetConstraintOpts{WithAllowedValues([]string{"a"})},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := GetConstraints(mockQC, tt.name, tt.opts...)
			sort.Strings(actual)
			require.Equal(t, tt.expected, actual)
		})
	}

}
