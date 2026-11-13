package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMEGAsyncVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "4-part with build number", version: "6.7.0.3", want: "6.7.0"},
		{name: "4-part with zero build", version: "6.5.1.0", want: "6.5.1"},
		{name: "3-part version", version: "6.2.2", want: "6.2.2"},
		{name: "2-part version", version: "6.2", wantErr: "3 or 4 parts"},
		{name: "5-part version", version: "6.7.0.3.1", wantErr: "3 or 4 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "MEGAsync",
				Slug:             "megasync/windows",
				Version:          tc.version,
			}
			result, err := MEGAsyncVersionShortener(app)
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
