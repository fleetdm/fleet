package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20230916091643(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := `
INSERT INTO policies (
	name, 
	query, 
	description, 
	platforms, 
	critical
)
VALUES
	(?, ?, ?, ?, ?)`

	args := []interface{}{
		"test-policy",
		"SELECT * FROM apps",
		"Test policy description",
		"darwin,linux",
		1,
	}
	execNoErr(t, db, insertStmt, args...)

	applyNext(t, db)

	// retrieve the stored value
	var policy fleet.PolicyData

	selectStmt := "SELECT * FROM policies WHERE name = ?"
	require.NoError(t, db.Get(&policy, selectStmt, "test-policy"))
	require.Equal(t, "test-policy", policy.Name)
	require.Equal(t, "SELECT * FROM apps", policy.Query)
	require.Equal(t, "Test policy description", policy.Description)
	require.Equal(t, "darwin,linux", policy.Platform)
	require.Equal(t, true, policy.Critical)
	require.Equal(t, uint(0), policy.FailingHostCount) // Default value
	require.Equal(t, uint(0), policy.PassingHostCount) // Default value

	// Update the counts and verify
	updateStmt := "UPDATE policies SET failing_host_count = ?, passing_host_count = ? WHERE name = ?"
	updateArgs := []interface{}{
		500000, // Setting a large value to test the capacity of the new columns
		300000,
		"test-policy",
	}
	execNoErr(t, db, updateStmt, updateArgs...)

	require.NoError(t, db.Get(&policy, selectStmt, "test-policy"))
	require.Equal(t, uint(500000), policy.FailingHostCount)
	require.Equal(t, uint(300000), policy.PassingHostCount)
}
