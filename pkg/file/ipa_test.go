package file_test

import (
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestExtractIPAMetadata(t *testing.T) {
	// a valid ipa returns metadata without error
	tfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "software-installers", "ipa_test.ipa"))
	require.NoError(t, err)
	defer tfr.Close()

	meta, err := file.ExtractIPAMetadata(tfr)
	require.NoError(t, err)
	require.NotNil(t, meta)

	// a zip-based package without a Payload/ directory (an .msix here) is not an ipa
	msixTfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "software-installers", "msix_test.msix"))
	require.NoError(t, err)
	defer msixTfr.Close()

	meta, err = file.ExtractIPAMetadata(msixTfr)
	require.ErrorIs(t, err, file.ErrInvalidType)
	require.Nil(t, meta)

	// the same msix renamed to a .msi extension still returns invalid type
	obfuscatedPath := filepath.Join(t.TempDir(), "not-really-an.msi")
	require.NoError(t, file.Copy(filepath.Join("testdata", "software-installers", "msix_test.msix"), obfuscatedPath, 0o644))

	obfuscatedTfr, err := fleet.NewKeepFileReader(obfuscatedPath)
	require.NoError(t, err)
	defer obfuscatedTfr.Close()

	meta, err = file.ExtractIPAMetadata(obfuscatedTfr)
	require.ErrorIs(t, err, file.ErrInvalidType)
	require.Nil(t, meta)
}
