package service

// Android MDM tests for the MDM suite.
//
// Belongs here: Android enrollment and unenrollment, AMAPI error mapping, enterprise deletion
// detection, lock/wipe/clear passcode, and custom commands.
//
// Does not belong here: Android software (integration_android_software_test.go) and Android
// profiles (integration_mdm_profiles_test.go).

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mdm/android/tests"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestAndroidHostUnenrollMDM() {
	t := s.T()
	ctx := t.Context()

	// Per #41683: BYO unenroll runs an AMAPI WIPE (work-profile only), not EnterprisesDevicesDelete.
	// COBO unenroll keeps the existing Delete behavior (terminates management without factory reset).
	var (
		didCallAMAPIDelete    bool
		didCallAMAPIIssueWipe bool
		issuedCommand         *androidmanagement.Command
		issuedToDeviceName    string
		issueOperationName    = "enterprises/ultimate-fake/devices/dev-byo/operations/wipe-byo-1"
	)
	s.androidAPIClient.EnterprisesDevicesDeleteFunc = func(_ context.Context, _ string) error {
		didCallAMAPIDelete = true
		return nil
	}
	s.androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, deviceName string, cmd *androidmanagement.Command) (*androidmanagement.Operation, error) {
		didCallAMAPIIssueWipe = true
		issuedCommand = cmd
		issuedToDeviceName = deviceName
		return &androidmanagement.Operation{Name: issueOperationName}, nil
	}

	enterpriseId, err := s.ds.CreateEnterprise(ctx, s.users["admin1"].ID)
	require.NoError(t, err)
	err = s.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		ID:           enterpriseId,
		EnterpriseID: "ultimate-fake",
		SignupName:   "fake",
		SignupToken:  "value",
		TopicID:      "yep",
	})
	require.NoError(t, err)

	// BYO host -> WIPE command, no Delete call.
	byoHostID := createAndroidHostForTest(t, s.ds, nil, false)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", byoHostID), nil, http.StatusNoContent)
	require.True(t, didCallAMAPIIssueWipe, "expected BYO unenroll to call EnterprisesDevicesIssueCommand")
	require.False(t, didCallAMAPIDelete, "BYO unenroll must not call EnterprisesDevicesDelete")
	require.NotNil(t, issuedCommand)
	require.Equal(t, string(android.MDMAndroidCommandTypeWipe), issuedCommand.Type)
	require.NotNil(t, issuedCommand.WipeParams, "WIPE requires non-nil WipeParams")
	require.Equal(t, "315360000s", issuedCommand.Duration, "android commands must use the long duration to mirror Apple/Windows queue semantics")
	require.NotEmpty(t, issuedToDeviceName)

	// Per #41683 the transient work-profile-wipe status is suppressed for BYO Android at the API layer, so the host must NOT surface
	// a pending wipe. The host simply transitions to "Off" once it removes its work profile.
	var byoResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", byoHostID), nil, http.StatusOK, &byoResp)
	require.NotNil(t, byoResp.Host.MDM.PendingAction)
	require.Empty(t, *byoResp.Host.MDM.PendingAction, "BYO Android unenroll must not surface a pending action")
	require.NotNil(t, byoResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *byoResp.Host.MDM.DeviceStatus, "BYO Android unenroll must not surface a wipe device_status")

	// The wipe_ref is still written under the hood (suppression is presentation-only), so the work-profile wipe runs.
	byoHost, err := s.ds.Host(ctx, byoHostID)
	require.NoError(t, err)
	byoLockWipe, err := s.ds.GetHostLockWipeStatus(ctx, byoHost)
	require.NoError(t, err)
	require.True(t, byoLockWipe.IsPendingWipe(), "BYO unenroll must still write wipe_ref so the work-profile wipe runs")

	// Reset between sub-cases.
	didCallAMAPIDelete = false
	didCallAMAPIIssueWipe = false
	issuedCommand = nil

	// COBO host -> Delete call, no WIPE.
	coboHostID := createAndroidHostForTest(t, s.ds, nil, true)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", coboHostID), nil, http.StatusNoContent)
	require.True(t, didCallAMAPIDelete, "COBO unenroll must call EnterprisesDevicesDelete")
	require.False(t, didCallAMAPIIssueWipe, "COBO unenroll must not call EnterprisesDevicesIssueCommand")

	// host_mdm.enrolled must stay 1 here. EnterprisesDevicesDelete returns 200 when AMAPI accepts the request, not when the device has processed it.
	coboHostMDM, err := s.ds.GetHostMDM(ctx, coboHostID)
	require.NoError(t, err)
	require.True(t, coboHostMDM.Enrolled, "host_mdm.enrolled must stay true until the device acks via Pub/Sub DELETED")

	// Reset between sub-cases.
	didCallAMAPIDelete = false
	didCallAMAPIIssueWipe = false
	issuedCommand = nil

	// BYO host where AMAPI returns 404 (device already removed from Google) -> should succeed (204), not 500.
	s.androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, _ string, _ *androidmanagement.Command) (*androidmanagement.Operation, error) {
		didCallAMAPIIssueWipe = true
		return nil, &googleapi.Error{Code: http.StatusNotFound, Message: "Not Found"}
	}
	byoGoneHostID := createAndroidHostForTest(t, s.ds, nil, false)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", byoGoneHostID), nil, http.StatusNoContent)
	require.True(t, didCallAMAPIIssueWipe, "expected BYO unenroll to attempt IssueCommand even when device is gone")
}

