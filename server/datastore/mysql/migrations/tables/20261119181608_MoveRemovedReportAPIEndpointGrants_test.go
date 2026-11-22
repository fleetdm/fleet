package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261119181608(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		oldRun      = "/api/v1/fleet/queries/run"
		newRun      = "/api/v1/fleet/reports/run"
		oldDelete   = "/api/v1/fleet/reports/id/:id"
		otherDelete = "/api/v1/fleet/reports/id/{report_id}" // same fingerprint, other placeholder name
		newDelete   = "/api/v1/fleet/reports/:id"
		otherPath   = "/api/v1/fleet/reports"
	)
	newUser := func(name string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO users (password, salt, name, email, api_only) VALUES ('p', 's', ?, ?, 1)`,
			name, fmt.Sprintf("%s@example.com", name))
	}
	grant := func(userID int64, method, path string) {
		execNoErr(t, db, `INSERT INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, ?)`, userID, path, method)
	}

	// A user with no grants can call every endpoint, so the only grant must be replaced, not just removed.
	runOnly := newUser("run-only")
	grant(runOnly, "GET", oldRun)
	runAndOther := newUser("run-and-other")
	grant(runAndOther, "GET", oldRun)
	grant(runAndOther, "GET", otherPath)
	runBoth := newUser("run-both")
	grant(runBoth, "GET", oldRun)
	grant(runBoth, "GET", newRun)
	deleteOnly := newUser("delete-only")
	grant(deleteOnly, "DELETE", oldDelete)
	deleteBoth := newUser("delete-both")
	grant(deleteBoth, "DELETE", otherDelete)
	grant(deleteBoth, "DELETE", newDelete)
	unrelated := newUser("unrelated")
	grant(unrelated, "GET", otherPath)
	grant(unrelated, "DELETE", newDelete)

	applyNext(t, db)

	grantsOf := func(userID int64) []string {
		var grants []string
		require.NoError(t, db.Select(&grants,
			`SELECT CONCAT(method, ' ', path) FROM user_api_endpoints WHERE user_id = ? ORDER BY method, path`, userID))
		return grants
	}
	assert.Equal(t, []string{"GET " + newRun}, grantsOf(runOnly))
	assert.Equal(t, []string{"GET " + otherPath, "GET " + newRun}, grantsOf(runAndOther))
	assert.Equal(t, []string{"GET " + newRun}, grantsOf(runBoth))
	assert.Equal(t, []string{"DELETE " + newDelete}, grantsOf(deleteOnly))
	assert.Equal(t, []string{"DELETE " + newDelete}, grantsOf(deleteBoth))
	assert.Equal(t, []string{"DELETE " + newDelete, "GET " + otherPath}, grantsOf(unrelated))
}
