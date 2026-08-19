package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260529120000(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// app_config_json is a singleton (id PRIMARY KEY DEFAULT '1'); target it explicitly so the assertion can't pick a
	// stray row if one is ever introduced.
	var raw json.RawMessage
	require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value FROM app_config_json WHERE id = 1;`))

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(raw, &cfg))

	mdm, ok := cfg["mdm"].(map[string]any)
	require.True(t, ok, "mdm object should be present in app config")

	clientIDs, ok := mdm["windows_entra_client_ids"].([]any)
	require.True(t, ok, "windows_entra_client_ids should be initialized to an empty array")
	require.Empty(t, clientIDs)
}
