package mysql

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsSCEPStatusProbeCommands(t *testing.T) {
	installCmdUUID := uuid.NewString()
	profileUUID := "w" + uuid.NewString()

	probe := windowsSCEPStatusProbeCommand(installCmdUUID, profileUUID, false, 1)
	node := "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/" + profileUUID
	assert.Equal(t, node+"/Status", probe.TargetLocURI)
	assert.Contains(t, string(probe.RawCommand), "<CmdID>"+probe.CommandUUID+"</CmdID>")
	assert.Contains(t, string(probe.RawCommand), node+"/ErrorCode")
	assert.True(t, fleet.IsWindowsSCEPStatusProbeCmdUUID(probe.CommandUUID))

	gotInstall, gotAttempt, ok := parseWindowsSCEPStatusProbeCmdUUID(probe.CommandUUID)
	require.True(t, ok)
	assert.Equal(t, installCmdUUID, gotInstall)
	assert.Equal(t, 1, gotAttempt)
	_, _, ok = parseWindowsSCEPStatusProbeCmdUUID(installCmdUUID)
	assert.False(t, ok)

	cases := []struct {
		name        string
		userScoped  bool
		attempt     int
		wantAttempt int // 0 when no further probe is sent
	}{
		{name: "device scope", attempt: 1, wantAttempt: 2},
		{name: "user scope", userScoped: true, attempt: 3, wantAttempt: 4},
		{name: "last attempt", attempt: windowsSCEPStatusProbeMaxAttempts},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next := nextWindowsSCEPStatusProbeCommand(*windowsSCEPStatusProbeCommand(installCmdUUID, profileUUID, c.userScoped, c.attempt))
			if c.wantAttempt == 0 {
				assert.Nil(t, next)
				return
			}
			require.NotNil(t, next)
			assert.Equal(t, *windowsSCEPStatusProbeCommand(installCmdUUID, profileUUID, c.userScoped, c.wantAttempt), *next)
		})
	}
}

func TestWindowsSCEPStatusProbes(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Probe queued when a SCEP install is ACKed", testWindowsSCEPStatusProbeQueuedOnAck},
		{"Apply probe results", testApplyWindowsSCEPStatusProbeResults},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

type windowsSCEPInstall struct {
	device         *fleet.MDMWindowsEnrolledDevice
	profileUUID    string
	installCmdUUID string
}

// setUpWindowsSCEPInstall queues a SCEP profile install for a new enrolled host. An install proxied through a CA
// (caType set) has the managed certificate row Fleet's SCEP proxy creates.
func setUpWindowsSCEPInstall(t *testing.T, ds *Datastore, userScoped bool, caType fleet.CAConfigAssetType) windowsSCEPInstall {
	t.Helper()
	ctx := t.Context()
	in := windowsSCEPInstall{
		device:         createEnrolledDevice(t, ds),
		profileUUID:    "w" + uuid.NewString(),
		installCmdUUID: uuid.NewString(),
	}

	scope := "./Device"
	if userScoped {
		scope = "./User"
	}
	install := &fleet.MDMWindowsCommand{
		CommandUUID: in.installCmdUUID,
		RawCommand: fmt.Appendf(nil, `<Atomic><CmdID>%s</CmdID><Exec><CmdID>%s</CmdID><Item><Target>`+
			`<LocURI>%s/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Enroll</LocURI></Target></Item></Exec></Atomic>`,
			in.installCmdUUID, uuid.NewString(), scope, in.profileUUID),
	}
	require.NoError(t, ds.mdmWindowsInsertCommandForEnrollmentIDsDB(t.Context(), ds.primary, []uint{in.device.ID}, install))
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, profile_name, status, operation_type, command_uuid)
			VALUES (?, ?, ?, ?, ?, ?)`,
			in.device.HostUUID, in.profileUUID, "scep-"+in.profileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall, in.installCmdUUID)
		return err
	})
	if caType != "" {
		require.NoError(t, ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{{
			HostUUID: in.device.HostUUID, ProfileUUID: in.profileUUID, Type: caType, CAName: "CA-" + string(caType),
		}}))
	}
	return in
}

// queuedSCEPStatusProbes returns the probe commands queued for the enrollment.
func queuedSCEPStatusProbes(t *testing.T, ds *Datastore, enrollmentID uint) []fleet.MDMWindowsCommand {
	t.Helper()
	var probes []fleet.MDMWindowsCommand
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(t.Context(), q, &probes, `
			SELECT c.command_uuid, c.raw_command, c.target_loc_uri
			FROM windows_mdm_command_queue q JOIN windows_mdm_commands c ON c.command_uuid = q.command_uuid
			WHERE q.enrollment_id = ? AND c.command_uuid LIKE 'scep-status-%'
			ORDER BY c.command_uuid`, enrollmentID)
	})
	return probes
}

// windowsMDMResponse wraps SyncML body commands in a device message, the way a Windows 11 host sends them.
func windowsMDMResponse(t *testing.T, device *fleet.MDMWindowsEnrolledDevice, body string) fleet.EnrichedSyncML {
	t.Helper()
	raw := fmt.Sprintf(`<SyncML xmlns="SYNCML:SYNCML1.2"><SyncHdr><VerDTD>1.2</VerDTD><VerProto>DM/1.2</VerProto><SessionID>1</SessionID>`+
		`<MsgID>3</MsgID><Target><LocURI>https://example.com/api/mdm/microsoft/management</LocURI></Target><Source><LocURI>%s</LocURI></Source>`+
		`</SyncHdr><SyncBody>`+
		`<Status><CmdID>1</CmdID><MsgRef>2</MsgRef><CmdRef>0</CmdRef><Cmd>SyncHdr</Cmd><Data>200</Data></Status>`+
		`%s<Final/></SyncBody></SyncML>`, device.MDMDeviceID, body)
	syncML := &fleet.SyncML{}
	require.NoError(t, xml.Unmarshal([]byte(raw), syncML))
	syncML.Raw = []byte(raw)
	return fleet.NewEnrichedSyncML(syncML)
}

