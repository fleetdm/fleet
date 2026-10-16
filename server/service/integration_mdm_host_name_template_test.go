package service

// Host name template tests for the MDM suite.
//
// Belongs here: end-to-end host name templates for fleets and "No team", re-enrollment with stale
// names, IdP variables and group changes, secrets, fleet specs, and moving hosts between fleets.
//
// Does not belong here: secret variable scoping for host name templates
// (integration_mdm_host_name_template_scope_test.go).

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestHostNameTemplateEndToEnd() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	setFleetMDMData := func(hostID uint, personal bool) {
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", manualProfileIf(personal)))
	}

	// A macOS host that walks the whole pipeline: command, ack, osquery verify, drift.
	macHost, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(macHost.ID, false)

	// A macOS host whose name already matches the resolved template and that has
	// reported since it enrolled: verified directly, no command.
	matchingHost, matchingDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(matchingHost.ID, false)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(ctx, `UPDATE hosts SET computer_name = ?, detail_updated_at = NOW() WHERE id = ?`,
			"WS-"+matchingHost.HardwareSerial, matchingHost.ID); err != nil {
			return err
		}
		_, err := q.ExecContext(ctx, `UPDATE nano_devices SET authenticate_at = DATE_SUB(NOW(), INTERVAL 1 HOUR) WHERE id = ?`, matchingHost.UUID)
		return err
	})

	// An iOS host that acks the rename and later verifies through the
	// DeviceInformation refetch path.
	iosHost, iosDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	setFleetMDMData(iosHost.ID, false)

	// An iOS host whose device rejects the command (e.g. unsupervised).
	iosFailHost, iosFailDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	setFleetMDMData(iosFailHost.ID, false)

	// A BYOD (personal enrollment) host: never enforced.
	byodHost, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	setFleetMDMData(byodHost.ID, true)

	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID,
		[]uint{macHost.ID, matchingHost.ID, iosHost.ID, iosFailHost.ID, byodHost.ID})))

	requireRowStatus := func(hostUUID string, want *fleet.MDMDeliveryStatus) *fleet.HostDeviceNameEnforcement {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
		return row
	}
	requireNoRow := func(hostUUID string) {
		_, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.True(t, fleet.IsNotFound(err))
	}
	runDeviceNameCron := func() {
		require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	}

	// --- endpoint validation ---
	// (An omitted fleet_id targets "No team", covered by
	// TestHostNameTemplateNoTeamEndToEnd; this test drives the team scope.)
	res := s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: "WS-$FLEET_VAR_NDES_SCEP_CHALLENGE"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "not supported in host name templates")
	// A custom (secret) variable is allowed, but an undefined one is rejected at
	// save time with the missing-secret error.
	res = s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: "WS-$FLEET_SECRET_UNDEFINED"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "missing from database")
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: "WS-\x07"}, http.StatusUnprocessableEntity)
	// fixed text alone longer than the 63-byte device-name limit is rejected up front
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: strings.Repeat("N", 64)}, http.StatusUnprocessableEntity)
	// nothing was created or logged by the failed attempts
	requireNoRow(macHost.UUID)

	// --- set the template ---
	const tmpl = "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL"
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: tmpl}, http.StatusNoContent)
	activityID := s.lastActivityMatches("edited_host_name_template",
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "name_template": %q}`, team.ID, team.Name, tmpl), 0)

	// eligible hosts are queued; the BYOD host has no row
	requireRowStatus(macHost.UUID, nil)
	requireRowStatus(matchingHost.UUID, nil)
	requireRowStatus(iosHost.UUID, nil)
	requireRowStatus(iosFailHost.UUID, nil)
	requireNoRow(byodHost.UUID)

	// re-saving the identical template emits no duplicate activity
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: tmpl}, http.StatusNoContent)
	s.lastActivityMatches("edited_host_name_template", "", activityID)

	// --- cron: resolve + enqueue ---
	runDeviceNameCron()

	// the already-matching host goes straight to verified, and its device got no command
	matchingRow := requireRowStatus(matchingHost.UUID, &fleet.MDMDeliveryVerified)
	require.NotNil(t, matchingRow.ExpectedDeviceName)
	require.Equal(t, "WS-"+matchingHost.HardwareSerial, *matchingRow.ExpectedDeviceName)
	cmd, err := matchingDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// the other eligible hosts are pending on a DEVNAME- command with the resolved name
	macRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, macRow.CommandUUID)
	require.True(t, strings.HasPrefix(*macRow.CommandUUID, fleet.DeviceNameCommandUUIDPrefix))
	require.NotNil(t, macRow.ExpectedDeviceName)
	require.Equal(t, "WS-"+macHost.HardwareSerial, *macRow.ExpectedDeviceName)

	// --- mac: device receives the Settings command and acknowledges ---
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	require.Equal(t, *macRow.CommandUUID, cmd.CommandUUID)
	var settingsCmd struct {
		Command struct {
			Settings []struct {
				Item       string
				DeviceName string
			}
		}
	}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &settingsCmd))
	require.Len(t, settingsCmd.Command.Settings, 1)
	require.Equal(t, "DeviceName", settingsCmd.Command.Settings[0].Item)
	require.Equal(t, "WS-"+macHost.HardwareSerial, settingsCmd.Command.Settings[0].DeviceName)

	_, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// the ack renamed the host in Fleet and moved the row to verifying
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerifying)
	renamedMac, err := s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Equal(t, "WS-"+macHost.HardwareSerial, renamedMac.ComputerName)
	require.Equal(t, "WS-"+macHost.HardwareSerial, renamedMac.Hostname)
	require.Equal(t, "WS-"+macHost.HardwareSerial, renamedMac.DisplayName())

	// --- mac: osquery reports the matching name -> verified ---
	submitSystemInfo := func(computerName string) {
		distributedReq := SubmitDistributedQueryResultsRequest{
			NodeKey: *macHost.NodeKey,
			Results: map[string][]map[string]string{
				"fleet_detail_query_system_info": {
					{
						"computer_name":      computerName,
						"hostname":           computerName,
						"uuid":               macHost.UUID,
						"hardware_serial":    macHost.HardwareSerial,
						"hardware_model":     "MacBookPro16,1",
						"physical_memory":    "16000000000",
						"cpu_physical_cores": "8",
						"cpu_logical_cores":  "8",
					},
				},
			},
			Statuses: map[string]fleet.OsqueryStatus{
				"fleet_detail_query_system_info": 0,
			},
		}
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
	}

	// a report that was generated before the device applied the rename (still
	// carrying the old name) arriving right after the ack is not drift: the row
	// stays verifying until a fresh report decides it
	submitSystemInfo("stale-pre-rename-name")
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerifying)

	submitSystemInfo("WS-" + macHost.HardwareSerial)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerified)

	// --- mac: the end user renames the device -> drift -> re-enforced ---
	submitSystemInfo("Renamed by user")
	requireRowStatus(macHost.UUID, nil)
	runDeviceNameCron()
	reenforcedRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotEqual(t, *macRow.CommandUUID, *reenforcedRow.CommandUUID)
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, *reenforcedRow.CommandUUID, cmd.CommandUUID)
	var reenforceCmd struct {
		Command struct {
			Settings []struct {
				Item       string
				DeviceName string
			}
		}
	}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &reenforceCmd))
	require.Len(t, reenforceCmd.Command.Settings, 1)
	require.Equal(t, "WS-"+macHost.HardwareSerial, reenforceCmd.Command.Settings[0].DeviceName)
	_, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerifying)
	submitSystemInfo("WS-" + macHost.HardwareSerial)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerified)

	// --- iOS: ack then verify through the DeviceInformation refetch path ---
	cmd, err = iosDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	_, err = iosDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(iosHost.UUID, &fleet.MDMDeliveryVerifying)

	refetchIOS := func(deviceName string) {
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", iosHost.ID), nil, http.StatusOK)
		cmd, err := iosDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				cmd, err = iosDevice.AcknowledgeInstalledApplicationList(iosDevice.UUID, cmd.CommandUUID, nil)
			case "CertificateList":
				cmd, err = iosDevice.AcknowledgeCertificateList(iosDevice.UUID, cmd.CommandUUID, nil)
			case "DeviceInformation":
				cmd, err = iosDevice.AcknowledgeDeviceInformation(iosDevice.UUID, cmd.CommandUUID,
					deviceName, "iPhone14,6", "America/Los_Angeles")
			default:
				cmd, err = iosDevice.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)
		}
	}
	refetchIOS("WS-" + iosHost.HardwareSerial)
	requireRowStatus(iosHost.UUID, &fleet.MDMDeliveryVerified)

	// --- iOS: the end user renames the device -> drift -> re-enforced ---
	refetchIOS("Renamed by user")
	requireRowStatus(iosHost.UUID, nil)
	runDeviceNameCron()
	requireRowStatus(iosHost.UUID, &fleet.MDMDeliveryPending)
	cmd, err = iosDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	_, err = iosDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(iosHost.UUID, &fleet.MDMDeliveryVerifying)

	// --- iOS failure: the device errors the command ---
	// Each error re-queues the row until the retries are used up, then it fails.
	failIOSCommand := func(errChain mdm.ErrorChain) {
		cmd, err := iosFailDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "Settings", cmd.Command.RequestType)
		_, err = iosFailDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{errChain})
		require.NoError(t, err)
	}
	transientErr := mdm.ErrorChain{ErrorCode: 99, ErrorDomain: "MCMDMErrorDomain", USEnglishDescription: "Something went wrong."}
	for i := 1; i <= servermdm.MaxAppleDeviceNameRetries; i++ {
		failIOSCommand(transientErr)
		retriedRow := requireRowStatus(iosFailHost.UUID, nil)
		require.EqualValues(t, i, retriedRow.Retries)
		runDeviceNameCron()
		requireRowStatus(iosFailHost.UUID, &fleet.MDMDeliveryPending)
	}
	failIOSCommand(transientErr)
	failedRow := requireRowStatus(iosFailHost.UUID, &fleet.MDMDeliveryFailed)
	require.Contains(t, failedRow.Detail, "Something went wrong.")

	// once retries are used up, the failed command is not re-sent by subsequent cron runs
	runDeviceNameCron()
	requireRowStatus(iosFailHost.UUID, &fleet.MDMDeliveryFailed)
	cmd, err = iosFailDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// An unsupervised device can never apply the rename, so after a resend that
	// rejection fails the row right away instead of using retries.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/name_template/resend", iosFailHost.ID), nil, http.StatusAccepted)
	runDeviceNameCron()
	failIOSCommand(mdm.ErrorChain{ErrorCode: 12026, ErrorDomain: "MCMDMErrorDomain", USEnglishDescription: "The device is not supervised."})
	failedRow = requireRowStatus(iosFailHost.UUID, &fleet.MDMDeliveryFailed)
	require.Contains(t, failedRow.Detail, "The device is not supervised.")
	require.Zero(t, failedRow.Retries)
	runDeviceNameCron()
	cmd, err = iosFailDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// --- a template resolving past 63 bytes fails without sending a command ---
	// The fixed text (50 bytes) passes save-time validation, but expanding the
	// UUID (36 chars) pushes the resolved name over the 63-byte limit — that
	// per-host overflow is only detectable at resolve time, by the cron.
	longTemplate := strings.Repeat("N", 50) + "$FLEET_VAR_HOST_UUID"
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: longTemplate}, http.StatusNoContent)
	requireRowStatus(macHost.UUID, nil) // rows re-queued by the new template
	runDeviceNameCron()
	longRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryFailed)
	require.Equal(t, "Resolved name exceeds 63 bytes.", longRow.Detail)
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// --- an APNs push failure doesn't lose or duplicate the command ---
	// The mac already carries the template name, so rename it off-template (its
	// failed row ignores the report) to make the cron send a command.
	submitSystemInfo("Renamed by user")
	// re-save the resolvable template to queue rows again
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: tmpl}, http.StatusNoContent)
	originalPushMock := s.pushProvider.PushFunc
	// Restore via defer so a failed assertion below can't leak the failing push
	// mock into later tests. Nothing after this in the test pushes (the second
	// cron run has no pending rows; Idle/Acknowledge are device-driven), so
	// keeping the mock active until the test ends is harmless.
	defer func() { s.pushProvider.PushFunc = originalPushMock }()
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		res := make(map[string]*push.Response, len(pushes))
		for _, p := range pushes {
			res[p.Token.String()] = &push.Response{Id: uuid.New().String(), Err: errors.New("APNs is down")}
		}
		return res, nil
	}
	runDeviceNameCron()

	// the command was persisted despite the failed push, so the row is pending
	// on it and a later cron run does not enqueue a duplicate
	apnsRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, apnsRow.CommandUUID)
	runDeviceNameCron()
	afterRetry := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, afterRetry.CommandUUID)
	require.Equal(t, *apnsRow.CommandUUID, *afterRetry.CommandUUID)

	// the device still receives that single command on its next check-in
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	require.Equal(t, *apnsRow.CommandUUID, cmd.CommandUUID)
	cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	require.Nil(t, cmd)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerifying)

	// --- clearing deletes all rows and never renames ---
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: ""}, http.StatusNoContent)
	s.lastActivityMatches("edited_host_name_template",
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "name_template": null}`, team.ID, team.Name), 0)
	for _, h := range []*fleet.Host{macHost, matchingHost, iosHost, iosFailHost} {
		requireNoRow(h.UUID)
	}
	// the host keeps the last name it was given; clearing renames nothing
	unchangedMac, err := s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Equal(t, "WS-"+macHost.HardwareSerial, unchangedMac.ComputerName)

	// --- custom (secret) variable: expanded into the resolved name ---
	secretID, err := s.ds.CreateSecretVariable(ctx, "SITE", "HQ")
	require.NoError(t, err)
	const secretTmpl = "${FLEET_SECRET_SITE}-$FLEET_VAR_HOST_HARDWARE_SERIAL" //nolint:gosec // G101: name template string, not a credential
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: secretTmpl}, http.StatusNoContent)
	// the activity records the unexpanded template (the secret placeholder is
	// never stored expanded)
	s.lastActivityMatches("edited_host_name_template",
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "name_template": %q}`, team.ID, team.Name, secretTmpl), 0)

	requireRowStatus(macHost.UUID, nil)
	runDeviceNameCron()
	secretRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, secretRow.ExpectedDeviceName)
	// the secret value ("HQ") is expanded alongside the built-in serial variable
	require.Equal(t, "HQ-"+macHost.HardwareSerial, *secretRow.ExpectedDeviceName)
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	var secretSettingsCmd struct {
		Command struct {
			Settings []struct {
				Item       string
				DeviceName string
			}
		}
	}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &secretSettingsCmd))
	require.Len(t, secretSettingsCmd.Command.Settings, 1)
	require.Equal(t, "HQ-"+macHost.HardwareSerial, secretSettingsCmd.Command.Settings[0].DeviceName)
	_, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerifying)

	// --- changing the secret value re-enqueues a fresh command ---
	_, _, err = s.ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{{Name: "SITE", Value: "NYC"}})
	require.NoError(t, err)
	// the secret change reset the enforcement row back to queued
	requireRowStatus(macHost.UUID, nil)
	runDeviceNameCron()
	changedSecretRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, changedSecretRow.ExpectedDeviceName)
	require.Equal(t, "NYC-"+macHost.HardwareSerial, *changedSecretRow.ExpectedDeviceName)
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	var changedSecretCmd struct {
		Command struct {
			Settings []struct {
				Item       string
				DeviceName string
			}
		}
	}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &changedSecretCmd))
	require.Len(t, changedSecretCmd.Command.Settings, 1)
	require.Equal(t, "NYC-"+macHost.HardwareSerial, changedSecretCmd.Command.Settings[0].DeviceName)

	// the secret can't be deleted while a host name template references it
	_, err = s.ds.DeleteSecretVariable(ctx, secretID)
	require.Error(t, err)
	var secretUsed *fleet.SecretUsedError
	require.ErrorAs(t, err, &secretUsed)
	require.Equal(t, "host_name_template", secretUsed.Entity.Type) // nolint:nilaway // set by the previous require.ErrorAs

	// clearing the template releases the secret for deletion
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: ""}, http.StatusNoContent)
	_, err = s.ds.DeleteSecretVariable(ctx, secretID)
	require.NoError(t, err)
}

func (s *integrationMDMTestSuite) TestHostNameTemplateReenrollWithStaleName() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, device := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", manualProfileIf(false)))
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL"}, http.StatusNoContent)
	want := "WS-" + host.HardwareSerial

	// A re-provisioned device: Fleet's host record still carries the template name
	// from before the device was erased, and its last report predates the
	// re-enrollment below.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE hosts SET computer_name = ?, detail_updated_at = DATE_SUB(NOW(), INTERVAL 1 HOUR) WHERE id = ?`, want, host.ID)
		return err
	})
	require.NoError(t, device.Enroll())

	row, err := s.ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.NoError(t, err)
	require.Nil(t, row.Status, "re-enrollment should queue the host")

	// The stored name is stale, so the cron must send the rename instead of
	// trusting it and marking the host verified.
	require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	row, err = s.ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, row.Status)
	require.Equal(t, fleet.MDMDeliveryPending, *row.Status)
	require.NotNil(t, row.CommandUUID)

	cmd, err := device.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	require.Equal(t, *row.CommandUUID, cmd.CommandUUID)
	_, err = device.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// The first report after re-enrollment matches, so no retry is spent.
	_, err = s.ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, want)
	require.NoError(t, err)
	row, err = s.ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMDeliveryVerified, *row.Status)
	require.Zero(t, row.Retries)
}

