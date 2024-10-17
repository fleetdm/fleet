package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20230503101418(t *testing.T) {
	db := applyUpToPrev(t)

	r, err := db.Exec(`INSERT INTO jobs (name, args, state) VALUES (?, ?, ?)`, "Test", "{}", "queued")
	require.NoError(t, err)
	id, _ := r.LastInsertId()

	// Apply current migration.
	applyNext(t, db)

	type job struct {
		ID        uint      `db:"id"`
		Name      string    `db:"name"`
		UpdatedAt time.Time `db:"updated_at"`
		NotBefore time.Time `db:"not_before"`
	}
	var j job
	err = db.Get(&j, `SELECT id, name, updated_at, not_before FROM jobs WHERE id = ?`, id)
	require.NoError(t, err)
	require.NotZero(t, j.UpdatedAt)
	require.NotZero(t, j.NotBefore)
	j.UpdatedAt = time.Time{}
	j.NotBefore = time.Time{}
	require.Equal(t, job{ID: uint(id), Name: "Test"}, j) //nolint:gosec // dismiss G115
}
