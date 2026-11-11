package file

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzParseDistributionFile(f *testing.F) {
	fixtures, err := filepath.Glob("testdata/distribution/*.xml")
	require.NoError(f, err)
	require.NotEmpty(f, fixtures)
	for _, fixture := range fixtures {
		data, err := os.ReadFile(fixture)
		require.NoError(f, err)
		f.Add(data)
	}
	f.Add([]byte{})
	f.Add([]byte(`<installer-gui-script/>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256<<10 {
			t.Skip("Distribution XML exceeds fuzz input budget")
		}
		meta, err := parseDistributionFile(data)
		if err != nil {
			return
		}
		require.True(t, slices.IsSorted(meta.PackageIDs))
		for i, id := range meta.PackageIDs {
			require.NotEmpty(t, id)
			if i > 0 {
				require.NotEqual(t, meta.PackageIDs[i-1], id)
			}
		}
		again, err := parseDistributionFile(data)
		require.NoError(t, err)
		require.Equal(t, meta, again, "metadata must not depend on map iteration order")
	})
}
