package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230913141311(t *testing.T) {
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

	var softwareCVE struct {
		SoftwareID        uint   `db:"software_id"`
		Source            uint   `db:"source"`
		CVE               string `db:"cve"`
		ResolvedInVersion string `db:"resolved_in_version"`
	}

	insertStmt = `
		INSERT INTO software_cve (
			software_id,
			source,
			cve,
			resolved_in_version
		)
		VALUES (?, ?, ?, ?)
	`
	args = []interface{}{
		1,
		0,
		"CVE-2021-1235",
		"6.0.2-76060002.202210150739~1666289067~22.04~fe0ce53", // This is a long linux kernel version string to test the capacity of the new column
	}

	execNoErr(t, db, insertStmt, args...)

	selectStmt := `
		SELECT software_id, source, cve, resolved_in_version
		FROM software_cve
		WHERE software_id = ? AND source = ? AND cve = ?
	`

	require.NoError(t, db.Get(&softwareCVE, selectStmt, 1, 0, "CVE-2021-1235"))
	require.Equal(t, uint(1), softwareCVE.SoftwareID)
	require.Equal(t, uint(0), softwareCVE.Source)
	require.Equal(t, "CVE-2021-1235", softwareCVE.CVE)
	require.Equal(t, "6.0.2-76060002.202210150739~1666289067~22.04~fe0ce53", softwareCVE.ResolvedInVersion)
}
