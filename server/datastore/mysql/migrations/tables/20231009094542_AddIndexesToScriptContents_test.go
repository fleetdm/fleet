package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231009094542(t *testing.T) {
	// skipping old migration tests as migrations don't change and we're getting
	// timeouts in CI
	t.Skip("old migration test, not longer required to run")
	db := applyUpToPrev(t)

	idxExists := indexExists(db, "scripts", "idx_scripts_team_name")
	require.False(t, idxExists)

	applyNext(t, db)

	idxExists = indexExists(db, "scripts", "idx_scripts_team_name")
	require.True(t, idxExists)
}
