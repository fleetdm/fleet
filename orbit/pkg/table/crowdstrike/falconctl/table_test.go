// based on github.com/kolide/launcher/pkg/osquery/tables
package falconctl

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestOptionRestrictions tests that the table only allows the options we expect.
func TestOptionRestrictions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		options           []string
		expectedExecs     int
		expectedDisallows int
	}{
		{
			name:              "default",
			expectedExecs:     1,
			expectedDisallows: 0,
		},
		{
			name:              "allowed options as array",
			options:           []string{"--aid", "--aph"},
			expectedExecs:     2,
			expectedDisallows: 0,
		},
		{
			name:              "allowed options as string",
			options:           []string{"--aid --aph"},
			expectedExecs:     1,
			expectedDisallows: 0,
		},
		{
			name:              "disallowed option as array",
			options:           []string{"--not-allowed", "--definitely-not-allowed", "--aid", "--aph"},
			expectedExecs:     2,
			expectedDisallows: 2,
		},
		{
			name:              "disallowed option as string",
			options:           []string{"--aid --aph --not-allowed"},
			expectedExecs:     0,
			expectedDisallows: 1,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logBytes bytes.Buffer

			testTable := &falconctlOptionsTable{
				logger:   zerolog.New(zerolog.ConsoleWriter{Out: &logBytes}),
				execFunc: noopExec,
			}

			mockQC := tablehelpers.MockQueryContext(map[string][]string{
				"options": tt.options,
			})

			_, err := testTable.generate(context.TODO(), mockQC)
			require.NoError(t, err)

			// test the number of times exec was called
			require.Equal(t, tt.expectedExecs, strings.Count(logBytes.String(), "exec-in-test"))

			// test the number of times we disallowed an option
			require.Equal(t, tt.expectedDisallows, strings.Count(logBytes.String(), "requested option not allowed"))
		})
	}
}

func noopExec(_ context.Context, log zerolog.Logger, _ int, _ []string, args []string, _ bool) ([]byte, error) {
	log.Info().Str("args", strings.Join(args, " ")).Msg("exec-in-test")
	return []byte{}, nil
}
