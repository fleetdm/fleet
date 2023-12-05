package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221101103952(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO host_disks (host_id, gigs_disk_space_available, percent_disk_space_available) VALUES (1, 35, 70.5)`)
	require.NoError(t, err)

	applyNext(t, db)

	var gigs, percent float64
	var encrypted *bool
	err = db.QueryRow(`SELECT gigs_disk_space_available, percent_disk_space_available, encrypted FROM host_disks WHERE host_id = ?`, 1).Scan(&gigs, &percent, &encrypted)
	require.NoError(t, err)
	require.Equal(t, 35.0, gigs)
	require.Equal(t, 70.5, percent)
	require.Nil(t, encrypted)

	// create a new row with encrypted set
	_, err = db.Exec(`INSERT INTO host_disks (host_id, encrypted) VALUES (2, 1)`)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT gigs_disk_space_available, percent_disk_space_available, encrypted FROM host_disks WHERE host_id = ?`, 2).Scan(&gigs, &percent, &encrypted)
	require.NoError(t, err)
	require.Equal(t, 0.0, gigs)
	require.Equal(t, 0.0, percent)
	require.NotNil(t, encrypted)
	require.True(t, *encrypted)
}
