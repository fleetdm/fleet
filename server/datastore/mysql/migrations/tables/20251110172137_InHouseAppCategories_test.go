package tables

import "testing"

func TestUp_20251110172137(t *testing.T) {
	db := applyUpToPrev(t)

	// New table

	// Apply current migration.
	applyNext(t, db)
}
