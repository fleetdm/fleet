package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramVersionShortener(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
		wantErr string
	}{
		{name: "zero patch part", version: "7.3.0", want: "7.3"},
		{name: "major release", version: "8.0.0", want: "8.0"},
		{name: "non-zero patch part", version: "7.3.3", want: "7.3.3"},
		{name: "already major.minor", version: "7.3", want: "7.3"},
		{name: "4-part version", version: "7.3.1.1", wantErr: "2 or 3 parts"},
		{name: "single part", version: "7", wantErr: "2 or 3 parts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				UniqueIdentifier: "Telegram Desktop",
				Slug:             "telegram/windows",
				Version:          tc.version,
			}
			result, err := TelegramVersionShortener(app)
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
