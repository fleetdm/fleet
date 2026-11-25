package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBandicamVersionedURL(t *testing.T) {
	const versioned = "https://dl.bandicam.com/old/BDCAMSETUP_27_0_6_2686.EXE"

	cases := []struct {
		name    string
		url     string
		version string
		want    string
		wantErr string
	}{
		{name: "rolling URL", url: bandicamRollingURL, version: "27.0.7.2687", want: "https://dl.bandicam.com/old/BDCAMSETUP_27_0_7_2687.EXE"},
		{name: "rolling URL, different case", url: "https://dl.bandicam.com/BDCAMSETUP.EXE", version: "27.0.7.2687", want: "https://dl.bandicam.com/old/BDCAMSETUP_27_0_7_2687.EXE"},
		{name: "already versioned", url: versioned, version: "27.0.6.2686", want: versioned},
		{name: "already versioned, odd version", url: versioned, version: "27.0.6", want: versioned},
		{name: "three parts", url: bandicamRollingURL, version: "27.0.7", wantErr: "4 parts"},
		{name: "non-numeric part", url: bandicamRollingURL, version: "27.0.7.2687-beta", wantErr: "numeric"},
		{name: "empty part", url: bandicamRollingURL, version: "27.0..2687", wantErr: "numeric"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Bandicam",
				Slug:             "bandicam/windows",
				Version:          tc.version,
				InstallerURL:     tc.url,
			}
			result, err := BandicamVersionedURL(app)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Equal(t, tc.url, result.InstallerURL)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, result.InstallerURL)
			assert.Equal(t, tc.version, result.Version)
		})
	}
}