func (s *integrationMDMTestSuite) TestHostNameTemplateNoTeamEndToEnd() {
	t := s.T()
	ctx := t.Context()

	setFleetMDMData := func(hostID uint, personal bool) {
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", manualProfileIf(personal)))
	}

	// The host stays in "No team" (we never call AddHostsToTeam). BYOD exclusion
	// for the No-team scope is covered deterministically by the datastore test
	// (testHostDeviceNamesNoTeam); here we focus on the cron/ack/verify/host-detail
	// /resend pipeline resolving the template from the global app config.
	macHost, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(macHost.ID, false)

	require.Nil(t, macHost.TeamID)

	requireRowStatus := func(hostUUID string, want *fleet.MDMDeliveryStatus) *fleet.HostDeviceNameEnforcement {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
		return row
	}
	requireNoRow := func(hostUUID string) {
		_, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.True(t, fleet.IsNotFound(err))
	}
	runDeviceNameCron := func() {
		require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	}

	// --- set the No-team template (nil fleet_id) ---
	// An invalid variable is still rejected on the No-team path.
	res := s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: "WS-$FLEET_VAR_NDES_SCEP_CHALLENGE"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "not supported in host name templates")

	const tmpl = "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL"
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: tmpl}, http.StatusNoContent)
	// the activity carries a null fleet_id/fleet_name for No team
	activityID := s.lastActivityMatches("edited_host_name_template",
		fmt.Sprintf(`{"fleet_id": null, "fleet_name": null, "name_template": %q}`, tmpl), 0)

	// the global app config now carries the template
	ac, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, tmpl, ac.MDM.HostNameTemplate.Value)

	// eligible No-team host queued
	requireRowStatus(macHost.UUID, nil)

	// re-saving the identical template is a no-op (no duplicate activity)
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: tmpl}, http.StatusNoContent)
	s.lastActivityMatches("edited_host_name_template", "", activityID)

	// --- cron resolves the No-team template from app config and enqueues ---
	runDeviceNameCron()
	macRow := requireRowStatus(macHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, macRow.CommandUUID)
	require.True(t, strings.HasPrefix(*macRow.CommandUUID, fleet.DeviceNameCommandUUIDPrefix))
	require.Equal(t, "WS-"+macHost.HardwareSerial, *macRow.ExpectedDeviceName)

	// device receives the Settings command and acknowledges → rename + verifying
	cmd, err := macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	_, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerifying)
	renamedMac, err := s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Equal(t, "WS-"+macHost.HardwareSerial, renamedMac.ComputerName)

	submitSystemInfo := func(computerName string) {
		distributedReq := SubmitDistributedQueryResultsRequest{
			NodeKey: *macHost.NodeKey,
			Results: map[string][]map[string]string{
				"fleet_detail_query_system_info": {{
					"computer_name":      computerName,
					"hostname":           computerName,
					"uuid":               macHost.UUID,
					"hardware_serial":    macHost.HardwareSerial,
					"hardware_model":     "MacBookPro16,1",
					"physical_memory":    "16000000000",
					"cpu_physical_cores": "8",
					"cpu_logical_cores":  "8",
				}},
			},
			Statuses: map[string]fleet.OsqueryStatus{"fleet_detail_query_system_info": 0},
		}
		s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &submitDistributedQueryResultsResponse{})
	}

	// osquery reports the matching name → verified
	submitSystemInfo("WS-" + macHost.HardwareSerial)
	requireRowStatus(macHost.UUID, &fleet.MDMDeliveryVerified)

	// end user renames the device off-template → drift → re-queued
	submitSystemInfo("Renamed by user")
	requireRowStatus(macHost.UUID, nil)

	// --- host detail exposes the host_name object for the No-team host ---
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", macHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.HostName)
	require.Equal(t, fleet.HostNameSettingPending, getHostResp.Host.MDM.OSSettings.HostName.Status)

	// --- resend works for the No-team host (host-keyed, nil TeamID allowed) ---
	// Queued rows can't be resent, so first simulate the device rejecting a command.
	require.NoError(t, s.ds.SetHostDeviceNameStatus(ctx, macHost.UUID, fleet.MDMDeliveryFailed, nil, "", "rejected"))
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/name_template/resend", macHost.ID), nil, http.StatusAccepted)
	requireRowStatus(macHost.UUID, nil) // reset to queued

	// --- aggregate: the No-team OS-settings summary folds the rename in ---
	// (exact bucket counts are asserted in the datastore test; here we confirm the
	// No-team host list filter surfaces the queued host under the No-team scope.)
	noTeam := uint(0)
	pendingHosts, err := s.ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin},
		fleet.HostListOptions{TeamFilter: &noTeam, OSSettingsFilter: fleet.OSSettingsPending})
	require.NoError(t, err)
	pendingIDs := make([]uint, 0, len(pendingHosts))
	for _, h := range pendingHosts {
		pendingIDs = append(pendingIDs, h.ID)
	}
	require.Contains(t, pendingIDs, macHost.ID)

	// --- clearing deletes the row and emits a null activity (nil fleet_id) ---
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: ""}, http.StatusNoContent)
	s.lastActivityMatches("edited_host_name_template",
		`{"fleet_id": null, "fleet_name": null, "name_template": null}`, 0)
	requireNoRow(macHost.UUID)
}

