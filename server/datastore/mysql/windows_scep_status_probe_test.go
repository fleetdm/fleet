package mysql

import (
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

func TestNextWindowsSCEPStatusProbeCommand(t *testing.T) {
	installCmdUUID := uuid.NewString()
	profileUUID := "w" + uuid.NewString()

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

	_, _, ok := parseWindowsSCEPStatusProbeCmdUUID(installCmdUUID)
	assert.False(t, ok, "an ordinary command UUID is not a probe")
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
	require.NoError(t, ds.MDMWindowsInsertCommandForHosts(ctx, []string{in.device.HostUUID}, &fleet.MDMWindowsCommand{
		CommandUUID: in.installCmdUUID,
		RawCommand: fmt.Appendf(nil, `<Atomic><CmdID>%s</CmdID><Exec><CmdID>%s</CmdID><Item><Target>`+
			`<LocURI>%s/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Enroll</LocURI></Target></Item></Exec></Atomic>`,
			in.installCmdUUID, uuid.NewString(), scope, in.profileUUID),
	}))
	setWindowsSCEPInstallRow(t, ds, in, in.installCmdUUID, fleet.MDMDeliveryPending, 0)
	if caType != "" {
		require.NoError(t, ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{{
			HostUUID: in.device.HostUUID, ProfileUUID: in.profileUUID, Type: caType, CAName: "CA-" + string(caType),
		}}))
	}
	return in
}

// setWindowsSCEPInstallRow sets the host profile row for the install's profile.
func setWindowsSCEPInstallRow(t *testing.T, ds *Datastore, in windowsSCEPInstall, cmdUUID string, status fleet.MDMDeliveryStatus,
	retries int,
) {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{{
		ProfileUUID:   in.profileUUID,
		ProfileName:   "scep-" + in.profileUUID,
		HostUUID:      in.device.HostUUID,
		CommandUUID:   cmdUUID,
		OperationType: fleet.MDMOperationTypeInstall,
		Status:        &status,
		Checksum:      []byte{1},
	}}))
	if retries == 0 {
		return
	}
	// No datastore method sets the retry count directly.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_windows_profiles SET retries = ? WHERE host_uuid = ? AND profile_uuid = ?`,
			retries, in.device.HostUUID, in.profileUUID)
		return err
	})
}

// pendingSCEPStatusProbes returns the probes queued for the enrollment that the device hasn't answered yet.
func pendingSCEPStatusProbes(t *testing.T, ds *Datastore, enrollmentID uint) []*fleet.MDMWindowsCommand {
	t.Helper()
	cmds, err := ds.MDMWindowsGetPendingCommands(t.Context(), enrollmentID)
	require.NoError(t, err)
	var probes []*fleet.MDMWindowsCommand
	for _, cmd := range cmds {
		if fleet.IsWindowsSCEPStatusProbeCmdUUID(cmd.CommandUUID) {
			probes = append(probes, cmd)
		}
	}
	return probes
}

// probeResults is the <Results> a Windows 11 host sends for a probe.
func probeResults(probe *fleet.MDMWindowsCommand, status, errorCode string) string {
	node := strings.TrimSuffix(probe.TargetLocURI, "/Status")
	return fmt.Sprintf(`<Results><CmdID>%s</CmdID><MsgRef>1</MsgRef><CmdRef>%s</CmdRef>`+
		`<Item><Source><LocURI>%s/Status</LocURI></Source><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>%s</Data></Item>`+
		`<Item><Source><LocURI>%s/ErrorCode</LocURI></Source><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>%s</Data></Item>`+
		`</Results>`, uuid.NewString(), probe.CommandUUID, node, status, node, errorCode)
}

func ackInstall(t *testing.T, ds *Datastore, in windowsSCEPInstall) {
	t.Helper()
	_, err := ds.MDMWindowsSaveResponse(t.Context(), in.device, createResponseAsEnrichedSyncML(t, in.device, []enrichResponseEntry{
		{Type: "Atomic", StatusCode: 200, UUID: in.installCmdUUID},
	}), nil)
	require.NoError(t, err)
}

func answerProbe(t *testing.T, ds *Datastore, in windowsSCEPInstall, probe *fleet.MDMWindowsCommand, status, errorCode string) {
	t.Helper()
	entries := []enrichResponseEntry{{Type: "Get", StatusCode: 404, UUID: probe.CommandUUID}}
	var results []string
	if status != "" {
		entries[0].StatusCode = 200
		results = append(results, probeResults(probe, status, errorCode))
	}
	_, err := ds.MDMWindowsSaveResponse(t.Context(), in.device, createResponseAsEnrichedSyncML(t, in.device, entries, results...), nil)
	require.NoError(t, err)
}

