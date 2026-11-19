package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyBurnVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "4-part bot version", version: "6.9.0.0", want: "6.9"},
		{name: "3-part version", version: "6.9.0", want: "6.9"},
		{name: "already major.minor", version: "6.9", want: "6.9"},
		{name: "major bump keeps .0 minor", version: "7.0.0.0", want: "7.0"},
		{name: "non-zero third part", version: "6.9.1.0", wantErr: "only zero parts"},
		{name: "non-zero fourth part", version: "6.9.0.1", wantErr: "only zero parts"},
		{name: "single part", version: "6", wantErr: "at least 2 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "AnyBurn",
				Slug:             "anyburn/windows",
				Version:          tc.version,
			}
			result, err := AnyBurnVersionShortener(app)
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