func (s *integrationMDMTestSuite) TestHostNameTemplateIDPVariables() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	setFleetMDMData := func(hostID uint) {
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}

	// A host mapped to an IdP user.
	idpHost, idpDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(idpHost.ID)
	scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{
		UserName:   "jdoe@example.com",
		GivenName:  new("Jane"),
		FamilyName: new("Doe"),
		Department: new("Engineering"),
	})
	require.NoError(t, err)
	_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, idpHost.ID, scimUserID)
	require.NoError(t, err)

	// A host with no IdP user mapped.
	noIDPHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(noIDPHost.ID)

	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID,
		[]uint{idpHost.ID, noIDPHost.ID})))

	requireRowStatus := func(hostUUID string, want *fleet.MDMDeliveryStatus) *fleet.HostDeviceNameEnforcement {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
		return row
	}
	runDeviceNameCron := func() {
		require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	}
	deviceNameFromCmd := func(raw []byte) string {
		var sc struct {
			Command struct {
				Settings []struct {
					Item       string
					DeviceName string
				}
			}
		}
		require.NoError(t, plist.Unmarshal(raw, &sc))
		require.Len(t, sc.Command.Settings, 1)
		require.Equal(t, "DeviceName", sc.Command.Settings[0].Item)
		return sc.Command.Settings[0].DeviceName
	}

	// --- set a template that uses an IdP variable ---
	const tmpl = "WS-$FLEET_VAR_HOST_END_USER_IDP_USERNAME"
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: tmpl}, http.StatusNoContent)

	// --- cron: resolves the IdP value for the mapped host; fails the unmapped one ---
	runDeviceNameCron()

	idpRow := requireRowStatus(idpHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, idpRow.ExpectedDeviceName)
	require.Equal(t, "WS-jdoe@example.com", *idpRow.ExpectedDeviceName)

	// a host with no IdP user fails exactly like a profile would, with the same detail
	noIDPRow := requireRowStatus(noIDPHost.UUID, &fleet.MDMDeliveryFailed)
	require.Contains(t, noIDPRow.Detail, "no IdP username for this host")

	// the mapped host received the DeviceName command carrying the resolved IdP value
	cmd, err := idpDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "Settings", cmd.Command.RequestType)
	require.Equal(t, "WS-jdoe@example.com", deviceNameFromCmd(cmd.Raw))
	_, err = idpDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(idpHost.UUID, &fleet.MDMDeliveryVerifying)

	// A host in another team with an identity-only template, mapped to the *same*
	// IdP user, driven to a settled (verifying) state. An IdP change must NOT
	// re-enqueue it — its resolved name can't depend on IdP data.
	identityTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-identity"})
	require.NoError(t, err)
	identityHost, identityDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(identityHost.ID)
	_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, identityHost.ID, scimUserID)
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&identityTeam.ID, []uint{identityHost.ID})))
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &identityTeam.ID, HostNameTemplate: "SN-$FLEET_VAR_HOST_HARDWARE_SERIAL"}, http.StatusNoContent)
	runDeviceNameCron()
	requireRowStatus(identityHost.UUID, &fleet.MDMDeliveryPending)
	icmd, err := identityDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, icmd)
	_, err = identityDevice.Acknowledge(icmd.CommandUUID)
	require.NoError(t, err)
	requireRowStatus(identityHost.UUID, &fleet.MDMDeliveryVerifying)

	// --- changing the IdP value re-enqueues a fresh command (mirrors profile resend) ---
	_, err = s.ds.ReplaceScimUser(ctx, &fleet.ScimUser{
		ID:         scimUserID,
		UserName:   "jsmith@example.com",
		GivenName:  new("Jane"),
		FamilyName: new("Smith"),
		Department: new("Engineering"),
	})
	require.NoError(t, err)

	// the IdP-template host's row was reset back to queued...
	requireRowStatus(idpHost.UUID, nil)
	// ...but the identity-only-template host mapped to the same IdP user is untouched.
	requireRowStatus(identityHost.UUID, &fleet.MDMDeliveryVerifying)

	runDeviceNameCron()
	changedRow := requireRowStatus(idpHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, changedRow.ExpectedDeviceName)
	require.Equal(t, "WS-jsmith@example.com", *changedRow.ExpectedDeviceName)
	cmd, err = idpDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "WS-jsmith@example.com", deviceNameFromCmd(cmd.Raw))
}

