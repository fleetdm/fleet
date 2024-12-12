package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241203130032(t *testing.T) {
	db := applyUpToPrev(t)

	// create a couple users and an invite
	u1 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u1", "u1@b.c", "1234", "salt")
	u2 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u2", "u2@b.c", "1234", "salt")
	inviteId := execNoErrLastID(t, db, `INSERT INTO invites
    	( invited_by, email, name, position, token, sso_enabled, global_role ) VALUES ( ?, ?, ?, ?, ?, ?, ?)
    	`, u1, "foo@example.com", "Foo User", "positron", "a123b123", false, "admin")

	// Apply current migration.
	applyNext(t, db)

	mfaEnabled := true
	err := db.Get(&mfaEnabled, "SELECT mfa_enabled FROM users WHERE id = ?", u1)
	require.NoError(t, err)
	require.False(t, mfaEnabled)
	mfaEnabled = true
	err = db.Get(&mfaEnabled, "SELECT mfa_enabled FROM invites WHERE id = ?", inviteId)
	require.NoError(t, err)
	require.False(t, mfaEnabled)

	execNoErr(t, db, `INSERT INTO verification_tokens (user_id, token) VALUES (?, ?)`, u2, "a123b1234")
	mfaCount := 0
	err = db.Get(&mfaCount, "SELECT COUNT(*) FROM verification_tokens")
	require.NoError(t, err)
	require.Equal(t, 1, mfaCount)

	execNoErr(t, db, `DELETE FROM users WHERE id = ?`, u2)
	err = db.Get(&mfaCount, "SELECT COUNT(*) FROM verification_tokens")
	require.NoError(t, err)
	require.Equal(t, 0, mfaCount)
}
