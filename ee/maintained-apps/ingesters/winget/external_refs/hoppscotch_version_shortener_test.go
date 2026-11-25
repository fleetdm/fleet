package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoppscotchVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "zero revision", version: "26.11.0-0", want: "26.11.0"},
		{name: "non-zero revision", version: "26.10.1-2", want: "26.10.1"},
		{name: "no revision", version: "26.11.0", want: "26.11.0"},
		{name: "two parts", version: "26.11-0", wantErr: "expected Hoppscotch version"},
		{name: "non-numeric suffix", version: "26.11.0-beta", wantErr: "expected Hoppscotch version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Hoppscotch",
				Slug:             "hoppscotch/windows",
				Version:          tc.version,
			}
			result, err := HoppscotchVersionShortener(app)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Equal(t, tc.version, result.Version)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, result.Version)
		})
	}
}
