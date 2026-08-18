package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

// reconcileNow is the fixed "current time" the reconcile tests run at, so command ages are exact.
var reconcileNow = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

// reconcileTestCallInterval keeps the reconciler's AMAPI pacing out of the tests' wall-clock time.
const reconcileTestCallInterval = time.Nanosecond

// pendingCommandForReconcile builds a pending command row of the given type, created age ago.
func pendingCommandForReconcile(cmdUUID, cmdType string, age time.Duration) *android.MDMAndroidCommand {
	return &android.MDMAndroidCommand{
		CommandUUID:   cmdUUID,
		HostUUID:      "host-uuid-" + cmdUUID,
		OperationName: "enterprises/E/devices/D/operations/" + cmdUUID,
		CommandType:   cmdType,
		Status:        string(android.MDMAndroidCommandStatusPending),
		CreatedAt:     reconcileNow.Add(-age),
	}
}

// newReconcileFixture wires a mock datastore and AMAPI client for the reconciler. cmds is what
// ListPendingMDMAndroidCommands returns; the caller shapes the client's operations.get behavior.
func newReconcileFixture(t *testing.T, cmds ...*android.MDMAndroidCommand) (*AndroidMockDS, *android_mock.Client, *slog.Logger) {
	t.Helper()
	mockDS := InitCommonDSMocks()
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	mockDS.ListPendingMDMAndroidCommandsFunc = func(ctx context.Context, createdBefore time.Time, limit int) ([]*android.MDMAndroidCommand, error) {
		require.Equal(t, reconcileNow.Add(-androidCommandReconcileMinAge), createdBefore)
		require.Equal(t, androidCommandReconcileBatchSize, limit)
		return cmds, nil
	}
	client := &android_mock.Client{}
	client.InitCommonMocks()
	// Discard log output: these tests assert on datastore effects, not on log lines.
	return mockDS, client, slog.New(slog.NewTextHandler(io.Discard, nil))
}

// googleAPIError builds the *googleapi.Error shape the AMAPI clients return, so the reconciler's
// status-code classification is exercised the way it is in production.
func googleAPIError(code int, message string) error {
	return &googleapi.Error{Code: code, Message: message}
}

