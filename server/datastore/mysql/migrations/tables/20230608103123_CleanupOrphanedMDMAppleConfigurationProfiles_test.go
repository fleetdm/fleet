package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230608103123(t *testing.T) {
	db := applyUpToPrev(t)

	insertProfStmt := "INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum) VALUES (?, ?, ?, ?, 'made up')"

	// insert a global profile
	_, err := db.Exec(insertProfStmt, 0, "TestPayloadIdentifier", "TestPayloadName", `<?xml version="1.0"`)
	require.NoError(t, err)

	// insert a profile for a team that doesn't exist
	_, err = db.Exec(insertProfStmt, 999, "TestPayloadIdentifier", "TestPayloadName", `<?xml version="1.0"`)
	require.NoError(t, err)

	// insert a profile for a team that exists
	r, err := db.Exec(`INSERT INTO teams (name) VALUES (?)`, "Test Team")
	require.NoError(t, err)
	tmID, _ := r.LastInsertId()
	_, err = db.Exec(insertProfStmt, tmID, "TestPayloadIdentifier", "TestPayloadName", `<?xml version="1.0"`)
	require.NoError(t, err)

	applyNext(t, db)

	var teamIDs []uint
	err = db.Select(&teamIDs, "SELECT team_id FROM mdm_apple_configuration_profiles GROUP BY team_id")
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{0, uint(tmID)}, teamIDs) //nolint:gosec // dismiss G115
}
