package dataflattentable

import (
	"os"
	"testing"

	"github.com/kolide/launcher/pkg/osquery/tables/execparsers/dsregcmd"
	"github.com/stretchr/testify/require"
)

// TestFlattenerFromParser tests flattening. This really shouldn't be needed -- the parsers are tested, and Flatten is
// pretty tested as well. But, it's here to make sure that if we swap between `dataflatten.Flatten` and
// `dataflatten.Json` we make sure to handle some of the unknown types.
func TestFlattenerFromParser(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name         string
		input        []byte
		parser       parser
		expectedRows int
	}{
		{
			name:         "dsreg",
			input:        readTestFile(t, "../execparsers/dsregcmd/test-data/not_configured.txt"),
			parser:       dsregcmd.Parser,
			expectedRows: 25,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flattener := flattenerFromParser(tt.parser)
			rows, err := flattener.FlattenBytes(tt.input)
			require.NoError(t, err)
			require.Len(t, rows, tt.expectedRows)
		})
	}

}

func readTestFile(t *testing.T, filepath string) []byte {
	b, err := os.ReadFile(filepath)
	require.NoError(t, err)
	return b
}
