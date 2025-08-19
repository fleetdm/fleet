package tables

import "testing"

func TestUp_20250818165154(t *testing.T) {
	db := applyUpToPrev(t)

	// TODO: test at least for the mdm_profile_labels check constraint (only one
	// of windows, apple or google profile uuid).

	// Apply current migration.
	applyNext(t, db)

}
