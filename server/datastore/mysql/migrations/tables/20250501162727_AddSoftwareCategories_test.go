package tables

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20250501162727(t *testing.T) {
	require.NoError(t, os.Setenv("FLEET_TEST_DISABLE_COLLATION_UPDATES", "true"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("FLEET_TEST_DISABLE_COLLATION_UPDATES"))
	})

	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	checkCollation(t, db)

	// Check that default values are there
	var gotCategories []fleet.SoftwareCategory
	err := db.Select(&gotCategories, "SELECT id, name FROM software_categories")
	require.NoError(t, err)
	require.Len(t, gotCategories, 4)
	expectedNames := []string{"Developer tools", "Browsers", "Communication", "Productivity"}
	var gotNames []string
	for _, c := range gotCategories {
		gotNames = append(gotNames, c.Name)
	}
	require.ElementsMatch(t, expectedNames, gotNames)
}
