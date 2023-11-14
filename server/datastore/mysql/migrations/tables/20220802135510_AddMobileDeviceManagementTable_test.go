package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220802135510(t *testing.T) {
	// skipping old migration tests as migrations don't change and we're getting
	// timeouts in CI
	t.Skip("old migration test, not longer required to run")
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
INSERT INTO mobile_device_management_solutions (
	name, server_url
)
VALUES (?, ?)
`
	res, err := db.Exec(query, "test", "http://localhost:8080")
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	var (
		name string
		url  string
	)
	err = db.QueryRow(`SELECT name, server_url FROM mobile_device_management_solutions WHERE id = ?`, id).
		Scan(&name, &url)
	require.NoError(t, err)
	require.Equal(t, "test", name)
	require.Equal(t, "http://localhost:8080", url)
}
