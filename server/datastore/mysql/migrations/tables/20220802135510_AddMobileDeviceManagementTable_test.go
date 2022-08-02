package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220802135510(t *testing.T) {
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