// TestAndroidLockWipeClearPasscode exercises the three commands end-to-end against the real Fleet HTTP handler stack with a
// mocked AMAPI client: lock (BYO + COBO), wipe (both), clear-passcode (both), and the Pub/Sub COMMAND ack that
// transitions a pending row to acknowledged. Each assertion checks that (1) the mock IssueCommand was called with the right
// type/duration, (2) the mdm_android_commands row is persisted with the right status, and where applicable (3) host_mdm_actions
// is updated so the API surfaces PendingAction=lock/wipe.
// TestAndroidAMAPIErrorStatusMapping checks that an AMAPI failure surfaces the matching
// HTTP status on every Android command endpoint, not just on unenroll. Per #52547 an AMAPI
// 404 encoded as a 500 on lock/wipe/clear-passcode even though the mapped error reported
// IsNotFound, so these assertions are on the response status, not on the error value.
func (s *integrationMDMTestSuite) TestAndroidAMAPIErrorStatusMapping() {
	t := s.T()
	ctx := t.Context()

	enterpriseID, err := s.ds.CreateEnterprise(ctx, s.users["admin1"].ID)
	require.NoError(t, err)
	require.NoError(t, s.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		ID:           enterpriseID,
		EnterpriseID: "ultimate-fake",
		SignupName:   "fake",
		SignupToken:  "value",
		TopicID:      "yep",
	}))

	amapiErrors := []struct {
		name string
		err  *googleapi.Error
		// want is the status for lock/wipe/clear-passcode. Unenroll treats not-found as
		// "already unenrolled" and returns 204, so it is listed separately.
		want         int
		wantUnenroll int
	}{
		{"404", &googleapi.Error{Code: http.StatusNotFound, Message: "Not Found"}, http.StatusNotFound, http.StatusNoContent},
		{"500 entity not found", &googleapi.Error{Code: http.StatusInternalServerError, Body: "Requested entity was not found"}, http.StatusNotFound, http.StatusNoContent},
		{"400", &googleapi.Error{Code: http.StatusBadRequest, Message: "invalid command"}, http.StatusBadRequest, http.StatusBadRequest},
		{"409", &googleapi.Error{Code: http.StatusConflict, Message: "device state incompatible"}, http.StatusConflict, http.StatusConflict},
		{"403 unmapped", &googleapi.Error{Code: http.StatusForbidden, Message: "no access"}, http.StatusInternalServerError, http.StatusInternalServerError},
	}

	// The mock client is shared by the whole suite and there is no SetupTest, so restore
	// it rather than leaving every later test with a command that fails.
	t.Cleanup(func() { s.androidAPIClient.InitCommonMocks() })

	for _, ae := range amapiErrors {
		t.Run(ae.name, func(t *testing.T) {
			amapiErr := ae.err
			s.androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, _ string, _ *androidmanagement.Command) (*androidmanagement.Operation, error) {
				return nil, amapiErr
			}

			// COBO hosts: lock, wipe and clear passcode are all allowed.
			lockHostID := createAndroidHostForTest(t, s.ds, nil, true)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", lockHostID), nil, ae.want)

			wipeHostID := createAndroidHostForTest(t, s.ds, nil, true)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", wipeHostID), nil, ae.want)

			passcodeHostID := createAndroidHostForTest(t, s.ds, nil, true)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", passcodeHostID), nil, ae.want)

			// BYO hosts may be locked too, and take the same path as COBO.
			byoLockHostID := createAndroidHostForTest(t, s.ds, nil, false)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", byoLockHostID), nil, ae.want)

			// Custom commands reach the mapping through a different service entry point
			// (server/service/mdm.go) than the three commands above, so cover it separately.
			customHostID := createAndroidHostForTest(t, s.ds, nil, true)
			var customHost getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", customHostID), nil, http.StatusOK, &customHost)
			s.Do("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
				Command:   base64.StdEncoding.EncodeToString([]byte(`{"type":"REBOOT"}`)),
				HostUUIDs: []string{customHost.Host.UUID},
			}, ae.want)

			// BYO unenroll issues the work-profile wipe, so it goes through the same mapping.
			unenrollHostID := createAndroidHostForTest(t, s.ds, nil, false)
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", unenrollHostID), nil, ae.wantUnenroll)
		})
	}
}

