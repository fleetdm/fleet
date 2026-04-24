package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260423153950(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO windows_updates (host_id, date_epoch, kb_id) VALUES (?, ?, ?)`, 1, 1, 123)
	require.NoError(t, err)

	applyNext(t, db)

	_, err = db.Exec(`SELECT 1 FROM windows_updates LIMIT 1`)
	require.Error(t, err)
}
