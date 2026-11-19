package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261118222516(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		oldPath   = "/api/v1/fleet/hosts/identifier/:identifier/query"
		newPath   = "/api/v1/fleet/hosts/:id/query"
		otherPath = "/api/v1/fleet/hosts/:id/refetch"
	)
	newUser := func(name string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO users (password, salt, name, email, api_only) VALUES ('p', 's', ?, ?, 1)`,
			name, fmt.Sprintf("%s@example.com", name))
	}
	grant := func(userID int64, path string) {
		execNoErr(t, db, `INSERT INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, 'POST')`, userID, path)
	}

	identifierAndOther := newUser("identifier-and-other")
	grant(identifierAndOther, oldPath)
	grant(identifierAndOther, otherPath)
	bothGrants := newUser("both-grants")
	grant(bothGrants, oldPath)
	grant(bothGrants, newPath)
	unrelated := newUser("unrelated")
	grant(unrelated, otherPath)
	// A user with no grants can call every endpoint, so a user's only grant must be replaced, not just removed. The server accepts any
	// placeholder spelling and the version placeholder, so the stored path can differ from the catalog's.
	alternateSpellings := []string{
		oldPath,
		"/api/v1/fleet/hosts/identifier/{identifier}/query",
		"/api/v1/fleet/hosts/identifier/:serial/query",
		"/api/_version_/fleet/hosts/identifier/:identifier/query",
	}
	alternateUsers := make([]int64, len(alternateSpellings))
	for i, path := range alternateSpellings {
		alternateUsers[i] = newUser(fmt.Sprintf("alternate-spelling-%d", i))
		grant(alternateUsers[i], path)
	}

	applyNext(t, db)

	grantsOf := func(userID int64) []string {
		var paths []string
		require.NoError(t, db.Select(&paths, `SELECT path FROM user_api_endpoints WHERE user_id = ? AND method = 'POST' ORDER BY path`, userID))
		return paths
	}
	assert.Equal(t, []string{newPath, otherPath}, grantsOf(identifierAndOther))
	assert.Equal(t, []string{newPath}, grantsOf(bothGrants))
	assert.Equal(t, []string{otherPath}, grantsOf(unrelated))
	for i, userID := range alternateUsers {
		assert.Equal(t, []string{newPath}, grantsOf(userID), alternateSpellings[i])
	}
}