func (s *integrationMDMTestSuite) TestAndroidLockWipeClearPasscode() {
	t := s.T()
	ctx := t.Context()

	// Long-duration AMAPI command timeout (10 years). Mirrors longCommandDuration in
	// server/mdm/android/service/service.go (private to that package, so duplicated here).
	const longCommandDuration = "315360000s"

	enterpriseID, err := s.ds.CreateEnterprise(ctx, s.users["admin1"].ID)
	require.NoError(t, err)
	require.NoError(t, s.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		ID: enterpriseID, EnterpriseID: "ultimate-fake",
		SignupName:  "fake",
		SignupToken: "value",
		TopicID:     "yep",
	}))

	// The Pub/Sub COMMAND handler authenticates against the AndroidPubSubToken asset. The real signup flow inserts this when AMAPI
	// creates the enterprise; in this test we shortcut the signup and seed the token directly so the COMMAND notification path can run.
	const pubsubToken = "test-android-pubsub-token"
	require.NoError(t, s.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
		Name:  fleet.MDMAssetAndroidPubSubToken,
		Value: []byte(pubsubToken),
	}))

	// Capture every IssueCommand call so each subtest can assert against the latest invocation without resetting global state between subtests.
	var (
		issueCallsMu     sync.Mutex
		issueCalls       []*androidmanagement.Command
		issueDeviceNames []string
		issueOpNames     []string
		issueCounter     int
	)
	s.androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, deviceName string, cmd *androidmanagement.Command) (*androidmanagement.Operation, error) {
		issueCallsMu.Lock()
		defer issueCallsMu.Unlock()
		issueCounter++
		issueCalls = append(issueCalls, cmd)
		issueDeviceNames = append(issueDeviceNames, deviceName)
		opName := fmt.Sprintf("%s/operations/op-%d", deviceName, issueCounter)
		issueOpNames = append(issueOpNames, opName)
		return &androidmanagement.Operation{Name: opName}, nil
	}

	lastIssue := func() (*androidmanagement.Command, string, string) {
		t.Helper()
		issueCallsMu.Lock()
		defer issueCallsMu.Unlock()
		require.NotEmpty(t, issueCalls, "expected IssueCommand to have been called")
		i := len(issueCalls) - 1
		return issueCalls[i], issueDeviceNames[i], issueOpNames[i]
	}

	// deliverPubSubCommand simulates an AMAPI COMMAND notification arriving at Fleet's Pub/Sub endpoint with the supplied Operation envelope.
	deliverPubSubCommand := func(t *testing.T, op androidmanagement.Operation) {
		t.Helper()
		body, err := json.Marshal(op)
		require.NoError(t, err)
		req := android_service.PubSubPushRequest{
			Attributes: map[string]string{"notificationType": string(android.PubSubCommand)},
			Data:       base64.StdEncoding.EncodeToString(body),
		}
		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", pubsubToken)
	}

	// assertHostMDMStatus fetches /hosts/{id} and asserts DeviceStatus + PendingAction. Both fields are always set to non-nil string
	// pointers by getHostDetails (zero-value status = pointer to empty string), so the assertion is unconditional.
	assertHostMDMStatus := func(t *testing.T, hostID uint, wantPendingAction, wantDeviceStatus string) {
		t.Helper()
		var resp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Host.MDM.PendingAction)
		require.Equal(t, wantPendingAction, *resp.Host.MDM.PendingAction)
		require.NotNil(t, resp.Host.MDM.DeviceStatus)
		require.Equal(t, wantDeviceStatus, *resp.Host.MDM.DeviceStatus)
	}

	coboHostID := createAndroidHostForTest(t, s.ds, nil, true)
	byoHostID := createAndroidHostForTest(t, s.ds, nil, false)

	t.Run("Lock COBO transitions to pending then locked via Pub/Sub ack", func(t *testing.T) {
		var lockResp fleet.LockHostResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", coboHostID), nil, http.StatusOK, &lockResp)
		require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
		require.Equal(t, fleet.DeviceStatusUnlocked, lockResp.DeviceStatus)

		cmd, _, opName := lastIssue()
		require.Equal(t, string(android.MDMAndroidCommandTypeLock), cmd.Type)
		require.Equal(t, longCommandDuration, cmd.Duration)
		require.Nil(t, cmd.WipeParams)

		row, err := s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusPending), row.Status)
		require.Equal(t, string(android.MDMAndroidCommandTypeLock), row.CommandType)

		// host_mdm_actions.lock_ref is the link the host page reads to surface PendingAction=lock.
		assertHostMDMStatus(t, coboHostID, "lock", "unlocked")

		// Simulate the AMAPI Pub/Sub COMMAND notification reporting a successful device ack.
		deliverPubSubCommand(t, androidmanagement.Operation{Name: opName, Done: true})

		row, err = s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), row.Status)

		// After ack, the host page surfaces DeviceStatus=unlocked: Fleet can't observe the device-side unlock (no AMAPI signal), so we
		// never report "locked" -- only "pending lock" during the brief window before the ack. See HostLockWipeStatus.IsLocked.
		assertHostMDMStatus(t, coboHostID, "", "unlocked")
	})

	t.Run("Wipe BYO issues WIPE and unenrolls via Pub/Sub ack", func(t *testing.T) {
		var wipeResp fleet.WipeHostResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", byoHostID), nil, http.StatusOK, &wipeResp)
		// The wipe response reports the pending wipe even though GET host suppresses it for BYO below.
		require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)

		cmd, _, opName := lastIssue()
		require.Equal(t, string(android.MDMAndroidCommandTypeWipe), cmd.Type)
		require.Equal(t, longCommandDuration, cmd.Duration)
		require.NotNil(t, cmd.WipeParams, "AMAPI requires a non-nil WipeParams even when empty")

		row, err := s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusPending), row.Status)
		require.Equal(t, string(android.MDMAndroidCommandTypeWipe), row.CommandType)
		s.lastHostActivityMatches(byoHostID, fleet.ActivityTypeWipedHost{}.ActivityName(), "", 0)

		// BYO wipe_ref is suppressed on the host page, the same as BYO Unenroll which also runs a WIPE.
		assertHostMDMStatus(t, byoHostID, "", "unlocked")

		deliverPubSubCommand(t, androidmanagement.Operation{Name: opName, Done: true})

		row, err = s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), row.Status)

		byoHostMDM, err := s.ds.GetHostMDM(ctx, byoHostID)
		require.NoError(t, err)
		require.False(t, byoHostMDM.Enrolled, "BYO Wipe ack must flip host_mdm.enrolled to false")
		s.lastHostActivityMatches(byoHostID, fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), "", 0)

		// Only the work profile was removed, so the ack clears wipe_ref and the host is not shown as wiped.
		assertHostMDMStatus(t, byoHostID, "", "unlocked")
	})

	t.Run("Custom WIPE command on BYO unenrolls via Pub/Sub ack", func(t *testing.T) {
		customByoHostID := createAndroidHostForTest(t, s.ds, nil, false)
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", customByoHostID), nil, http.StatusOK, &hostResp)

		var runResp runMDMCommandResponse
		s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
			Command:   base64.StdEncoding.EncodeToString([]byte(`{"wipeParams":{}}`)),
			HostUUIDs: []string{hostResp.Host.UUID},
		}, http.StatusOK, &runResp)
		require.Equal(t, string(android.MDMAndroidCommandTypeWipe), runResp.RequestType)

		cmd, _, opName := lastIssue()
		require.NotNil(t, cmd.WipeParams)

		row, err := s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandTypeWipe), row.CommandType)

		deliverPubSubCommand(t, androidmanagement.Operation{Name: opName, Done: true})

		customByoHostMDM, err := s.ds.GetHostMDM(ctx, customByoHostID)
		require.NoError(t, err)
		require.False(t, customByoHostMDM.Enrolled, "BYO custom WIPE ack must flip host_mdm.enrolled to false")
		s.lastHostActivityMatches(customByoHostID, fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), "", 0)
		assertHostMDMStatus(t, customByoHostID, "", "unlocked")
	})

	t.Run("Wipe COBO uses WIPE with empty WipeParams and long duration", func(t *testing.T) {
		// COBO host above is already locked; queueing a wipe on a locked host should still be allowed (wipe takes precedence). Re-create
		// a fresh COBO host to keep the assertion focused on the wipe-issue path, not on the lock/wipe interaction.
		wipeHostID := createAndroidHostForTest(t, s.ds, nil, true)

		var wipeResp fleet.WipeHostResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", wipeHostID), nil, http.StatusOK, &wipeResp)
		require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)

		cmd, _, opName := lastIssue()
		require.Equal(t, string(android.MDMAndroidCommandTypeWipe), cmd.Type)
		require.Equal(t, longCommandDuration, cmd.Duration)
		require.NotNil(t, cmd.WipeParams, "AMAPI requires a non-nil WipeParams even when empty")

		row, err := s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusPending), row.Status)
		require.Equal(t, string(android.MDMAndroidCommandTypeWipe), row.CommandType)

		assertHostMDMStatus(t, wipeHostID, "wipe", "unlocked")

		// COBO Wipe ack must flip host_mdm.enrolled directly. AMAPI does not reliably send a STATUS_REPORT / ENROLLMENT with
		// state=DELETED for a factory-reset COBO device (the agent is gone), so the COMMAND ack is the only authoritative unenroll
		// signal. Without this, the host page sticks at "MDM On" + "Wiped" badge forever.
		deliverPubSubCommand(t, androidmanagement.Operation{Name: opName, Done: true})

		row, err = s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), row.Status)

		wipeHostMDM, err := s.ds.GetHostMDM(ctx, wipeHostID)
		require.NoError(t, err)
		require.False(t, wipeHostMDM.Enrolled, "COBO Wipe ack must flip host_mdm.enrolled to false")

		// wipe_ref stays so the host page surfaces "Wiped" -- COBO Wipe is destructive and the
		// badge is correct. The host stays in Fleet until an admin deletes it manually or device re-enrolls.
		assertHostMDMStatus(t, wipeHostID, "", "wiped")
	})

	t.Run("Clear passcode uses RESET_PASSWORD with empty newPassword and transitions clear_passcode_ref through Pub/Sub ack", func(t *testing.T) {
		// Fresh host to keep assertions self-contained; BYO vs COBO doesn't matter for
		// clear-passcode (both surface the same "Clear passcode pending" device status).
		passHostID := createAndroidHostForTest(t, s.ds, nil, false)

		var cpResp fleet.ClearPasscodeResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", passHostID), nil, http.StatusOK, &cpResp)
		require.NotNil(t, cpResp.CommandEnqueueResult)
		require.Equal(t, string(android.MDMAndroidCommandTypeResetPassword), cpResp.RequestType)
		require.NotEmpty(t, cpResp.CommandUUID, "clear-passcode response must include the persisted command UUID for lookup")

		cmd, _, opName := lastIssue()
		require.Equal(t, string(android.MDMAndroidCommandTypeResetPassword), cmd.Type)
		require.Empty(t, cmd.NewPassword, "clear-passcode must send an empty newPassword to actually clear (not regenerate) the passcode")
		require.Equal(t, longCommandDuration, cmd.Duration)

		row, err := s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusPending), row.Status)
		require.Equal(t, string(android.MDMAndroidCommandTypeResetPassword), row.CommandType)
		require.Equal(t, row.CommandUUID, cpResp.CommandUUID,
			"the CommandUUID returned to API consumers must match the persisted row (#41683 review fix)")

		// host_mdm_actions.clear_passcode_ref is written so device_status flips to "clearing passcode" while the AMAPI command is in flight
		assertHostMDMStatus(t, passHostID, "clear_passcode", "unlocked")

		// Pub/Sub COMMAND ack clears the ref and returns the host to its baseline (unlocked, no pending action). Mirrors the symmetric
		// round-trip the Lock subtest above performs.
		deliverPubSubCommand(t, androidmanagement.Operation{Name: opName, Done: true})

		row, err = s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), row.Status)

		assertHostMDMStatus(t, passHostID, "", "unlocked")
	})

	t.Run("Pub/Sub COMMAND with op.Error transitions row to error", func(t *testing.T) {
		// Issue a lock against a fresh COBO host so we have a known pending row to fail.
		errHostID := createAndroidHostForTest(t, s.ds, nil, true)
		var lockResp fleet.LockHostResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", errHostID), nil, http.StatusOK, &lockResp)
		_, _, opName := lastIssue()

		// Deliver an Operation with an Error set -- AMAPI's signal that the device rejected the command.
		deliverPubSubCommand(t, androidmanagement.Operation{
			Name: opName,
			Done: true,
			Error: &androidmanagement.Status{
				Code:    13,
				Message: "internal error executing command",
			},
		})

		row, err := s.ds.GetMDMAndroidCommandByOperationName(ctx, opName)
		require.NoError(t, err)
		require.Equal(t, string(android.MDMAndroidCommandStatusError), row.Status)
		require.True(t, row.ErrorCode.Valid)
		require.Equal(t, "13", row.ErrorCode.V)
		require.True(t, row.ErrorMessage.Valid)
		require.Equal(t, "internal error executing command", row.ErrorMessage.V)
	})
}

