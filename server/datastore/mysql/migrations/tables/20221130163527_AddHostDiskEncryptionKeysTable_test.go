package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestUp_20221130163527(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
		INSERT INTO hosts (hostname, osquery_host_id) VALUES ('foo.example.com', 'foo');
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO hosts (hostname, osquery_host_id, computer_name) VALUES ('bar.example.com', 'bar', 'bar');
	`)
	require.NoError(t, err)

	applyNext(t, db)

	insertStmt := `
      INSERT INTO host_disk_encryption_keys (host_id, disk_encryption_key)
      VALUES (?, ?)`

	_, err = db.Exec(insertStmt, 1, "key1")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, 2, "key2")
	require.NoError(t, err)

	type hostDiskEncryptionKey struct {
		HostID            uint64    `db:"host_id"`
		DiskEncryptionKey string    `db:"disk_encryption_key"`
		CreatedAt         time.Time `db:"created_at"`
		UpdatedAt         time.Time `db:"updated_at"`
	}

	var keys []hostDiskEncryptionKey
	err = db.Select(&keys, "SELECT * FROM host_disk_encryption_keys ORDER BY host_id")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	for i, key := range keys {
		require.Equal(t, uint64(i+1), key.HostID)
		require.Equal(t, fmt.Sprintf("key%d", i+1), key.DiskEncryptionKey)
		require.NotEmpty(t, key.CreatedAt)
		require.NotEmpty(t, key.UpdatedAt)
	}

	// validate that the host_id column is a primary key
	_, err = db.Exec(insertStmt, 1, "key3")
	require.Error(t, err)
	driverErr, ok := err.(*mysql.MySQLError)
	require.True(t, ok)
	require.Equal(t, mysqlerr.ER_DUP_ENTRY, int(driverErr.Number))
}
