package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestGoogleDriveVersionShortener(t *testing.T) {
	tcs := []struct {
		Name     string
		Version  string
		Expected string
	}{
		{
			Name:     "no subversion",
			Version:  "123",
			Expected: "123",
		},
		{
			Name:     "one subversion",
			Version:  "123.45",
			Expected: "123.45",
		},
		{
			Name:     "two subversion",
			Version:  "123.45.67",
			Expected: "123.45",
		},
		{
			Name:     "three subversions",
			Version:  "123.45.67.89",
			Expected: "123.45",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			fma := &maintained_apps.FMAManifestApp{
				Version: tc.Version,
			}
			fma2, err := GoogleDriveVersionShortener(fma)
			assert.Equal(t, tc.Expected, fma.Version)
			assert.Equal(t, tc.Expected, fma2.Version)
			assert.NoError(t, err)
		})
	}
}