func ackStatus(cmdUUID, cmd, code string) string {
	return fmt.Sprintf(`<Status><CmdID>%s</CmdID><MsgRef>2</MsgRef><CmdRef>%s</CmdRef><Cmd>%s</Cmd><Data>%s</Data></Status>`,
		uuid.NewString(), cmdUUID, cmd, code)
}

func probeResults(probe *fleet.MDMWindowsCommand, status, errorCode string) string {
	node := strings.TrimSuffix(probe.TargetLocURI, "/Status")
	return fmt.Sprintf(`<Results><CmdID>%s</CmdID><MsgRef>2</MsgRef><CmdRef>%s</CmdRef>`+
		`<Item><Source><LocURI>%s/Status</LocURI></Source><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>%s</Data></Item>`+
		`<Item><Source><LocURI>%s/ErrorCode</LocURI></Source><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>%s</Data></Item>`+
		`</Results>`, uuid.NewString(), probe.CommandUUID, node, status, node, errorCode)
}

func testWindowsSCEPStatusProbeQueuedOnAck(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	cases := []struct {
		name       string
		userScoped bool
		caType     fleet.CAConfigAssetType
		wantStatus fleet.MDMDeliveryStatus
		wantProbe  bool
	}{
		{name: "device-scoped custom SCEP", caType: fleet.CAConfigCustomSCEPProxy, wantStatus: fleet.MDMDeliveryVerifying, wantProbe: true},
		{name: "user-scoped custom SCEP", userScoped: true, caType: fleet.CAConfigCustomSCEPProxy, wantStatus: fleet.MDMDeliveryVerifying, wantProbe: true},
		{name: "device-scoped NDES", caType: fleet.CAConfigNDES, wantStatus: fleet.MDMDeliveryVerifying, wantProbe: true},
		{name: "user-scoped NDES", userScoped: true, caType: fleet.CAConfigNDES, wantStatus: fleet.MDMDeliveryVerifying, wantProbe: true},
		{name: "not proxied by Fleet", wantStatus: fleet.MDMDeliveryVerified},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			in := setUpWindowsSCEPInstall(t, ds, c.userScoped, c.caType)
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `UPDATE mdm_windows_enrollments SET has_pending_commands = 0 WHERE id = ?`, in.device.ID)
				return err
			})

			ack := windowsMDMResponse(t, in.device, ackStatus(in.installCmdUUID, "Atomic", "200"))
			_, err := ds.MDMWindowsSaveResponse(ctx, in.device, ack, nil)
			require.NoError(t, err)

			status, _, _ := readWindowsHostProfile(t, ds, in.device.HostUUID, in.profileUUID)
			assert.Equal(t, c.wantStatus, status)
			probes := queuedSCEPStatusProbes(t, ds, in.device.ID)
			if !c.wantProbe {
				assert.Empty(t, probes)
				return
			}
			require.Len(t, probes, 1)
			assert.Equal(t, *windowsSCEPStatusProbeCommand(in.installCmdUUID, in.profileUUID, c.userScoped, 1), probes[0])

			// The queued probe is what has fleetd wake the device for the next session.
			var hasPending bool
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &hasPending, `SELECT has_pending_commands FROM mdm_windows_enrollments WHERE id = ?`, in.device.ID)
			})
			assert.True(t, hasPending)

			// A repeated ACK doesn't queue a second probe.
			_, err = ds.MDMWindowsSaveResponse(ctx, in.device, windowsMDMResponse(t, in.device, ackStatus(in.installCmdUUID, "Atomic", "200")), nil)
			require.NoError(t, err)
			assert.Len(t, queuedSCEPStatusProbes(t, ds, in.device.ID), 1)
		})
	}
}