func (s *integrationMDMTestSuite) TestAndroidEnterpriseDeletedDetection() {
	t := s.T()
	ctx := t.Context()

	// ----- SETUP

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	originalAppConfig := *appConfig
	appConfig.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := s.ds.SaveAppConfig(context.Background(), &originalAppConfig)
		require.NoError(t, err)
	})

	// ---- SETUP DONE

	s.androidAPIClient.InitCommonMocks()

	var proxyCallbackURL string
	s.androidAPIClient.SignupURLsCreateFunc = func(_ context.Context, _, callbackURL string) (*android.SignupDetails, error) {
		url, err := url.Parse(callbackURL)
		require.NoError(t, err)
		proxyCallbackURL = url.EscapedPath() + "?" + url.RawQuery
		return &android.SignupDetails{
			Url:  tests.EnterpriseSignupURL,
			Name: "signupUrls/Cb0812d0999c464f",
		}, nil
	}
	s.androidAPIClient.EnterprisesCreateFunc = func(_ context.Context, _ androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
		return androidmgmt.EnterprisesCreateResponse{
			EnterpriseName: "enterprises/" + tests.EnterpriseID,
			TopicName:      "projects/android/topics/ae98ed130-5ce2-4ddb-a90a-191ec76976d5",
		}, nil
	}
	s.androidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{}, nil
	}

	// First, create an enterprise to test with
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/latest/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)

	const enterpriseToken = "enterpriseToken"
	s.Do("GET", proxyCallbackURL, nil, http.StatusOK, "enterpriseToken", enterpriseToken)

	// Update LIST mock to return the enterprise after "creation"
	s.androidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{
			{Name: "enterprises/" + tests.EnterpriseID},
		}, nil
	}

	// Verify enterprise exists
	var resp android.GetEnterpriseResponse
	s.DoJSON("GET", "/api/latest/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(t, tests.EnterpriseID, resp.EnterpriseID)

	t.Run("enterprise deleted detection", func(t *testing.T) {
		// Mock EnterprisesListFunc to return empty list (enterprise deleted)
		s.androidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
			return []*androidmanagement.Enterprise{}, nil
		}

		// Attempt to get enterprise - should return 200 until cron is ran
		var getResp android.GetEnterpriseResponse
		s.DoJSON("GET", "/api/latest/fleet/android_enterprise", nil, http.StatusOK, &getResp)
		// Verify LIST API was not called
		assert.False(t, s.androidAPIClient.EnterprisesListFuncInvoked)

		// Trigger cron
		s.awaitRunCleanupSchedule()

		// Verify LIST API was called
		assert.True(t, s.androidAPIClient.EnterprisesListFuncInvoked)

		// Attempt to get enterprise - should now return 404 since cron detected deletion
		s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusNotFound, &getResp)
	})
}

