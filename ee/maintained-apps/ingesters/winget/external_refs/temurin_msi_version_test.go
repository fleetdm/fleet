package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemurinMSIVersion(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "regular release", version: "17.0.21.9", want: "17.0.21.9"},
		{name: "regular release two-digit build", version: "11.0.31.11", want: "11.0.31.11"},
		{name: "point release", version: "17.0.21.102", want: "17.0.21.10002"},
		{name: "point release jdk 21", version: "21.0.13.102", want: "21.0.13.10002"},
		{name: "point release build 1", version: "11.0.32.101", want: "11.0.32.10001"},
		{name: "second point release", version: "21.0.13.212", want: "21.0.13.20012"},
		{name: "three parts", version: "17.0.21", wantErr: "4 parts"},
		{name: "non-numeric", version: "17.0.21.x", wantErr: "numeric"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Eclipse Temurin JDK with Hotspot",
				Slug:             "eclipse-temurin-jdk-17/windows",
				Version:          tc.version,
			}
			result, err := TemurinMSIVersion(app)
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
