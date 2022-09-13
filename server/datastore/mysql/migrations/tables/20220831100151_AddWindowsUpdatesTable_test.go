package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220831100151(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	stmt := `INSERT INTO windows_updates (host_id, date_epoch, kb_id) VALUES (?, ?, ?)`

	_, err := db.Exec(stmt, 1, 1, 123)
	require.NoError(t, err)

	// This should raise an error
	_, err = db.Exec(stmt, 1, 1, 123)
	require.Error(t, err)

	// Test windows_updates has no duplicates
	var n uint
	err = db.QueryRow(`SELECT COUNT(1) FROM windows_updates WHERE host_id=1`).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)
}
