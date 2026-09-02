package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
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

	t.Run("no installs left to verify releases the verify command", func(t *testing.T) {
		ds := setupMockDS(t)
		ds.GetUnverifiedVPPInstallsForHostFunc = func(_ context.Context, _ string) ([]*fleet.HostVPPSoftwareInstall, error) {
			return nil, nil
		}
		var removedHostUUID, removedCmdType string
		ds.RemoveHostMDMCommandByHostUUIDFunc = func(_ context.Context, hUUID, cmdType string) error {
			removedHostUUID, removedCmdType = hUUID, cmdType
			return nil
		}

		handler := NewInstalledApplicationListResultsHandler(ds, nil, logger, verifyTimeout, verifyRequestDelay, newNoopActivityFn)

		err := handler(ctx, &testInstalledAppListResult{
			uuid:          cmdUUID,
			hostUUID:      hostUUID,
			hostPlatform:  "darwin",
			availableApps: []fleet.Software{},
		})
		require.NoError(t, err)

		// Holding the verify command here would suppress the next install's acknowledgement on
		// this host until the daily cleanup removes it.
		require.True(t, ds.RemoveHostMDMCommandByHostUUIDFuncInvoked)
		assert.Equal(t, hostUUID, removedHostUUID)
		assert.Equal(t, fleet.VerifySoftwareInstallVPPPrefix, removedCmdType)
		assert.False(t, ds.NewJobFuncInvoked, "nothing left to verify, so no polling job")
	})
}

// newRecoveryLockTestCommander returns a commander whose enqueued commands are captured
// in enqueued, keyed by RequestType.
func newRecoveryLockTestCommander(t *testing.T) (*apple_mdm.MDMAppleCommander, map[string]*mdm.CommandWithSubtype) {
	t.Helper()
	enqueued := make(map[string]*mdm.CommandWithSubtype)
	mdmStorage := &mdmmock.MDMAppleStore{}
	mdmStorage.EnqueueCommandFunc = func(_ context.Context, _ []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		enqueued[cmd.Command.Command.RequestType] = cmd
		return nil, nil
	}
	mdmStorage.RetrievePushInfoFunc = func(_ context.Context, ids []string) (map[string]*mdm.Push, error) {
		return map[string]*mdm.Push{}, nil
	}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.DiscardHandler)),
	)
	return apple_mdm.NewMDMAppleCommander(mdmStorage, pusher), enqueued
}

func TestSetRecoveryLockResultsHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	hostUUID := "test-host-uuid"
	cmdUUID := "set-recovery-lock-cmd-uuid"

	newTestCommander := newRecoveryLockTestCommander

	// pendingFn mocks GetPendingRecoveryLock for a host awaiting the SetRecoveryLock result.
	pendingFn := func(op fleet.MDMOperationType, retries int) func(context.Context, string) (*fleet.HostRecoveryLockPending, error) {
		return func(_ context.Context, hUUID string) (*fleet.HostRecoveryLockPending, error) {
			return &fleet.HostRecoveryLockPending{
				PendingSetCommandUUID: new(cmdUUID),
				OperationType:         op,
				Retries:               retries,
			}, nil
		}
	}

	newResult := func(status string, errChain []mdm.ErrorChain) fleet.MDMCommandResults {
		return NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: cmdUUID,
			Status:      status,
			ErrorChain:  errChain,
			Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>`),
		})
	}

	t.Run("acknowledged promotes to verifying and enqueues VerifyRecoveryLock", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, 0)

		var verifyingCalled bool
		var capturedSetCmdUUID, capturedVerifyCmdUUID string
		ds.SetRecoveryLockVerifyingFunc = func(_ context.Context, hUUID, commandUUID, pendingVerifyCommandUUID string) error {
			verifyingCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedSetCmdUUID = commandUUID
			capturedVerifyCmdUUID = pendingVerifyCommandUUID
			return nil
		}

		commander, enqueued := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)

		assert.True(t, verifyingCalled)
		assert.Equal(t, cmdUUID, capturedSetCmdUUID)

		// The verify command is enqueued under the same UUID recorded as pending.
		verifyCmd, ok := enqueued[fleet.VerifyRecoveryLockCmdName]
		require.True(t, ok, "VerifyRecoveryLock should be enqueued")
		assert.Equal(t, capturedVerifyCmdUUID, verifyCmd.CommandUUID)
	})

	t.Run("acknowledged clear enqueues VerifyRecoveryLock with empty password", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeRemove, 0)
		ds.SetRecoveryLockVerifyingFunc = func(_ context.Context, _, _, _ string) error { return nil }

		commander, enqueued := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)

		verifyCmd, ok := enqueued[fleet.VerifyRecoveryLockCmdName]
		require.True(t, ok, "VerifyRecoveryLock should be enqueued for a clear")
		// Clearing is verified with an empty password, so no host secret placeholder.
		assert.NotContains(t, string(verifyCmd.Raw), fleet.HostSecretPrefix)
	})

	t.Run("error with retries exhausted sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, maxRecoveryLockRetries)

		var failedCalled bool
		var capturedError, capturedCmdUUID string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID, commandUUID, errorMsg string) error {
			failedCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedCmdUUID = commandUUID
			capturedError = errorMsg
			return nil
		}

		commander, _ := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusError,
			[]mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "test", LocalizedDescription: "Test error"}}))
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Equal(t, cmdUUID, capturedCmdUUID)
		assert.Contains(t, capturedError, "Test error")
	})

	t.Run("error with retries remaining retries", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, 0)

		var retryCalled bool
		ds.RetryRecoveryLockFunc = func(_ context.Context, hUUID, cmdUUID string) error {
			retryCalled = true
			assert.Equal(t, hostUUID, hUUID)
			return nil
		}
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, _ string) error {
			t.Fatal("SetRecoveryLockFailed should not be called while retries remain")
			return nil
		}

		commander, _ := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusError,
			[]mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "SomeTransientError", LocalizedDescription: "Network timeout or temporary failure"}}))
		require.NoError(t, err)

		assert.True(t, retryCalled, "RetryRecoveryLock should be called for transient errors")
	})

	t.Run("command format error sets failed with default message", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, 0)

		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, errorMsg string) error {
			capturedError = errorMsg
			return nil
		}
		ds.RetryRecoveryLockFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("RetryRecoveryLock should not be called for command format errors")
			return nil
		}

		commander, _ := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusCommandFormatError, nil))
		require.NoError(t, err)

		assert.Equal(t, "SetRecoveryLock command failed", capturedError)
	})

	t.Run("password mismatch on clear sets failed", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeRemove, 0)

		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, errorMsg string) error {
			failedCalled = true
			capturedError = errorMsg
			return nil
		}

		commander, _ := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusError,
			[]mdm.ErrorChain{{ErrorCode: 70, ErrorDomain: "MDMClientError", LocalizedDescription: "Existing recovery lock password not provided"}}))
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Contains(t, capturedError, "Existing recovery lock password not provided")
	})

	t.Run("clear errors are never retried", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeRemove, 0)

		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, errorMsg string) error {
			failedCalled = true
			capturedError = errorMsg
			return nil
		}
		ds.RetryRecoveryLockFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("RetryRecoveryLock should not be called for clear operations")
			return nil
		}

		commander, _ := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusError,
			[]mdm.ErrorChain{{ErrorCode: 8, ErrorDomain: "ROSLockoutServiceDaemonErrorDomain", LocalizedDescription: "The provided recovery password failed to validate."}}))
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Contains(t, capturedError, "The provided recovery password failed to validate")
	})

	t.Run("result for a stale command UUID is ignored", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = func(_ context.Context, _ string) (*fleet.HostRecoveryLockPending, error) {
			return &fleet.HostRecoveryLockPending{
				PendingSetCommandUUID: new("some-other-cmd-uuid"),
				OperationType:         fleet.MDMOperationTypeInstall,
			}, nil
		}
		ds.SetRecoveryLockVerifyingFunc = func(_ context.Context, _, _, _ string) error {
			t.Fatal("SetRecoveryLockVerifying should not be called for a stale result")
			return nil
		}

		commander, _ := newTestCommander(t)
		handler := NewSetRecoveryLockResultsHandler(ds, logger, commander)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)
	})
}

func TestVerifyRecoveryLockResultsHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	hostUUID := "test-host-uuid"
	verifyCmdUUID := "verify-recovery-lock-cmd-uuid"

	// Shared by the subtests that never reach an enqueue; the retry subtest builds its own
	// so it can inspect what was enqueued.
	commander, _ := newRecoveryLockTestCommander(t)

	// pendingFn mocks GetPendingRecoveryLock for a host awaiting the VerifyRecoveryLock result.
	pendingFn := func(op fleet.MDMOperationType, hasCurrentPassword bool, retries int) func(context.Context, string) (*fleet.HostRecoveryLockPending, error) {
		return func(_ context.Context, _ string) (*fleet.HostRecoveryLockPending, error) {
			return &fleet.HostRecoveryLockPending{
				PendingVerifyCommandUUID: new(verifyCmdUUID),
				OperationType:            op,
				HasCurrentPassword:       hasCurrentPassword,
				Retries:                  retries,
			}, nil
		}
	}

	// A device answers a VerifyRecoveryLock in the payload, not in the status: it
	// acknowledges either way and reports the verdict in PasswordVerified.
	rawVerifyResponse := func(passwordVerified bool) []byte {
		verdict := "<false/>"
		if passwordVerified {
			verdict = "<true/>"
		}
		return []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>PasswordVerified</key>` + verdict + `</dict></plist>`)
	}

	newResult := func(status string, errChain []mdm.ErrorChain) fleet.MDMCommandResults {
		return NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: verifyCmdUUID,
			Status:      status,
			ErrorChain:  errChain,
			Raw:         rawVerifyResponse(true),
		})
	}

	// newUnverifiedResult is an acknowledgment that says the password did not match.
	newUnverifiedResult := func() fleet.MDMCommandResults {
		return NewRecoveryLockResult(&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: verifyCmdUUID,
			Status:      fleet.MDMAppleStatusAcknowledged,
			Raw:         rawVerifyResponse(false),
		})
	}

	t.Run("acknowledged sets verified and logs activity", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, false, 0)

		var verifiedCalled bool
		var capturedCmdUUID string
		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, hUUID, commandUUID string) error {
			verifiedCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedCmdUUID = commandUUID
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

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, newActivityFn)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)

		assert.True(t, verifiedCalled)
		assert.Equal(t, verifyCmdUUID, capturedCmdUUID)
		assert.True(t, activityCalled)
		assert.Equal(t, uint(1), capturedHostID)
		assert.Equal(t, "Test Host", capturedDisplayName)
	})

	t.Run("acknowledged rotation sets verified without activity", func(t *testing.T) {
		ds := new(mock.DataStore)
		// A host that already had a password is a rotation; its activity is logged at
		// rotation enqueue time, not here.
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, true, 0)

		var verifiedCalled bool
		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, _, _ string) error {
			verifiedCalled = true
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("activity should not be created for a rotation")
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, newActivityFn)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)

		assert.True(t, verifiedCalled)
	})

	t.Run("acknowledged clear deletes password", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeRemove, true, 0)

		var deleteCalled bool
		var capturedCmdUUID string
		ds.DeleteHostRecoveryLockPasswordFunc = func(_ context.Context, hUUID, commandUUID string) error {
			deleteCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedCmdUUID = commandUUID
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("activity should not be created for a clear")
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, newActivityFn)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)

		assert.True(t, deleteCalled)
		assert.Equal(t, verifyCmdUUID, capturedCmdUUID)
	})

	t.Run("acknowledged with PasswordVerified false fails instead of verifying", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, false, 0)

		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("an acknowledgment the device did not verify must not mark the host verified")
			return nil
		}
		ds.RetryRecoveryLockVerifyFunc = func(_ context.Context, _, _, _ string) error {
			t.Fatal("re-asking the device the same question gets the same answer")
			return nil
		}

		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, hUUID, commandUUID, errorMsg string) error {
			assert.Equal(t, hostUUID, hUUID)
			assert.Equal(t, verifyCmdUUID, commandUUID)
			capturedError = errorMsg
			return nil
		}

		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("no set activity for a password the device never confirmed")
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, newActivityFn)

		err := handler(ctx, newUnverifiedResult())
		require.NoError(t, err)

		assert.True(t, ds.SetRecoveryLockFailedFuncInvoked)
		assert.Contains(t, capturedError, "does not match")
	})

	t.Run("acknowledged clear with PasswordVerified false keeps the password", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeRemove, true, 0)

		// The lock is still on the device, so dropping Fleet's copy would strand it.
		ds.DeleteHostRecoveryLockPasswordFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("the stored password must not be deleted while the device still has a lock")
			return nil
		}

		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, errorMsg string) error {
			capturedError = errorMsg
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, nil)

		err := handler(ctx, newUnverifiedResult())
		require.NoError(t, err)

		assert.True(t, ds.SetRecoveryLockFailedFuncInvoked)
		assert.Contains(t, capturedError, "still set")
	})

	t.Run("password not set error is terminal", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, false, 0)

		var failedCalled bool
		var capturedError string
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, errorMsg string) error {
			failedCalled = true
			capturedError = errorMsg
			return nil
		}
		ds.RetryRecoveryLockFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("RetryRecoveryLock should not be called when the password is not set")
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, nil)

		err := handler(ctx, newResult(fleet.MDMAppleStatusError,
			[]mdm.ErrorChain{{ErrorCode: 70, ErrorDomain: "MDMClientError", LocalizedDescription: "Recovery lock password not set"}}))
		require.NoError(t, err)

		assert.True(t, failedCalled)
		assert.Contains(t, capturedError, "Recovery lock password not set")
	})

	t.Run("error with retries remaining re-issues the verify", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = pendingFn(fleet.MDMOperationTypeInstall, false, 0)

		// The set was already acknowledged, so the device holds the pending password while
		// encrypted_password still holds the superseded one. Retrying via RetryRecoveryLock
		// would hand the row to the cron, which enqueues RotateRecoveryLock with that stale
		// value as CurrentPassword and fails on every attempt.
		ds.RetryRecoveryLockFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("RetryRecoveryLock must not be used for a verify retry")
			return nil
		}
		ds.SetRecoveryLockFailedFunc = func(_ context.Context, _, _, _ string) error {
			t.Fatal("SetRecoveryLockFailed should not be called while retries remain")
			return nil
		}

		var retryVerifyCalled bool
		var capturedOldUUID, capturedNewUUID string
		ds.RetryRecoveryLockVerifyFunc = func(_ context.Context, hUUID, oldUUID, newUUID string) error {
			retryVerifyCalled = true
			assert.Equal(t, hostUUID, hUUID)
			capturedOldUUID, capturedNewUUID = oldUUID, newUUID
			return nil
		}

		commander, enqueued := newRecoveryLockTestCommander(t)
		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, nil)

		err := handler(ctx, newResult(fleet.MDMAppleStatusError,
			[]mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "SomeTransientError", LocalizedDescription: "Network timeout"}}))
		require.NoError(t, err)

		require.True(t, retryVerifyCalled)
		assert.Equal(t, verifyCmdUUID, capturedOldUUID)
		assert.NotEmpty(t, capturedNewUUID)
		assert.NotEqual(t, verifyCmdUUID, capturedNewUUID, "the retry needs its own command UUID")

		// A fresh VerifyRecoveryLock goes out under the new UUID, and no SetRecoveryLock does.
		cmd, ok := enqueued[fleet.VerifyRecoveryLockCmdName]
		require.True(t, ok, "a replacement VerifyRecoveryLock should be enqueued")
		assert.Equal(t, capturedNewUUID, cmd.CommandUUID)
		assert.NotContains(t, enqueued, fleet.SetRecoveryLockCmdName,
			"a verify retry must never re-run the set")
	})

	t.Run("result for a stale command UUID is ignored", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetPendingRecoveryLockFunc = func(_ context.Context, _ string) (*fleet.HostRecoveryLockPending, error) {
			return &fleet.HostRecoveryLockPending{
				PendingVerifyCommandUUID: new("some-other-cmd-uuid"),
				OperationType:            fleet.MDMOperationTypeInstall,
			}, nil
		}
		ds.SetRecoveryLockVerifiedFunc = func(_ context.Context, _, _ string) error {
			t.Fatal("SetRecoveryLockVerified should not be called for a stale result")
			return nil
		}

		handler := NewVerifyRecoveryLockResultsHandler(ds, logger, commander, nil)

		err := handler(ctx, newResult(fleet.MDMAppleStatusAcknowledged, nil))
		require.NoError(t, err)
	})
}
