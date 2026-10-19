package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261019171901(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.server_settings.ai_features_disabled', true) WHERE id = 1`)

	applyNext(t, db)

	var mcpEnabled, otherSetting string
	require.NoError(t, sqlx.Get(db, &mcpEnabled, `SELECT JSON_EXTRACT(json_value, '$.server_settings.mcp_enabled') FROM app_config_json WHERE id = 1`))
	require.Equal(t, "true", mcpEnabled)
	require.NoError(t, sqlx.Get(db, &otherSetting, `SELECT JSON_EXTRACT(json_value, '$.server_settings.ai_features_disabled') FROM app_config_json WHERE id = 1`))
	require.Equal(t, "true", otherSetting, "sibling server settings are preserved")
}
