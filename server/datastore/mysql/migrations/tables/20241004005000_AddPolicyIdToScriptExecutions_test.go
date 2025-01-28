package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241004005000(t *testing.T) {
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

	// insert a script result
	hostScriptResultID := execNoErrLastID(t, db, `INSERT INTO host_script_results
    	(host_id, execution_id, script_content_id, output, script_id, policy_id, user_id, sync_request)
		VALUES (1, 'a123b123', ?, '', ?, ?, NULL, FALSE)`, scriptContentID, scriptID, policyID)

	// delete the associated policy
	execNoErr(t, db, `DELETE FROM policies WHERE id = ?`, policyID)

	// policy ID should be null but script result should still exist
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM host_script_results WHERE policy_id IS NULL AND id = ?", hostScriptResultID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
