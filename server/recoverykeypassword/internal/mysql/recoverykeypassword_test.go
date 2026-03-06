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
// This is a synthetic test-only value, not a real secret.
const testPrivateKey = "FLEET_TEST_KEY_DO_NOT_USE_IN_PRD"

type testEnv struct {
	db     *sqlx.DB
	logger *slog.Logger
	ds     *Datastore
}

func (env *testEnv) truncateTables(t *testing.T) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, env.db, env.logger, nil,
		"host_recovery_key_passwords", "host_operating_system", "operating_systems",
		"hosts", "teams", "nano_enrollments", "nano_command_results", "nano_commands",
		"nano_devices", "app_config_json")
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

// insertHostFull inserts a host with all fields needed for recovery lock testing.
// Also inserts into operating_systems and host_operating_system tables.
func (env *testEnv) insertHostFull(t *testing.T, hostname, uuid, platform, osVersion string, teamID *uint) uint {
	t.Helper()
	ctx := t.Context()

	result, err := env.db.ExecContext(ctx, `
		INSERT INTO hosts (hostname, uuid, platform, os_version, team_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`, hostname, uuid, platform, osVersion, teamID)
	require.NoError(t, err)

	hostID, err := result.LastInsertId()
	require.NoError(t, err)

	// Parse osVersion (e.g., "macOS 15.7" -> name="macOS", version="15.7")
	// For non-darwin platforms, use osVersion as-is
	osName := osVersion
	osVersionNum := ""
	if platform == "darwin" && len(osVersion) > 6 && osVersion[:6] == "macOS " {
		osName = "macOS"
		osVersionNum = osVersion[6:]
	}

	// Insert into operating_systems
	osResult, err := env.db.ExecContext(ctx, `
		INSERT INTO operating_systems (name, version, arch, kernel_version, platform, display_version)
		VALUES (?, ?, 'x86_64', '', ?, '')
	`, osName, osVersionNum, platform)
	require.NoError(t, err)

	osID, err := osResult.LastInsertId()
	require.NoError(t, err)

	// Link host to operating system
	_, err = env.db.ExecContext(ctx, `
		INSERT INTO host_operating_system (host_id, os_id)
		VALUES (?, ?)
	`, hostID, osID)
	require.NoError(t, err)

	return uint(hostID)
}

