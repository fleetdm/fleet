package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260203170911(t *testing.T) {
	db := applyUpToPrev(t)

	db.Exec(`
		INSERT INTO policies (name, query) VALUES
			('test1', 'SELECT 1'),
			('test2', 'SELECT 1')
	`)

	// Apply current migration.
	applyNext(t, db)

	// Existing policies should have the new column set to true after migration
	// New policies should default to true when no value defined
	db.Exec(`
		INSERT INTO policies (name, query) VALUES
			('test3', 'SELECT 1'),
	`)

	// Only new policies with explicit false should be set to false
	db.Exec(`
		INSERT INTO policies (name, query, conditional_access_bypass_enabled) VALUES
			('test4', 'SELECT 1', false),
	`)

	var policies []struct {
		Name          string `db:"name"`
		BypassEnabled bool   `db:"conditional_access_bypass_enabled"`
	}

	err := sqlx.Select(db, &policies, `SELECT name, conditional_access_bypass_enabled FROM policies`)
	require.NoError(t, err)

	for _, p := range policies {
		if p.Name != "test4" {
			require.True(t, p.BypassEnabled)
		} else {
			require.False(t, p.BypassEnabled)
		}
	}
}