// TestHostNameTemplateIDPResolution covers resolving a template that mixes
// identity, IdP, and custom (secret) variables in one string, the
// username-local-part variable, and the fail-hard behavior when a referenced IdP
// field is empty for a host (matching how configuration profiles treat it).
func (s *integrationMDMTestSuite) TestHostNameTemplateIDPResolution() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	setFleetMDMData := func(hostID uint) {
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}

	// A host whose IdP user has a username and a department.
	fullHost, fullDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(fullHost.ID)
	fullUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{
		UserName:   "jdoe@corp.com",
		GivenName:  new("Jane"),
		FamilyName: new("Doe"),
		Department: new("Eng"),
	})
	require.NoError(t, err)
	_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, fullHost.ID, fullUserID)
	require.NoError(t, err)

	// A host whose IdP user has a username but no department.
	noDeptHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(noDeptHost.ID)
	noDeptUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "nodept@corp.com"})
	require.NoError(t, err)
	_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, noDeptHost.ID, noDeptUserID)
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID,
		[]uint{fullHost.ID, noDeptHost.ID})))

	requireRowStatus := func(hostUUID string, want *fleet.MDMDeliveryStatus) *fleet.HostDeviceNameEnforcement {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
		return row
	}
	deviceNameFromCmd := func(raw []byte) string {
		var sc struct {
			Command struct {
				Settings []struct {
					Item       string
					DeviceName string
				}
			}
		}
		require.NoError(t, plist.Unmarshal(raw, &sc))
		require.Len(t, sc.Command.Settings, 1)
		require.Equal(t, "DeviceName", sc.Command.Settings[0].Item)
		return sc.Command.Settings[0].DeviceName
	}

	// A single template mixing a custom (secret) variable, an identity variable, the
	// username-local-part IdP variable, and the department IdP variable.
	secretID, err := s.ds.CreateSecretVariable(ctx, "SITE", "HQ")
	require.NoError(t, err)
	const tmpl = "${FLEET_SECRET_SITE}-${FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART}-$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT" //nolint:gosec // G101: name template string, not a credential
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: tmpl}, http.StatusNoContent)

	require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))

	// the fully-populated host resolves all three variable kinds: secret "HQ", the
	// local part of the username ("jdoe"), and the department ("Eng").
	fullRow := requireRowStatus(fullHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, fullRow.ExpectedDeviceName)
	require.Equal(t, "HQ-jdoe-Eng", *fullRow.ExpectedDeviceName)
	cmd, err := fullDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "HQ-jdoe-Eng", deviceNameFromCmd(cmd.Raw))

	// the host whose IdP user has no department fails the same way a profile would —
	// an empty IdP field is treated like a missing one.
	noDeptRow := requireRowStatus(noDeptHost.UUID, &fleet.MDMDeliveryFailed)
	require.Contains(t, noDeptRow.Detail, "no IdP department for this host")

	// cleanup: clearing the template releases the secret for deletion.
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: ""}, http.StatusNoContent)
	_, err = s.ds.DeleteSecretVariable(ctx, secretID)
	require.NoError(t, err)
}

