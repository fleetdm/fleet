package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261106154955(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		oldPath   = "/api/v1/fleet/hosts/identifier/:identifier"
		newPath   = "/api/v1/fleet/hosts/:id"
		otherPath = "/api/v1/fleet/hosts"
	)
	newUser := func(name string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO users (password, salt, name, email, api_only) VALUES ('p', 's', ?, ?, 1)`,
			name, fmt.Sprintf("%s@example.com", name))
	}
	grant := func(userID int64, path string) {
		execNoErr(t, db, `INSERT INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, 'GET')`, userID, path)
	}

	identifierOnly := newUser("identifier-only")
	grant(identifierOnly, oldPath)
	grant(identifierOnly, otherPath)
	bothGrants := newUser("both-grants")
	grant(bothGrants, oldPath)
	grant(bothGrants, newPath)
	unrelated := newUser("unrelated")
	grant(unrelated, otherPath)

	applyNext(t, db)

	grantsOf := func(userID int64) []string {
		var paths []string
		require.NoError(t, db.Select(&paths, `SELECT path FROM user_api_endpoints WHERE user_id = ? AND method = 'GET' ORDER BY path`, userID))
		return paths
	}
	assert.Equal(t, []string{otherPath, newPath}, grantsOf(identifierOnly))
	assert.Equal(t, []string{newPath}, grantsOf(bothGrants))
	assert.Equal(t, []string{otherPath}, grantsOf(unrelated))
}
