package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20230918132351(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
		INSERT INTO software_cve (
			software_id,
			source,
			cve
		)
		VALUES (?, ?, ?)
	`
	args := []interface{}{
		1,
		0,
		"CVE-2021-1234",
	}

	execNoErr(t, db, insertStmt, args...)

	// Apply current migration.
	applyNext(t, db)

	// check for null resolved_in_version default
	selectAndAssert(t, db, 1, 0, "CVE-2021-1234", nil)

	// Update the resolved_in_version and verify
	updateVersion := "6.0.2-76060002.202210150739~1666289067~22.04~fe0ce53" // long string to test the capacity of the new column
	updateStmt := `
		UPDATE software_cve
		SET resolved_in_version = ?
		WHERE software_id = ? AND source = ? AND cve = ?
		`

	execNoErr(t, db, updateStmt, updateVersion, 1, 0, "CVE-2021-1234")
	selectAndAssert(t, db, 1, 0, "CVE-2021-1234", &updateVersion)

	// Insert a new record and verify
	insertStmt = `
		INSERT INTO software_cve (
			software_id,
			source,
			cve,
			resolved_in_version
		)
		VALUES (?, ?, ?, ?)
	`

	execNoErr(t, db, insertStmt, 1, 0, "CVE-2021-1235", updateVersion)
	selectAndAssert(t, db, 1, 0, "CVE-2021-1235", &updateVersion)
}

func selectAndAssert(t *testing.T, db *sqlx.DB, softwareID uint, source uint, cve string, resolvedInVersion *string) {
	var softwareCVE struct {
		SoftwareID        uint    `db:"software_id"`
		Source            uint    `db:"source"`
		CVE               string  `db:"cve"`
		ResolvedInVersion *string `db:"resolved_in_version"`
	}

	selectStmt := `
		SELECT software_id, source, cve, resolved_in_version
		FROM software_cve
		WHERE software_id = ? AND source = ? AND cve = ?
	`

	require.NoError(t, db.Get(&softwareCVE, selectStmt, softwareID, source, cve))
	require.Equal(t, softwareID, softwareCVE.SoftwareID)
	require.Equal(t, source, softwareCVE.Source)
	require.Equal(t, cve, softwareCVE.CVE)
	require.Equal(t, resolvedInVersion, softwareCVE.ResolvedInVersion)
}
