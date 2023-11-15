package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220627104817(t *testing.T) {
	// skipping old migration tests as migrations don't change and we're getting
	// timeouts in CI
	t.Skip("old migration test, not longer required to run")
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
INSERT INTO host_batteries (
    host_id,
    serial_number,
    cycle_count,
    health
)
VALUES (?, ?, ?, ?)
`
	_, err := db.Exec(query, 1, "abc", 2, "Good")
	require.NoError(t, err)

	var (
		hostID       uint
		serialNumber string
		cycleCount   int
		health       string
	)
	err = db.QueryRow(`SELECT host_id, serial_number, cycle_count, health FROM host_batteries WHERE host_id = ?`, 1).
		Scan(&hostID, &serialNumber, &cycleCount, &health)
	require.NoError(t, err)
	require.Equal(t, uint(1), hostID)
	require.Equal(t, "abc", serialNumber)
	require.Equal(t, 2, cycleCount)
	require.Equal(t, "Good", health)
}
