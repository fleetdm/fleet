package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220330100659(t *testing.T) {
	// skipping old migration tests as migrations don't change and we're getting
	// timeouts in CI
	t.Skip("old migration test, not longer required to run")
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
INSERT INTO jobs (
    name,
    args,
    state,
    retries,
    error
)
VALUES (?, ?, ?, ?, ?)
`
	_, err := db.Exec(query, "test", nil, "queued", 0, "")
	require.NoError(t, err)
}