func testApplyWindowsSCEPStatusProbeResults(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	cases := []struct {
		name string
		// status and errorCode are what the device reports, as observed on a Windows 11 host. An empty status means the
		// device answered the Get with an error and no values.
		status, errorCode string
		attempt           int
		retries           int
		// installStatus and newerDelivery describe the profile row when the answer arrives.
		installStatus fleet.MDMDeliveryStatus
		newerDelivery bool
		wantStatus    fleet.MDMDeliveryStatus
		wantRetries   int
		wantDetail    string
		wantNextProbe bool
	}{
		{
			name: "installed", status: "1", errorCode: "0", attempt: 1,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: fleet.MDMDeliveryVerified,
		},
		{
			name: "still enrolling", status: "32", errorCode: "0", attempt: 1,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: fleet.MDMDeliveryVerifying, wantNextProbe: true,
		},
		{
			name: "still enrolling on the last attempt", status: "32", errorCode: "0", attempt: windowsSCEPStatusProbeMaxAttempts,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: fleet.MDMDeliveryVerifying,
		},
		{
			name: "node not found", attempt: 2,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: fleet.MDMDeliveryVerifying, wantNextProbe: true,
		},
		{
			name: "failed with retries left", status: "16", errorCode: "-2145844748", attempt: 1,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: "", wantRetries: 1,
		},
		{
			name: "failed with no retries left", status: "16", errorCode: "-2145844748", attempt: 1, retries: mdm.MaxWindowsProfileRetries,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: fleet.MDMDeliveryFailed, wantRetries: mdm.MaxWindowsProfileRetries,
			wantDetail: "The host reported that the SCEP certificate isn't installed (error code 0x801901F4).",
		},
		{
			name: "certificate removed after install", status: "16", errorCode: "-2146885628", attempt: 1, retries: mdm.MaxWindowsProfileRetries,
			installStatus: fleet.MDMDeliveryVerifying, wantStatus: fleet.MDMDeliveryFailed, wantRetries: mdm.MaxWindowsProfileRetries,
			wantDetail: "The host reported that the SCEP certificate isn't installed (error code 0x80092004).",
		},
		{
			name: "answer for an earlier delivery", status: "32", errorCode: "0", attempt: 1,
			installStatus: fleet.MDMDeliveryVerifying, newerDelivery: true, wantStatus: fleet.MDMDeliveryVerifying,
		},
		{
			name: "proxy already failed the delivery", status: "1", errorCode: "0", attempt: 1, retries: mdm.MaxWindowsProfileRetries,
			installStatus: fleet.MDMDeliveryFailed, wantStatus: fleet.MDMDeliveryFailed, wantRetries: mdm.MaxWindowsProfileRetries,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			in := setUpWindowsSCEPInstall(t, ds, false, fleet.CAConfigNDES)
			probe := windowsSCEPStatusProbeCommand(in.installCmdUUID, in.profileUUID, false, c.attempt)
			require.NoError(t, ds.mdmWindowsInsertCommandForEnrollmentIDsDB(ctx, ds.primary, []uint{in.device.ID}, probe))

			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				cmdUUID := in.installCmdUUID
				if c.newerDelivery {
					cmdUUID = uuid.NewString()
				}
				_, err := q.ExecContext(ctx, `UPDATE host_mdm_windows_profiles SET status = ?, retries = ?, command_uuid = ? WHERE host_uuid = ?`,
					c.installStatus, c.retries, cmdUUID, in.device.HostUUID)
				return err
			})
			require.NoError(t, ds.ReconcileWindowsProfilesStatus(ctx))

			body := ackStatus(probe.CommandUUID, "Get", "404")
			if c.status != "" {
				body = ackStatus(probe.CommandUUID, "Get", "200") + probeResults(probe, c.status, c.errorCode)
			}
			_, err := ds.MDMWindowsSaveResponse(ctx, in.device, windowsMDMResponse(t, in.device, body), nil)
			require.NoError(t, err)

			status, detail, retries := readWindowsHostProfile(t, ds, in.device.HostUUID, in.profileUUID)
			assert.Equal(t, c.wantStatus, status)
			assert.Equal(t, c.wantRetries, retries)
			assert.Equal(t, c.wantDetail, detail)

			probes := queuedSCEPStatusProbes(t, ds, in.device.ID)
			if c.wantNextProbe {
				require.Len(t, probes, 2)
				assert.Contains(t, probes, *nextWindowsSCEPStatusProbeCommand(*probe))
			} else {
				assert.Len(t, probes, 1)
			}

			// The rollup must follow without waiting for the reconcile cron.
			rollup := readWindowsProfilesStatusRollup(t, ds)[in.device.HostUUID]
			require.NoError(t, ds.ReconcileWindowsProfilesStatus(ctx))
			assert.Equal(t, readWindowsProfilesStatusRollup(t, ds)[in.device.HostUUID], rollup)

			if c.status != "" {
				// The answer is kept with the command result.
				var rawResult string
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(ctx, q, &rawResult, `SELECT raw_result FROM windows_mdm_command_results WHERE command_uuid = ?`,
						probe.CommandUUID)
				})
				assert.Contains(t, rawResult, "<Data>"+c.status+"</Data>")
			}
		})
	}
}
