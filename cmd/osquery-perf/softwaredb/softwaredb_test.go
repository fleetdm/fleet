package softwaredb

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromDatabaseSeededPool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "software.db")
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE software (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, version TEXT NOT NULL, source TEXT NOT NULL,
		bundle_identifier TEXT DEFAULT '', vendor TEXT DEFAULT '', arch TEXT DEFAULT '', release TEXT DEFAULT '',
		extension_id TEXT DEFAULT '', extension_for TEXT DEFAULT '', upgrade_code TEXT DEFAULT NULL)`)
	require.NoError(t, err)
	tx, err := db.Begin()
	require.NoError(t, err)
	stmt, err := tx.Prepare(`INSERT INTO software (name, version, source) VALUES (?, '1.0', 'apps')`)
	require.NoError(t, err)
	defer stmt.Close()
	for i := range MaxSoftwarePerPlatform + 10000 {
		_, err := stmt.Exec(fmt.Sprintf("app-%d", i))
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
	require.NoError(t, db.Close())

	pool := func(seed int64) []string {
		sdb, err := LoadFromDatabase(path, seed)
		require.NoError(t, err)
		names := make([]string, 0, len(sdb.Darwin))
		for _, s := range sdb.Darwin {
			names = append(names, s.Name)
		}
		return names
	}

	first := pool(1)
	require.Len(t, first, MaxSoftwarePerPlatform)
	// A restarted osquery-perf with the same --seed loads the same pool.
	assert.Equal(t, first, pool(1))
	assert.NotEqual(t, first, pool(2))
}
