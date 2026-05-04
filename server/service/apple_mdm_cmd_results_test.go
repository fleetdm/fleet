package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInstalledAppListResult implements InstalledApplicationListResult for testing.
type testInstalledAppListResult struct {
	raw           []byte
	uuid          string
	hostUUID      string
	hostPlatform  string
	availableApps []fleet.Software
}

func (t *testInstalledAppListResult) Raw() []byte                     { return t.raw }
func (t *testInstalledAppListResult) UUID() string                    { return t.uuid }
func (t *testInstalledAppListResult) HostUUID() string                { return t.hostUUID }
func (t *testInstalledAppListResult) HostPlatform() string            { return t.hostPlatform }
func (t *testInstalledAppListResult) AvailableApps() []fleet.Software { return t.availableApps }

func TestInstalledApplicationListHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	verifyTimeout := 10 * time.Minute
	verifyRequestDelay := 5 * time.Second

	hostUUID := "host-uuid-1"
	hostID := uint(42)
	cmdUUID := fleet.VerifySoftwareInstallVPPPrefix + "test-cmd-uuid"
	bundleID := "com.example.app"

	ackTime := time.Now().Add(-1 * time.Minute)

	newNoopActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
		return nil
	}

	// setupMockDS creates a mock datastore with common function stubs.
	setupMockDS := func(t *testing.T) *mock.DataStore {
		ds := new(mock.DataStore)
		ds.GetUnverifiedInHouseAppInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return nil, nil
		}
		ds.IsAutoUpdateVPPInstallFunc = func(_ context.Context, _ string) (bool, error) {
			return false, nil
		}
		ds.UpdateSetupExperienceStatusResultFunc = func(_ context.Context, _ *fleet.SetupExperienceStatusResult) error {
			return nil
		}
		ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(_ context.Context, _ string, _ string, _ fleet.SetupExperienceStatusResultStatus) (bool, error) {
			return false, nil
		}
		ds.GetPastActivityDataForVPPAppInstallFunc = func(_ context.Context, _ *mdm.CommandResults) (*fleet.User, *fleet.ActivityInstalledAppStoreApp, error) {
			return &fleet.User{}, &fleet.ActivityInstalledAppStoreApp{}, nil
		}
		ds.RemoveHostMDMCommandFunc = func(_ context.Context, _ fleet.HostMDMCommand) error {
			return nil
		}
		ds.UpdateHostRefetchRequestedFunc = func(_ context.Context, _ uint, _ bool) error {
			return nil
		}
		return ds
	}

	t.Run("app installed with matching version is verified", func(t *testing.T) {
		ds := setupMockDS(t)

		var verifiedCalled bool
		ds.SetVPPInstallAsVerifiedFunc = func(_ context.Context, hID uint, installUUID string, verifyUUID string) error {
			verifiedCalled = true
			assert.Equal(t, hostID, hID)
			assert.Equal(t, cmdUUID, installUUID)
			return nil
		}
		ds.SetVPPInstallAsFailedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("fail should not be called")
			return nil
		}
		ds.GetUnverifiedVPPInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return []*fleet.HostVPPSoftwareInstall{
				{
					InstallCommandUUID:  cmdUUID,
					InstallCommandAckAt: &ackTime,
					HostID:              hostID,
					BundleIdentifier:    bundleID,
					ExpectedVersion:     "1.0.0",
				},
			}, nil
		}

		handler := NewInstalledApplicationListResultsHandler(ds, nil, logger, verifyTimeout, verifyRequestDelay, newNoopActivityFn)

		result := &testInstalledAppListResult{
			uuid:         cmdUUID,
			hostUUID:     hostUUID,
			hostPlatform: "darwin",
			availableApps: []fleet.Software{
				{BundleIdentifier: bundleID, Version: "1.0.0", Installed: true},
			},
		}

		err := handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, verifiedCalled, "verify should have been called")
	})

	t.Run("app installed with different version is verified (bug fix)", func(t *testing.T) {
		ds := setupMockDS(t)

		var verifiedCalled bool
		ds.SetVPPInstallAsVerifiedFunc = func(_ context.Context, hID uint, installUUID string, _ string) error {
			verifiedCalled = true
			assert.Equal(t, hostID, hID)
			assert.Equal(t, cmdUUID, installUUID)
			return nil
		}
		ds.SetVPPInstallAsFailedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("fail should not be called for version mismatch")
			return nil
		}
		ds.GetUnverifiedVPPInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return []*fleet.HostVPPSoftwareInstall{
				{
					InstallCommandUUID:  cmdUUID,
					InstallCommandAckAt: &ackTime,
					HostID:              hostID,
					BundleIdentifier:    bundleID,
					ExpectedVersion:     "26.01.40",
				},
			}, nil
		}

		handler := NewInstalledApplicationListResultsHandler(ds, nil, logger, verifyTimeout, verifyRequestDelay, newNoopActivityFn)

		result := &testInstalledAppListResult{
			uuid:         cmdUUID,
			hostUUID:     hostUUID,
			hostPlatform: "darwin",
			availableApps: []fleet.Software{
				{BundleIdentifier: bundleID, Version: "24.10.50", Installed: true},
			},
		}

		err := handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, verifiedCalled, "verify should be called even with version mismatch")
		// Key assertion: should NOT be polling (NewJob should not be called)
		assert.False(t, ds.NewJobFuncInvoked, "should not queue a polling job when app is installed")
	})

	t.Run("app not installed within timeout continues polling", func(t *testing.T) {
		ds := setupMockDS(t)

		ds.SetVPPInstallAsVerifiedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("verify should not be called")
			return nil
		}
		ds.SetVPPInstallAsFailedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("fail should not be called")
			return nil
		}
		ds.GetUnverifiedVPPInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return []*fleet.HostVPPSoftwareInstall{
				{
					InstallCommandUUID:  cmdUUID,
					InstallCommandAckAt: &ackTime, // 1 minute ago, within 10-minute timeout
					HostID:              hostID,
					BundleIdentifier:    bundleID,
					ExpectedVersion:     "1.0.0",
				},
			}, nil
		}
		ds.NewJobFunc = func(_ context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}

		handler := NewInstalledApplicationListResultsHandler(ds, nil, logger, verifyTimeout, verifyRequestDelay, newNoopActivityFn)

		// App not in the list at all
		result := &testInstalledAppListResult{
			uuid:          cmdUUID,
			hostUUID:      hostUUID,
			hostPlatform:  "darwin",
			availableApps: []fleet.Software{},
		}

		err := handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, ds.NewJobFuncInvoked, "should queue a polling job when app not yet installed")
	})

	t.Run("app not installed timeout exceeded is marked failed", func(t *testing.T) {
		ds := setupMockDS(t)

		expiredAckTime := time.Now().Add(-15 * time.Minute) // well past the 10-minute timeout

		var failedCalled bool
		ds.SetVPPInstallAsVerifiedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("verify should not be called")
			return nil
		}
		ds.SetVPPInstallAsFailedFunc = func(_ context.Context, hID uint, installUUID string, _ string) error {
			failedCalled = true
			assert.Equal(t, hostID, hID)
			assert.Equal(t, cmdUUID, installUUID)
			return nil
		}
		ds.GetUnverifiedVPPInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return []*fleet.HostVPPSoftwareInstall{
				{
					InstallCommandUUID:  cmdUUID,
					InstallCommandAckAt: &expiredAckTime,
					HostID:              hostID,
					BundleIdentifier:    bundleID,
					ExpectedVersion:     "1.0.0",
				},
			}, nil
		}

		handler := NewInstalledApplicationListResultsHandler(ds, nil, logger, verifyTimeout, verifyRequestDelay, newNoopActivityFn)

		result := &testInstalledAppListResult{
			uuid:          cmdUUID,
			hostUUID:      hostUUID,
			hostPlatform:  "darwin",
			availableApps: []fleet.Software{},
		}

		err := handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, failedCalled, "fail should be called when timeout exceeded")
	})

	t.Run("app not reported in list continues polling", func(t *testing.T) {
		ds := setupMockDS(t)

		ds.SetVPPInstallAsVerifiedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("verify should not be called")
			return nil
		}
		ds.SetVPPInstallAsFailedFunc = func(_ context.Context, _ uint, _ string, _ string) error {
			t.Fatal("fail should not be called")
			return nil
		}
		ds.GetUnverifiedVPPInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return []*fleet.HostVPPSoftwareInstall{
				{
					InstallCommandUUID:  cmdUUID,
					InstallCommandAckAt: &ackTime,
					HostID:              hostID,
					BundleIdentifier:    bundleID,
					ExpectedVersion:     "1.0.0",
				},
			}, nil
		}
		ds.NewJobFunc = func(_ context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}

		handler := NewInstalledApplicationListResultsHandler(ds, nil, logger, verifyTimeout, verifyRequestDelay, newNoopActivityFn)

		// Different app is reported but not our expected one
		result := &testInstalledAppListResult{
			uuid:         cmdUUID,
			hostUUID:     hostUUID,
			hostPlatform: "darwin",
			availableApps: []fleet.Software{
				{BundleIdentifier: "com.other.app", Version: "2.0.0", Installed: true},
			},
		}

		err := handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, ds.NewJobFuncInvoked, "should queue a polling job when expected app not in list")
	})
}

func TestSetRecoveryLockResultsHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	hostUUID := "test-host-uuid"
	cmdUUID := "set-recovery-lock-cmd-uuid"

	t.Run("acknowledged sets verified", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock GetRecoveryLockOperationType to return 'install' (SET operation)
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeInstall, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var verifiedCalled bool
		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, hUUID string) error {
			verifiedCalled = true
			assert.Equal(t, hostUUID, hUUID)
			return nil
		}

		ds.HostLiteByIdentifierFunc = func(_ context.Context, identifier string) (*fleet.HostLite, error) {
			assert.Equal(t, hostUUID, identifier)
			return &fleet.HostLite{ID: 1, Hostname: "Test Host"}, nil
		}

		var activityCalled bool
		var capturedHostID uint
		var capturedDisplayName string
		newActivityFn := func(_ context.Context, _ *fleet.User, activity fleet.ActivityDetails) error {
			activityCalled = true
			act, ok := activity.(fleet.ActivityTypeSetHostRecoveryLockPassword)
			require.True(t, ok)
			capturedHostID = act.HostID
			capturedDisplayName = act.HostDisplayName
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		// Verify status was set to verified
		assert.True(t, verifiedCalled)
		assert.True(t, activityCalled)
		assert.Equal(t, uint(1), capturedHostID)
		assert.Equal(t, "Test Host", capturedDisplayName)
	})

	t.Run("error status sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock GetRecoveryLockOperationType to return 'install' (SET operation)
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeInstall, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			failedCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedError = errorMsg
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("activity should not be called on error")
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusError,
			ErrorChain:  []mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "test", LocalizedDescription: "Test error"}},
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Contains(t, capturedError, "Test error")
	})

	t.Run("command format error sets failed with default message", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock GetRecoveryLockOperationType to return 'install' (SET operation)
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeInstall, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			capturedError = errorMsg
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("activity should not be called on error")
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusCommandFormatError,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.Equal(t, "SetRecoveryLock command failed", capturedError)
	})

	t.Run("acknowledged clear deletes password", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock GetRecoveryLockOperationType to return 'remove' (CLEAR operation)
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeRemove, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var deleteCalled bool
		ds.DeleteHostRecoveryLockPasswordFunc = func(_ context.Context, hUUID string) error {
			deleteCalled = true
			assert.Equal(t, hostUUID, hUUID)
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, deleteCalled)
	})

	t.Run("error clear with password mismatch sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock GetRecoveryLockOperationType to return 'remove' (CLEAR operation)
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeRemove, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			failedCalled = true
			capturedError = errorMsg
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		// Test MDMClientError 70 (password not provided)
		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusError,
			ErrorChain:  []mdm.ErrorChain{{ErrorCode: 70, ErrorDomain: "MDMClientError", LocalizedDescription: "Existing recovery lock password not provided"}},
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Contains(t, capturedError, "Existing recovery lock password not provided")
	})

	t.Run("error clear with ROSLockoutService password validation error sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeRemove, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			failedCalled = true
			capturedError = errorMsg
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		// Test ROSLockoutServiceDaemonErrorDomain 8 (password failed to validate)
		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusError,
			ErrorChain:  []mdm.ErrorChain{{ErrorCode: 8, ErrorDomain: "ROSLockoutServiceDaemonErrorDomain", LocalizedDescription: "The provided recovery password failed to validate."}},
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Contains(t, capturedError, "The provided recovery password failed to validate")
	})

	t.Run("error clear with transient error resets for retry", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeRemove, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var resetCalled bool
		ds.ResetRecoveryLockForRetryFunc = func(_ context.Context, hUUID string) error {
			resetCalled = true
			assert.Equal(t, hostUUID, hUUID)
			return nil
		}
		// SetRecoveryLockFailed should NOT be called for transient errors
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			t.Fatal("SetRecoveryLockFailed should not be called for transient errors")
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		// Test a generic transient error (not password mismatch)
		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusError,
			ErrorChain:  []mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "SomeTransientError", LocalizedDescription: "Network timeout or temporary failure"}},
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, resetCalled, "ResetRecoveryLockForRetry should be called for transient errors")
	})

	t.Run("command format error clear sets failed not retry", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, hUUID string) (fleet.MDMOperationType, error) {
			return fleet.MDMOperationTypeRemove, nil
		}
		// Mock HasPendingRecoveryLockRotation to return false (no rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return false, nil
		}
		var failedCalled bool
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			failedCalled = true
			assert.Equal(t, hostUUID, hUUID)
			return nil
		}
		// ResetRecoveryLockForRetry should NOT be called for command format errors
		ds.ResetRecoveryLockForRetryFunc = func(_ context.Context, hUUID string) error {
			t.Fatal("ResetRecoveryLockForRetry should not be called for command format errors")
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		// CommandFormatError is terminal - command is malformed and will never succeed
		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusCommandFormatError,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, failedCalled, "SetRecoveryLockFailed should be called for command format errors")
	})

	// Rotation tests - verify rotation branch doesn't fall through to SET/CLEAR logic

	t.Run("rotation acknowledged completes rotation", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock HasPendingRecoveryLockRotation to return true (rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			assert.Equal(t, hostUUID, hUUID)
			return true, nil
		}

		var completeRotationCalled bool
		ds.CompleteRecoveryLockRotationFunc = func(_ context.Context, hUUID string) error {
			completeRotationCalled = true
			assert.Equal(t, hostUUID, hUUID)
			return nil
		}

		// These should NOT be called for rotation
		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, _ string) error {
			t.Fatal("SetRecoveryLockVerified should not be called for rotation")
			return nil
		}
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, _ string) (fleet.MDMOperationType, error) {
			t.Fatal("GetRecoveryLockOperationType should not be called for rotation")
			return "", nil
		}

		// No activity should be created for manual rotation (activity logged at initiation)
		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("Activity should not be created for manual rotation completion")
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, completeRotationCalled, "CompleteRecoveryLockRotation should be called")
	})

	t.Run("rotation error fails rotation", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock HasPendingRecoveryLockRotation to return true (rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return true, nil
		}

		var failRotationCalled bool
		var capturedError string
		ds.FailRecoveryLockRotationFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			failRotationCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedError = errorMsg
			return nil
		}

		// These should NOT be called for rotation
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _ string, _ string) error {
			t.Fatal("SetRecoveryLockFailed should not be called for rotation")
			return nil
		}
		ds.GetRecoveryLockOperationTypeFunc = func(_ context.Context, _ string) (fleet.MDMOperationType, error) {
			t.Fatal("GetRecoveryLockOperationType should not be called for rotation")
			return "", nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusError,
			ErrorChain:  []mdm.ErrorChain{{ErrorCode: 8, ErrorDomain: "ROSLockoutServiceDaemonErrorDomain", LocalizedDescription: "Password mismatch during rotation"}},
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, failRotationCalled, "FailRecoveryLockRotation should be called")
		assert.Contains(t, capturedError, "Password mismatch during rotation")
	})

	t.Run("rotation command format error fails rotation", func(t *testing.T) {
		ds := new(mock.DataStore)

		// Mock HasPendingRecoveryLockRotation to return true (rotation pending)
		ds.HasPendingRecoveryLockRotationFunc = func(_ context.Context, hUUID string) (bool, error) {
			return true, nil
		}

		var failRotationCalled bool
		var capturedError string
		ds.FailRecoveryLockRotationFunc = func(_ context.Context, hUUID string, errorMsg string) error {
			failRotationCalled = true
			capturedError = errorMsg
			return nil
		}

		// These should NOT be called for rotation
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _ string, _ string) error {
			t.Fatal("SetRecoveryLockFailed should not be called for rotation")
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, logger, newActivityFn)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusCommandFormatError,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		assert.True(t, failRotationCalled, "FailRecoveryLockRotation should be called for command format errors")
		assert.Equal(t, "RotateRecoveryLock command failed", capturedError)
	})

	// Note: Activity logging for auto-rotation now happens at initiation time
	// (in the cron job's sendAutoRotationCommands), not at completion time.
	// Manual rotation activity is logged at the API handler level.
}