// insertTeamWithRecoveryLock inserts a team with enable_recovery_lock_password setting.
func (env *testEnv) insertTeamWithRecoveryLock(t *testing.T, name string, enabled bool) uint {
	t.Helper()
	ctx := t.Context()

	config := `{"mdm": {"enable_recovery_lock_password": false}}`
	if enabled {
		config = `{"mdm": {"enable_recovery_lock_password": true}}`
	}

	result, err := env.db.ExecContext(ctx, `
		INSERT INTO teams (name, config, created_at)
		VALUES (?, ?, NOW())
	`, name, config)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

// insertNanoDevice inserts a nano_device record (required for nano_enrollments FK).
func (env *testEnv) insertNanoDevice(t *testing.T, deviceID string) {
	t.Helper()
	ctx := t.Context()

	_, err := env.db.ExecContext(ctx, `
		INSERT INTO nano_devices (id, authenticate, authenticate_at, created_at, updated_at)
		VALUES (?, '<?xml version="1.0"?><plist></plist>', NOW(), NOW(), NOW())
	`, deviceID)
	require.NoError(t, err)
}

// insertNanoEnrollment inserts a nano_enrollment record for MDM enrollment.
// Also inserts the required nano_device record.
func (env *testEnv) insertNanoEnrollment(t *testing.T, deviceID, enrollmentType string, enabled bool) {
	t.Helper()
	ctx := t.Context()

	// Insert nano_device first (required by FK constraint)
	env.insertNanoDevice(t, deviceID)

	_, err := env.db.ExecContext(ctx, `
		INSERT INTO nano_enrollments (id, device_id, user_id, type, topic, push_magic, token_hex, enabled, token_update_tally, last_seen_at, created_at, updated_at)
		VALUES (?, ?, NULL, ?, 'topic', 'push_magic', 'token_hex', ?, 0, NOW(), NOW(), NOW())
	`, deviceID, deviceID, enrollmentType, enabled)
	require.NoError(t, err)
}

// insertNanoCommand inserts a nano_command record.
func (env *testEnv) insertNanoCommand(t *testing.T, commandUUID, requestType string) {
	t.Helper()
	ctx := t.Context()

	_, err := env.db.ExecContext(ctx, `
		INSERT INTO nano_commands (command_uuid, request_type, command, created_at, updated_at)
		VALUES (?, ?, '<?xml version="1.0"?><plist></plist>', NOW(), NOW())
	`, commandUUID, requestType)
	require.NoError(t, err)
}

// insertNanoCommandResult inserts a nano_command_results record.
func (env *testEnv) insertNanoCommandResult(t *testing.T, commandUUID, deviceID, status, result string) {
	t.Helper()
	ctx := t.Context()

	_, err := env.db.ExecContext(ctx, `
		INSERT INTO nano_command_results (command_uuid, id, status, result, not_now_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, NULL, NOW(), NOW())
	`, commandUUID, deviceID, status, result)
	require.NoError(t, err)
}

// setAppConfigRecoveryLock sets the enable_recovery_lock_password setting in appconfig.
func (env *testEnv) setAppConfigRecoveryLock(t *testing.T, enabled bool) {
	t.Helper()
	ctx := t.Context()

	config := `{"mdm": {"enable_recovery_lock_password": false}}`
	if enabled {
		config = `{"mdm": {"enable_recovery_lock_password": true}}`
	}

	_, err := env.db.ExecContext(ctx, `
		INSERT INTO app_config_json (json_value)
		VALUES (?)
		ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)
	`, config)
	require.NoError(t, err)
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

func TestRecoveryLockStatusMethods(t *testing.T) {
	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "recoverykeypassword_status_" + t.Name(),
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
		{"SetRecoveryLockPending", testSetRecoveryLockPending},
		{"SetRecoveryLockVerifying", testSetRecoveryLockVerifying},
		{"SetRecoveryLockVerified", testSetRecoveryLockVerified},
		{"SetRecoveryLockFailed", testSetRecoveryLockFailed},
		{"GetHostIDByVerifyCommandUUID", testGetHostIDByVerifyCommandUUID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.truncateTables(t)
			c.fn(t, env)
		})
	}
}

func testSetRecoveryLockPending(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-pending")

	// Set password first (creates the record)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Set pending status
	err = env.ds.SetRecoveryLockPending(ctx, hostID, "test-set-uuid-123")
	require.NoError(t, err)

	// Verify status
	var status string
	err = env.db.GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_id = ?", hostID)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)
}

func testSetRecoveryLockVerifying(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-verifying")

	// Set password first (creates the record)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Set verifying status
	err = env.ds.SetRecoveryLockVerifying(ctx, hostID, "test-verify-uuid-456")
	require.NoError(t, err)

	// Verify status and verify_command_uuid
	var status, verifyUUID string
	err = env.db.GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_id = ?", hostID)
	require.NoError(t, err)
	assert.Equal(t, "verifying", status)

	err = env.db.GetContext(ctx, &verifyUUID, "SELECT verify_command_uuid FROM host_recovery_key_passwords WHERE host_id = ?", hostID)
	require.NoError(t, err)
	assert.Equal(t, "test-verify-uuid-456", verifyUUID)
}

func testSetRecoveryLockVerified(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-verified")

	// Set password first (creates the record)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Set verified status
	err = env.ds.SetRecoveryLockVerified(ctx, hostID)
	require.NoError(t, err)

	// Verify status
	var status string
	err = env.db.GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_id = ?", hostID)
	require.NoError(t, err)
	assert.Equal(t, "verified", status)
}

