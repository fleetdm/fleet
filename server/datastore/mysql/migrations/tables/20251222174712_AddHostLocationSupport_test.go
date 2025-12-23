package tables

import "testing"

func TestUp_20251222174712(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)
}
