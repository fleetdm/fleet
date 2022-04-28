package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220428140039(t *testing.T) {
	db := applyUpToPrev(t)

	const insStmt = `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`

	// insert a row with an empty email
	_, err := db.Exec(insStmt, 1, "", "google_chrome_profiles")
	require.NoError(t, err)

	// insert a row with an email
	_, err = db.Exec(insStmt, 1, "test@example.com", "google_chrome_profiles")
	require.NoError(t, err)

	var count int
	const countStmt = `SELECT COUNT(*) FROM host_emails`

	err = db.Get(&count, countStmt)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Apply current migration.
	applyNext(t, db)

	err = db.Get(&count, countStmt)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
