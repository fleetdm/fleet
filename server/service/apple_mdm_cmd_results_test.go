package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/micromdm/plist"
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

// mockRecoveryLockVerifier implements RecoveryLockVerifier for testing.
type mockRecoveryLockVerifier struct {
	verifyRecoveryLockFn    func(ctx context.Context, hostUUIDs []string, cmdUUID, password string) error
	verifyRecoveryLockCalls []struct {
		hostUUIDs []string
		cmdUUID   string
		password  string
	}
}

func (m *mockRecoveryLockVerifier) VerifyRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID, password string) error {
	m.verifyRecoveryLockCalls = append(m.verifyRecoveryLockCalls, struct {
		hostUUIDs []string
		cmdUUID   string
		password  string
	}{hostUUIDs, cmdUUID, password})
	if m.verifyRecoveryLockFn != nil {
		return m.verifyRecoveryLockFn(ctx, hostUUIDs, cmdUUID, password)
	}
	return nil
}

func TestSetRecoveryLockResultsHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	hostUUID := "test-host-uuid"
	hostID := uint(42)
	cmdUUID := "set-recovery-lock-cmd-uuid"
	password := "test-password-123"

	t.Run("acknowledged sends verify command and sets verifying", func(t *testing.T) {
		ds := new(mock.DataStore)
		commander := &mockRecoveryLockVerifier{}

		ds.HostByIdentifierFunc = func(_ context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: hostUUID}, nil
		}
		ds.GetHostRecoveryLockPasswordFunc = func(_ context.Context, hID uint) (*fleet.HostRecoveryLockPassword, error) {
			assert.Equal(t, hostID, hID)
			return &fleet.HostRecoveryLockPassword{Password: password}, nil
		}
		var verifyingCalled bool
		var capturedVerifyUUID string
		ds.SetRecoveryLockVerifyingFunc = func(_ context.Context, hID uint, verifyCmdUUID string) error {
			verifyingCalled = true
			assert.Equal(t, hostID, hID)
			capturedVerifyUUID = verifyCmdUUID
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, commander, logger)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})

		err := handler(ctx, result)
		require.NoError(t, err)

		// Verify VerifyRecoveryLock was called
		require.Len(t, commander.verifyRecoveryLockCalls, 1)
		assert.Equal(t, []string{hostUUID}, commander.verifyRecoveryLockCalls[0].hostUUIDs)
		assert.Equal(t, password, commander.verifyRecoveryLockCalls[0].password)
		assert.True(t, strings.HasPrefix(commander.verifyRecoveryLockCalls[0].cmdUUID, fleet.VerifyRecoveryLockCommandPrefix))

		// Verify status was set to verifying
		assert.True(t, verifyingCalled)
		assert.True(t, strings.HasPrefix(capturedVerifyUUID, fleet.VerifyRecoveryLockCommandPrefix))
	})

	t.Run("error status sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)
		commander := &mockRecoveryLockVerifier{}

		ds.HostByIdentifierFunc = func(_ context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: hostUUID}, nil
		}
		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hID uint, errorMsg string) error {
			failedCalled = true
			assert.Equal(t, hostID, hID)
			capturedError = errorMsg
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, commander, logger)

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
		// Verify VerifyRecoveryLock was NOT called
		assert.Empty(t, commander.verifyRecoveryLockCalls)
	})

	t.Run("command format error sets failed with default message", func(t *testing.T) {
		ds := new(mock.DataStore)
		commander := &mockRecoveryLockVerifier{}

		ds.HostByIdentifierFunc = func(_ context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: hostUUID}, nil
		}
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hID uint, errorMsg string) error {
			capturedError = errorMsg
			return nil
		}

		handler := NewSetRecoveryLockResultsHandler(ds, commander, logger)

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
}

func TestVerifyRecoveryLockResultsHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	hostUUID := "test-host-uuid"
	hostID := uint(42)
	cmdUUID := fleet.VerifyRecoveryLockCommandPrefix + "test-cmd-uuid"

	t.Run("acknowledged with password verified sets verified", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetHostIDByVerifyRecoveryLockCommandUUIDFunc = func(_ context.Context, verifyCmdUUID string) (uint, error) {
			assert.Equal(t, cmdUUID, verifyCmdUUID)
			return hostID, nil
		}
		var verifiedCalled bool
		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, hID uint) error {
			verifiedCalled = true
			assert.Equal(t, hostID, hID)
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger)

		// Create plist response with PasswordVerified=true
		response := struct {
			Status           string `plist:"Status"`
			PasswordVerified bool   `plist:"PasswordVerified"`
		}{
			Status:           "Acknowledged",
			PasswordVerified: true,
		}
		rawPlist, err := plist.Marshal(response)
		require.NoError(t, err)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         rawPlist,
		})
		result.(*recoveryLockResult).cmdResult.UDID = hostUUID

		err = handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, verifiedCalled)
	})

	t.Run("acknowledged with password not verified sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetHostIDByVerifyRecoveryLockCommandUUIDFunc = func(_ context.Context, _ string) (uint, error) {
			return hostID, nil
		}
		var failedCalled bool
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hID uint, errorMsg string) error {
			failedCalled = true
			assert.Equal(t, hostID, hID)
			assert.Contains(t, errorMsg, "password does not match")
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger)

		// Create plist response with PasswordVerified=false
		response := struct {
			Status           string `plist:"Status"`
			PasswordVerified bool   `plist:"PasswordVerified"`
		}{
			Status:           "Acknowledged",
			PasswordVerified: false,
		}
		rawPlist, err := plist.Marshal(response)
		require.NoError(t, err)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         rawPlist,
		})
		result.(*recoveryLockResult).cmdResult.UDID = hostUUID

		err = handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, failedCalled)
	})

	t.Run("error status sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetHostIDByVerifyRecoveryLockCommandUUIDFunc = func(_ context.Context, _ string) (uint, error) {
			return hostID, nil
		}
		var failedCalled bool
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hID uint, errorMsg string) error {
			failedCalled = true
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusError,
			ErrorChain:  []mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "test", LocalizedDescription: "Verification error"}},
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})
		result.(*recoveryLockResult).cmdResult.UDID = hostUUID

		err := handler(ctx, result)
		require.NoError(t, err)
		assert.True(t, failedCalled)
	})

	t.Run("skips commands without verify prefix", func(t *testing.T) {
		ds := new(mock.DataStore)

		// This should not be called
		ds.GetHostIDByVerifyRecoveryLockCommandUUIDFunc = func(_ context.Context, _ string) (uint, error) {
			t.Fatal("should not be called")
			return 0, nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger)

		// Command UUID without the prefix
		result := NewRecoveryLockResult(&mdm.CommandResults{
			CommandUUID: "not-a-verify-command",
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})
		result.(*recoveryLockResult).cmdResult.UDID = hostUUID

		err := handler(ctx, result)
		require.NoError(t, err)
	})

	t.Run("skips commands with not found error", func(t *testing.T) {
		ds := new(mock.DataStore)

		ds.GetHostIDByVerifyRecoveryLockCommandUUIDFunc = func(_ context.Context, _ string) (uint, error) {
			return 0, &notFoundError{}
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger)

		result := NewRecoveryLockResult(&mdm.CommandResults{
			CommandUUID: cmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})
		result.(*recoveryLockResult).cmdResult.UDID = hostUUID

		err := handler(ctx, result)
		require.NoError(t, err)
	})
}
