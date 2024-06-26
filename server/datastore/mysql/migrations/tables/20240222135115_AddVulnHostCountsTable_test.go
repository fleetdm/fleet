package tables

import "testing"

func TestUp_20240222135115(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	insertStmt := `
		INSERT INTO vulnerability_host_counts (cve, team_id, host_count)
		VALUES ('CVE-2024-1234', 1, 1)
		`
	_, err := db.Exec(insertStmt)
	if err != nil {
		t.Errorf("Error inserting data into vulnerability_host_counts: %v", err)
	}

	// Verify unique constraint on cve, team_id
	insertStmt = `

		INSERT INTO vulnerability_host_counts (cve, team_id, host_count)
		VALUES ('CVE-2024-1234', 1, 1)
		`
	_, err = db.Exec(insertStmt)
	if err == nil {
		t.Errorf("Expected error inserting duplicate data into vulnerability_host_counts")
	}
}