// TestHostNameTemplateSecretReenqueue covers the secret-value change trigger: it
// re-queues teams and "No team" whose template references the changed secret,
// leaves scopes referencing a similarly-named secret (SITE vs SITE_CODE) alone, and
// is a no-op when a secret is re-upserted with an unchanged value.
func (s *integrationMDMTestSuite) TestHostNameTemplateSecretReenqueue() {
	t := s.T()
	ctx := t.Context()

	setFleetMDMData := func(hostID uint) {
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}
	requireRowStatus := func(hostUUID string, want *fleet.MDMDeliveryStatus) {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
	}

	_, err := s.ds.CreateSecretVariable(ctx, "SITE", "HQ")
	require.NoError(t, err)
	_, err = s.ds.CreateSecretVariable(ctx, "SITE_CODE", "C1")
	require.NoError(t, err)

	// A team whose template uses $FLEET_SECRET_SITE.
	siteTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-site"})
	require.NoError(t, err)
	siteHost, siteDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(siteHost.ID)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&siteTeam.ID, []uint{siteHost.ID})))

	// A team whose template uses the similarly-named $FLEET_SECRET_SITE_CODE.
	codeTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-code"})
	require.NoError(t, err)
	codeHost, codeDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(codeHost.ID)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&codeTeam.ID, []uint{codeHost.ID})))

	// A "No team" host whose global template uses $FLEET_SECRET_SITE.
	noTeamHost, noTeamDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(noTeamHost.ID)
	require.Nil(t, noTeamHost.TeamID)

	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &siteTeam.ID, HostNameTemplate: "${FLEET_SECRET_SITE}-x"}, http.StatusNoContent) //nolint:gosec // G101: template
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &codeTeam.ID, HostNameTemplate: "${FLEET_SECRET_SITE_CODE}-x"}, http.StatusNoContent) //nolint:gosec // G101: template
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: "${FLEET_SECRET_SITE}-y"}, http.StatusNoContent) //nolint:gosec // G101: template

	// Drive all three to a settled (verifying) state.
	require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	for _, d := range []*mdmtest.TestAppleMDMClient{siteDevice, codeDevice, noTeamDevice} {
		cmd, err := d.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "Settings", cmd.Command.RequestType)
		_, err = d.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	requireRowStatus(siteHost.UUID, &fleet.MDMDeliveryVerifying)
	requireRowStatus(codeHost.UUID, &fleet.MDMDeliveryVerifying)
	requireRowStatus(noTeamHost.UUID, &fleet.MDMDeliveryVerifying)

	// Re-upserting SITE with its current value changes nothing → no re-queue.
	_, _, err = s.ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{{Name: "SITE", Value: "HQ"}})
	require.NoError(t, err)
	requireRowStatus(siteHost.UUID, &fleet.MDMDeliveryVerifying)
	requireRowStatus(noTeamHost.UUID, &fleet.MDMDeliveryVerifying)

	// Changing SITE re-queues the SITE team and the No-team host, but not the
	// SITE_CODE team (the trailing word boundary prevents SITE from matching
	// SITE_CODE).
	_, _, err = s.ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{{Name: "SITE", Value: "NYC"}})
	require.NoError(t, err)
	requireRowStatus(siteHost.UUID, nil)
	requireRowStatus(noTeamHost.UUID, nil)
	requireRowStatus(codeHost.UUID, &fleet.MDMDeliveryVerifying)
}

