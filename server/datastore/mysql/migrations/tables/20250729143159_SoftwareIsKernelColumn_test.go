package tables

import "testing"

func TestUp_20250729143159(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

}
