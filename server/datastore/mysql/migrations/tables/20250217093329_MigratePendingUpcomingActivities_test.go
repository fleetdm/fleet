package tables

import "testing"

func TestUp_20250217093329_None(t *testing.T) {
	db := applyUpToPrev(t)
	// Apply current migration.
	applyNext(t, db)
	assertRowCount(t, db, "upcoming_activities", 0)
}