func (s *integrationMDMTestSuite) TestAndroidEnroll() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	s.enableAndroidMDM(t)

	s.runWorkerUntilDoneWithChecks(true)

	// enroll a couple android devices on no-team - host1 company-owned and host2 personally-owned
	host1, _, _ := s.createAndEnrollAndroidDevice(t, "test-android1", nil, true)
	host2, _, _ := s.createAndEnrollAndroidDevice(t, "test-android2", nil, false)

	require.NotNil(t, host1.MDM.EnrollmentStatus)
	require.Equal(t, "On (automatic)", *host1.MDM.EnrollmentStatus)
	require.NotNil(t, host2.MDM.EnrollmentStatus)
	require.Equal(t, fleet.MDMEnrollmentStatusPersonal, *host2.MDM.EnrollmentStatus)

	// Do the same but with a team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test team", Secrets: []*fleet.EnrollSecret{{Secret: uuid.NewString()}}})
	require.NoError(t, err)

	host3, _, _ := s.createAndEnrollAndroidDevice(t, "test-android3", &tm.ID, true)
	host4, _, _ := s.createAndEnrollAndroidDevice(t, "test-android4", &tm.ID, false)

	require.NotNil(t, host3.MDM.EnrollmentStatus)
	require.Equal(t, "On (automatic)", *host3.MDM.EnrollmentStatus)
	require.NotNil(t, host4.MDM.EnrollmentStatus)
	require.Equal(t, fleet.MDMEnrollmentStatusPersonal, *host4.MDM.EnrollmentStatus)
}

