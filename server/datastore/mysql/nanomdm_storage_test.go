package mysql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNanoMDMStorage(t *testing.T) {
	ds := CreateMySQLDS(t)
	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestEnqueueDeviceLockCommand", testEnqueueDeviceLockCommand},
		{"TestGetPendingLockCommand", testGetPendingLockCommand},
		{"TestEnqueueDeviceLockCommandRaceCondition", testEnqueueDeviceLockCommandRaceCondition},
		{"TestEnqueueDeviceUnlockCommand", testEnqueueDeviceUnlockCommand},
		{"TestStoreAuthenticatePreservesBootstrapTokenDuringSCEPRenewal", testStoreAuthenticatePreservesBootstrapTokenDuringSCEPRenewal},
		{"TestRetrievePushCert", testRetrievePushCert},
		{"TestIsPushCertStale", testIsPushCertStale},
		{"TestStorePushCert", testStorePushCert},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testEnqueueDeviceLockCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	// no commands yet
	res, err := ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, res)

	cmd := &mdm.Command{}
	cmd.CommandUUID = "cmd-uuid"
	cmd.Command.RequestType = "DeviceLock"
	cmd.Raw = []byte("<?xml")

	err = ns.EnqueueDeviceLockCommand(ctx, host, cmd, "123456")
	require.NoError(t, err)

	// command has no results yet, so the status is empty
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 1)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}

	require.ElementsMatch(t, []*fleet.MDMAppleCommand{
		{
			DeviceID:    host.UUID,
			CommandUUID: "cmd-uuid",
			Status:      "Pending",
			RequestType: "DeviceLock",
			Hostname:    host.Hostname,
			TeamID:      nil,
		},
	}, res)

	status, err := ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "cmd-uuid", status.LockMDMCommand.CommandUUID)
	require.Equal(t, "123456", status.UnlockPIN)
}

func testEnqueueDeviceUnlockCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "ios",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	// no commands yet
	res, err := ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, res)

	cmd := &mdm.Command{}
	cmd.CommandUUID = "cmd-uuid"
	cmd.Command.RequestType = "DisableLostMode"
	cmd.Raw = []byte("<?xml")

	err = ns.EnqueueDeviceUnlockCommand(ctx, host, cmd)
	require.NoError(t, err)

	// command has no results yet, so the status is empty
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 1)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}

	require.ElementsMatch(t, []*fleet.MDMAppleCommand{
		{
			DeviceID:    host.UUID,
			CommandUUID: "cmd-uuid",
			Status:      "Pending",
			RequestType: "DisableLostMode",
			Hostname:    host.Hostname,
			TeamID:      nil,
		},
	}, res)

	status, err := ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "cmd-uuid", status.UnlockMDMCommand.CommandUUID)
}

func testGetPendingLockCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host2-name",
		OsqueryHostID: ptr.String("1338"),
		NodeKey:       ptr.String("1338"),
		UUID:          "test-uuid-2",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	// Test 1: No pending commands should return nil
	cmd, pin, err := ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.Nil(t, cmd)
	require.Empty(t, pin)

	// Test 2: Enqueue a lock command
	lockCmd := &mdm.Command{}
	lockCmd.CommandUUID = "lock-cmd-uuid"
	lockCmd.Command.RequestType = "DeviceLock"
	lockCmd.Raw = []byte("<?xml")

	err = ns.EnqueueDeviceLockCommand(ctx, host, lockCmd, "654321")
	require.NoError(t, err)

	// Test 3: Should find the pending command
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "lock-cmd-uuid", cmd.CommandUUID)
	require.Equal(t, "DeviceLock", cmd.Command.RequestType)
	require.Equal(t, "654321", pin)

	// Test 4: Multiple commands should fail due to conflict
	lockCmd2 := &mdm.Command{}
	lockCmd2.CommandUUID = "lock-cmd-uuid-2"
	lockCmd2.Command.RequestType = "DeviceLock"
	lockCmd2.Raw = []byte("<?xml2")

	// This should fail with conflict error since a lock is already pending
	err = ns.EnqueueDeviceLockCommand(ctx, host, lockCmd2, "111111")
	require.Error(t, err)
	require.True(t, isConflict(err), "Should get conflict error for duplicate lock")

	// The pending command should still be the original one
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "lock-cmd-uuid", cmd.CommandUUID)
	require.Equal(t, "654321", pin)

	// Test 5: After acknowledgment, should not find the command
	// First acknowledge the existing command
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO nano_command_results (id, command_uuid, status, result)
		VALUES (?, ?, 'Acknowledged', '<?xml version="1.0"?><plist></plist>')`,
		host.UUID, "lock-cmd-uuid")
	require.NoError(t, err)

	// Now no pending command should exist
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.Nil(t, cmd)
	require.Empty(t, pin)

	// Test 6: After acknowledgment, the lock_ref still exists in host_mdm_actions
	// This is expected behavior - the device remains locked until manually unlocked
	// Therefore, attempting to create a new lock command should still fail
	lockCmd3 := &mdm.Command{}
	lockCmd3.CommandUUID = "lock-cmd-uuid-3"
	lockCmd3.Command.RequestType = "DeviceLock"
	lockCmd3.Raw = []byte("<?xml3")

	err = ns.EnqueueDeviceLockCommand(ctx, host, lockCmd3, "222222")
	require.Error(t, err)
	require.True(t, isConflict(err), "Should still get conflict error since device is locked")

	// No pending command should exist since the previous was acknowledged
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.Nil(t, cmd)
	require.Empty(t, pin)
}

// testStoreAuthenticatePreservesBootstrapTokenDuringSCEPRenewal verifies that
// StoreAuthenticate does NOT clear the bootstrap token when a SCEP renewal is
// in progress (renew_command_uuid is set in nano_cert_auth_associations), and
// DOES clear it on a normal (re-)enrollment.
// See https://github.com/fleetdm/fleet/issues/41167
func testStoreAuthenticatePreservesBootstrapTokenDuringSCEPRenewal(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	deviceUUID := uuid.NewString()

	// --- Set up device with a bootstrap token ---

	// Insert into nano_devices with a bootstrap token.
	bootstrapToken := base64.StdEncoding.EncodeToString([]byte("my-secret-bootstrap-token"))
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO nano_devices (id, serial_number, authenticate, authenticate_at, bootstrap_token_b64, bootstrap_token_at)
		 VALUES (?, 'SERIAL1', 'auth-raw', CURRENT_TIMESTAMP, ?, CURRENT_TIMESTAMP)`,
		deviceUUID, bootstrapToken)
	require.NoError(t, err)

	// Insert a nano_enrollment so cert auth association can reference it.
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, token_update_tally, last_seen_at)
		 VALUES (?, ?, 'Device', 'topic', 'magic', 'deadbeef', 1, NOW())`,
		deviceUUID, deviceUUID)
	require.NoError(t, err)

	// Insert cert auth association (no SCEP renewal yet).
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO nano_cert_auth_associations (id, sha256) VALUES (?, '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef')`,
		deviceUUID)
	require.NoError(t, err)

	// Helper to read bootstrap_token_b64 from nano_devices.
	getBootstrapToken := func() sql.NullString {
		var token sql.NullString
		err := ds.writer(ctx).QueryRowContext(ctx,
			`SELECT bootstrap_token_b64 FROM nano_devices WHERE id = ?`, deviceUUID,
		).Scan(&token)
		require.NoError(t, err)
		return token
	}

	// Verify token is set.
	token := getBootstrapToken()
	require.True(t, token.Valid)
	require.Equal(t, bootstrapToken, token.String)

	// --- Case 1: Normal re-enrollment (no SCEP renewal) should clear the bootstrap token ---

	authMsg := &mdm.Authenticate{
		Enrollment: mdm.Enrollment{UDID: deviceUUID},
		Raw:        []byte("auth-raw-reenroll"),
	}
	authMsg.SerialNumber = "SERIAL1"
	req := &mdm.Request{
		EnrollID: &mdm.EnrollID{ID: deviceUUID, Type: mdm.Device},
		Context:  ctx,
	}

	err = ns.StoreAuthenticate(req, authMsg)
	require.NoError(t, err)

	token = getBootstrapToken()
	require.False(t, token.Valid, "bootstrap token should be cleared on normal re-enrollment")

	// --- Restore the bootstrap token for the next test case ---

	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE nano_devices SET bootstrap_token_b64 = ?, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = ?`,
		bootstrapToken, deviceUUID)
	require.NoError(t, err)

	// --- Case 2: SCEP renewal in progress should preserve the bootstrap token ---

	// Simulate SCEP renewal by inserting a nano_command and setting renew_command_uuid.
	renewCmdUUID := uuid.NewString()
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, 'InstallProfile', '<?xml version="1.0"?>')`,
		renewCmdUUID)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE nano_cert_auth_associations SET renew_command_uuid = ? WHERE id = ?`,
		renewCmdUUID, deviceUUID)
	require.NoError(t, err)

	// Now call StoreAuthenticate again — this simulates the device checking in during SCEP renewal.
	authMsg2 := &mdm.Authenticate{
		Enrollment: mdm.Enrollment{UDID: deviceUUID},
		Raw:        []byte("auth-raw-scep-renewal"),
	}
	authMsg2.SerialNumber = "SERIAL1"

	err = ns.StoreAuthenticate(req, authMsg2)
	require.NoError(t, err)

	token = getBootstrapToken()
	require.True(t, token.Valid, "bootstrap token should be preserved during SCEP renewal")
	require.Equal(t, bootstrapToken, token.String)

	// --- Case 3: After SCEP renewal completes (renew_command_uuid cleared), token should be cleared again ---

	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE nano_cert_auth_associations SET renew_command_uuid = NULL WHERE id = ?`,
		deviceUUID)
	require.NoError(t, err)

	authMsg3 := &mdm.Authenticate{
		Enrollment: mdm.Enrollment{UDID: deviceUUID},
		Raw:        []byte("auth-raw-post-renewal"),
	}
	authMsg3.SerialNumber = "SERIAL1"

	err = ns.StoreAuthenticate(req, authMsg3)
	require.NoError(t, err)

	token = getBootstrapToken()
	require.False(t, token.Valid, "bootstrap token should be cleared after SCEP renewal completes")
}

