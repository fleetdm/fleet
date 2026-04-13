package tables

import "testing"

func TestUp_20260413175411(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)
}
