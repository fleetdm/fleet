package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20241125150614(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	var appCfg json.RawMessage
	err := sqlx.Get(db, &appCfg, `SELECT json_value FROM app_config_json LIMIT 1;`)
	require.NoError(t, err)

	var config map[string]interface{}
	err = json.Unmarshal(appCfg, &config)
	require.NoError(t, err)

	mdm, ok := config["mdm"]
	require.True(t, ok)
	mdmMap, ok := mdm.(map[string]interface{})
	require.True(t, ok)

	_, ok = mdmMap["windows_enabled_and_configured"].(bool)
	require.True(t, ok)
	require.False(t, mdmMap["windows_migration_enabled"].(bool))
}
