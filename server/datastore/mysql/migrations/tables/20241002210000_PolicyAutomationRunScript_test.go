package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241002210000(t *testing.T) {
	db := applyUpToPrev(t)

	// insert a team
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ("Foo")`)

	// insert a policy
	policyID := execNoErrLastID(t, db, `INSERT INTO policies (name, query, description, team_id, checksum)
		VALUES ('test_policy', "SELECT 1", "", ?, "a123b123")`, teamID)

	// insert a script
	scriptContentID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES ("md5", "echo 'Hello World'")`)
	scriptID := execNoErrLastID(t, db, `INSERT INTO scripts (
			team_id, global_or_team_id, name, script_content_id
		) VALUES (?, ?, "hello-world.sh", ?)`, teamID, teamID, scriptContentID)

	// Apply current migration.
	applyNext(t, db)

	// associate the policy to the script
	execNoErr(t, db, `UPDATE policies SET script_id = ? WHERE id = ?`, scriptID, policyID)

	// attempt to delete the script; should error
	_, err := db.Exec(`DELETE FROM scripts WHERE id = ?`, scriptID)
	require.Error(t, err, "Foo")

	// dissociate the policy
	execNoErr(t, db, `UPDATE policies SET script_id = NULL WHERE id = ?`, policyID)

	// attempt to delete the script; should succeed
	execNoErr(t, db, `DELETE FROM scripts WHERE id = ?`, scriptID)
}
