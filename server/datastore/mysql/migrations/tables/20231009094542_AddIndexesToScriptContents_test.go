package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231009094542(t *testing.T) {
	db := applyUpToPrev(t)

	idxExists := indexExists(db, "scripts", "idx_scripts_team_name")
	require.False(t, idxExists)

	applyNext(t, db)

	idxExists = indexExists(db, "scripts", "idx_scripts_team_name")
	require.True(t, idxExists)
}
