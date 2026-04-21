package tables

import "testing"

func TestUp_20260421182057(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)
}
