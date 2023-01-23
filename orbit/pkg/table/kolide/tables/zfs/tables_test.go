package zfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColumns(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		infile string
		len    int
	}{
		{
			infile: "zfs.txt",
			len:    438,
		},
		{
			infile: "zpool.txt",
			len:    120,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.infile, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(filepath.Join("testdata", tt.infile))
			require.NoError(t, err, "read input file")

			rows, err := parseColumns(input)
			require.NoError(t, err, "parse columns")
			assert.Equal(t, tt.len, len(rows), "expected number of rows")

		})
	}
}
