package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAOMEIBackupperVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "4-part version", version: "8.5.0.0", want: "8.5.0"},
		{name: "major bump", version: "9.0.0.0", want: "9.0.0"},
		{name: "3-part version", version: "7.3.1", want: "7.3.1"},
		{name: "non-zero fourth part", version: "8.5.0.1", wantErr: "end in '.0'"},
		{name: "2-part version", version: "8.5", wantErr: "3 or 4 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "AOMEI Backupper",
				Slug:             "aomei-backupper-standard/windows",
				Version:          tc.version,
			}
			result, err := AOMEIBackupperVersionShortener(app)
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
