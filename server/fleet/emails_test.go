package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestEmailLocalPart(t *testing.T) {
	cases := map[string]string{
		"alice@example.com":       "alice",
		"alice.smith@example.com": "alice.smith",
		"jdoe":                    "jdoe",
		"":                        "",
	}
	for email, want := range cases {
		require.Equal(t, want, EmailLocalPart(email), email)
	}
}
