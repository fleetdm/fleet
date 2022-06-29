package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectMatchType(t *testing.T) {
	t.Run("NewObjectMatchType", func(t *testing.T) {
		cases := []struct {
			input    string
			expected ObjectMatchType
		}{
			{"all_exist", AllExist},
			{"any_exist", AnyExist},
			{"at_least_one_exists", AtLeastOneExists},
			{"none_exist", NoneExist},
			{"only_one_exists", OnlyOneExists},
			{"", AtLeastOneExists},
		}

		for _, c := range cases {
			require.Equal(t, c.expected, NewObjectMatchType(c.input))
		}
	})

	t.Run("#Eval", func(t *testing.T) {
		cases := []struct {
			op       ObjectMatchType
			total    int
			matches  int
			expected bool
		}{
			{AllExist, 1, 1, true},
			{AllExist, 5, 1, false},
			{AnyExist, 1, 0, true},
			{AnyExist, 1, 1, true},
			{AnyExist, 5, 1, true},
			{AtLeastOneExists, 1, 0, false},
			{AtLeastOneExists, 1, 1, true},
			{AtLeastOneExists, 5, 1, true},
			{NoneExist, 1, 1, false},
			{NoneExist, 5, 1, false},
			{NoneExist, 1, 0, true},
			{OnlyOneExists, 5, 1, true},
			{OnlyOneExists, 5, 0, false},
			{OnlyOneExists, 5, 5, false},
		}

		for _, c := range cases {
			require.Equal(t, c.expected, c.op.Eval(c.matches, c.total))
		}
	})
}
