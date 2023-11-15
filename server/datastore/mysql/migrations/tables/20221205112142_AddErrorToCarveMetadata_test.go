package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221205112142(t *testing.T) {
	db := applyUpToPrev(t)
	query := `
INSERT INTO carve_metadata
  (host_id, block_count, block_size, carve_size, carve_id, request_id, session_id)
VALUES
  (1, 10, 1000, 10000, "carve_id", "request_id", ?)
`

	execNoErr(t, db, "INSERT INTO hosts (hostname, osquery_host_id) VALUES ('foo.example.com', 'foo')")
	execNoErr(t, db, query, 1)
	execNoErr(t, db, query, 2)

	// Apply current migration.
	applyNext(t, db)

	// Okay if we don't provide an error
	execNoErr(t, db, query, 3)
	// Insert with an error
	execNoErr(t, db, `
INSERT INTO carve_metadata
  (host_id, block_count, block_size, carve_size, carve_id, request_id, session_id, error)
VALUES
  (1, 10, 1000, 10000, "carve_id", "request_id", 4, "made_up_error")
`)
	// Update an existing row to add an error
	execNoErr(t, db, `UPDATE carve_metadata SET error = "updated_error" WHERE session_id = 3`)

	var storedErr sql.NullString
	row := db.QueryRow(`SELECT error FROM carve_metadata WHERE session_id = 3`)
	err := row.Scan(&storedErr)
	require.NoError(t, err)
	require.Equal(t, "updated_error", storedErr.String)

	row = db.QueryRow(`SELECT error FROM carve_metadata WHERE session_id = 4`)
	err = row.Scan(&storedErr)
	require.NoError(t, err)
	require.Equal(t, "made_up_error", storedErr.String)

	row = db.QueryRow(`SELECT error FROM carve_metadata WHERE session_id = 1`)
	err = row.Scan(&storedErr)
	require.NoError(t, err)
	require.Equal(t, "", storedErr.String)
}
