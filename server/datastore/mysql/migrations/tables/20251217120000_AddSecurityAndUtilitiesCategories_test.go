package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20251217120000(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Check that Security and Utilities categories are added
	var gotCategories []fleet.SoftwareCategory
	err := db.Select(&gotCategories, "SELECT id, name FROM software_categories WHERE name IN ('Security', 'Utilities')")
	require.NoError(t, err)
	require.Len(t, gotCategories, 2)

	var gotNames []string
	for _, c := range gotCategories {
		gotNames = append(gotNames, c.Name)
	}
	require.Contains(t, gotNames, "Security")
	require.Contains(t, gotNames, "Utilities")
}
