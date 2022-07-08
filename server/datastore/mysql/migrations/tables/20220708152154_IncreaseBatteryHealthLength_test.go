package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220708152154(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
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

	_, err := db.Exec(query, 1, "abc", 2, "Check Battery")
	require.NoError(t, err)

	_, err = db.Exec(query, 2, "def", 3, "Good")
	require.NoError(t, err)

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM host_batteries`)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
