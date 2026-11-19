package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatadogAgentVersionNormalizer(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "package release 1", version: "7.85.2.1", want: "7.85.2.0"},
		{name: "package release 2", version: "7.85.2.2", want: "7.85.2.0"},
		{name: "already normalized", version: "7.85.2.0", want: "7.85.2.0"},
		{name: "3-part version", version: "7.86.0", want: "7.86.0.0"},
		{name: "2-part version", version: "7.86", wantErr: "3 or 4 parts"},
		{name: "5-part version", version: "7.86.0.1.1", wantErr: "3 or 4 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Datadog Agent",
				Slug:             "datadog-agent/windows",
				Version:          tc.version,
			}
			result, err := DatadogAgentVersionNormalizer(app)
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
