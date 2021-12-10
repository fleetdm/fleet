package mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocks(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"LockUnlock", func(t *testing.T, ds *Datastore) { testLocksLockUnlock(t, ds) }},
		{"DBLocks", func(t *testing.T, ds *Datastore) { testLocksDBLocks(t, ds) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testLocksLockUnlock(t *testing.T, ds *Datastore) {
	owner1, err := server.GenerateRandomText(64)
	require.NoError(t, err)
	owner2, err := server.GenerateRandomText(64)
	require.NoError(t, err)

	// get first lock
	locked, err := ds.Lock(context.Background(), "test", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)

	// renew current lock
	locked, err = ds.Lock(context.Background(), "test", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)

	// owner2 tries to get the lock but fails
	locked, err = ds.Lock(context.Background(), "test", owner2, 1*time.Minute)
	require.NoError(t, err)
	assert.False(t, locked)

	// owner2 gets a new lock that expires quickly
	locked, err = ds.Lock(context.Background(), "test-expired", owner2, 1*time.Second)
	require.NoError(t, err)
	assert.True(t, locked)

	time.Sleep(3 * time.Second)

	// owner1 gets the same lock because it's now expired
	locked, err = ds.Lock(context.Background(), "test-expired", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)

	// unlocking clears the lock
	locked, err = ds.Lock(context.Background(), "test", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)
	err = ds.Unlock(context.Background(), "test", owner1)
	require.NoError(t, err)

	// owner2 tries to get the lock but fails
	locked, err = ds.Lock(context.Background(), "test", owner2, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)
}

func testLocksDBLocks(t *testing.T, ds *Datastore) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	_, err := ds.writer.ExecContext(ctx, `CREATE TABLE deadlocks(a int primary key)`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := ds.writer.ExecContext(ctx, `DROP TABLE deadlocks`)
		require.NoError(t, err)
	})

	_, err = ds.writer.ExecContext(ctx, `INSERT INTO deadlocks(a) VALUES (0), (1)`)
	require.NoError(t, err)

	// cause a deadlock (see https://stackoverflow.com/a/31552794/1094941)
	tx1, err := ds.writer.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	require.NoError(t, err)
	defer tx1.Rollback()
	tx2, err := ds.writer.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	require.NoError(t, err)
	defer tx2.Rollback()

	wait := make(chan struct{})
	go func() {
		var dst []int
		err = tx1.SelectContext(ctx, &dst, `SELECT * FROM deadlocks WHERE a = 0`)
		require.NoError(t, err)
		err = tx2.SelectContext(ctx, &dst, `SELECT * FROM deadlocks WHERE a = 1`)
		require.NoError(t, err)

		close(wait)
		_, err = tx1.ExecContext(ctx, `UPDATE deadlocks SET a = 0 WHERE a != 0`)
		require.Error(t, err)
		_, err = tx2.ExecContext(ctx, `UPDATE deadlocks SET a = 1 WHERE a != 1`)
		require.Error(t, err)
	}()

	<-wait
	locks, err := ds.DBLocks(ctx)
	require.NoError(t, err)
	require.Len(t, locks, 1)
	require.NotNil(t, locks[0].WaitingQuery)
	require.Equal(t, *locks[0].WaitingQuery, `UPDATE deadlocks SET a = 0 WHERE a != 0`)
	require.NotEmpty(t, locks[0].BlockingTrxID)
	require.NotEmpty(t, locks[0].WaitingTrxID)
	require.NotZero(t, locks[0].BlockingThread)
	require.NotZero(t, locks[0].WaitingThread)
}
