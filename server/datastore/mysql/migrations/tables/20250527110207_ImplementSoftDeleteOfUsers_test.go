package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20250527110207(t *testing.T) {
	db := applyUpToPrev(t)

	// Add a user to the users table
	_, err := db.Exec(`
		INSERT INTO users (name, email, password, salt)
		VALUES ('Test User', 'test@example.com', 'abc123', 'xxx')
	`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Retrieve the user from the users view
	var users []struct {
		ID    uint   `db:"id"`
		Name  string `db:"name"`
		Email string `db:"email"`
	}
	err = sqlx.Select(db, &users, `
		SELECT id, name, email
		FROM users
		`)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "Test User", users[0].Name)
	require.Equal(t, "test@example.com", users[0].Email)

	// Update the user to mark it as deleted
	_, err = db.Exec(`
		UPDATE users_all
		SET deleted_at = NOW(), deleted_by_user_id = 1
		WHERE id = ?
	`, users[0].ID)
	require.NoError(t, err)

	// Check that the user is no longer visible in the users view
	err = sqlx.Select(db, &users, `
		SELECT id, name, email
		FROM users
		WHERE id = ?
	`, users[0].ID,
	)
	require.NoError(t, err)
	require.Len(t, users, 0)
}
