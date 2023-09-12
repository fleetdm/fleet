package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230912125945(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a record without the new columns before applying the migration
	insertStmt := `
		INSERT INTO software_cve (
			cve,
			source,
			software_id
		)
		VALUES
			(?, ?, ?)
	`

	args := []interface{}{
		"test-cve",
		0,
		1, // Assuming a valid software_id exists with this ID
	}
	execNoErr(t, db, insertStmt, args...)

	applyNext(t, db)

	// Insert a new record with the new columns after applying the migration
	insertStmt = `
		INSERT INTO software_cve (
			cve, 
			source, 
			software_id, 
			versionStartIncluding, 
			versionEndExcluding
		)
		VALUES
			(?, ?, ?, ?, ?)
	`

	args = []interface{}{
		"test-cve-2",
		0,
		1,
		"1.0",
		"2.0",
	}
	execNoErr(t, db, insertStmt, args...)

	// retrieve the stored value for the new record
	var scve struct {
		ID                    uint   `db:"id"`
		CVE                   string `db:"cve"`
		Source                int    `db:"source"`
		SoftwareID            uint   `db:"software_id"`
		VersionStartIncluding string `db:"versionStartIncluding"`
		VersionEndExcluding   string `db:"versionEndExcluding"`
	}

	selectStmt := "SELECT * FROM software_cve WHERE cve = ?"
	require.NoError(t, db.Get(&scve, selectStmt, "test-cve-2"))
	require.Equal(t, "test-cve-2", scve.CVE)
	require.Equal(t, 0, scve.Source)
	require.Equal(t, uint(1), scve.SoftwareID)
	require.Equal(t, "1.0", scve.VersionStartIncluding)
	require.Equal(t, "2.0", scve.VersionEndExcluding)
}
