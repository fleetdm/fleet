package tables

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type VersionRange struct {
	VersionStartIncluding string `db:"versionStartIncluding,omitempty"`
	VersionEndExcluding   string `db:"versionEndExcluding,omitempty"`
}

func TestUp_20230912125945(t *testing.T) {
	db := applyUpToPrev(t)

	// Define the struct to scan the version_ranges data into
	var scve struct {
		ID            uint           `db:"id"`
		SoftwareID    uint           `db:"software_id"`
		CreatedAt     string         `db:"created_at"`
		UpdatedAt     string         `db:"updated_at"`
		CPE           string         `db:"cpe"`
		VersionRanges sql.NullString `db:"version_ranges"`
	}

	insertStmt := `
	INSERT INTO software_cpe (
		software_id, 
		cpe, 
		version_ranges) 
	VALUES 
		(?, ?, ?)`

	applyNext(t, db)

	// Prepare version ranges data as a JSON string
	versionRanges := []VersionRange{
		{
			VersionStartIncluding: "1.0.0",
			VersionEndExcluding:   "2.0.0",
		},
		{
			VersionStartIncluding: "3.0.0",
			VersionEndExcluding:   "4.0.0",
		},
	}
	versionRangesJSON, err := json.Marshal(versionRanges)
	require.NoError(t, err)

	// Insert data including a JSON field
	args := []interface{}{
		1,
		"test-cpe",
		versionRangesJSON,
	}
	execNoErr(t, db, insertStmt, args...)

	// Query the inserted data
	selectStmt := "SELECT * FROM software_cpe WHERE cpe = ?"
	err = db.Get(&scve, selectStmt, "test-cpe")
	require.NoError(t, err)

	// Parse the JSON data from the version_ranges column into a slice of VersionRange structs
	var retrievedVersionRanges []VersionRange
	require.NoError(t, json.Unmarshal([]byte(scve.VersionRanges.String), &retrievedVersionRanges))

	require.Equal(t, versionRanges, retrievedVersionRanges)
}
