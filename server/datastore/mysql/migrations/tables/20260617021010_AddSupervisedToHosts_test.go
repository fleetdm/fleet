package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// 20260608070039_AddSupervisedToHosts_test.go
func TestUp_20260617021010(t *testing.T) {
	db := applyUpToPrev(t)

	// applies current migration
	applyNext(t, db)

	// column should now exist and default to NULL

	// Insert a host and verify supervised is NULL by default.
	var supervised sql.NullBool
	_, err := db.Exec(`INSERT INTO hosts (osquery_host_id, node_key) VALUES (?, ?)`,
		"test-osquery-id", "test-node-key")
	require.NoError(t, err)
	require.NoError(t, db.Get(&supervised, `SELECT supervised FROM hosts WHERE osquery_host_id = ?`, "test-osquery-id"))
	require.False(t, supervised.Valid)
}
