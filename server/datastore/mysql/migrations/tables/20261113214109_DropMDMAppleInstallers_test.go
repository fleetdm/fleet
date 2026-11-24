package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261113214109(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO mdm_apple_installers (name, size, manifest, installer, url_token) VALUES ('t.pkg', 3, '', 'abc', 'tok')`)
	require.NoError(t, err)

	applyNext(t, db)

	tableExists := func() bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'mdm_apple_installers'`).Scan(&n))
		return n > 0
	}
	require.False(t, tableExists())

	t.Run("a second run changes nothing", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261113214109(tx))
		require.NoError(t, tx.Commit())
		require.False(t, tableExists())
	})
}