func testWindowsSCEPStatusProbeQueuedOnAck(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	cases := []struct {
		name       string
		userScoped bool
		caType     fleet.CAConfigAssetType
		wantStatus fleet.MDMDeliveryStatus
		wantScope  string // empty when no probe should be queued
	}{
		{name: "device-scoped custom SCEP", caType: fleet.CAConfigCustomSCEPProxy, wantStatus: fleet.MDMDeliveryVerifying, wantScope: "./Device"},
		{name: "user-scoped custom SCEP", userScoped: true, caType: fleet.CAConfigCustomSCEPProxy, wantStatus: fleet.MDMDeliveryVerifying, wantScope: "./User"},
		{name: "device-scoped NDES", caType: fleet.CAConfigNDES, wantStatus: fleet.MDMDeliveryVerifying, wantScope: "./Device"},
		{name: "user-scoped NDES", userScoped: true, caType: fleet.CAConfigNDES, wantStatus: fleet.MDMDeliveryVerifying, wantScope: "./User"},
		{name: "not proxied by Fleet", wantStatus: fleet.MDMDeliveryVerified},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			in := setUpWindowsSCEPInstall(t, ds, c.userScoped, c.caType)

			ackInstall(t, ds, in)

			status, _, _ := readWindowsHostProfile(t, ds, in.device.HostUUID, in.profileUUID)
			assert.Equal(t, c.wantStatus, status)

			// The session refreshes has_pending_commands once nothing deliverable is left; a queued probe keeps it set, which
			// is what has fleetd wake the device for the probe.
			require.NoError(t, ds.MDMWindowsRefreshHasPendingCommands(ctx, in.device.ID))
			device, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, in.device.MDMDeviceID)
			require.NoError(t, err)

			probes := pendingSCEPStatusProbes(t, ds, in.device.ID)
			if c.wantScope == "" {
				assert.Empty(t, probes)
				assert.False(t, device.HasPendingCommands)
				return
			}
			require.Len(t, probes, 1)
			assert.Equal(t, fmt.Sprintf("%s/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Status", c.wantScope, in.profileUUID),
				probes[0].TargetLocURI)
			assert.True(t, device.HasPendingCommands)
		})
	}
}

func testWindowsSCEPStatusProbeRepeatedAck(t *testing.T, ds *Datastore) {
	in := setUpWindowsSCEPInstall(t, ds, false, fleet.CAConfigNDES)

	ackInstall(t, ds, in)
	ackInstall(t, ds, in)
	probes := pendingSCEPStatusProbes(t, ds, in.device.ID)
	require.Len(t, probes, 1, "a repeated ACK must not queue a second probe")

	answerProbe(t, ds, in, probes[0], "1", "0")
	status, _, _ := readWindowsHostProfile(t, ds, in.device.HostUUID, in.profileUUID)
	require.Equal(t, fleet.MDMDeliveryVerified, status)

	// A late repeat of the ACK must not send the delivery back to verifying.
	ackInstall(t, ds, in)
	status, _, _ = readWindowsHostProfile(t, ds, in.device.HostUUID, in.profileUUID)
	assert.Equal(t, fleet.MDMDeliveryVerified, status)
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
			// Back to NULL (pending) so the profile manager redelivers it.
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
			cmdUUID := in.installCmdUUID
			if c.newerDelivery {
				cmdUUID = uuid.NewString()
			}
			setWindowsSCEPInstallRow(t, ds, in, cmdUUID, c.installStatus, c.retries)
			require.NoError(t, ds.ReconcileWindowsProfilesStatus(ctx))
			probe := windowsSCEPStatusProbeCommand(in.installCmdUUID, in.profileUUID, false, c.attempt)
			require.NoError(t, ds.MDMWindowsInsertCommandForHosts(ctx, []string{in.device.HostUUID}, probe))

			answerProbe(t, ds, in, probe, c.status, c.errorCode)
			// A late repeat of the install ACK must leave the outcome as it is.
			ackInstall(t, ds, in)

			status, detail, retries := readWindowsHostProfile(t, ds, in.device.HostUUID, in.profileUUID)
			assert.Equal(t, c.wantStatus, status)
			assert.Equal(t, c.wantRetries, retries)
			assert.Equal(t, c.wantDetail, detail)

			probes := pendingSCEPStatusProbes(t, ds, in.device.ID)
			if c.wantNextProbe {
				require.Len(t, probes, 1)
				assert.Equal(t, nextWindowsSCEPStatusProbeCommand(*probe).CommandUUID, probes[0].CommandUUID)
			} else {
				assert.Empty(t, probes)
			}

			// The rollup must follow without waiting for the reconcile cron.
			rollup := readWindowsProfilesStatusRollup(t, ds)[in.device.HostUUID]
			require.NoError(t, ds.ReconcileWindowsProfilesStatus(ctx))
			assert.Equal(t, readWindowsProfilesStatusRollup(t, ds)[in.device.HostUUID], rollup)
		})
	}
}
