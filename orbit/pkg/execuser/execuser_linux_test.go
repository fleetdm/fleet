package execuser

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWhoOutputForDisplay(t *testing.T) {
	testCases := []struct {
		name            string
		output          string
		user            string
		expectedDisplay string
	}{
		{
			"Ubuntu 22.04.2 (X11)",
			`foo      :0           2024-05-14 17:34 (:0)`,
			"foo",
			":0",
		},
		{
			"Ubuntu 22.04.2 (X11) - user not listed",
			`foo      :0           2024-05-14 17:34 (:0)`,
			"bar",
			"",
		},
		{
			"Ubuntu 24.04 (X11)",
			`foo      seat0        2024-05-14 17:42 (login screen)
foo      :1           2024-05-14 17:42 (:1)`,
			"foo",
			":1",
		},
		{
			"Ubuntu 24.04 (Wayland) - DISPLAY not found",
			`foo      seat0        2024-05-14 18:11 (login screen)
foo      tty2         2024-05-14 18:11 (tty2)`,
			"foo",
			"",
		},
		{
			"Empty",
			``,
			"foo",
			"",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			display, err := parseWhoOutputForDisplay(bytes.NewReader([]byte(tc.output)), tc.user)
			require.NoError(t, err)
			require.Equal(t, tc.expectedDisplay, display)
		})
	}
}
