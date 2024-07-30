package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLooseEmail(t *testing.T) {
	testCases := []struct {
		str   string
		match bool
	}{
		{"foo", false},
		{"", false},
		{"foo@example", false},
		{"foo@example.com", true},
		{"foo+bar@example.com", true},
		{"foo.bar@example.com", true},
		{"foo.bar@baz.example.com", true},
	}

	for _, tc := range testCases {
		t.Run(tc.str, func(t *testing.T) {
			assert.Equal(t, tc.match, IsLooseEmail(tc.str))
		})
	}
}
