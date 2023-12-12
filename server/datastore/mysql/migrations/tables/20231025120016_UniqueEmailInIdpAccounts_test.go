package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231025120016(t *testing.T) {
	db := applyUpToPrev(t)

	type idpAcc struct {
		Email    string `db:"email"`
		UUID     string `db:"uuid"`
		Username string `db:"username"`
		Fullname string `db:"fullname"`
	}

	insertStmt := `INSERT INTO mdm_idp_accounts (email, uuid, username, fullname) VALUES (?, ?, ?, ?)`

	loadAccountsStmt := `
            SELECT email, uuid, username, fullname
            FROM mdm_idp_accounts ORDER BY uuid
	`

	rowsToInsert := []idpAcc{
		{Email: "foo@example.com", UUID: "UUID1", Username: "foo", Fullname: "Foo"},
		{Email: "foo@example.com", UUID: "UUID2", Username: "foo", Fullname: "Foo"},
		{Email: "bar@example.com", UUID: "UUID3", Username: "bar", Fullname: "Bar"},
		{Email: "baz@example.com", UUID: "UUID4", Username: "baz", Fullname: "Baz"},
		{Email: "baz@example.com", UUID: "UUID5", Username: "baz", Fullname: "Baz"},
	}
	for _, r := range rowsToInsert {
		_, err := db.Exec(insertStmt, r.Email, r.UUID, r.Username, r.Fullname)
		require.NoError(t, err)
	}

	var results []idpAcc
	err := db.Select(&results, loadAccountsStmt)
	require.NoError(t, err)
	require.Len(t, results, 5)
	require.Equal(t, results, rowsToInsert)

	// Apply current migration.
	applyNext(t, db)

	// check that duplicates are gone
	results = []idpAcc{}
	err = db.Select(&results, loadAccountsStmt)
	require.NoError(t, err)
	require.Len(t, results, 3)
	require.Equal(t, results, []idpAcc{
		{Email: "foo@example.com", UUID: "UUID2", Username: "foo", Fullname: "Foo"},
		{Email: "bar@example.com", UUID: "UUID3", Username: "bar", Fullname: "Bar"},
		{Email: "baz@example.com", UUID: "UUID5", Username: "baz", Fullname: "Baz"},
	})
}