func testEnqueueDeviceLockCommandRaceCondition(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		UUID:          "test-host-race-" + uuid.NewString(),
		Platform:      "darwin",
		OsqueryHostID: ptr.String("test-osquery-id"),
		NodeKey:       ptr.String("test-node-key"),
		Hostname:      "test-host.local",
	})
	require.NoError(t, err)

	// Enable MDM for the host
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://test.local", false, "test-ref", "", false)
	require.NoError(t, err)

	// Create nano_devices record first
	deviceID := "device-" + host.UUID
	_, err = ds.writer(ctx).Exec(`
		INSERT INTO nano_devices (id, authenticate, token_update) VALUES (?, 'Authenticate', 0)`, deviceID)
	require.NoError(t, err)

	// Create nano_enrollments record (required for MDM commands)
	_, err = ds.writer(ctx).Exec(`
		INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at)
		VALUES (?, ?, 'Device', 'com.apple.mgmt.test', 'test-magic', 'deadbeef', NOW())`,
		host.UUID, deviceID)
	require.NoError(t, err)

	// Create NanoMDMStorage
	storage := &NanoMDMStorage{
		db:     ds.writer(ctx),
		logger: slog.New(slog.DiscardHandler),
		ds:     ds,
	}

	// Number of concurrent lock attempts
	numGoroutines := 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track successful locks
	successCount := 0
	var successMu sync.Mutex

	// Track conflict errors
	conflictCount := 0

	// Collect all PINs that were generated
	var pins []string

	// Barrier to ensure all goroutines start at the same time
	barrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			// Wait for barrier
			<-barrier

			// Create a unique command for this goroutine
			cmdUUID := fmt.Sprintf("test-lock-%03d", idx)
			pin := fmt.Sprintf("%06d", 100000+idx) // Unique PIN for each request

			cmd := &mdm.Command{
				CommandUUID: cmdUUID,
				Command: struct {
					RequestType string
				}{
					RequestType: "DeviceLock",
				},
				Raw: []byte(fmt.Sprintf(`<?xml version="1.0"?><plist><dict><key>PIN</key><string>%s</string></dict></plist>`, pin)),
			}

			// Try to enqueue the lock command
			err := storage.EnqueueDeviceLockCommand(ctx, host, cmd, pin)

			switch {
			case err == nil:
				successMu.Lock()
				successCount++
				pins = append(pins, pin)
				successMu.Unlock()
			case isConflict(err):
				successMu.Lock()
				conflictCount++
				successMu.Unlock()
			default:
				// Unexpected error
				t.Logf("Request %d got unexpected error: %v", idx, err)
			}
		}(i)
	}

	// Release all goroutines at once
	close(barrier)

	// Wait for all to complete
	wg.Wait()

	// Check the database state

	// 1. Count how many DeviceLock commands were created
	var commandCount int
	err = ds.writer(ctx).Get(&commandCount,
		`SELECT COUNT(*) FROM nano_commands WHERE command_uuid LIKE 'test-lock-%'`)
	require.NoError(t, err)

	// 2. Check what's stored in host_mdm_actions
	var storedPIN string
	var lockRef string
	err = ds.writer(ctx).QueryRow(
		`SELECT COALESCE(unlock_pin, ''), COALESCE(lock_ref, '') FROM host_mdm_actions WHERE host_id = ?`,
		host.ID).Scan(&storedPIN, &lockRef)
	require.NoError(t, err)

	// Log the results
	t.Logf("===== RACE CONDITION TEST RESULTS =====")
	t.Logf("Concurrent requests sent: %d", numGoroutines)
	t.Logf("Successful lock commands: %d", successCount)
	t.Logf("Conflict errors: %d", conflictCount)
	t.Logf("Commands in nano_commands table: %d", commandCount)
	t.Logf("Final PIN stored in database: %s", storedPIN)
	t.Logf("Final lock_ref in database: %s", lockRef)

	// Assertions - only one lock should succeed
	require.Equal(t, 1, successCount, "Only one lock command should succeed")
	require.Equal(t, numGoroutines-1, conflictCount, "All other requests should get conflict error")
	require.Equal(t, 1, commandCount, "Only one command should be in nano_commands table")
	require.Len(t, pins, 1, "Only one PIN should be generated")
	require.Equal(t, pins[0], storedPIN, "Stored PIN should match the successful request")
}

