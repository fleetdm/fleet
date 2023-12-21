package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231221130911(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	insertStmt := `
		INSERT INTO operating_system_cve (
			cve, operating_system_id
		) VALUES (?, ?)
		`

	_, err := db.Exec(insertStmt, "CVE-2023-12345", 1)
	require.NoError(t, err)

	// unique constraint applies to cve+operating_system_id
	_, err = db.Exec(insertStmt, "CVE-2023-12345", 1)
	require.ErrorContains(t, err, "Duplicate entry")

	_, err = db.Exec(insertStmt, "CVE-2023-12345", 2)
	require.NoError(t, err)

	selectStmt := `
		SELECT cve, operating_system_id
		FROM operating_system_cve
		`
	var rows []struct {
		CVE                string `db:"cve"`
		OperatingSystemID  uint   `db:"operating_system_id"`
	}
	err = db.Select(&rows, selectStmt)
	require.NoError(t, err)
}
