package tables

import "testing"

func TestUp_20251121124239(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}
