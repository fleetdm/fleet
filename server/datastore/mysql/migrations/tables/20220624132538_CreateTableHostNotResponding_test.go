package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220624132538(t *testing.T) {
	db := applyUpToPrev(t)

	stmt := `
INSERT INTO host_not_responding (host_id) VALUE (1) 
`
	_, err := db.Exec(stmt)
	require.ErrorContains(t, err, "doesn't exist")

	applyNext(t, db)
	_, err = db.Exec(stmt)
	require.NoError(t, err)
}
