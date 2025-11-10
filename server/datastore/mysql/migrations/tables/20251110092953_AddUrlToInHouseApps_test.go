package tables

import "testing"

func TestUp_20251110092953(t *testing.T) {
	db := applyUpToPrev(t)

	// Just a new column, so no logic to test here.
	// Leaving it in because it's nice to validate that the migration applies successfully.

	// Apply current migration.
	applyNext(t, db)
}