func testSetRecoveryLockFailed(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-failed")

	// Set password first (creates the record)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Set failed status
	err = env.ds.SetRecoveryLockFailed(ctx, hostID, "test error message")
	require.NoError(t, err)

	// Verify status and error message
	var status, errorMsg string
	err = env.db.GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_id = ?", hostID)
	require.NoError(t, err)
	assert.Equal(t, "failed", status)

	err = env.db.GetContext(ctx, &errorMsg, "SELECT error_message FROM host_recovery_key_passwords WHERE host_id = ?", hostID)
	require.NoError(t, err)
	assert.Equal(t, "test error message", errorMsg)
}

func testGetHostIDByVerifyCommandUUID(t *testing.T, env *testEnv) {
	ctx := t.Context()
	hostID := env.insertHost(t, "test-host-verify-cmd")

	// Set password first (creates the record)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)

	// Set verifying status with a command UUID
	err = env.ds.SetRecoveryLockVerifying(ctx, hostID, "unique-verify-cmd-uuid")
	require.NoError(t, err)

	// Get host ID by command UUID
	foundHostID, err := env.ds.GetHostIDByVerifyCommandUUID(ctx, "unique-verify-cmd-uuid")
	require.NoError(t, err)
	assert.Equal(t, hostID, foundHostID)

	// Test not found
	_, err = env.ds.GetHostIDByVerifyCommandUUID(ctx, "non-existent-uuid")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func TestGetPendingRecoveryLockHosts(t *testing.T) {
	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "recoverykeypassword_pending_" + t.Name(),
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
		{"NoPendingHosts", testGetPendingRecoveryLockHostsEmpty},
		{"PendingHostNoResult", testGetPendingRecoveryLockHostsNoResult},
		{"PendingHostAcknowledged", testGetPendingRecoveryLockHostsAcknowledged},
		{"PendingHostError", testGetPendingRecoveryLockHostsError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.truncateTables(t)
			c.fn(t, env)
		})
	}
}