func (s *integrationMDMTestSuite) TestAndroidReenrollKeepsDiskSpace() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	s.enableAndroidMDM(t)

	// The enrollment helper's payload carries MEASURED memory events: 35 GB free of 128 GB.
	host, deviceInfo, pubSubToken := s.createAndEnrollAndroidDevice(t, "android-disk-reenroll", nil, false)

	// AMAPI omits memory events from enrollment payloads and from status reports sent before
	// the device has measured storage, but still reports the 64 GB internal total.
	withoutMemoryEvents := func(msg *android.PubSubMessage) *android.PubSubMessage {
		data, err := base64.StdEncoding.DecodeString(msg.Data)
		require.NoError(t, err)
		var device androidmanagement.Device
		require.NoError(t, json.Unmarshal(data, &device))
		device.MemoryEvents = nil
		data, err = json.Marshal(device)
		require.NoError(t, err)
		msg.Data = base64.StdEncoding.EncodeToString(data)
		return msg
	}
	push := func(msg *android.PubSubMessage) {
		req := android_service.PubSubPushRequest{PubSubMessage: *msg}
		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))
	}
	requireDisk := func(hostID uint, available, percent, total float64) {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
		assert.InDelta(t, available, hostResp.Host.GigsDiskSpaceAvailable, 0.01)
		assert.InDelta(t, percent, hostResp.Host.PercentDiskSpaceAvailable, 0.01)
		assert.InDelta(t, total, hostResp.Host.GigsTotalDiskSpace, 0.01)
	}

	// Moves the host's enrollment and stored measurement an hour into the past, so their order
	// relative to later writes doesn't depend on last_enrolled_at's second precision.
	age := func(hostID uint) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(t.Context(), `UPDATE hosts SET last_enrolled_at = last_enrolled_at - INTERVAL 1 HOUR WHERE id = ?`, hostID); err != nil {
				return err
			}
			_, err := q.ExecContext(t.Context(), `UPDATE host_disks SET updated_at = updated_at - INTERVAL 1 HOUR WHERE host_id = ?`, hostID)
			return err
		})
	}

	requireDisk(host.ID, 35, 27.34, 128)
	age(host.ID)

	// Re-enroll, then a status report, neither carrying a measurement.
	push(withoutMemoryEvents(enrollmentMessageWithEnterpriseSpecificID(t, deviceInfo, host.UUID)))
	requireDisk(host.ID, 35, 27.34, 128)
	push(withoutMemoryEvents(statusReportMessageWithEnterpriseSpecificID(t, deviceInfo, host.UUID)))
	requireDisk(host.ID, 35, 27.34, 128)

	// A host that has measured since it enrolled reports what it sends: a status report without
	// a measurement reads as not supported.
	measuredHost, measuredDeviceInfo, _ := s.createAndEnrollAndroidDevice(t, "android-disk-measured", nil, false)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), `UPDATE hosts SET last_enrolled_at = last_enrolled_at - INTERVAL 1 HOUR WHERE id = ?`, measuredHost.ID)
		return err
	})
	push(withoutMemoryEvents(statusReportMessageWithEnterpriseSpecificID(t, measuredDeviceInfo, measuredHost.UUID)))
	requireDisk(measuredHost.ID, -1, -1, 64)

	// A device new to Fleet that has never measured storage still reads as not supported.
	newDeviceID := strings.ToUpper(uuid.New().String())
	push(withoutMemoryEvents(enrollmentMessageWithEnterpriseSpecificID(t, androidmanagement.Device{
		Name:                createAndroidDeviceID("android-disk-unmeasured"),
		EnrollmentTokenData: deviceInfo.EnrollmentTokenData,
	}, newDeviceID)))
	var hosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts, "query", newDeviceID)
	require.Len(t, hosts.Hosts, 1)
	requireDisk(hosts.Hosts[0].ID, -1, -1, 64)

	// The first host is unaffected by the other host's write.
	requireDisk(host.ID, 35, 27.34, 128)
}

