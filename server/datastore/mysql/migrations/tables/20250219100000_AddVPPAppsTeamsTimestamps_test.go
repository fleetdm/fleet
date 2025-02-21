package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250219100000(t *testing.T) {
	db := applyUpToPrev(t)

	appCreatedAt := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	appUpdatedAt := time.Date(2025, 2, 2, 1, 0, 0, 0, time.UTC)

	adamID := "a"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform, created_at, updated_at) VALUES (?,?,?,?),(?,?,?,?),(?,?,?,?)`,
		adamID, "darwin", appCreatedAt, appUpdatedAt,
		adamID, "ios", appCreatedAt, appUpdatedAt,
		adamID, "ipados", appCreatedAt, appUpdatedAt,
	)
	vppTokenID := execNoErrLastID(t, db, `
	INSERT INTO vpp_tokens (
		organization_name,
		location,
		renew_at,
		token
	) VALUES
		(?, ?, ?, ?)
	`,
		"org1", "loc1", "2030-01-01 10:10:10", "blob1",
	)
	execNoErr(t, db, `INSERT INTO teams (name, id) VALUES ("Foo", 1)`)
	execNoErr(
		t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id, vpp_token_id)
			VALUES (?,?,?,?,?),(?,?,?,?,?),(?,?,?,?,?),(?,?,?,?,?)`,
		adamID, "darwin", nil, 0, vppTokenID,
		adamID, "darwin", 1, 1, vppTokenID,
		adamID, "ios", 1, 1, vppTokenID,
		adamID, "ipados", 1, 1, vppTokenID,
	)

	// Apply current migration.
	applyNext(t, db)

	var row struct {
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	require.NoError(t, db.Get(&row, `SELECT created_at, updated_at FROM vpp_apps_teams WHERE adam_id = ?`, adamID))
	assert.Equal(t, appCreatedAt, row.CreatedAt)
	assert.Equal(t, appUpdatedAt, row.UpdatedAt)
}
