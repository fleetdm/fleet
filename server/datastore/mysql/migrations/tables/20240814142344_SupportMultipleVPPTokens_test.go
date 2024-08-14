package tables

import "testing"

func TestUp_20240814142344(t *testing.T) {
	db := applyUpToPrev(t)

	// TODO(mna): test migration

	// Apply current migration.
	applyNext(t, db)
}