// TestHostNameTemplateIDPGroupChange covers the SCIM group-change re-enqueue path:
// renaming a group re-queues hosts whose template uses the groups variable, but not
// a host (mapped to the same user) whose template uses only the username variable.
func (s *integrationMDMTestSuite) TestHostNameTemplateIDPGroupChange() {
	t := s.T()
	ctx := t.Context()

	setFleetMDMData := func(hostID uint) {
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}
	requireRowStatus := func(hostUUID string, want *fleet.MDMDeliveryStatus) *fleet.HostDeviceNameEnforcement {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, hostUUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
		return row
	}
	deviceNameFromCmd := func(raw []byte) string {
		var sc struct {
			Command struct {
				Settings []struct {
					Item       string
					DeviceName string
				}
			}
		}
		require.NoError(t, plist.Unmarshal(raw, &sc))
		require.Len(t, sc.Command.Settings, 1)
		return sc.Command.Settings[0].DeviceName
	}

	// A user who belongs to a SCIM group. (Usernames/group names are unique per
	// test — the integration suite shares one DB and scim_users.user_name is unique.)
	userID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "jdoe@group.example.com"})
	require.NoError(t, err)
	groupID, err := s.ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: "GCAdmins", ScimUsers: []uint{userID}})
	require.NoError(t, err)

	// A team whose template uses the groups variable.
	groupTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-groups"})
	require.NoError(t, err)
	groupHost, groupDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(groupHost.ID)
	_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, groupHost.ID, userID)
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&groupTeam.ID, []uint{groupHost.ID})))

	// A team whose template uses only the username variable, with a host mapped to
	// the *same* user.
	userTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-user"})
	require.NoError(t, err)
	userHost, userDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setFleetMDMData(userHost.ID)
	_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, userHost.ID, userID)
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&userTeam.ID, []uint{userHost.ID})))

	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &groupTeam.ID, HostNameTemplate: "G-$FLEET_VAR_HOST_END_USER_IDP_GROUPS"}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &userTeam.ID, HostNameTemplate: "U-$FLEET_VAR_HOST_END_USER_IDP_USERNAME"}, http.StatusNoContent)

	// Drive both to a settled (verifying) state.
	require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	groupRow := requireRowStatus(groupHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, groupRow.ExpectedDeviceName)
	require.Equal(t, "G-GCAdmins", *groupRow.ExpectedDeviceName)
	for _, d := range []*mdmtest.TestAppleMDMClient{groupDevice, userDevice} {
		cmd, err := d.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		_, err = d.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	requireRowStatus(groupHost.UUID, &fleet.MDMDeliveryVerifying)
	requireRowStatus(userHost.UUID, &fleet.MDMDeliveryVerifying)

	// Renaming the group re-queues the groups-template host (its resolved name
	// changes) but leaves the username-template host — mapped to the same user —
	// untouched.
	require.NoError(t, s.ds.ReplaceScimGroup(ctx, &fleet.ScimGroup{ID: groupID, DisplayName: "GCSupers", ScimUsers: []uint{userID}}))
	requireRowStatus(groupHost.UUID, nil)
	requireRowStatus(userHost.UUID, &fleet.MDMDeliveryVerifying)

	require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	newRow := requireRowStatus(groupHost.UUID, &fleet.MDMDeliveryPending)
	require.NotNil(t, newRow.ExpectedDeviceName)
	require.Equal(t, "G-GCSupers", *newRow.ExpectedDeviceName)
	cmd, err := groupDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "G-GCSupers", deviceNameFromCmd(cmd.Raw))
}

