package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261123220227(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	execNoErr(t, db, `INSERT INTO host_android_agent_vitals (host_uuid, reported_at) VALUES ('host-without-roaming', NOW(6))`)
	execNoErr(t, db, `INSERT INTO host_android_agent_vitals (host_uuid, roaming, reported_at) VALUES ('roaming-host', 1, NOW(6))`)

	var row struct {
		Roaming    sql.NullBool `db:"roaming"`
		ReportedAt time.Time    `db:"reported_at"`
	}
	require.NoError(t, db.Get(&row, `SELECT roaming, reported_at FROM host_android_agent_vitals WHERE host_uuid = 'host-without-roaming'`))
	require.False(t, row.Roaming.Valid, "a vital the agent didn't report is NULL")
	require.False(t, row.ReportedAt.IsZero())

	require.NoError(t, db.Get(&row, `SELECT roaming, reported_at FROM host_android_agent_vitals WHERE host_uuid = 'roaming-host'`))
	require.True(t, row.Roaming.Valid)
	require.True(t, row.Roaming.Bool)

	// Retrying after a partial run is a no-op.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261123220227(tx))
	require.NoError(t, tx.Commit())
}
