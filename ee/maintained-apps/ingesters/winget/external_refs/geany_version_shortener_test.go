package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeanyVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "zero patch tag", version: "2.1.0", want: "2.1"},
		{name: "major bump keeps .0 minor", version: "3.0.0", want: "3.0"},
		{name: "non-zero patch kept", version: "1.37.1", want: "1.37.1"},
		{name: "already major.minor", version: "1.38", want: "1.38"},
		{name: "single part", version: "2", wantErr: "2 or 3 parts"},
		{name: "four parts", version: "2.1.0.0", wantErr: "2 or 3 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Geany",
				Slug:             "geany/windows",
				Version:          tc.version,
			}
			result, err := GeanyVersionShortener(app)
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
