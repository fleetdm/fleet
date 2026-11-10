package tables

import (
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestUp_20261110123702(t *testing.T) {
	db := applyUpToPrev(t)

	team1 := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1')`)
	team2 := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team2')`)
	execNoErr(t, db, `INSERT INTO labels (name, query, team_id) VALUES ('global', '', NULL)`)
	execNoErr(t, db, `INSERT INTO labels (name, query, team_id) VALUES ('engineering', '', ?)`, team1)

	_, err := db.Exec(`INSERT INTO labels (name, query, team_id) VALUES ('engineering', '', ?)`, team2)
	requireDuplicateEntry(t, err)

	applyNext(t, db)

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM labels WHERE name IN ('global', 'engineering')`))
	require.Equal(t, 2, count)

	execNoErr(t, db, `INSERT INTO labels (name, query, team_id) VALUES ('engineering', '', ?)`, team2)

	// Still unique within a fleet and among global labels, including case variants.
	_, err = db.Exec(`INSERT INTO labels (name, query, team_id) VALUES ('Engineering', '', ?)`, team2)
	requireDuplicateEntry(t, err)
	_, err = db.Exec(`INSERT INTO labels (name, query, team_id) VALUES ('GLOBAL', '', NULL)`)
	requireDuplicateEntry(t, err)

	// The upsert used when applying label specs now keys on name and fleet.
	execNoErr(t, db, `INSERT INTO labels (name, query, description, team_id) VALUES ('engineering', '', 'updated', ?)
		ON DUPLICATE KEY UPDATE description = VALUES(description)`, team1)
	var descriptions []string
	require.NoError(t, db.Select(&descriptions, `SELECT COALESCE(description, '') FROM labels WHERE name = 'engineering' ORDER BY team_id`))
	require.Equal(t, []string{"updated", ""}, descriptions)
}

func requireDuplicateEntry(t *testing.T, err error) {
	t.Helper()
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	require.ErrorContains(t, err, "Duplicate entry")
}