func TestReconcileAndroidCommands(t *testing.T) {
	t.Run("done operation transitions the row to its terminal status", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			opError        *androidmanagement.Status
			expectedStatus string
			expectedCode   string
			expectedMsg    string
		}{
			{
				name:           "no error means the device executed the command",
				opError:        nil,
				expectedStatus: string(android.MDMAndroidCommandStatusAcknowledged),
			},
			{
				name:           "populated error records the google.rpc code and message",
				opError:        &androidmanagement.Status{Code: 13, Message: "device does not support LOCK"},
				expectedStatus: string(android.MDMAndroidCommandStatusError),
				expectedCode:   "13",
				expectedMsg:    "device does not support LOCK",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cmd := pendingCommandForReconcile("cmd-done", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
				mockDS, client, logger := newReconcileFixture(t, cmd)
				client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
					require.Equal(t, cmd.OperationName, operationName)
					return &androidmanagement.Operation{Name: operationName, Done: true, Error: tc.opError}, nil
				}
				var gotStatus string
				var gotCode, gotMsg, gotRawResult *string
				mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
					require.Equal(t, cmd.CommandUUID, commandUUID)
					gotStatus, gotCode, gotMsg, gotRawResult = status, errorCode, errorMessage, rawResult
					return nil
				}

				require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))

				require.True(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
				assert.Equal(t, tc.expectedStatus, gotStatus)
				require.NotNil(t, gotRawResult, "reconciled operation result should be persisted")
				assert.Contains(t, *gotRawResult, `"done":true`, "raw_result should contain the operation")
				if tc.expectedCode == "" {
					assert.Nil(t, gotCode)
					assert.Nil(t, gotMsg)
				} else {
					require.NotNil(t, gotCode)
					require.NotNil(t, gotMsg)
					assert.Equal(t, tc.expectedCode, *gotCode)
					assert.Equal(t, tc.expectedMsg, *gotMsg)
				}
			})
		}
	})

	t.Run("operation still running is left pending", func(t *testing.T) {
		cmd := pendingCommandForReconcile("cmd-running", string(android.MDMAndroidCommandTypeWipe), 48*time.Hour)
		mockDS, client, logger := newReconcileFixture(t, cmd)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			return &androidmanagement.Operation{Name: operationName, Done: false}, nil
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			t.Fatalf("a command AMAPI is still working on must not be transitioned")
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))
		require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
	})

	t.Run("unknown operation inside the grace period is left pending", func(t *testing.T) {
		// AMAPI 404s for an operation it has already discarded, but a notification can still arrive while
		// the row is younger than Pub/Sub's retention, so we keep waiting.
		cmd := pendingCommandForReconcile("cmd-404-young", string(android.MDMAndroidCommandTypeLock),
			androidCommandReconcileNotFoundGrace-time.Hour)
		mockDS, client, logger := newReconcileFixture(t, cmd)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			return nil, googleAPIError(http.StatusNotFound, "Requested entity was not found.")
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			t.Fatalf("a command still inside the not-found grace period must not be transitioned")
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))
		require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
	})

	t.Run("unknown operation past the grace period is marked error", func(t *testing.T) {
		// Past Pub/Sub's retention no notification can arrive anymore, so the row can only be stuck.
		cmd := pendingCommandForReconcile("cmd-404-old", string(android.MDMAndroidCommandTypeLock),
			androidCommandReconcileNotFoundGrace+time.Hour)
		mockDS, client, logger := newReconcileFixture(t, cmd)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			return nil, googleAPIError(http.StatusNotFound, "Requested entity was not found.")
		}
		var gotStatus string
		var gotCode, gotMsg *string
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			gotStatus, gotCode, gotMsg = status, errorCode, errorMessage
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))

		require.True(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
		assert.Equal(t, string(android.MDMAndroidCommandStatusError), gotStatus)
		require.NotNil(t, gotCode)
		assert.Equal(t, "5", *gotCode, "google.rpc.Code NOT_FOUND")
		require.NotNil(t, gotMsg)
		assert.NotEmpty(t, *gotMsg)
	})

	t.Run("acknowledged WIPE runs the unenroll side effects", func(t *testing.T) {
		const hostID uint = 42
		cmd := pendingCommandForReconcile("cmd-wipe", string(android.MDMAndroidCommandTypeWipe), 48*time.Hour)
		mockDS, client, logger := newReconcileFixture(t, cmd)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			return &androidmanagement.Operation{Name: operationName, Done: true}, nil
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			return nil
		}
		mockDS.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
			require.Equal(t, cmd.HostUUID, hostUUID)
			return &fleet.AndroidHost{Host: &fleet.Host{ID: hostID, UUID: hostUUID}}, nil
		}
		// BYO: the work profile was removed, so host_mdm_actions must be cleared for the "Wiped" badge to drop.
		mockDS.GetHostMDMFunc = func(ctx context.Context, id uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{IsPersonalEnrollment: true}, nil
		}
		mockDS.ClearHostMDMActionsFunc = func(ctx context.Context, id uint) error { return nil }
		mockDS.SetAndroidHostUnenrolledFunc = func(ctx context.Context, id uint) (bool, error) {
			require.Equal(t, hostID, id)
			return true, nil
		}
		mockDS.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
			return []*fleet.Host{{ID: hostID, Hostname: "wiped-host"}}, nil
		}
		var activities []fleet.ActivityDetails
		newActivity := func(_ context.Context, _ *fleet.User, details fleet.ActivityDetails) error {
			activities = append(activities, details)
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, newActivity, reconcileNow, reconcileTestCallInterval))

		require.True(t, mockDS.ClearHostMDMActionsFuncInvoked, "BYO wipe must clear host_mdm_actions")
		require.True(t, mockDS.SetAndroidHostUnenrolledFuncInvoked, "a wiped host must be flipped to unenrolled")
		require.Len(t, activities, 1)
		require.IsType(t, fleet.ActivityTypeMDMUnenrolled{}, activities[0])
	})

	t.Run("a failed WIPE side effect leaves the command pending so the next run retries it", func(t *testing.T) {
		// The reconciler only ever selects pending rows, so writing the terminal status before the
		// unenroll side effect succeeds would strand the host: acknowledged, still enrolled, and never
		// looked at again.
		cmd := pendingCommandForReconcile("cmd-wipe-transient", string(android.MDMAndroidCommandTypeWipe), 48*time.Hour)
		mockDS, client, logger := newReconcileFixture(t, cmd)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			return &androidmanagement.Operation{Name: operationName, Done: true}, nil
		}
		mockDS.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{Host: &fleet.Host{ID: 55, UUID: hostUUID}}, nil
		}
		mockDS.GetHostMDMFunc = func(ctx context.Context, id uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{IsPersonalEnrollment: false}, nil
		}
		mockDS.SetAndroidHostUnenrolledFunc = func(ctx context.Context, id uint) (bool, error) {
			return false, errors.New("simulated transient DB connection drop")
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			t.Fatalf("the command must stay pending when its wipe side effect fails")
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))
		require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
	})

	t.Run("errored WIPE does not unenroll the host", func(t *testing.T) {
		cmd := pendingCommandForReconcile("cmd-wipe-failed", string(android.MDMAndroidCommandTypeWipe), 48*time.Hour)
		mockDS, client, logger := newReconcileFixture(t, cmd)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			return &androidmanagement.Operation{
				Name:  operationName,
				Done:  true,
				Error: &androidmanagement.Status{Code: 13, Message: "wipe failed"},
			}, nil
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			require.Equal(t, string(android.MDMAndroidCommandStatusError), status)
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))
		require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked, "a failed wipe must leave the host enrolled")
	})

	t.Run("a failure on one command does not stop the rest of the batch", func(t *testing.T) {
		failing := pendingCommandForReconcile("cmd-transient", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
		updateFailing := pendingCommandForReconcile("cmd-update-fails", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
		succeeding := pendingCommandForReconcile("cmd-ok", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
		mockDS, client, logger := newReconcileFixture(t, failing, updateFailing, succeeding)
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			if operationName == failing.OperationName {
				return nil, errors.New("simulated transient network failure")
			}
			return &androidmanagement.Operation{Name: operationName, Done: true}, nil
		}
		var updated []string
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage, rawResult *string) error {
			if commandUUID == updateFailing.CommandUUID {
				return errors.New("simulated transient DB failure")
			}
			updated = append(updated, commandUUID)
			return nil
		}

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))
		require.Equal(t, []string{succeeding.CommandUUID}, updated,
			"the reconciler must keep going past both an AMAPI failure and a DB failure")
	})

	t.Run("AMAPI quota error stops the run and surfaces an error", func(t *testing.T) {
		first := pendingCommandForReconcile("cmd-quota", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
		second := pendingCommandForReconcile("cmd-after-quota", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
		mockDS, client, logger := newReconcileFixture(t, first, second)
		var calls int
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
			calls++
			return nil, googleAPIError(http.StatusTooManyRequests, "Quota exceeded")
		}

		err := reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval)
		require.Error(t, err)
		require.Equal(t, 1, calls, "the run must stop at the first quota error instead of hammering AMAPI")
		require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
	})

	t.Run("AMAPI rejecting our credentials stops the run and surfaces an error", func(t *testing.T) {
		// A missing or stale Fleet server secret, lost access to the enterprise, or (on the proxy path)
		// fleetdm.com having no record of the enterprise, rejects every call identically -- working
		// through the batch would only produce noise.
		for _, statusCode := range []int{http.StatusUnauthorized, http.StatusForbidden} {
			first := pendingCommandForReconcile("cmd-rejected", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
			second := pendingCommandForReconcile("cmd-after-rejected", string(android.MDMAndroidCommandTypeLock), 48*time.Hour)
			mockDS, client, logger := newReconcileFixture(t, first, second)
			var calls int
			client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
				calls++
				return nil, googleAPIError(statusCode, "rejected")
			}

			err := reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval)
			require.Error(t, err, "status code %d", statusCode)
			require.Equal(t, 1, calls, "status code %d must stop the run at the first rejection", statusCode)
			require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked,
				"a rejected call says nothing about the command, so nothing may be marked failed")
		}
	})

	t.Run("nothing pending makes no AMAPI calls", func(t *testing.T) {
		mockDS, client, logger := newReconcileFixture(t)

		require.NoError(t, reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval))
		require.False(t, client.EnterprisesDevicesOperationsGetFuncInvoked)
	})

	t.Run("a datastore failure surfaces so the cron run is marked failed", func(t *testing.T) {
		mockDS, client, logger := newReconcileFixture(t)
		mockDS.ListPendingMDMAndroidCommandsFunc = func(ctx context.Context, createdBefore time.Time, limit int) ([]*android.MDMAndroidCommand, error) {
			return nil, errors.New("simulated DB outage")
		}

		err := reconcileAndroidCommands(t.Context(), &mockDS.DataStore, client, logger, noopNewActivity, reconcileNow, reconcileTestCallInterval)
		require.ErrorContains(t, err, "simulated DB outage")
	})

	t.Run("android MDM turned off skips the run entirely", func(t *testing.T) {
		mockDS, _, logger := newReconcileFixture(t)
		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: false}}, nil
		}

		require.NoError(t, ReconcileAndroidCommands(t.Context(), &mockDS.DataStore, logger, "", noopNewActivity))
		require.False(t, mockDS.ListPendingMDMAndroidCommandsFuncInvoked)
	})
}
