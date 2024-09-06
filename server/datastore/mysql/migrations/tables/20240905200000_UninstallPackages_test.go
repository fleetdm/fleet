package tables

import "testing"

func TestUp_20240905200000(t *testing.T) {
	db := applyUpToPrev(t)

	// TODO: Create test

	//
	// Insert data to test the migration
	//
	// ...

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}