func testGetPendingRecoveryLockHostsEmpty(t *testing.T, env *testEnv) {
	ctx := t.Context()

	hosts, err := env.ds.GetPendingRecoveryLockHosts(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetPendingRecoveryLockHostsNoResult(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create a host with pending status but no command result yet
	hostID := env.insertHostFull(t, "pending-host", "uuid-pending-1", "darwin", "macOS 15.7", nil)

	// Set password and pending status
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockPending(ctx, hostID, "set-cmd-uuid-1")
	require.NoError(t, err)

	// Get pending hosts - should return host with empty status
	hosts, err := env.ds.GetPendingRecoveryLockHosts(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
	assert.Equal(t, "uuid-pending-1", hosts[0].HostUUID)
	assert.Equal(t, "set-cmd-uuid-1", hosts[0].SetCommandUUID)
	assert.Empty(t, hosts[0].SetCommandStatus)
}

func testGetPendingRecoveryLockHostsAcknowledged(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create a host with pending status and acknowledged command result
	hostID := env.insertHostFull(t, "ack-host", "uuid-ack-1", "darwin", "macOS 15.7", nil)

	// Set password and pending status
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockPending(ctx, hostID, "set-cmd-uuid-ack")
	require.NoError(t, err)

	// Insert enrollment, command and result
	env.insertNanoEnrollment(t, "uuid-ack-1", "Device", true)
	env.insertNanoCommand(t, "set-cmd-uuid-ack", "SetRecoveryLock")
	env.insertNanoCommandResult(t, "set-cmd-uuid-ack", "uuid-ack-1", "Acknowledged", "<?xml version=\"1.0\"?><plist></plist>")

	// Get pending hosts
	hosts, err := env.ds.GetPendingRecoveryLockHosts(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
	assert.Equal(t, "Acknowledged", hosts[0].SetCommandStatus)
}

func testGetPendingRecoveryLockHostsError(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create a host with pending status and error command result
	hostID := env.insertHostFull(t, "err-host", "uuid-err-1", "darwin", "macOS 15.7", nil)

	// Set password and pending status
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockPending(ctx, hostID, "set-cmd-uuid-err")
	require.NoError(t, err)

	// Insert enrollment, command and error result
	env.insertNanoEnrollment(t, "uuid-err-1", "Device", true)
	env.insertNanoCommand(t, "set-cmd-uuid-err", "SetRecoveryLock")
	env.insertNanoCommandResult(t, "set-cmd-uuid-err", "uuid-err-1", "Error", "<?xml version=\"1.0\"?><plist><key>ErrorChain</key></plist>")

	// Get pending hosts
	hosts, err := env.ds.GetPendingRecoveryLockHosts(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
	assert.Equal(t, "Error", hosts[0].SetCommandStatus)
	assert.Contains(t, hosts[0].SetCommandErrorInfo, "ErrorChain")
}

func TestGetHostsForRecoveryLockAction(t *testing.T) {
	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "recoverykeypassword_action_" + t.Name(),
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
		{"NoEligibleHosts", testGetHostsForRecoveryLockActionEmpty},
		{"EligibleMacOS15", testGetHostsForRecoveryLockActionMacOS15},
		{"EligibleMacOS11_5", testGetHostsForRecoveryLockActionMacOS11_5},
		{"EligibleMacOS11_5_2", testGetHostsForRecoveryLockActionMacOS11_5_2},
		{"IneligibleMacOS11_4", testGetHostsForRecoveryLockActionMacOS11_4},
		{"IneligibleMacOS10", testGetHostsForRecoveryLockActionMacOS10},
		{"IneligibleTeamDisabled", testGetHostsForRecoveryLockActionTeamDisabled},
		{"IneligibleNotEnrolled", testGetHostsForRecoveryLockActionNotEnrolled},
		{"IneligibleNotDarwin", testGetHostsForRecoveryLockActionNotDarwin},
		{"IneligiblePending", testGetHostsForRecoveryLockActionPending},
		{"IneligibleVerifying", testGetHostsForRecoveryLockActionVerifying},
		{"IneligibleVerified", testGetHostsForRecoveryLockActionVerified},
		{"IneligibleFailed", testGetHostsForRecoveryLockActionFailed},
		{"EligibleNoTeamEnabled", testGetHostsForRecoveryLockActionNoTeamEnabled},
		{"IneligibleNoTeamDisabled", testGetHostsForRecoveryLockActionNoTeamDisabled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.truncateTables(t)
			// Ensure app_config_json has a row (required by CROSS JOIN in query).
			// Individual tests can override this with their own settings.
			env.setAppConfigRecoveryLock(t, false)
			c.fn(t, env)
		})
	}
}

func testGetHostsForRecoveryLockActionEmpty(t *testing.T, env *testEnv) {
	ctx := t.Context()

	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionMacOS15(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-macos15", true)

	// Create eligible macOS 15.7 host
	hostID := env.insertHostFull(t, "macos15-host", "uuid-macos15", "darwin", "macOS 15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-macos15", "Device", true)

	// Get eligible hosts
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
	assert.Equal(t, "uuid-macos15", hosts[0].HostUUID)
}

func testGetHostsForRecoveryLockActionMacOS11_5(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-macos11-5", true)

	// Create eligible macOS 11.5 host (minimum supported version)
	hostID := env.insertHostFull(t, "macos11-5-host", "uuid-macos11-5", "darwin", "macOS 11.5", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-macos11-5", "Device", true)

	// Get eligible hosts
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
}

func testGetHostsForRecoveryLockActionMacOS11_5_2(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-macos11-5-2", true)

	// Create eligible macOS 11.5.2 host (patch version)
	hostID := env.insertHostFull(t, "macos11-5-2-host", "uuid-macos11-5-2", "darwin", "macOS 11.5.2", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-macos11-5-2", "Device", true)

	// Get eligible hosts
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
}

func testGetHostsForRecoveryLockActionMacOS11_4(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-macos11-4", true)

	// Create ineligible macOS 11.4 host (below minimum)
	_ = env.insertHostFull(t, "macos11-4-host", "uuid-macos11-4", "darwin", "macOS 11.4", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-macos11-4", "Device", true)

	// Should not return host
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionMacOS10(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-macos10", true)

	// Create ineligible macOS 10.15 host
	_ = env.insertHostFull(t, "macos10-host", "uuid-macos10", "darwin", "macOS 10.15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-macos10", "Device", true)

	// Should not return host
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionTeamDisabled(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock DISABLED
	teamID := env.insertTeamWithRecoveryLock(t, "team-disabled", false)

	// Create macOS host in team with disabled setting
	_ = env.insertHostFull(t, "disabled-team-host", "uuid-disabled-team", "darwin", "macOS 15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-disabled-team", "Device", true)

	// Should not return host
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionNotEnrolled(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-not-enrolled", true)

	// Create macOS host without MDM enrollment
	_ = env.insertHostFull(t, "not-enrolled-host", "uuid-not-enrolled", "darwin", "macOS 15.7", &teamID)

	// No nano_enrollment record

	// Should not return host
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionNotDarwin(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-not-darwin", true)

	// Create Windows host
	_ = env.insertHostFull(t, "windows-host", "uuid-windows", "windows", "Windows 11", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-windows", "Device", true)

	// Should not return host
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionPending(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-pending", true)

	// Create eligible macOS host
	hostID := env.insertHostFull(t, "pending-host", "uuid-pending", "darwin", "macOS 15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-pending", "Device", true)

	// Set password and mark as pending (waiting for SetRecoveryLock to be acknowledged)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockPending(ctx, hostID, "set-cmd-uuid-pending")
	require.NoError(t, err)

	// Should NOT return host (pending means command is in progress)
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionVerifying(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-verifying", true)

	// Create eligible macOS host
	hostID := env.insertHostFull(t, "verifying-host", "uuid-verifying", "darwin", "macOS 15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-verifying", "Device", true)

	// Set password and mark as verifying (waiting for VerifyRecoveryLock to be acknowledged)
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockVerifying(ctx, hostID, "verify-cmd-uuid-verifying")
	require.NoError(t, err)

	// Should NOT return host (verifying means command is in progress)
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionVerified(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-verified", true)

	// Create eligible macOS host
	hostID := env.insertHostFull(t, "verified-host", "uuid-verified", "darwin", "macOS 15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-verified", "Device", true)

	// Set password and mark as verified
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockVerified(ctx, hostID)
	require.NoError(t, err)

	// Should not return host (already verified)
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionFailed(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := env.insertTeamWithRecoveryLock(t, "team-failed", true)

	// Create eligible macOS host
	hostID := env.insertHostFull(t, "failed-host", "uuid-failed", "darwin", "macOS 15.7", &teamID)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-failed", "Device", true)

	// Set password and mark as failed
	_, err := env.ds.SetHostRecoveryKeyPassword(ctx, hostID)
	require.NoError(t, err)
	err = env.ds.SetRecoveryLockFailed(ctx, hostID, "previous error")
	require.NoError(t, err)

	// Should NOT return host (failed hosts don't auto-retry)
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func testGetHostsForRecoveryLockActionNoTeamEnabled(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Set appconfig with recovery lock enabled
	env.setAppConfigRecoveryLock(t, true)

	// Create eligible macOS host with no team
	hostID := env.insertHostFull(t, "no-team-host", "uuid-no-team", "darwin", "macOS 15.7", nil)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-no-team", "Device", true)

	// Get eligible hosts
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, hostID, hosts[0].HostID)
	assert.Equal(t, "uuid-no-team", hosts[0].HostUUID)
}

func testGetHostsForRecoveryLockActionNoTeamDisabled(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// Set appconfig with recovery lock disabled
	env.setAppConfigRecoveryLock(t, false)

	// Create eligible macOS host with no team
	_ = env.insertHostFull(t, "no-team-disabled-host", "uuid-no-team-disabled", "darwin", "macOS 15.7", nil)

	// Create MDM enrollment
	env.insertNanoEnrollment(t, "uuid-no-team-disabled", "Device", true)

	// Should not return host (appconfig has recovery lock disabled)
	hosts, err := env.ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}
