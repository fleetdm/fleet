package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcilerIntegration(t *testing.T) {
	s := setupIntegrationTest(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, s *integrationTestSuite)
	}{
		{"NoHostsNeedingRecoveryLock", testNoHostsNeedingRecoveryLock},
		{"SendsSetRecoveryLockToEligibleHosts", testSendsSetRecoveryLockToEligibleHosts},
		{"ProcessesAcknowledgedSetCommand", testProcessesAcknowledgedSetCommand},
		{"ProcessesFailedSetCommand", testProcessesFailedSetCommand},
		{"SkipsHostsWithNoCommandResult", testSkipsHostsWithNoCommandResult},
		{"EnqueueErrorContinuesWithOtherHosts", testEnqueueErrorContinuesWithOtherHosts},
		{"NotificationErrorContinues", testNotificationErrorContinues},
		{"FullFlowSetThenVerify", testFullFlowSetThenVerify},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer s.truncateTables(t)
			s.resetCommander()
			c.fn(t, s)
		})
	}
}

func testNoHostsNeedingRecoveryLock(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	assert.Empty(t, s.commander.EnqueueCommandCalls)
	assert.Empty(t, s.commander.SendNotificationsCalls)
}

func testSendsSetRecoveryLockToEligibleHosts(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create two eligible macOS hosts
	s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	s.insertHostFull(t, "host2", "uuid-2", "darwin", "macOS 14.0", &teamID)
	s.insertNanoEnrollment(t, "uuid-2", "Device", true)

	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	// Should enqueue SetRecoveryLock for both hosts
	require.Len(t, s.commander.EnqueueCommandCalls, 2)
	assert.Contains(t, s.commander.EnqueueCommandCalls[0].RawCommand, "SetRecoveryLock")
	assert.Contains(t, s.commander.EnqueueCommandCalls[1].RawCommand, "SetRecoveryLock")

	// Should send notifications for both
	require.Len(t, s.commander.SendNotificationsCalls, 2)
}

func testProcessesAcknowledgedSetCommand(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create eligible host
	hostID := s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	// First reconcile: sends SetRecoveryLock
	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)
	require.Len(t, s.commander.EnqueueCommandCalls, 1)

	// Get the command UUID that was used
	setCommandUUID := extractCommandUUID(t, s.commander.EnqueueCommandCalls[0].RawCommand)

	// Simulate device acknowledging the SetRecoveryLock command
	s.insertNanoCommand(t, setCommandUUID, "SetRecoveryLock")
	s.insertNanoCommandResult(t, setCommandUUID, "uuid-1", fleet.MDMAppleStatusAcknowledged, "")

	// Reset commander to track new calls
	s.resetCommander()

	// Second reconcile: should send VerifyRecoveryLock
	err = recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	// Should have sent VerifyRecoveryLock (not SetRecoveryLock again)
	require.Len(t, s.commander.EnqueueCommandCalls, 1)
	assert.Contains(t, s.commander.EnqueueCommandCalls[0].RawCommand, "VerifyRecoveryLock")
	assert.Equal(t, []string{"uuid-1"}, s.commander.EnqueueCommandCalls[0].HostUUIDs)

	// Verify status changed to verifying
	status := s.getRecoveryLockStatus(t, hostID)
	require.NotNil(t, status)
	assert.Equal(t, fleet.MDMDeliveryVerifying, *status)
}

func testProcessesFailedSetCommand(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create eligible host
	hostID := s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	// First reconcile: sends SetRecoveryLock
	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	setCommandUUID := extractCommandUUID(t, s.commander.EnqueueCommandCalls[0].RawCommand)

	// Simulate device returning error for SetRecoveryLock command
	s.insertNanoCommand(t, setCommandUUID, "SetRecoveryLock")
	s.insertNanoCommandResult(t, setCommandUUID, "uuid-1", fleet.MDMAppleStatusError, "")

	s.resetCommander()

	// Second reconcile: should mark as failed, not send more commands
	err = recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	// Should NOT send any new commands (host is marked failed)
	assert.Empty(t, s.commander.EnqueueCommandCalls)

	// Verify status changed to failed
	status := s.getRecoveryLockStatus(t, hostID)
	require.NotNil(t, status)
	assert.Equal(t, fleet.MDMDeliveryFailed, *status)
}

