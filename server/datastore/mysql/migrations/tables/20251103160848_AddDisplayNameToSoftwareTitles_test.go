package tables

import "testing"

func TestUp_20251103160848(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

}
