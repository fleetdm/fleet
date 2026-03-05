package integrationtest

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/gdmf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetAssetMetadata checks that we can fetch OS information from Apple.
func TestGetAssetMetadata(t *testing.T) {
	resp, err := gdmf.GetAssetMetadata()
	require.NoError(t, err)
	assert.Greater(t, len(resp.AssetSets.MacOS), 0)
}
