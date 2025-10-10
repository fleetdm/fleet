package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20251009091733(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM labels WHERE label_type = ? AND platform != ''`, fleet.LabelTypeBuiltIn)
	require.NoError(t, err)
	require.Zero(t, count)
}
