package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260512173249(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a vpp token to satisfy the FK.
	tokenID := execNoErrLastID(t, db,
		`INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES (?, ?, ?, ?)`,
		"org", "loc", time.Now(), "token",
	)

	// Apply current migration.
	applyNext(t, db)

	// Insert a row with explicit values.
	_, err := db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id, apple_user_id, status)
		VALUES (?, ?, ?, ?, ?)`,
		tokenID, "user@example.com", "11111111-1111-1111-1111-111111111111", "apple-user-1", "registered",
	)
	require.NoError(t, err)

	// Insert a row that exercises the default status ('pending') and NULL apple_user_id.
	_, err = db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id)
		VALUES (?, ?, ?)`,
		tokenID, "other@example.com", "22222222-2222-2222-2222-222222222222",
	)
	require.NoError(t, err)

	var (
		status      string
		appleUserID *string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err = db.QueryRow(`
		SELECT status, apple_user_id, created_at, updated_at
		FROM vpp_client_users
		WHERE managed_apple_id = ?`, "other@example.com",
	).Scan(&status, &appleUserID, &createdAt, &updatedAt)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)
	assert.Nil(t, appleUserID)
	assert.False(t, createdAt.IsZero())
	assert.False(t, updatedAt.IsZero())

	// Duplicate (vpp_token_id, managed_apple_id) is rejected by unique constraint.
	_, err = db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id)
		VALUES (?, ?, ?)`,
		tokenID, "user@example.com", "33333333-3333-3333-3333-333333333333",
	)
	require.Error(t, err)

	// Duplicate (vpp_token_id, client_user_id) is rejected by unique constraint.
	_, err = db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id)
		VALUES (?, ?, ?)`,
		tokenID, "third@example.com", "11111111-1111-1111-1111-111111111111",
	)
	require.Error(t, err)

	// Same managed_apple_id under a different vpp_token_id is allowed.
	otherTokenID := execNoErrLastID(t, db,
		`INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES (?, ?, ?, ?)`,
		"org2", "loc2", time.Now(), "token2",
	)
	_, err = db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id)
		VALUES (?, ?, ?)`,
		otherTokenID, "user@example.com", "44444444-4444-4444-4444-444444444444",
	)
	require.NoError(t, err)

	// Invalid status value is rejected by ENUM.
	_, err = db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id, status)
		VALUES (?, ?, ?, ?)`,
		tokenID, "bogus@example.com", "55555555-5555-5555-5555-555555555555", "bogus",
	)
	require.Error(t, err)

	// FK to vpp_tokens is enforced.
	_, err = db.Exec(`
		INSERT INTO vpp_client_users
			(vpp_token_id, managed_apple_id, client_user_id)
		VALUES (?, ?, ?)`,
		999999, "ghost@example.com", "66666666-6666-6666-6666-666666666666",
	)
	require.Error(t, err)

	// ON DELETE CASCADE: deleting the parent vpp_token removes its rows.
	_, err = db.Exec(`DELETE FROM vpp_tokens WHERE id = ?`, otherTokenID)
	require.NoError(t, err)
	var remaining int
	err = db.QueryRow(`SELECT COUNT(*) FROM vpp_client_users WHERE vpp_token_id = ?`, otherTokenID).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}
