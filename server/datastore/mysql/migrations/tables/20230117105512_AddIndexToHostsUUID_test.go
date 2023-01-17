package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230117105512(t *testing.T) {
	db := applyUpToPrev(t)
	require.False(t, idxExists(db, "hosts", "idx_hosts_uuid"), "index should not exist before migration")
	applyNext(t, db)
	require.True(t, idxExists(db, "hosts", "idx_hosts_uuid"), "index should exist after migration")
}
