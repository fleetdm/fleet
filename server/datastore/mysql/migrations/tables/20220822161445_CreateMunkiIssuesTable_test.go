package tables

import (
	"testing"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestUp_20220822161445(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	res, err := db.Exec(`INSERT INTO munki_issues (name, issue_type) VALUES ('a', 'error')`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	assertDuplicate := func(err error) {
		driverErr, ok := err.(*mysql.MySQLError)
		require.True(t, ok)
		require.Equal(t, mysqlerr.ER_DUP_ENTRY, int(driverErr.Number))
	}

	// insert same name + issue type again, fails with duplicate error
	_, err = db.Exec(`INSERT INTO munki_issues (name, issue_type) VALUES ('a', 'error')`)
	require.Error(t, err)
	assertDuplicate(err)

	var existID int64
	err = db.Get(&existID, `SELECT id FROM munki_issues WHERE name = 'a'`)
	require.NoError(t, err)
	require.Equal(t, id, existID)

	_, err = db.Exec(`INSERT INTO host_munki_issues (host_id, munki_issue_id) VALUES (1, ?)`, id)
	require.NoError(t, err)

	// insert same host/issue ids again, fails with duplicate error
	_, err = db.Exec(`INSERT INTO host_munki_issues (host_id, munki_issue_id) VALUES (1, ?)`, id)
	require.Error(t, err)
	assertDuplicate(err)
}
