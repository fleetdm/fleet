package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221109100749(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO hosts (hostname, osquery_host_id, gigs_disk_space_available, percent_disk_space_available) VALUES ('h1', 'ohid', 35, 70.5)`)
	require.NoError(t, err)

	applyNext(t, db)

	var name string
	err = db.QueryRow(`SELECT hostname FROM hosts WHERE hostname = ?`, "h1").Scan(&name)
	require.NoError(t, err)
	require.Equal(t, "h1", name)
}
