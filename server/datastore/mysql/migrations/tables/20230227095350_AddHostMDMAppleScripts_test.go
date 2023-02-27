package tables

import "testing"

func TestUp_20230227095350(t *testing.T) {
	db := applyUpToPrev(t)

	t.Fatal("implement test")
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
