package tables

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261111132618(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert invites with cleartext tokens.
	execNoErr(t, db, `INSERT INTO users (password, salt, name, email) VALUES ('p', 's', 'inviter', 'inviter@example.com')`)

	execNoErr(t, db, `INSERT INTO invites (invited_by, email, name, token) VALUES (1, 'a@example.com', 'A', 'invite-token-aaa')`)
	execNoErr(t, db, `INSERT INTO invites (invited_by, email, name, token) VALUES (1, 'b@example.com', 'B', 'invite-token-bbb')`)

	// Insert password reset requests with cleartext tokens.
	execNoErr(t, db, `INSERT INTO password_reset_requests (user_id, token, expires_at)
		VALUES (1, 'reset-token-111', DATE_ADD(NOW(), INTERVAL 1 DAY))`)
	execNoErr(t, db, `INSERT INTO password_reset_requests (user_id, token, expires_at)
		VALUES (1, 'reset-token-222', DATE_ADD(NOW(), INTERVAL 1 DAY))`)

	// Run the migration.
	applyNext(t, db)

	// Verify invites: token column is gone, token_hash is populated.
	var inviteCols []string
	require.NoError(t, db.Select(&inviteCols,
		`SELECT COLUMN_NAME FROM information_schema.columns WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'invites' AND COLUMN_NAME = 'token'`))
	assert.Empty(t, inviteCols, "cleartext token column should be dropped from invites")

	type inviteRow struct {
		Email     string `db:"email"`
		TokenHash []byte `db:"token_hash"`
	}
	var invites []inviteRow
	require.NoError(t, db.Select(&invites, `SELECT email, token_hash FROM invites ORDER BY email`))
	require.Len(t, invites, 2)

	wantHashA := sha256.Sum256([]byte("invite-token-aaa"))
	wantHashB := sha256.Sum256([]byte("invite-token-bbb"))
	assert.Equal(t, hex.EncodeToString(wantHashA[:]), hex.EncodeToString(invites[0].TokenHash))
	assert.Equal(t, hex.EncodeToString(wantHashB[:]), hex.EncodeToString(invites[1].TokenHash))

	// Verify lookup by hash works for invites.
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM invites WHERE token_hash = UNHEX(SHA2(?, 256))`, "invite-token-aaa"))
	assert.Equal(t, 1, count)

	// Verify password_reset_requests: token column is gone, token_hash is populated.
	var resetCols []string
	require.NoError(t, db.Select(&resetCols,
		`SELECT COLUMN_NAME FROM information_schema.columns WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'password_reset_requests' AND COLUMN_NAME = 'token'`))
	assert.Empty(t, resetCols, "cleartext token column should be dropped from password_reset_requests")

	type resetRow struct {
		UserID    uint   `db:"user_id"`
		TokenHash []byte `db:"token_hash"`
	}
	var resets []resetRow
	require.NoError(t, db.Select(&resets, `SELECT user_id, token_hash FROM password_reset_requests ORDER BY id`))
	require.Len(t, resets, 2)

	wantHash1 := sha256.Sum256([]byte("reset-token-111"))
	wantHash2 := sha256.Sum256([]byte("reset-token-222"))
	assert.Equal(t, hex.EncodeToString(wantHash1[:]), hex.EncodeToString(resets[0].TokenHash))
	assert.Equal(t, hex.EncodeToString(wantHash2[:]), hex.EncodeToString(resets[1].TokenHash))

	// Verify lookup by hash works for password resets.
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM password_reset_requests WHERE token_hash = UNHEX(SHA2(?, 256))`, "reset-token-111"))
	assert.Equal(t, 1, count)

	// Verify unique index exists on invites.token_hash.
	var idxCount int
	require.NoError(t, db.Get(&idxCount, `SELECT COUNT(1) FROM INFORMATION_SCHEMA.STATISTICS
		WHERE table_schema = DATABASE() AND table_name = 'invites' AND index_name = 'idx_invite_token_hash'`))
	assert.Positive(t, idxCount, "unique index on invites.token_hash should exist")

	// Verify index exists on password_reset_requests.token_hash.
	require.NoError(t, db.Get(&idxCount, `SELECT COUNT(1) FROM INFORMATION_SCHEMA.STATISTICS
		WHERE table_schema = DATABASE() AND table_name = 'password_reset_requests' AND index_name = 'idx_password_reset_token_hash'`))
	assert.Positive(t, idxCount, "index on password_reset_requests.token_hash should exist")
}
