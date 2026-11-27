package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeePassVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "zero patch part", version: "2.62.0", want: "2.62"},
		{name: "major release", version: "3.0.0", want: "3.0"},
		{name: "non-zero patch part", version: "2.61.1", want: "2.61.1"},
		{name: "already major.minor", version: "2.62", want: "2.62"},
		{name: "4-part version", version: "2.62.0.0", wantErr: "2 or 3 parts"},
		{name: "single part", version: "2", wantErr: "2 or 3 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "KeePass Password Safe",
				Slug:             "keepass/windows",
				Version:          tc.version,
			}
			result, err := KeePassVersionShortener(app)
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
