package tables

import "testing"

func TestUp_20251030174959(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

}
