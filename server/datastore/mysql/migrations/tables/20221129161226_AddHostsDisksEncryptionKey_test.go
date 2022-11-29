package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221129161226(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
    INSERT INTO host_disks
      (host_id, gigs_disk_space_available, percent_disk_space_available, encrypted)
    VALUES
      (1, 35, 70.5, 1)
  `)
	require.NoError(t, err)

	applyNext(t, db)

	var query = `
    SELECT gigs_disk_space_available, percent_disk_space_available, encrypted, encryption_key
    FROM host_disks
    WHERE host_id = ?
  `
	var gigs, percent float64
	var encrypted sql.NullBool
	var key sql.NullString
	err = db.QueryRow(query, 1).Scan(&gigs, &percent, &encrypted, &key)
	require.NoError(t, err)
	require.Equal(t, 35.0, gigs)
	require.Equal(t, 70.5, percent)
	require.True(t, encrypted.Bool)
	require.False(t, key.Valid)

	// create a new row with an encryption key set
	_, err = db.Exec(`INSERT INTO host_disks (host_id, encryption_key) VALUES (2, 'AAA-BBB-CCC')`)
	require.NoError(t, err)

	err = db.QueryRow(query, 2).Scan(&gigs, &percent, &encrypted, &key)
	require.NoError(t, err)
	require.Equal(t, 0.0, gigs)
	require.Equal(t, 0.0, percent)
	require.False(t, encrypted.Bool)
	require.True(t, key.Valid)
	require.Equal(t, "AAA-BBB-CCC", key.String)
}
