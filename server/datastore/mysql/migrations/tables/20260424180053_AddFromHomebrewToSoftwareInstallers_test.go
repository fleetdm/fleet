package tables

import "testing"

func TestUp_20260424180053(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)
}
