package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261005121016(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO mdm_apple_enrollment_profiles (token, type, dep_profile) VALUES ('static-token', 'automatic', '{}')`)
	execNoErr(t, db, `INSERT INTO mdm_idp_accounts (uuid, username, email) VALUES ('acct-1', 'user1', 'user1@example.com'), ('acct-2', 'user2', 'user2@example.com')`)

	applyNext(t, db)

	var prof struct {
		Token                  string  `db:"token"`
		PreviousToken          *string `db:"previous_token"`
		PreviousTokenExpiresAt *string `db:"previous_token_expires_at"`
	}
	require.NoError(t, db.Get(&prof, `SELECT token, previous_token, previous_token_expires_at FROM mdm_apple_enrollment_profiles WHERE type = 'automatic'`))
	require.Equal(t, "static-token", prof.Token)
	require.Nil(t, prof.PreviousToken)
	require.Nil(t, prof.PreviousTokenExpiresAt)

	insertChallenge := `INSERT INTO mdm_apple_dep_enrollment_challenges (challenge, idp_account_uuid, hardware_serial, host_uuid, expires_at)
		VALUES (?, ?, 'SERIAL1', 'UDID1', DATE_ADD(NOW(6), INTERVAL 1 HOUR))`
	execNoErr(t, db, insertChallenge, "abc", "acct-1")
	// binary collation: a challenge differing only by case is a distinct value
	execNoErr(t, db, insertChallenge, "ABC", "acct-1")
	execNoErr(t, db, insertChallenge, "def", "acct-2")

	_, err := db.Exec(insertChallenge, "abc", "acct-2")
	require.ErrorContains(t, err, "Duplicate entry")

	_, err = db.Exec(insertChallenge, "ghi", "no-such-account")
	require.ErrorContains(t, err, "foreign key constraint fails")

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM mdm_apple_dep_enrollment_challenges WHERE challenge = 'abc'`))
	require.Equal(t, 1, count)

	execNoErr(t, db, `DELETE FROM mdm_idp_accounts WHERE uuid = 'acct-1'`)

	var remaining []string
	require.NoError(t, db.Select(&remaining, `SELECT challenge FROM mdm_apple_dep_enrollment_challenges`))
	require.Equal(t, []string{"def"}, remaining)
}
