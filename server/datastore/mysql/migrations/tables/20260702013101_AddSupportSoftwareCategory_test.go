package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260702013101(t *testing.T) {
	db := applyUpToPrev(t)

	// The global defaults seeded by earlier migrations should not yet include
	// the new "🛟 Support" category.
	var preCount int
	require.NoError(t, db.Get(&preCount,
		`SELECT COUNT(*) FROM software_categories WHERE team_id = 0 AND name = '🛟 Support'`))
	require.Equal(t, 0, preCount, "Support should not exist before this migration")

	// Two existing fleets so the per-fleet backfill has rows to touch.
	teamA := uint(execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "team-a")) //nolint:gosec // dismiss G115
	teamB := uint(execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "team-b")) //nolint:gosec // dismiss G115

	applyNext(t, db)

	// The global default now exists exactly once at team_id=0.
	type categoryRow struct {
		ID        uint      `db:"id"`
		Name      string    `db:"name"`
		TeamID    uint      `db:"team_id"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	var globalRows []categoryRow
	require.NoError(t, db.Select(&globalRows,
		`SELECT id, name, team_id, created_at, updated_at FROM software_categories WHERE team_id = 0 AND name = '🛟 Support'`))
	require.Len(t, globalRows, 1, "Support should be added exactly once at team_id=0")
	// Timestamps are pinned to a constant so the generated schema stays stable.
	require.Equal(t, "2026-05-29", globalRows[0].CreatedAt.Format("2006-01-02"), "timestamps should be pinned for schema-dump stability")
	require.Equal(t, "2026-05-29", globalRows[0].UpdatedAt.Format("2006-01-02"))

	// Each existing fleet got its own copy.
	for _, teamID := range []uint{teamA, teamB} {
		var count int
		require.NoError(t, db.Get(&count,
			`SELECT COUNT(*) FROM software_categories WHERE team_id = ? AND name = '🛟 Support'`, teamID))
		require.Equal(t, 1, count, "fleet %d should have its own Support category", teamID)
	}
}