// TestAndroidCustomCommandsListAndResults exercises the full API flow for Android custom commands:
// send a command via POST /commands/run, verify it appears in GET /commands (list), deliver a
// Pub/Sub ack, and verify GET /commands/results returns the stored result.
func (s *integrationMDMTestSuite) TestAndroidCustomCommandsListAndResults() {
	t := s.T()
	ctx := t.Context()

	enterpriseID, err := s.ds.CreateEnterprise(ctx, s.users["admin1"].ID)
	require.NoError(t, err)
	require.NoError(t, s.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		ID: enterpriseID, EnterpriseID: "custom-cmd-fake",
		SignupName:  "fake",
		SignupToken: "value",
		TopicID:     "yep",
	}))

	const pubsubToken = "test-android-custom-cmd-pubsub-token"
	require.NoError(t, s.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
		Name:  fleet.MDMAssetAndroidPubSubToken,
		Value: []byte(pubsubToken),
	}))

	var (
		issueCallsMu sync.Mutex
		issueCounter int
	)
	s.androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, deviceName string, cmd *androidmanagement.Command) (*androidmanagement.Operation, error) {
		issueCallsMu.Lock()
		defer issueCallsMu.Unlock()
		issueCounter++
		opName := fmt.Sprintf("%s/operations/custom-op-%d", deviceName, issueCounter)
		return &androidmanagement.Operation{Name: opName}, nil
	}

	coboHostID := createAndroidHostForTest(t, s.ds, nil, true)

	// Get the host UUID for API calls.
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", coboHostID), nil, http.StatusOK, &hostResp)
	hostUUID := hostResp.Host.UUID

	// Send a custom REBOOT command via the API.
	rebootJSON := `{"type":"REBOOT"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(rebootJSON))
	var runResp runMDMCommandResponse
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   encoded,
		HostUUIDs: []string{hostUUID},
	}, http.StatusOK, &runResp)

	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, "REBOOT", runResp.RequestType)
	require.Equal(t, "android", runResp.Platform)

	cmdUUID := runResp.CommandUUID

	// Verify the command appears in GET /commands?host_identifier=<uuid>.
	var listResp listMDMCommandsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands?host_identifier=%s", hostUUID), nil, http.StatusOK, &listResp)
	var found bool
	for _, cmd := range listResp.Results {
		if cmd.CommandUUID == cmdUUID {
			found = true
			assert.Equal(t, "REBOOT", cmd.RequestType)
			assert.Equal(t, hostUUID, cmd.HostUUID)
			break
		}
	}
	require.True(t, found, "expected the Android command to appear in the commands list")

	// Verify command results show pending (no Pub/Sub ack yet).
	var resultsResp getMDMCommandResultsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", cmdUUID), nil, http.StatusOK, &resultsResp)
	require.Len(t, resultsResp.Results, 1)
	assert.Equal(t, "Pending", resultsResp.Results[0].Status)
	assert.Equal(t, "REBOOT", resultsResp.Results[0].RequestType)

	// Deliver a Pub/Sub COMMAND ack.
	row, err := s.ds.GetMDMAndroidCommandByUUID(ctx, cmdUUID)
	require.NoError(t, err)

	ackOp := androidmanagement.Operation{
		Name: row.OperationName,
		Done: true,
	}
	body, err := json.Marshal(ackOp)
	require.NoError(t, err)
	req := android_service.PubSubPushRequest{
		Attributes: map[string]string{"notificationType": string(android.PubSubCommand)},
		Data:       base64.StdEncoding.EncodeToString(body),
	}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", pubsubToken)

	// Verify command results now show acknowledged with raw_result.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", cmdUUID), nil, http.StatusOK, &resultsResp)
	require.Len(t, resultsResp.Results, 1)
	assert.Equal(t, "Acknowledged", resultsResp.Results[0].Status)
	assert.NotEmpty(t, resultsResp.Results[0].Result, "raw_result should be populated after Pub/Sub ack")

	// Verify GetMDMCommandPlatform returns "android".
	p, err := s.ds.GetMDMCommandPlatform(ctx, cmdUUID)
	require.NoError(t, err)
	assert.Equal(t, "android", p)

	// Verify command_status filter works for Android hosts.
	var filteredResp listMDMCommandsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands?host_identifier=%s&command_status=ran", hostUUID), nil, http.StatusOK, &filteredResp)
	require.Len(t, filteredResp.Results, 1, "command_status=ran should return the acknowledged command")
	assert.Equal(t, cmdUUID, filteredResp.Results[0].CommandUUID)

	// Filtering by pending should return no results (the command already acked).
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands?host_identifier=%s&command_status=pending", hostUUID), nil, http.StatusOK, &filteredResp)
	require.Empty(t, filteredResp.Results, "command_status=pending should return no results after ack")

	// Verify raw_command stores the original payload (without injected duration).
	resultsResp = getMDMCommandResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", cmdUUID), nil, http.StatusOK, &resultsResp)
	require.Len(t, resultsResp.Results, 1)
	assert.JSONEq(t, rebootJSON, string(resultsResp.Results[0].Payload), "raw_command should match the original payload without injected duration")
}

// TestAndroidCustomCommandRejectedOnPersonallyOwnedHost covers the symptom reported in #53151: REBOOT on a BYOD work
// profile used to return 200 and record a "ran REBOOT" activity because AMAPI reports the operation as done with no
// error while the device ignores it. The command must now be refused outright, so no activity claims it ran.
func (s *integrationMDMTestSuite) TestAndroidCustomCommandRejectedOnPersonallyOwnedHost() {
	t := s.T()
	ctx := t.Context()

	enterpriseID, err := s.ds.CreateEnterprise(ctx, s.users["admin1"].ID)
	require.NoError(t, err)
	enterprise := android.Enterprise{ID: enterpriseID, EnterpriseID: "byod-cmd-fake"}
	require.NoError(t, s.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		Enterprise:  enterprise,
		SignupName:  "fake",
		SignupToken: "value",
		TopicID:     "yep",
	}))

	var issueCalls atomic.Int32
	s.androidAPIClient.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, deviceName string, _ *androidmanagement.Command) (*androidmanagement.Operation, error) {
		return &androidmanagement.Operation{Name: fmt.Sprintf("%s/operations/byod-op-%d", deviceName, issueCalls.Add(1))}, nil
	}

	byodHostID := createAndroidHostForTest(t, s.ds, nil, false)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", byodHostID), nil, http.StatusOK, &hostResp)
	hostUUID := hostResp.Host.UUID
	require.True(t, hostResp.Host.MDM.IsPersonalEnrollment)

	lastActivityID := s.lastActivityMatches("", "", 0)

	res := s.Do("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(`{"type":"REBOOT"}`)),
		HostUUIDs: []string{hostUUID},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body),
		"REBOOT is not supported for personally-owned Android hosts.")

	// Nothing reached AMAPI, so nothing can come back acknowledged.
	require.Zero(t, issueCalls.Load())

	// No command row, so the command does not show up as pending or ran for this host.
	var listResp listMDMCommandsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands?host_identifier=%s", hostUUID), nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.Results)

	// The reported symptom: no activity may claim the command ran. Asserting the newest activity is unchanged
	// also catches an activity written before the command is issued.
	require.Equal(t, lastActivityID, s.lastActivityMatches("", "", 0))

	// A command type AMAPI does support on a work profile is still issued, so the check is not over-broad.
	var runResp runMDMCommandResponse
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(`{"type":"CLEAR_APP_DATA","clearAppsDataParams":{"packageNames":["com.example.app"]}}`)),
		HostUUIDs: []string{hostUUID},
	}, http.StatusOK, &runResp)
	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, int32(1), issueCalls.Load())
	s.lastActivityMatches(fleet.ActivityTypeRanCustomMDMCommand{}.ActivityName(), "", 0)
}
