package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261118141311(t *testing.T) {
	db := applyUpToPrev(t)

	insertUser := func(email string) int64 {
		res, err := db.Exec(`INSERT INTO users (name, email, password, salt) VALUES (?, ?, 'pw', 'salt')`, email, email)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	u1 := insertUser("u1@example.com")
	u2 := insertUser("u2@example.com")

	applyNext(t, db)

	execNoErr(t, db, `INSERT INTO user_external_identities (user_id, issuer, subject) VALUES (?, 'gh', 's')`, u1)

	// One external identity can never map to two users.
	_, err := db.Exec(`INSERT INTO user_external_identities (user_id, issuer, subject) VALUES (?, 'gh', 's')`, u2)
	require.ErrorContains(t, err, "Duplicate entry")

	// Subjects compare byte-for-byte.
	execNoErr(t, db, `INSERT INTO user_external_identities (user_id, issuer, subject) VALUES (?, 'gh', 'S')`, u2)
	var matches int
	require.NoError(t, db.Get(&matches, `SELECT COUNT(*) FROM user_external_identities WHERE issuer = 'gh' AND subject = 's'`))
	require.Equal(t, 1, matches)

	execNoErr(t, db, `DELETE FROM users WHERE id = ?`, u1)
	var n int
	require.NoError(t, db.Get(&n, `SELECT COUNT(*) FROM user_external_identities`))
	require.Equal(t, 1, n)
}
