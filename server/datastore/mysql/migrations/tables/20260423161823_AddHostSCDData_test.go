package tables

import "testing"

func TestUp_20260423161823(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)
}
