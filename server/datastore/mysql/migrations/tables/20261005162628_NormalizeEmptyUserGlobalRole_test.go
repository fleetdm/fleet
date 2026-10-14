package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261005162628(t *testing.T) {
	db := applyUpToPrev(t)

	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	insertUser := func(email string, role any) int64 {
		return execNoErrLastID(t, db, `INSERT INTO users (email, password, salt, global_role, updated_at) VALUES (?, 'x', 'x', ?, ?)`,
			email, role, past)
	}
	emptyUser := insertUser("empty@example.com", "")
	nullUser := insertUser("null@example.com", nil)
	adminUser := insertUser("admin@example.com", "admin")

	insertInvite := func(email string, role any) int64 {
		return execNoErrLastID(t, db, `INSERT INTO invites (invited_by, email, token, global_role, updated_at) VALUES (?, ?, ?, ?, ?)`,
			adminUser, email, email, role, past)
	}
	emptyInvite := insertInvite("empty-invite@example.com", "")
	observerInvite := insertInvite("observer-invite@example.com", "observer")

	applyNext(t, db)

	type row struct {
		GlobalRole sql.NullString `db:"global_role"`
		UpdatedAt  time.Time      `db:"updated_at"`
	}
	queries := map[string]string{
		"users":   `SELECT global_role, updated_at FROM users WHERE id = ?`,
		"invites": `SELECT global_role, updated_at FROM invites WHERE id = ?`,
	}
	get := func(table string, id int64) row {
		var r row
		require.NoError(t, db.Get(&r, queries[table], id))
		return r
	}

	for _, tc := range []struct {
		table string
		id    int64
		want  sql.NullString
	}{
		{"users", emptyUser, sql.NullString{}},
		{"users", nullUser, sql.NullString{}},
		{"users", adminUser, sql.NullString{String: "admin", Valid: true}},
		{"invites", emptyInvite, sql.NullString{}},
		{"invites", observerInvite, sql.NullString{String: "observer", Valid: true}},
	} {
		got := get(tc.table, tc.id)
		require.Equal(t, tc.want, got.GlobalRole, "%s %d", tc.table, tc.id)
		require.True(t, past.Equal(got.UpdatedAt), "%s %d updated_at changed to %s", tc.table, tc.id, got.UpdatedAt)
	}
}
