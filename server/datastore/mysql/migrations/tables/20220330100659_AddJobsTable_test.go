package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220330100659(t *testing.T) {
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
