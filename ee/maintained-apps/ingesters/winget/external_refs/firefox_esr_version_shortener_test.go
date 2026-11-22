package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirefoxESRVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "first release of a major", version: "153.0.0", want: "153.0"},
		{name: "minor release keeps zero patch", version: "153.1.0", want: "153.1.0"},
		{name: "patch release", version: "153.0.1", want: "153.0.1"},
		{name: "2-part version", version: "102.0", want: "102.0"},
		{name: "4-part version", version: "153.0.0.0", wantErr: "2 or 3 parts"},
		{name: "single part", version: "153", wantErr: "2 or 3 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Mozilla Firefox ESR",
				Slug:             "firefox@esr/windows",
				Version:          tc.version,
			}
			result, err := FirefoxESRVersionShortener(app)
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