func testRetrievePushCert(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ds.HardDeleteMDMConfigAsset(ctx, fleet.MDMAssetAPNSCert)
		_ = ds.HardDeleteMDMConfigAsset(ctx, fleet.MDMAssetAPNSKey)
		pushCertStaleness = nil
	})

	apnsCert, apnsKey, err := GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)

	err = ds.InsertMDMConfigAssets(ctx, []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetAPNSCert, Value: apnsCert},
		{Name: fleet.MDMAssetAPNSKey, Value: apnsKey},
	}, nil)
	require.NoError(t, err)

	cert, hash, err := ns.RetrievePushCert(ctx, "com.apple.mgmt.test")
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.NotEmpty(t, hash)
	require.NotNil(t, pushCertStaleness)
	require.Equal(t, hash, pushCertStaleness.hash)
	assert.WithinDuration(t, time.Now(), pushCertStaleness.updatedAt, 500*time.Millisecond)
	oldUpdatedAt := pushCertStaleness.updatedAt

	// Retrieve again with same cert - should not update staleness
	cert2, hash2, err := ns.RetrievePushCert(ctx, "com.apple.mgmt.test")
	require.NoError(t, err)
	require.NotNil(t, cert2)
	require.Equal(t, hash, hash2)
	require.Equal(t, oldUpdatedAt, pushCertStaleness.updatedAt)
	stalenessHash := pushCertStaleness.hash

	// Insert a new cert with different content to simulate cert rotation
	newApnsCert, newApnsKey, err := GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	require.NoError(t, ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{Name: fleet.MDMAssetAPNSCert, Value: newApnsCert}))
	require.NoError(t, ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{Name: fleet.MDMAssetAPNSKey, Value: newApnsKey}))

	cert3, hash3, err := ns.RetrievePushCert(ctx, "com.apple.mgmt.test")
	require.NoError(t, err)
	require.NotNil(t, cert3)
	require.NotEqual(t, hash, hash3)
	require.Equal(t, hash3, pushCertStaleness.hash)
	assert.WithinDuration(t, time.Now(), pushCertStaleness.updatedAt, 500*time.Millisecond)
	require.NotEqual(t, oldUpdatedAt, pushCertStaleness.updatedAt)
	require.NotEqual(t, stalenessHash, pushCertStaleness.hash)
}