func testSkipsHostsWithNoCommandResult(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create eligible host
	s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	// First reconcile: sends SetRecoveryLock
	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)
	require.Len(t, s.commander.EnqueueCommandCalls, 1)

	// Don't insert any command result - simulating device hasn't responded yet

	s.resetCommander()

	// Second reconcile: should not send any new commands (waiting for result)
	err = recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	// Should NOT send any commands - still waiting
	assert.Empty(t, s.commander.EnqueueCommandCalls)
}

func testEnqueueErrorContinuesWithOtherHosts(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create two eligible hosts
	s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	hostID2 := s.insertHostFull(t, "host2", "uuid-2", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-2", "Device", true)

	// Make first enqueue fail
	callCount := 0
	s.commander.EnqueueCommandFunc = func(ctx context.Context, hostUUIDs []string, rawCommand string) error {
		callCount++
		if callCount == 1 {
			return errors.New("enqueue failed")
		}
		return nil
	}

	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err) // Should not return error, just log and continue

	// Should have tried both hosts
	assert.Equal(t, 2, callCount)

	// Second host should have status set (first host failed before status update)
	status := s.getRecoveryLockStatus(t, hostID2)
	require.NotNil(t, status)
	assert.Equal(t, fleet.MDMDeliveryPending, *status)
}

func testNotificationErrorContinues(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create eligible host
	hostID := s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	// Make notifications fail
	s.commander.SendNotificationsFunc = func(ctx context.Context, hostUUIDs []string) error {
		return errors.New("APNs failed")
	}

	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)

	// Should still have set pending status despite notification failure
	status := s.getRecoveryLockStatus(t, hostID)
	require.NotNil(t, status)
	assert.Equal(t, fleet.MDMDeliveryPending, *status)
}

func testFullFlowSetThenVerify(t *testing.T, s *integrationTestSuite) {
	ctx := t.Context()

	// Create team with recovery lock enabled
	teamID := s.insertTeamWithRecoveryLock(t, "test-team", true)

	// Create eligible host
	hostID := s.insertHostFull(t, "host1", "uuid-1", "darwin", "macOS 15.7", &teamID)
	s.insertNanoEnrollment(t, "uuid-1", "Device", true)

	// Step 1: First reconcile sends SetRecoveryLock
	err := recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)
	require.Len(t, s.commander.EnqueueCommandCalls, 1)
	assert.Contains(t, s.commander.EnqueueCommandCalls[0].RawCommand, "SetRecoveryLock")

	status := s.getRecoveryLockStatus(t, hostID)
	require.NotNil(t, status)
	assert.Equal(t, fleet.MDMDeliveryPending, *status)

	// Step 2: Device acknowledges SetRecoveryLock
	setCommandUUID := extractCommandUUID(t, s.commander.EnqueueCommandCalls[0].RawCommand)
	s.insertNanoCommand(t, setCommandUUID, "SetRecoveryLock")
	s.insertNanoCommandResult(t, setCommandUUID, "uuid-1", fleet.MDMAppleStatusAcknowledged, "")

	s.resetCommander()

	// Step 3: Second reconcile sends VerifyRecoveryLock
	err = recoverykeypassword.ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
	require.NoError(t, err)
	require.Len(t, s.commander.EnqueueCommandCalls, 1)
	assert.Contains(t, s.commander.EnqueueCommandCalls[0].RawCommand, "VerifyRecoveryLock")

	status = s.getRecoveryLockStatus(t, hostID)
	require.NotNil(t, status)
	assert.Equal(t, fleet.MDMDeliveryVerifying, *status)

	// Step 4: Device acknowledges VerifyRecoveryLock (handled by results_handler, not reconciler)
	// This part would be tested in results_handler_test.go
}

// extractCommandUUID extracts the CommandUUID from a raw MDM command plist.
func extractCommandUUID(t *testing.T, rawCommand string) string {
	t.Helper()
	// Simple extraction - find <key>CommandUUID</key>\n\t<string>...</string>
	// This is a test helper, so we can be a bit naive
	const prefix = "<key>CommandUUID</key>"
	idx := findSubstring(rawCommand, prefix)
	require.Greater(t, idx, -1, "CommandUUID not found in command")

	// Find the <string> tag after CommandUUID
	rest := rawCommand[idx+len(prefix):]
	startIdx := findSubstring(rest, "<string>")
	require.Greater(t, startIdx, -1)

	rest = rest[startIdx+len("<string>"):]
	endIdx := findSubstring(rest, "</string>")
	require.Greater(t, endIdx, -1)

	return rest[:endIdx]
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
