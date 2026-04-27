package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260427134220_FreshInstall(t *testing.T) {
	db := applyUpToPrev(t)

	// No users inserted: this represents a fresh installation.

	applyNext(t, db)

	var raw json.RawMessage
	require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value FROM app_config_json LIMIT 1;`))

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(raw, &cfg))

	aes, ok := cfg["activity_expiry_settings"].(map[string]any)
	require.True(t, ok)
	v, ok := aes["preserve_host_activities_on_reenrollment"].(bool)
	require.True(t, ok)
	require.False(t, v)
}

func TestUp_20260427134220_UpgradedInstall(t *testing.T) {
	db := applyUpToPrev(t)

	// At least one user exists: this represents an upgrade.
	execNoErr(t, db,
		`INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?);`,
		"admin", "admin@example.com", "p", "s",
	)

	applyNext(t, db)

	var raw json.RawMessage
	require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value FROM app_config_json LIMIT 1;`))

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(raw, &cfg))

	aes, ok := cfg["activity_expiry_settings"].(map[string]any)
	require.True(t, ok)
	v, ok := aes["preserve_host_activities_on_reenrollment"].(bool)
	require.True(t, ok)
	require.True(t, v)
}
