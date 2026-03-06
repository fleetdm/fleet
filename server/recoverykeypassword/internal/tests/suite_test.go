package tests

import (
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword/internal/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// testPrivateKey is a 32-byte key for AES-256 encryption in tests.
// This is a synthetic test-only value, not a real secret.
const testPrivateKey = "FLEET_TEST_KEY_DO_NOT_USE_IN_PRD"

// integrationTestSuite holds all dependencies for integration tests.
type integrationTestSuite struct {
	db        *sqlx.DB
	logger    *slog.Logger
	ds        recoverykeypassword.Datastore
	commander *mockCommander
}

// setupIntegrationTest creates a new test suite with a real database.
func setupIntegrationTest(t *testing.T) *integrationTestSuite {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "rkp_integration_" + t.Name(),
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

	ds := mysql.NewDatastore(conns, logger)
	commander := &mockCommander{}

	return &integrationTestSuite{
		db:        db,
		logger:    logger,
		ds:        ds,
		commander: commander,
	}
}

// truncateTables clears all test data between tests.
func (s *integrationTestSuite) truncateTables(t *testing.T) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, s.db, s.logger, nil,
		"host_recovery_key_passwords", "host_operating_system", "operating_systems",
		"hosts", "teams", "nano_enrollments", "nano_devices", "nano_command_results", "nano_commands",
		"app_config_json")

	// Re-insert default app_config_json (required by CROSS JOIN in queries)
	_, err := s.db.ExecContext(t.Context(), `
		INSERT INTO app_config_json (json_value) VALUES ('{}')
	`)
	require.NoError(t, err)
}

// resetCommander resets the mock commander between tests.
func (s *integrationTestSuite) resetCommander() {
	s.commander = &mockCommander{}
}

// insertHostFull inserts a host with all fields needed for recovery lock testing.
func (s *integrationTestSuite) insertHostFull(t *testing.T, hostname, uuid, platform, osVersion string, teamID *uint) uint {
	t.Helper()
	ctx := t.Context()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO hosts (hostname, uuid, platform, os_version, team_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`, hostname, uuid, platform, osVersion, teamID)
	require.NoError(t, err)

	hostID, err := result.LastInsertId()
	require.NoError(t, err)

	// Parse osVersion (e.g., "macOS 15.7" -> name="macOS", version="15.7")
	osName := osVersion
	osVersionNum := ""
	if platform == "darwin" && len(osVersion) > 6 && osVersion[:6] == "macOS " {
		osName = "macOS"
		osVersionNum = osVersion[6:]
	}

	// Insert into operating_systems (or get existing)
	_, err = s.db.ExecContext(ctx, `
		INSERT IGNORE INTO operating_systems (name, version, arch, kernel_version, platform, display_version)
		VALUES (?, ?, 'x86_64', '', ?, '')
	`, osName, osVersionNum, platform)
	require.NoError(t, err)

	// Get the OS ID (whether just inserted or existing)
	var osID uint
	err = s.db.GetContext(ctx, &osID, `
		SELECT id FROM operating_systems WHERE name = ? AND version = ? AND platform = ?
	`, osName, osVersionNum, platform)
	require.NoError(t, err)

	// Link host to operating system
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO host_operating_system (host_id, os_id)
		VALUES (?, ?)
	`, hostID, osID)
	require.NoError(t, err)

	return uint(hostID)
}

// insertTeamWithRecoveryLock inserts a team with enable_recovery_lock_password setting.
func (s *integrationTestSuite) insertTeamWithRecoveryLock(t *testing.T, name string, enabled bool) uint {
	t.Helper()
	ctx := t.Context()

	config := `{"mdm": {"enable_recovery_lock_password": false}}`
	if enabled {
		config = `{"mdm": {"enable_recovery_lock_password": true}}`
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO teams (name, config)
		VALUES (?, ?)
	`, name, config)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

// insertNanoEnrollment inserts a nano_enrollment record for MDM enrollment.
// Also inserts the required nano_device record.
func (s *integrationTestSuite) insertNanoEnrollment(t *testing.T, deviceID, enrollmentType string, enabled bool) {
	t.Helper()
	ctx := t.Context()

	// Insert nano_device first (required by foreign key)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO nano_devices (id, authenticate, authenticate_at)
		VALUES (?, '{}', NOW())
	`, deviceID)
	require.NoError(t, err)

	// Insert nano_enrollment
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO nano_enrollments (id, device_id, user_id, type, topic, push_magic, token_hex, enabled, token_update_tally, last_seen_at)
		VALUES (?, ?, NULL, ?, 'topic', 'push_magic', 'token_hex', ?, 0, NOW())
	`, deviceID, deviceID, enrollmentType, enabled)
	require.NoError(t, err)
}

// insertNanoCommand inserts a nano_command record.
func (s *integrationTestSuite) insertNanoCommand(t *testing.T, commandUUID, requestType string) {
	t.Helper()
	ctx := t.Context()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO nano_commands (command_uuid, request_type, command)
		VALUES (?, ?, '<?xml version="1.0"?><plist></plist>')
	`, commandUUID, requestType)
	require.NoError(t, err)
}

// insertNanoCommandResult inserts a nano_command_results record.
func (s *integrationTestSuite) insertNanoCommandResult(t *testing.T, commandUUID, deviceID, status, result string) {
	t.Helper()
	ctx := t.Context()

	// result must start with <?xml per CHECK constraint
	if result == "" {
		result = "<?xml version=\"1.0\"?><plist></plist>"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO nano_command_results (command_uuid, id, status, result)
		VALUES (?, ?, ?, ?)
	`, commandUUID, deviceID, status, result)
	require.NoError(t, err)
}

// getRecoveryLockStatus returns the status of a host's recovery lock password.
// Returns nil if no row exists for the host. Fails the test on other DB errors.
func (s *integrationTestSuite) getRecoveryLockStatus(t *testing.T, hostID uint) *fleet.MDMDeliveryStatus {
	t.Helper()
	ctx := t.Context()

	var status *fleet.MDMDeliveryStatus
	err := s.db.GetContext(ctx, &status, `
		SELECT status FROM host_recovery_key_passwords WHERE host_id = ?
	`, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		require.NoError(t, err, "unexpected DB error querying recovery lock status")
	}
	return status
}
