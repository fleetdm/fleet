package mysql

import (
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPrivateKey is a 32-byte key for AES-256 encryption in tests.
const testPrivateKey = "abcdef0123456789abcdef0123456789"

type testEnv struct {
	db     *sqlx.DB
	logger *slog.Logger
	ds     *Datastore
}

func (env *testEnv) truncateTables(t *testing.T) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, env.db, env.logger, nil, "host_recovery_key_passwords", "hosts")
}

func (env *testEnv) insertHost(t *testing.T, hostname string) uint {
	t.Helper()
	ctx := t.Context()

	result, err := env.db.ExecContext(ctx, `
		INSERT INTO hosts (hostname, created_at, updated_at)
		VALUES (?, NOW(), NOW())
	`, hostname)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

func TestRecoveryKeyPassword(t *testing.T) {
	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "recoverykeypassword_mysql_" + t.Name(),
	})

	mysql_testing_utils.LoadDefaultSchema(t, testName, opts)
	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := platform_mysql.NewDB(config, &platform_mysql.DBOptions{}, "")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	logger := slog.New(slog.DiscardHandler)
	conns := &platform_mysql.DBConnections{
		Primary: db,
		Replica: db,
		Options: &platform_mysql.DBOptions{
			PrivateKey: testPrivateKey,
		},
	}

	ds := NewDatastore(conns, logger)
	env := &testEnv{db: db, logger: logger, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"SetAndGet", testSetAndGet},
		{"GetNotFound", testGetNotFound},
		{"SetOverwrite", testSetOverwrite},
		{"UpdatedAtChanges", testUpdatedAtChanges},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.truncateTables(t)
			c.fn(t, env)
		})
	}
}

func testSetAndGet(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-1")

	// Set password
	password, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	require.NotEmpty(t, password)

	// Get password and verify it matches
	result, err := env.ds.GetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	assert.Equal(t, password, result.Password)
	assert.False(t, result.UpdatedAt.IsZero())
}

func testGetNotFound(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Try to get password for non-existent host
	_, err := env.ds.GetHostRecoveryKeyPassword(ctx, 99999)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testSetOverwrite(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-2")

	// Set password first time
	password1, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Set password second time (should overwrite)
	password2, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Passwords should be different (randomly generated)
	assert.NotEqual(t, password1, password2)

	// Verify only the new password is stored
	result, err := env.ds.GetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	assert.Equal(t, password2, result.Password)
}

func testUpdatedAtChanges(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-3")

	// Set password first time
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	result1, err := env.ds.GetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Set password second time
	_, err = env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	result2, err := env.ds.GetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// updated_at should have changed
	assert.True(t, result2.UpdatedAt.After(result1.UpdatedAt), "updated_at should increase after overwrite")
}