func testIsPushCertStale(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ds.HardDeleteMDMConfigAsset(ctx, fleet.MDMAssetAPNSCert)
		_ = ds.HardDeleteMDMConfigAsset(ctx, fleet.MDMAssetAPNSKey)
		pushCertStaleness = nil
	})

	// Initially there is no cert, so it should be considered stale
	stale, err := ns.IsPushCertStale(ctx, "com.apple.mgmt.test", "nonexistent-token")
	require.NoError(t, err)
	require.True(t, stale)

	// Insert a cert
	apnsCert, apnsKey, err := GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)

	err = ds.InsertMDMConfigAssets(ctx, []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetAPNSCert, Value: apnsCert},
		{Name: fleet.MDMAssetAPNSKey, Value: apnsKey},
	}, nil)
	require.NoError(t, err)

	// Retrieve the cert to get the current hash
	cert, hash, err := ns.RetrievePushCert(ctx, "com.apple.mgmt.test")
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.NotEmpty(t, hash)

	// Check staleness with correct token - should not be stale
	stale, err = ns.IsPushCertStale(ctx, "com.apple.mgmt.test", hash)
	require.NoError(t, err)
	require.False(t, stale)

	// Check staleness with incorrect token - should be stale
	stale, err = ns.IsPushCertStale(ctx, "com.apple.mgmt.test", "invalid-token")
	require.NoError(t, err)
	require.True(t, stale)

	// Insert a new cert to simulate rotation
	newApnsCert, newApnsKey, err := GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	require.NoError(t, ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{Name: fleet.MDMAssetAPNSCert, Value: newApnsCert}))
	require.NoError(t, ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{Name: fleet.MDMAssetAPNSKey, Value: newApnsKey}))

	// Check staleness with old token - should not be stale since under 5 minutes
	stale, err = ns.IsPushCertStale(ctx, "com.apple.mgmt.test", hash)
	require.NoError(t, err)
	require.False(t, stale, "We allow the wrong cert for up to 5 minutes after rotation")
	require.WithinDuration(t, time.Now(), pushCertStaleness.updatedAt, 5*time.Minute)

	// Fake 5 minutes passing
	pushCertStaleness.updatedAt = time.Now().Add(-6 * time.Minute)

	// Check staleness with old token - should be stale since cert is now old
	stale, err = ns.IsPushCertStale(ctx, "com.apple.mgmt.test", hash)
	require.NoError(t, err)
	require.True(t, stale)
}

// ensure we always use our custom MDM config assets impl.
func testStorePushCert(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	err = ns.StorePushCert(ctx, nil, nil)
	require.Error(t, err)
	require.Equal(t, "please use fleet.Datastore to manage MDM assets", err.Error())
}