// TestHostNameTemplateTeamSpecSecret covers the team-spec (GitOps) apply path with a
// template referencing a custom (secret) variable: it validates on create and edit,
// is idempotent on re-apply, and rejects an undefined secret.
func (s *integrationMDMTestSuite) TestHostNameTemplateTeamSpecSecret() {
	t := s.T()
	ctx := t.Context()

	_, err := s.ds.CreateSecretVariable(ctx, "SPECSITE", "HQ")
	require.NoError(t, err)

	teamName := t.Name()
	applySpec := func(tmpl string, wantStatus int) *http.Response {
		return s.Do("POST", "/api/latest/fleet/spec/teams", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
			Name: teamName,
			MDM:  fleet.TeamSpecMDM{HostNameTemplate: optjson.SetString(tmpl)},
		}}}, wantStatus)
	}

	// Create the team via spec with a valid secret template — the team-spec path
	// (createTeamFromSpec) validates the referenced secret and stores the template.
	applySpec("iPad ${FLEET_SECRET_SPECSITE}", http.StatusOK) //nolint:gosec // G101: template
	team, err := s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.Equal(t, "iPad ${FLEET_SECRET_SPECSITE}", team.Config.MDM.HostNameTemplate)

	// Re-applying the identical spec succeeds and is idempotent — the change-gate
	// skips re-validation and leaves the stored template unchanged.
	applySpec("iPad ${FLEET_SECRET_SPECSITE}", http.StatusOK) //nolint:gosec // G101: template
	teamAfter, err := s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.Equal(t, "iPad ${FLEET_SECRET_SPECSITE}", teamAfter.Config.MDM.HostNameTemplate)

	// A spec referencing an undefined secret is rejected.
	res := applySpec("iPad ${FLEET_SECRET_NOPE}", http.StatusUnprocessableEntity) //nolint:gosec // G101: template
	require.Contains(t, extractServerErrorText(res.Body), "missing from database")
}

