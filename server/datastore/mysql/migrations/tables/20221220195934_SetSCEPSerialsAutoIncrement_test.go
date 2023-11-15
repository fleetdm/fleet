package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221220195934(t *testing.T) {
	// skipping old migration tests as migrations don't change and we're getting
	// timeouts in CI
	t.Skip("old migration test, not longer required to run")
	db := applyUpToPrev(t)
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM scep_serials")
	require.NoError(t, err)
	require.Equal(t, 0, count)
	applyNext(t, db)

	execNoErr(t, db, "INSERT INTO scep_serials () VALUES ()")
	var serial int
	err = db.Get(&serial, "SELECT serial FROM scep_serials")
	require.NoError(t, err)
	require.Equal(t, 2, serial)
}
