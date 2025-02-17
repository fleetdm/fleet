package feature_migrations

import (
	"testing"
)

func TestUp_20250101000000(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up ...

	// Apply current migration.
	applyNext(t, db)

	// Check results ...

}
