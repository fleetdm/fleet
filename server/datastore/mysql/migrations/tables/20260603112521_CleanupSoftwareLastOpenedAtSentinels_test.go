package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260603112521(t *testing.T) {
	db := applyUpToPrev(t)

	// Sentinel value: 1980-01-01 UTC (315532800), an app that was never opened.
	_, err := db.Exec(`INSERT INTO host_software (host_id, software_id, last_opened_at) VALUES (1, 1, '1980-01-01 00:00:00')`)
	require.NoError(t, err)
	// Boundary: exactly 2001-01-01, kept as a valid date.
	_, err = db.Exec(`INSERT INTO host_software (host_id, software_id, last_opened_at) VALUES (1, 2, '2001-01-01 00:00:00')`)
	require.NoError(t, err)
	// A real, recent timestamp that must be preserved.
	_, err = db.Exec(`INSERT INTO host_software (host_id, software_id, last_opened_at) VALUES (1, 3, '2020-11-22 03:36:49')`)
	require.NoError(t, err)
	// Already NULL (never reported), must stay NULL.
	_, err = db.Exec(`INSERT INTO host_software (host_id, software_id, last_opened_at) VALUES (1, 4, NULL)`)
	require.NoError(t, err)

	applyNext(t, db)

	lastOpenedAt := func(softwareID int) sql.NullString {
		var v sql.NullString
		require.NoError(t, db.QueryRow(`SELECT last_opened_at FROM host_software WHERE host_id = 1 AND software_id = ?`, softwareID).Scan(&v))
		return v
	}

	// Sentinel value cleared to NULL.
	require.False(t, lastOpenedAt(1).Valid)
	// Boundary and real values preserved.
	require.True(t, lastOpenedAt(2).Valid)
	require.True(t, lastOpenedAt(3).Valid)
	require.Contains(t, lastOpenedAt(3).String, "2020-11-22")
	// NULL stays NULL.
	require.False(t, lastOpenedAt(4).Valid)
}