func (s *integrationMDMTestSuite) TestHostNameTemplateTransferTeamToNoTeam() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	macHost, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, macHost.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))

	requireRowStatus := func(want *fleet.MDMDeliveryStatus) *fleet.HostDeviceNameEnforcement {
		row, err := s.ds.GetHostDeviceNameEnforcement(ctx, macHost.UUID)
		require.NoError(t, err)
		if want == nil {
			require.Nil(t, row.Status)
		} else {
			require.NotNil(t, row.Status)
			require.Equal(t, *want, *row.Status)
		}
		return row
	}
	runDeviceNameCron := func() {
		require.NoError(t, ReconcileHostDeviceNames(ctx, s.ds, s.mdmCommander, s.logger))
	}
	// drainSettingsCommand acks the pending Settings command and returns the
	// DeviceName the server told the device to apply.
	drainSettingsCommand := func() string {
		cmd, err := macDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "Settings", cmd.Command.RequestType)
		var settingsCmd struct {
			Command struct {
				Settings []struct {
					Item       string
					DeviceName string
				}
			}
		}
		require.NoError(t, plist.Unmarshal(cmd.Raw, &settingsCmd))
		require.Len(t, settingsCmd.Command.Settings, 1)
		_, err = macDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		return settingsCmd.Command.Settings[0].DeviceName
	}

	// --- both scopes carry a (distinct) template ---
	const teamTmpl = "TEAM-$FLEET_VAR_HOST_HARDWARE_SERIAL"
	const noTeamTmpl = "NOTEAM-$FLEET_VAR_HOST_HARDWARE_SERIAL"
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{FleetID: &team.ID, HostNameTemplate: teamTmpl}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: noTeamTmpl}, http.StatusNoContent)

	// --- host on the fleet: its team template governs ---
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{macHost.ID}}, http.StatusOK)
	requireRowStatus(nil) // queued by the transfer reconcile (team has a template)

	runDeviceNameCron()
	requireRowStatus(&fleet.MDMDeliveryPending)
	require.Equal(t, "TEAM-"+macHost.HardwareSerial, drainSettingsCommand())
	requireRowStatus(&fleet.MDMDeliveryVerifying)
	renamed, err := s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Equal(t, "TEAM-"+macHost.HardwareSerial, renamed.ComputerName)

	// --- transfer into "No team": the row is re-queued (No team has a template) ---
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{macHost.ID}}, http.StatusOK)
	movedHost, err := s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Nil(t, movedHost.TeamID, "host is now in No team")
	requireRowStatus(nil) // re-queued, not deleted (was: always deleted pre-#10)

	// --- cron now resolves the host name from the global No-team template ---
	runDeviceNameCron()
	requireRowStatus(&fleet.MDMDeliveryPending)
	require.Equal(t, "NOTEAM-"+macHost.HardwareSerial, drainSettingsCommand(),
		"after transfer to No team the cron must resolve the No-team template")
	requireRowStatus(&fleet.MDMDeliveryVerifying)
	renamed, err = s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Equal(t, "NOTEAM-"+macHost.HardwareSerial, renamed.ComputerName)

	requireNoRow := func() {
		_, err := s.ds.GetHostDeviceNameEnforcement(ctx, macHost.UUID)
		require.True(t, fleet.IsNotFound(err))
	}

	// --- reverse direction: No team -> fleet re-queues under the team template ---
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{macHost.ID}}, http.StatusOK)
	requireRowStatus(nil) // re-queued on the way back to the fleet
	runDeviceNameCron()
	require.Equal(t, "TEAM-"+macHost.HardwareSerial, drainSettingsCommand(),
		"back on the fleet the cron must resolve the team template again")
	requireRowStatus(&fleet.MDMDeliveryVerifying)

	// --- transfer to a template-less fleet deletes the row (stops enforcement) ---
	emptyTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-empty"})
	require.NoError(t, err)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &emptyTeam.ID, HostIDs: []uint{macHost.ID}}, http.StatusOK)
	requireNoRow()
	// no command is enqueued and the host keeps its last name
	runDeviceNameCron()
	cmd, err := macDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)
	kept, err := s.ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	require.Equal(t, "TEAM-"+macHost.HardwareSerial, kept.ComputerName, "transfer must not rename")

	// --- transfer into No team while No team has no template: not enforced ---
	s.Do("POST", "/api/latest/fleet/host_name_template",
		updateHostNameTemplateRequest{HostNameTemplate: ""}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{macHost.ID}}, http.StatusOK)
	requireNoRow()
	runDeviceNameCron()
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)
}

func manualProfileIf(personal bool) fleet.PersonalEnrollmentType {
	if personal {
		return fleet.PersonalEnrollmentTypeManualProfile
	}
	return fleet.PersonalEnrollmentTypeNone
}
