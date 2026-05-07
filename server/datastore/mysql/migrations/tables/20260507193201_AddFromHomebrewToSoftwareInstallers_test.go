package tables

import "testing"

func TestUp_20260507193201(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)
}
