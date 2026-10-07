package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUp_20261007130000 checks global backfill, timestamp preservation, and idempotent cloud migration.
func TestUp_20261007130000(t *testing.T) {
	db := applyUpToPrev(t)
	_, err := db.Exec(`INSERT INTO mdm_microsoft_graph_credentials
		(tenant_id, client_id, client_secret, updated_at)
		VALUES ('tenant', 'client', 'encrypted-secret', '2026-10-01 12:00:00.000000')`)
	require.NoError(t, err)
	applyNext(t, db)

	var row struct {
		Cloud     string    `db:"cloud"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	require.NoError(t, db.Get(&row, `SELECT cloud, updated_at FROM mdm_microsoft_graph_credentials WHERE tenant_id = 'tenant'`))
	assert.Equal(t, "global", row.Cloud)
	assert.Equal(t, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC), row.UpdatedAt)

	// Retrying after a partial migration must preserve the selected cloud.
	_, err = db.Exec(`UPDATE mdm_microsoft_graph_credentials SET cloud = 'gcc_high' WHERE tenant_id = 'tenant'`)
	require.NoError(t, err)
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261007130000(tx))
	require.NoError(t, tx.Commit())
	var cloud string
	require.NoError(t, db.Get(&cloud, `SELECT cloud FROM mdm_microsoft_graph_credentials WHERE tenant_id = 'tenant'`))
	assert.Equal(t, "gcc_high", cloud)
}
