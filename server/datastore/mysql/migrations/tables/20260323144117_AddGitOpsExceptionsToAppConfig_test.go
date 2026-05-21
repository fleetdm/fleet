package tables

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20260323144117(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply the migration
	applyNext(t, db)

	// Verify exceptions were set correctly for existing instance
	var rawAppConfig []byte
	err := db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&rawAppConfig)
	require.NoError(t, err)

	var config fleet.AppConfig
	err = json.Unmarshal(rawAppConfig, &config)
	require.NoError(t, err)

	// Existing instances should have labels and secrets excepted (preserving current behavior)
	require.True(t, config.GitOpsConfig.Exceptions.Labels)
	require.True(t, config.GitOpsConfig.Exceptions.Secrets)
	require.False(t, config.GitOpsConfig.Exceptions.Software)
}
