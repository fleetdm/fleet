package tables

import "testing"

func TestUp_20260422150000(t *testing.T) {
	db := applyUpToPrev(t)

	// Just a new column, so no logic to test here.
	// Leaving it in because it's nice to validate that the migration applies successfully.

	applyNext(t, db)
}
