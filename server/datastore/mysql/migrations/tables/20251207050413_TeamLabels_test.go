package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20251207050413(t *testing.T) {
	db := applyUpToPrev(t)

	// create a team to apply later
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('A Team')`)

	// create some labels
	idlA := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LA', 'select 1')`)
	idlB := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LB', 'select 1')`)

	// add team ID to labels
	applyNext(t, db)

	// make sure no team label doesn't work
	_, err := db.Exec(`INSERT INTO labels (name, query, team_id) VALUES ('no team', 'select 1', 0)`)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// make sure nonexistent team label doesn't work
	_, err = db.Exec(`INSERT INTO labels (name, query, team_id) VALUES ('fake team', 'select 1', ?)`, teamID+1)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// make sure inserting a label with no team specified still works
	idlC := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LC', 'select 1')`)

	// make sure inserting a label with a real team specified works
	idlD := execNoErrLastID(t, db, `INSERT INTO labels (name, query, team_id) VALUES ('LD', 'select 1', ?)`, teamID)

	var labels []struct {
		ID     uint  `db:"id"`
		TeamID *uint `db:"team_id"`
	}
	// only grab labels we've created (built-ins will have lower ID)
	err = sqlx.Select(db, &labels, `SELECT id, team_id FROM labels WHERE id >= ? ORDER BY id ASC`, idlA)
	require.NoError(t, err)
	require.Len(t, labels, 4)

	expected := []struct {
		ID     uint
		TeamID uint
	}{
		{ID: uint(idlA)},                       //nolint:gosec // dismiss G115
		{ID: uint(idlB)},                       //nolint:gosec // dismiss G115
		{ID: uint(idlC)},                       //nolint:gosec // dismiss G115
		{ID: uint(idlD), TeamID: uint(teamID)}, //nolint:gosec // dismiss G115
	}

	for index, actual := range labels {
		assert.Equal(t, expected[index].ID, actual.ID)
		if expected[index].TeamID > 0 {
			assert.Equal(t, expected[index].TeamID, *actual.TeamID)
		} else {
			assert.Nil(t, actual.TeamID)
		}
	}
}
