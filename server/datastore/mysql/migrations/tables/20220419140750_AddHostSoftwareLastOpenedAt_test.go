package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20220419140750(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO host_software (host_id, software_id) VALUES (1, 1)`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	now := time.Now()
	_, err = db.Exec(`INSERT INTO host_software (host_id, software_id, last_opened_at) VALUES (2, 2, ?)`, now)
	require.NoError(t, err)

	var lastOpened sql.NullTime

	row := db.QueryRow(`SELECT last_opened_at FROM host_software WHERE host_id = 1`)
	err = row.Scan(&lastOpened)
	require.NoError(t, err)
	require.False(t, lastOpened.Valid)

	row = db.QueryRow(`SELECT last_opened_at FROM host_software WHERE host_id = 2`)
	err = row.Scan(&lastOpened)
	require.NoError(t, err)
	require.True(t, lastOpened.Valid)
	require.WithinDuration(t, lastOpened.Time, now, time.Second)
}
