package file

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPEMetadata(t *testing.T) {
	t.Parallel()

	tfr, err := fleet.NewKeepFileReader("testdata/software-installers/hello-world-installer.exe")
	require.NoError(t, err)
	defer tfr.Close()

	meta, err := ExtractPEMetadata(tfr)
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "Hello world", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, []string{"Hello world"}, meta.PackageIDs)
}

func TestApplyPESpecialCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, version string
		wantName      string
	}{
		{"Notion", "3.11.1", "Notion 3.11.1"},
		{"Notion", "", "Notion"},
		{"Pritunl", "1.4.4791.77", "Pritunl Client"},
		{"Pritunl", "", "Pritunl Client"},
		{"Pritunl Client", "1.4.4791.77", "Pritunl Client"},
		{"Hello world", "1.0.0", "Hello world"},
	}
	for _, c := range cases {
		t.Run(c.name+"@"+c.version, func(t *testing.T) {
			t.Parallel()
			meta := applySpecialCases(&InstallerMetadata{
				Name:       c.name,
				Version:    c.version,
				PackageIDs: []string{c.name},
			}, nil)
			require.Equal(t, c.wantName, meta.Name)
			require.Equal(t, c.version, meta.Version)
			// Package IDs feed $PACKAGE_ID in install/uninstall scripts and keep the installer's own name.
			require.Equal(t, []string{c.name}, meta.PackageIDs)
		})
	}
}
