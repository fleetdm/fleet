package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockPusher struct{}

func (mockPusher) Push(ctx context.Context, ids []string) (map[string]*push.Response, error) {
	m := make(map[string]*push.Response, len(ids))
	for _, id := range ids {
		m[id] = &push.Response{Id: id}
	}
	return m, nil
}

func TestMDMRunCommand(t *testing.T) {
	// define some hosts to use in the tests
	macEnrolled := &fleet.Host{
		ID:       1,
		UUID:     "mac-enrolled",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolled := &fleet.Host{
		ID:       2,
		UUID:     "win-enrolled",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macUnenrolled := &fleet.Host{
		ID:       3,
		UUID:     "mac-unenrolled",
		Platform: "darwin",
	}
	winUnenrolled := &fleet.Host{
		ID:       4,
		UUID:     "win-unenrolled",
		Platform: "windows",
	}
	linuxUnenrolled := &fleet.Host{
		ID:       5,
		UUID:     "linux-unenrolled",
		Platform: "linux",
	}
	macEnrolled2 := &fleet.Host{
		ID:       6,
		UUID:     "mac-enrolled-2",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolled2 := &fleet.Host{
		ID:       7,
		UUID:     "win-enrolled-2",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macNonFleetEnrolled := &fleet.Host{
		ID:       8,
		UUID:     "mac-non-fleet-enrolled",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMJamf},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMJamf, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winNonFleetEnrolled := &fleet.Host{
		ID:       9,
		UUID:     "win-non-fleet-enrolled",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMIntune},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMIntune, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macPending := &fleet.Host{
		ID:       10,
		UUID:     "mac-pending",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winPending := &fleet.Host{
		ID:       11,
		UUID:     "win-pending",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	hostByUUID := make(map[string]*fleet.Host)
	hostByID := make(map[uint]*fleet.Host)
	for _, h := range []*fleet.Host{macEnrolled, winEnrolled, macUnenrolled, winUnenrolled, linuxUnenrolled, macEnrolled2, winEnrolled2, macNonFleetEnrolled, winNonFleetEnrolled, macPending, winPending} {
		hostByUUID[h.UUID] = h
		hostByID[h.ID] = h
	}

	// define some files to use in the tests
	yamlFilePath := writeTmpYml(t, `invalid`)
	mobileConfigFilePath := writeTmpMobileconfig(t, "Mobileconfig")
	appleCmdFilePath := writeTmpAppleMDMCmd(t, "FooBar")
	winCmdFilePath := writeTmpWindowsMDMCmd(t, "FooBar")
	applePremiumCmdFilePath := writeTmpAppleMDMCmd(t, "EraseDevice")
	winPremiumCmdFilePath := writeTmpWindowsMDMCmd(t, "./Device/Vendor/MSFT/RemoteWipe/doWipe")

	emptyAppleCmdFilePath, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = emptyAppleCmdFilePath.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
</plist>`))
	require.NoError(t, err)
	emptyAppleCmdFilePath.Close()

	emptyWinCmdFilePath, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = emptyWinCmdFilePath.WriteString(fmt.Sprintf(`<Exec>
</Exec>`))
	require.NoError(t, err)
	emptyWinCmdFilePath.Close()

	nonExecWinCmdFilePath, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = nonExecWinCmdFilePath.WriteString(fmt.Sprintf(`<Get>
	<CmdID>22</CmdID>
	<Item>
		<Target>
			<LocURI>FooBar</LocURI>
		</Target>
		<Meta>
			<Format xmlns="syncml:metinf">chr</Format>
			<Type xmlns="syncml:metinf">text/plain</Type>
		</Meta>
		<Data>NamedValuesList=MinPasswordLength,8;</Data>
	</Item>
</Get>`))
	require.NoError(t, err)
	nonExecWinCmdFilePath.Close()

	// define some app configs variations to use in the tests
	appCfgAllMDM := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true, WindowsEnabledAndConfigured: true}}
	appCfgWinMDM := &fleet.AppConfig{MDM: fleet.MDM{WindowsEnabledAndConfigured: true}}
	appCfgMacMDM := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
	appCfgNoMDM := &fleet.AppConfig{MDM: fleet.MDM{}}

	for _, lic := range []string{fleet.TierFree, fleet.TierPremium} {
		t.Run(lic, func(t *testing.T) {
			enqueuer := new(mock.MDMAppleStore)
			license := &fleet.LicenseInfo{Tier: lic, Expiration: time.Now().Add(24 * time.Hour)}

			_, ds := runServerWithMockedDS(t, &service.TestServerOpts{
				MDMStorage:       enqueuer,
				MDMPusher:        mockPusher{},
				License:          license,
				NoCacheDatastore: true,
			})

			ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
				h, ok := hostByUUID[identifier]
				if !ok {
					return nil, &notFoundError{}
				}
				return h, nil
			}
			ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
				return nil
			}
			ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
				return nil, nil
			}
			ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
				return nil, nil
			}
			ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*fleet.HostBattery, error) {
				return nil, nil
			}
			ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
				return nil, nil
			}
			ds.GetHostMDMAppleProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMAppleProfile, error) {
				return nil, nil
			}
			ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMWindowsProfile, error) {
				return nil, nil
			}
			ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDMMacOSSetup, error) {
				return nil, nil
			}
			ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
				return &fleet.HostLockWipeStatus{}, nil
			}
			ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
				if len(uuids) == 0 {
					return nil, nil
				}
				hosts := make([]*fleet.Host, 0, len(uuids))
				for _, uid := range uuids {
					if h := hostByUUID[uid]; h != nil {
						hosts = append(hosts, h)
					}
				}
				return hosts, nil
			}
			winCmds := map[string]struct{}{}
			ds.MDMWindowsInsertCommandForHostsFunc = func(ctx context.Context, deviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
				// every command uuid is different
				require.NotContains(t, winCmds, cmd.CommandUUID)
				winCmds[cmd.CommandUUID] = struct{}{}
				return nil
			}
			ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
				return &fleet.HostMDMDiskEncryption{}, nil
			}
			ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
				h, ok := hostByID[hostID]
				require.True(t, ok)
				if h.MDMInfo == nil {
					return nil, &notFoundError{}
				}
				return h.MDMInfo, nil
			}

			enqueuer.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
				return map[string]error{}, nil
			}

			cases := []struct {
				desc    string
				flags   []string
				appCfg  *fleet.AppConfig
				wantErr string
			}{
				{"no flags", nil, appCfgAllMDM, `Required flags "hosts, payload" not set`},
				{"no payload", []string{"--hosts", "abc"}, appCfgAllMDM, `Required flag "payload" not set`},
				{"no hosts", []string{"--payload", winCmdFilePath}, appCfgAllMDM, `Required flag "hosts" not set`},
				{"invalid payload", []string{"--hosts", "abc", "--payload", "no-such-file"}, appCfgAllMDM, `open no-such-file: no such file or directory`},
				{"macOS yaml payload", []string{"--hosts", "mac-enrolled", "--payload", yamlFilePath}, appCfgAllMDM, `The payload isn't valid XML`},
				{"win yaml payload", []string{"--hosts", "win-enrolled", "--payload", yamlFilePath}, appCfgAllMDM, `The payload isn't valid XML`},
				{"non-mdm-command plist payload", []string{"--hosts", "mac-enrolled", "--payload", mobileConfigFilePath}, appCfgAllMDM, `The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file:`},
				{"single host not found", []string{"--hosts", "no-such-host", "--payload", appleCmdFilePath}, appCfgAllMDM, `No hosts targeted.`},
				{"unenrolled macOS host", []string{"--hosts", "mac-unenrolled", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"unenrolled windows host", []string{"--hosts", "win-unenrolled", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"macOS non-fleet host", []string{"--hosts", "mac-non-fleet-enrolled", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"windows non-fleet host", []string{"--hosts", "win-non-fleet-enrolled", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"macOS pending host", []string{"--hosts", "mac-pending", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"windows pending host", []string{"--hosts", "win-pending", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"valid single mac", []string{"--hosts", "mac-enrolled", "--payload", appleCmdFilePath}, appCfgAllMDM, ""},
				{"valid single windows", []string{"--hosts", "win-enrolled", "--payload", winCmdFilePath}, appCfgAllMDM, ""},
				{"no mdm enabled", []string{"--hosts", "win-enrolled", "--payload", winCmdFilePath}, appCfgNoMDM, "MDM features aren't turned on in Fleet."},
				{"valid single mac only win mdm", []string{"--hosts", "mac-enrolled", "--payload", appleCmdFilePath}, appCfgWinMDM, "macOS MDM isn't turned on."},
				{"valid single win only mac mdm", []string{"--hosts", "win-enrolled", "--payload", winCmdFilePath}, appCfgMacMDM, "Windows MDM isn't turned on."},
				{"macOS premium cmd", []string{"--hosts", "mac-enrolled", "--payload", applePremiumCmdFilePath}, appCfgAllMDM, func() string {
					if lic == fleet.TierFree {
						return `missing or invalid license`
					}
					return ""
				}()},
				{"windows premium cmd", []string{"--hosts", "win-enrolled", "--payload", winPremiumCmdFilePath}, appCfgAllMDM, func() string {
					if lic == fleet.TierFree {
						return `Missing or invalid license. Wipe command is available in Fleet Premium only.`
					}
					return ""
				}()},
				{"empty plist file", []string{"--hosts", "mac-enrolled", "--payload", emptyAppleCmdFilePath.Name()}, appCfgAllMDM, `The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file.`},
				{"non-Exec win file", []string{"--hosts", "win-enrolled", "--payload", nonExecWinCmdFilePath.Name()}, appCfgAllMDM, `You can run only <Exec> command type.`},
				{"empty win file", []string{"--hosts", "win-enrolled", "--payload", emptyWinCmdFilePath.Name()}, appCfgAllMDM, `You can run only a single <Exec> command.`},
				{"hosts with different platforms", []string{"--hosts", "win-enrolled,mac-enrolled", "--payload", winCmdFilePath}, appCfgAllMDM, `Command can't run on hosts with different platforms.`},
				{"all hosts not found", []string{"--hosts", "no-such-1,no-such-2,no-such-3", "--payload", winCmdFilePath}, appCfgAllMDM, `No hosts targeted.`},
				{"one host not found", []string{"--hosts", "win-enrolled,no-such-2,win-enrolled-2", "--payload", winCmdFilePath}, appCfgAllMDM, `One or more targeted hosts don't exist.`},
				{"one windows host not enrolled", []string{"--hosts", "win-enrolled,win-unenrolled,win-enrolled-2", "--payload", winCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"one macOS host not enrolled", []string{"--hosts", "mac-enrolled,mac-unenrolled,mac-enrolled-2", "--payload", appleCmdFilePath}, appCfgAllMDM, `Can't run the MDM command because one or more hosts have MDM turned off.`},
				{"valid multiple mac", []string{"--hosts", "mac-enrolled,mac-enrolled-2", "--payload", appleCmdFilePath}, appCfgAllMDM, ""},
				{"valid multiple windows", []string{"--hosts", "win-enrolled,win-enrolled-2", "--payload", winCmdFilePath}, appCfgAllMDM, ""},
				{"valid multiple mac mac-enabled only", []string{"--hosts", "mac-enrolled,mac-enrolled-2", "--payload", appleCmdFilePath}, appCfgMacMDM, ""},
				{"valid multiple windows win-enabled only", []string{"--hosts", "win-enrolled,win-enrolled-2", "--payload", winCmdFilePath}, appCfgWinMDM, ""},
			}
			for _, c := range cases {
				t.Run(c.desc, func(t *testing.T) {
					ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
						return c.appCfg, nil
					}

					buf, err := runAppNoChecks(append([]string{"mdm", "run-command"}, c.flags...))
					if c.wantErr != "" {
						require.Error(t, err)
						require.ErrorContains(t, err, c.wantErr)
					} else {
						require.NoError(t, err)
						require.Contains(t, buf.String(), `Hosts will run the command the next time they check into Fleet.`)
						require.Contains(t, buf.String(), `fleetctl get mdm-command-results --id=`)
					}
				})
			}
		})
	}
}

func TestMDMLockCommand(t *testing.T) {
	macEnrolled := &fleet.Host{
		ID:       1,
		UUID:     "mac-enrolled",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolled := &fleet.Host{
		ID:       2,
		UUID:     "win-enrolled",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}

	linuxEnrolled := &fleet.Host{
		ID:       3,
		UUID:     "linux-enrolled",
		Platform: "linux",
	}
	winNotEnrolled := &fleet.Host{
		ID:       4,
		UUID:     "win-not-enrolled",
		Platform: "windows",
	}
	macNotEnrolled := &fleet.Host{
		ID:       5,
		UUID:     "mac-not-enrolled",
		Platform: "darwin",
	}
	macPending := &fleet.Host{
		ID:       6,
		UUID:     "mac-pending",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winPending := &fleet.Host{
		ID:       7,
		UUID:     "win-pending",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winEnrolledUP := &fleet.Host{
		ID:       8,
		UUID:     "win-enrolled-up",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledUP := &fleet.Host{
		ID:       9,
		UUID:     "mac-enrolled-up",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}

	winEnrolledLP := &fleet.Host{
		ID:       10,
		UUID:     "win-enrolled-lp",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledLP := &fleet.Host{
		ID:       11,
		UUID:     "mac-enrolled-lp",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledWP := &fleet.Host{
		ID:       12,
		UUID:     "win-enrolled-wp",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledWP := &fleet.Host{
		ID:       13,
		UUID:     "mac-enrolled-wp",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}

	hostByUUID := make(map[string]*fleet.Host)
	hostsByID := make(map[uint]*fleet.Host)
	for _, h := range []*fleet.Host{
		winEnrolled,
		macEnrolled,
		linuxEnrolled,
		macNotEnrolled,
		winNotEnrolled,
		macPending,
		winPending,
		winEnrolledUP,
		macEnrolledUP,
		winEnrolledLP,
		macEnrolledLP,
		winEnrolledWP,
		macEnrolledWP,
	} {
		hostByUUID[h.UUID] = h
		hostsByID[h.ID] = h
	}

	unlockPending := map[uint]*fleet.Host{
		winEnrolledUP.ID: winEnrolledUP,
		macEnrolledUP.ID: macEnrolledUP,
	}

	lockPending := map[uint]*fleet.Host{
		winEnrolledLP.ID: winEnrolledLP,
		macEnrolledLP.ID: macEnrolledLP,
	}

	wipePending := map[uint]*fleet.Host{
		winEnrolledWP.ID: winEnrolledWP,
		macEnrolledWP.ID: macEnrolledWP,
	}

	ds := setupTestServer(t)
	setupDSMocks(ds, hostByUUID, hostsByID)

	// custom ds mocks for these tests
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
		fleetPlatform := host.FleetPlatform()

		var status fleet.HostLockWipeStatus
		status.HostFleetPlatform = fleetPlatform

		if _, ok := unlockPending[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.UnlockPIN = "1234"
				status.UnlockRequestedAt = time.Now()
				return &status, nil
			}

			status.UnlockScript = &fleet.HostScriptResult{}
		}

		if _, ok := lockPending[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.LockMDMCommand = &fleet.MDMCommand{}
				return &status, nil
			}

			status.LockScript = &fleet.HostScriptResult{}
		}

		if _, ok := wipePending[host.ID]; ok {
			if fleetPlatform == "linux" {
				status.WipeScript = &fleet.HostScriptResult{ExitCode: nil}
				return &status, nil
			}

			status.WipeMDMCommand = &fleet.MDMCommand{}
			status.WipeMDMCommandResult = nil
			return &status, nil
		}

		return &status, nil
	}
	ds.LockHostViaScriptFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload, platform string) error {
		return nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok || h.MDMInfo == nil {
			return nil, &notFoundError{}
		}

		return h.MDMInfo, nil
	}

	ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*fleet.HostOrbitInfo, error) {
		hostIDMod := hostID % 3
		switch hostIDMod {
		case 0:
			return nil, &notFoundError{}
		case 1:
			return &fleet.HostOrbitInfo{}, nil
		case 2:
			return &fleet.HostOrbitInfo{ScriptsEnabled: ptr.Bool(true)}, nil
		default:
			t.Errorf("unexpected hostIDMod %v", hostIDMod)
			return nil, nil
		}
	}

	appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM := setupAppConigs()

	successfulOutput := func(ident string) string {
		return fmt.Sprintf(`
The host will lock when it comes online.

Copy and run this command to see lock status:

fleetctl get host %s

When you're ready to unlock the host, copy and run this command:

fleetctl mdm unlock --host=%s

`, ident, ident)
	}

	cases := []struct {
		appCfg  *fleet.AppConfig
		desc    string
		flags   []string
		wantErr string
	}{
		{appCfgAllMDM, "no flags", nil, `Required flag "host" not set`},
		{appCfgAllMDM, "host flag empty", []string{"--host", ""}, `No host targeted. Please provide --host.`},
		{appCfgAllMDM, "lock non-existent host", []string{"--host", "notfound"}, `The host doesn't exist. Please provide a valid host identifier.`},
		{appCfgMacMDM, "valid windows but only macos mdm", []string{"--host", winEnrolled.UUID}, `Windows MDM isn't turned on.`},
		{appCfgWinMDM, "valid macos but only windows mdm", []string{"--host", macEnrolled.UUID}, `macOS MDM isn't turned on.`},
		{appCfgAllMDM, "valid windows", []string{"--host", winEnrolled.UUID}, ""},
		{appCfgAllMDM, "valid macos", []string{"--host", macEnrolled.UUID}, ""},
		{appCfgNoMDM, "valid linux", []string{"--host", linuxEnrolled.UUID}, ""},
		{appCfgNoMDM, "valid windows but no mdm", []string{"--host", winEnrolled.UUID}, `Windows MDM isn't turned on.`},
		{appCfgNoMDM, "valid macos but no mdm", []string{"--host", macEnrolled.UUID}, `macOS MDM isn't turned on.`},
		{appCfgMacMDM, "valid macos but not enrolled", []string{"--host", macNotEnrolled.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but not enrolled", []string{"--host", winNotEnrolled.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but pending ", []string{"--host", winPending.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgMacMDM, "valid macos but pending", []string{"--host", macPending.UUID}, `Can't lock the host because it doesn't have MDM turned on.`},
		{appCfgAllMDM, "valid windows but pending unlock", []string{"--host", winEnrolledUP.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid macos but pending unlock", []string{"--host", macEnrolledUP.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid windows but pending lock", []string{"--host", winEnrolledLP.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid macos but pending lock", []string{"--host", macEnrolledLP.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid windows but pending wipe", []string{"--host", winEnrolledWP.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid macos but pending wipe", []string{"--host", macEnrolledWP.UUID}, "Host has pending wipe request."},
	}

	runTestCases(t, ds, "lock", successfulOutput, cases)
}

func TestMDMUnlockCommand(t *testing.T) {
	macEnrolled := &fleet.Host{
		ID:       1,
		UUID:     "mac-enrolled",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolled := &fleet.Host{
		ID:       2,
		UUID:     "win-enrolled",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	linuxEnrolled := &fleet.Host{
		ID:       3,
		UUID:     "linux-enrolled",
		Platform: "linux",
	}
	winNotEnrolled := &fleet.Host{
		ID:       4,
		UUID:     "win-not-enrolled",
		Platform: "windows",
	}
	macNotEnrolled := &fleet.Host{
		ID:       5,
		UUID:     "mac-not-enrolled",
		Platform: "darwin",
	}
	macPending := &fleet.Host{
		ID:       6,
		UUID:     "mac-pending",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winPending := &fleet.Host{
		ID:       7,
		UUID:     "win-pending",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winEnrolledUP := &fleet.Host{
		ID:       8,
		UUID:     "win-enrolled-up",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledUP := &fleet.Host{
		ID:       9,
		UUID:     "mac-enrolled-up",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledLP := &fleet.Host{
		ID:       10,
		UUID:     "win-enrolled-lp",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledLP := &fleet.Host{
		ID:       11,
		UUID:     "mac-enrolled-lp",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledWP := &fleet.Host{
		ID:       12,
		UUID:     "win-enrolled-wp",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledWP := &fleet.Host{
		ID:       13,
		UUID:     "mac-enrolled-wp",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}

	hostByUUID := make(map[string]*fleet.Host)
	hostsByID := make(map[uint]*fleet.Host)
	for _, h := range []*fleet.Host{
		winEnrolled,
		macEnrolled,
		linuxEnrolled,
		macNotEnrolled,
		winNotEnrolled,
		macPending,
		winPending,
		winEnrolledUP,
		macEnrolledUP,
		winEnrolledLP,
		macEnrolledLP,
		winEnrolledWP,
		macEnrolledWP,
	} {
		hostByUUID[h.UUID] = h
		hostsByID[h.ID] = h
	}

	locked := map[uint]*fleet.Host{
		winEnrolled.ID: winEnrolled,
		macEnrolled.ID: macEnrolled, linuxEnrolled.ID: linuxEnrolled,
	}

	unlockPending := map[uint]*fleet.Host{
		winEnrolledUP.ID: winEnrolledUP,
		macEnrolledUP.ID: macEnrolledUP,
	}

	lockPending := map[uint]*fleet.Host{
		winEnrolledLP.ID: winEnrolledLP,
		macEnrolledLP.ID: macEnrolledLP,
	}

	wipePending := map[uint]*fleet.Host{
		winEnrolledWP.ID: winEnrolledWP,
		macEnrolledWP.ID: macEnrolledWP,
	}

	ds := setupTestServer(t)
	setupDSMocks(ds, hostByUUID, hostsByID)

	// custom mocks for these test
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
		fleetPlatform := host.FleetPlatform()

		var status fleet.HostLockWipeStatus
		status.HostFleetPlatform = fleetPlatform
		if _, ok := locked[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.LockMDMCommand = &fleet.MDMCommand{}
				status.LockMDMCommandResult = &fleet.MDMCommandResult{Status: fleet.MDMAppleStatusAcknowledged}
				return &status, nil
			}

			status.LockScript = &fleet.HostScriptResult{ExitCode: ptr.Int64(0)}
		}

		if _, ok := unlockPending[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.UnlockPIN = "1234"
				status.UnlockRequestedAt = time.Now()
				return &status, nil
			}

			status.UnlockScript = &fleet.HostScriptResult{}
		}

		if _, ok := lockPending[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.LockMDMCommand = &fleet.MDMCommand{}
				return &status, nil
			}

			status.LockScript = &fleet.HostScriptResult{}
		}

		if _, ok := wipePending[host.ID]; ok {
			if fleetPlatform == "linux" {
				status.WipeScript = &fleet.HostScriptResult{ExitCode: nil}
				return &status, nil
			}

			status.WipeMDMCommand = &fleet.MDMCommand{}
			status.WipeMDMCommandResult = nil
			return &status, nil
		}

		return &status, nil
	}
	ds.UnlockHostViaScriptFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload, platform string) error {
		return nil
	}
	ds.UnlockHostManuallyFunc = func(ctx context.Context, hostID uint, platform string, ts time.Time) error {
		return nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok || h.MDMInfo == nil {
			return nil, &notFoundError{}
		}

		return h.MDMInfo, nil
	}

	ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*fleet.HostOrbitInfo, error) {
		hostIDMod := hostID % 3
		switch hostIDMod {
		case 0:
			return nil, &notFoundError{}
		case 1:
			return &fleet.HostOrbitInfo{}, nil
		case 2:
			return &fleet.HostOrbitInfo{ScriptsEnabled: ptr.Bool(true)}, nil
		default:
			t.Errorf("unexpected hostIDMod %v", hostIDMod)
			return nil, nil
		}
	}

	appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM := setupAppConigs()

	successfulOutput := func(ident string) string {
		h := hostByUUID[ident]
		if h.Platform == "darwin" {
			return `Use this 6 digit PIN to unlock the host:`
		}
		return fmt.Sprintf(`
The host will unlock when it comes online.

Copy and run this command to see results:

fleetctl get host %s

`, ident)
	}

	cases := []struct {
		appCfg  *fleet.AppConfig
		desc    string
		flags   []string
		wantErr string
	}{
		{appCfgAllMDM, "no flags", nil, `Required flag "host" not set`},
		{appCfgAllMDM, "host flag empty", []string{"--host", ""}, `No host targeted. Please provide --host.`},
		{appCfgAllMDM, "unlock non-existent host", []string{"--host", "notfound"}, `The host doesn't exist. Please provide a valid host identifier.`},
		{appCfgMacMDM, "valid windows but only macos mdm", []string{"--host", winEnrolled.UUID}, `Windows MDM isn't turned on.`},
		{appCfgAllMDM, "valid windows", []string{"--host", winEnrolled.UUID}, ""},
		{appCfgAllMDM, "valid macos", []string{"--host", macEnrolled.UUID}, ""},
		{appCfgNoMDM, "valid linux", []string{"--host", linuxEnrolled.UUID}, ""},
		{appCfgNoMDM, "valid windows but no mdm", []string{"--host", winEnrolled.UUID}, `Windows MDM isn't turned on.`},
		// TODO: should we error here?
		// {appCfgNoMDM, "valid macos but no mdm", []string{"--host", macEnrolled.UUID}, `macOS MDM isn't turned on.`},
		{appCfgMacMDM, "valid macos but not enrolled", []string{"--host", macNotEnrolled.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but not enrolled", []string{"--host", winNotEnrolled.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but pending mdm enroll", []string{"--host", winPending.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgMacMDM, "valid macos but pending mdm enroll", []string{"--host", macPending.UUID}, `Can't unlock the host because it doesn't have MDM turned on.`},
		{appCfgAllMDM, "valid windows but pending unlock", []string{"--host", winEnrolledUP.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid macos but pending unlock", []string{"--host", macEnrolledUP.UUID}, ""},
		{appCfgAllMDM, "valid windows but pending lock", []string{"--host", winEnrolledLP.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid macos but pending lock", []string{"--host", macEnrolledLP.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid windows but pending wipe", []string{"--host", winEnrolledWP.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid macos but pending wipe", []string{"--host", macEnrolledWP.UUID}, "Host has pending wipe request."},
	}

	runTestCases(t, ds, "unlock", successfulOutput, cases)
}

func TestMDMWipeCommand(t *testing.T) {
	macEnrolled := &fleet.Host{
		ID:       1,
		UUID:     "mac-enrolled",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolled := &fleet.Host{
		ID:       2,
		UUID:     "win-enrolled",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winNotEnrolled := &fleet.Host{
		ID:       4,
		UUID:     "win-not-enrolled",
		Platform: "windows",
	}
	macNotEnrolled := &fleet.Host{
		ID:       5,
		UUID:     "mac-not-enrolled",
		Platform: "darwin",
	}
	macPending := &fleet.Host{
		ID:       6,
		UUID:     "mac-pending",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winPending := &fleet.Host{
		ID:       7,
		UUID:     "win-pending",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: false, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")},
	}
	winEnrolledUP := &fleet.Host{
		ID:       8,
		UUID:     "win-enrolled-up",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledUP := &fleet.Host{
		ID:       9,
		UUID:     "mac-enrolled-up",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledLP := &fleet.Host{
		ID:       10,
		UUID:     "win-enrolled-lp",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledLP := &fleet.Host{
		ID:       11,
		UUID:     "mac-enrolled-lp",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledWP := &fleet.Host{
		ID:       12,
		UUID:     "win-enrolled-wp",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledWP := &fleet.Host{
		ID:       13,
		UUID:     "mac-enrolled-wp",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledWiped := &fleet.Host{
		ID:       14,
		UUID:     "win-enrolled-wiped",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	macEnrolledWiped := &fleet.Host{
		ID:       15,
		UUID:     "mac-enrolled-wiped",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
	}
	winEnrolledLocked := &fleet.Host{
		ID:       16,
		UUID:     "win-enrolled-locked",
		Platform: "windows",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual")},
	}
	macEnrolledLocked := &fleet.Host{
		ID:       17,
		UUID:     "mac-enrolled-locked",
		Platform: "darwin",
		MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
		MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual")},
	}
	linuxEnrolled := &fleet.Host{
		ID:       18,
		UUID:     "linux-enrolled",
		Platform: "linux",
	}
	linuxEnrolled2 := &fleet.Host{
		ID:       19,
		UUID:     "linux-enrolled",
		Platform: "linux",
	}
	linuxEnrolled3 := &fleet.Host{
		ID:       20,
		UUID:     "linux-enrolled",
		Platform: "linux",
	}

	linuxHostIDs := []uint{linuxEnrolled.ID, linuxEnrolled2.ID, linuxEnrolled3.ID}

	hostByUUID := make(map[string]*fleet.Host)
	hostsByID := make(map[uint]*fleet.Host)
	for _, h := range []*fleet.Host{
		winEnrolled,
		macEnrolled,
		linuxEnrolled,
		linuxEnrolled2,
		linuxEnrolled3,
		macNotEnrolled,
		winNotEnrolled,
		macPending,
		winPending,
		winEnrolledUP,
		macEnrolledUP,
		winEnrolledLP,
		macEnrolledLP,
		winEnrolledWP,
		macEnrolledWP,
		winEnrolledWiped,
		macEnrolledWiped,
		winEnrolledLocked,
		macEnrolledLocked,
	} {
		hostByUUID[h.UUID] = h
		hostsByID[h.ID] = h
	}

	locked := map[uint]*fleet.Host{
		winEnrolledLocked.ID: winEnrolledLocked,
		macEnrolledLocked.ID: macEnrolledLocked,
	}

	unlockPending := map[uint]*fleet.Host{
		winEnrolledUP.ID: winEnrolledUP,
		macEnrolledUP.ID: macEnrolledUP,
	}

	lockPending := map[uint]*fleet.Host{
		winEnrolledLP.ID: winEnrolledLP,
		macEnrolledLP.ID: macEnrolledLP,
	}

	wipePending := map[uint]*fleet.Host{
		winEnrolledWP.ID: winEnrolledWP,
		macEnrolledWP.ID: macEnrolledWP,
	}

	wiped := map[uint]*fleet.Host{
		winEnrolledWiped.ID: winEnrolledWiped,
		macEnrolledWiped.ID: macEnrolledWiped,
	}

	ds := setupTestServer(t)
	setupDSMocks(ds, hostByUUID, hostsByID)

	// TODO: custom ds mocks for these tests
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
		fleetPlatform := host.FleetPlatform()

		var status fleet.HostLockWipeStatus
		status.HostFleetPlatform = fleetPlatform
		if _, ok := locked[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.LockMDMCommand = &fleet.MDMCommand{}
				status.LockMDMCommandResult = &fleet.MDMCommandResult{Status: fleet.MDMAppleStatusAcknowledged}
				return &status, nil
			}

			status.LockScript = &fleet.HostScriptResult{ExitCode: ptr.Int64(0)}
		}

		if _, ok := unlockPending[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.UnlockPIN = "1234"
				status.UnlockRequestedAt = time.Now()
				return &status, nil
			}

			status.UnlockScript = &fleet.HostScriptResult{}
		}

		if _, ok := lockPending[host.ID]; ok {
			if fleetPlatform == "darwin" {
				status.LockMDMCommand = &fleet.MDMCommand{}
				return &status, nil
			}

			status.LockScript = &fleet.HostScriptResult{}
		}

		if _, ok := wipePending[host.ID]; ok {
			if fleetPlatform == "linux" {
				status.WipeScript = &fleet.HostScriptResult{ExitCode: nil}
				return &status, nil
			}

			status.WipeMDMCommand = &fleet.MDMCommand{}
			status.WipeMDMCommandResult = nil
			return &status, nil
		}

		if _, ok := wiped[host.ID]; ok {
			if fleetPlatform == "linux" {
				status.WipeScript = &fleet.HostScriptResult{ExitCode: ptr.Int64(0)}
			}

			if fleetPlatform == "darwin" {
				status.WipeMDMCommand = &fleet.MDMCommand{}
				status.WipeMDMCommandResult = &fleet.MDMCommandResult{
					Status: fleet.MDMAppleStatusAcknowledged,
				}
			}

			if fleetPlatform == "windows" {
				status.WipeMDMCommand = &fleet.MDMCommand{}
				status.WipeMDMCommandResult = &fleet.MDMCommandResult{
					Status: "200",
				}
			}

			return &status, nil
		}

		return &status, nil
	}
	ds.UnlockHostViaScriptFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload, hostFleetPlatform string) error {
		return nil
	}
	ds.UnlockHostManuallyFunc = func(ctx context.Context, hostID uint, hostFleetPlatform string, ts time.Time) error {
		return nil
	}
	ds.WipeHostViaWindowsMDMFunc = func(ctx context.Context, host *fleet.Host, cmd *fleet.MDMWindowsCommand) error {
		return nil
	}

	ds.WipeHostViaScriptFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload, hostFleetPlatform string) error {
		return nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok || h.MDMInfo == nil {
			return nil, &notFoundError{}
		}

		return h.MDMInfo, nil
	}

	// This function should only run on linux
	ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*fleet.HostOrbitInfo, error) {
		if !slices.Contains(linuxHostIDs, hostID) {
			t.Errorf("GetHostOrbitInfo should not be called for non-linux host %v", hostID)
			return nil, nil
		}
		hostIDMod := hostID % 3
		switch hostIDMod {
		case 0:
			return nil, &notFoundError{}
		case 1:
			return &fleet.HostOrbitInfo{}, nil
		case 2:
			return &fleet.HostOrbitInfo{ScriptsEnabled: ptr.Bool(true)}, nil
		default:
			t.Errorf("unexpected hostIDMod %v", hostIDMod)
			return nil, nil
		}
	}

	appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM := setupAppConigs()
	appCfgScriptsDisabled := &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ScriptsDisabled: true}}

	cases := []struct {
		appCfg  *fleet.AppConfig
		desc    string
		flags   []string
		wantErr string
	}{
		{appCfgAllMDM, "no flags", nil, `Required flag "host" not set`},
		{appCfgAllMDM, "host flag empty", []string{"--host", ""}, `No host targeted. Please provide --host.`},
		{appCfgAllMDM, "wipe non-existent host", []string{"--host", "notfound"}, `The host doesn't exist. Please provide a valid host identifier.`},
		{appCfgMacMDM, "valid windows but only macos mdm", []string{"--host", winEnrolled.UUID}, `Windows MDM isn't turned on.`},
		{appCfgAllMDM, "valid windows", []string{"--host", winEnrolled.UUID}, ""},
		{appCfgAllMDM, "valid macos", []string{"--host", macEnrolled.UUID}, ""},
		{appCfgNoMDM, "valid linux", []string{"--host", linuxEnrolled.UUID}, ""},
		{appCfgNoMDM, "valid linux 2", []string{"--host", linuxEnrolled2.UUID}, ""},
		{appCfgNoMDM, "valid linux 3", []string{"--host", linuxEnrolled3.UUID}, ""},
		{appCfgNoMDM, "valid windows but no mdm", []string{"--host", winEnrolled.UUID}, `Windows MDM isn't turned on.`},
		{appCfgMacMDM, "valid macos but not enrolled", []string{"--host", macNotEnrolled.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but not enrolled", []string{"--host", winNotEnrolled.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgWinMDM, "valid windows but pending mdm enroll", []string{"--host", winPending.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgMacMDM, "valid macos but pending mdm enroll", []string{"--host", macPending.UUID}, `Can't wipe the host because it doesn't have MDM turned on.`},
		{appCfgAllMDM, "valid windows but pending unlock", []string{"--host", winEnrolledUP.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid macos but pending unlock", []string{"--host", macEnrolledUP.UUID}, "Host has pending unlock request."},
		{appCfgAllMDM, "valid windows but pending lock", []string{"--host", winEnrolledLP.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid macos but pending lock", []string{"--host", macEnrolledLP.UUID}, "Host has pending lock request."},
		{appCfgAllMDM, "valid windows but pending wipe", []string{"--host", winEnrolledWP.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid macos but pending wipe", []string{"--host", macEnrolledWP.UUID}, "Host has pending wipe request."},
		{appCfgAllMDM, "valid windows but host wiped", []string{"--host", winEnrolledWiped.UUID}, "Host is already wiped."},
		{appCfgAllMDM, "valid macos but host wiped", []string{"--host", macEnrolledWiped.UUID}, "Host is already wiped."},
		{appCfgAllMDM, "valid windows but host is locked", []string{"--host", winEnrolledLocked.UUID}, "Host cannot be wiped until it is unlocked."},
		{appCfgAllMDM, "valid macos but host is locked", []string{"--host", macEnrolledLocked.UUID}, "Host cannot be wiped until it is unlocked."},
		{appCfgAllMDM, "valid macos but host is locked", []string{"--host", macEnrolledLocked.UUID}, "Host cannot be wiped until it is unlocked."},
		{appCfgScriptsDisabled, "valid linux but script are disabled", []string{"--host", linuxEnrolled.UUID}, "Can't wipe host because running scripts is disabled in organization settings."},
	}

	successfulOutput := func(ident string) string {
		return fmt.Sprintf(`
The host will wipe when it comes online.

Copy and run this command to see results:

fleetctl get host %s`, ident)
	}

	runTestCases(t, ds, "wipe", successfulOutput, cases)
}

func writeTmpAppleMDMCmd(t *testing.T, commandName string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
      <key>RequestType</key>
      <string>%s</string>
    </dict>
  </dict>
</plist>`, uuid.New().String(), commandName))
	require.NoError(t, err)
	return tmpFile.Name()
}

func writeTmpWindowsMDMCmd(t *testing.T, commandName string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(fmt.Sprintf(`<Exec>
	<CmdID>11</CmdID>
	<Item>
		<Target>
			<LocURI>%s</LocURI>
		</Target>
		<Meta>
			<Format xmlns="syncml:metinf">chr</Format>
			<Type xmlns="syncml:metinf">text/plain</Type>
		</Meta>
		<Data>NamedValuesList=MinPasswordLength,8;</Data>
	</Item>
</Exec>`, commandName))
	require.NoError(t, err)
	return tmpFile.Name()
}

func writeTmpMobileconfig(t *testing.T, name string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.mobileconfig")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(string(mobileconfigForTest(name, uuid.New().String())))
	require.NoError(t, err)
	return tmpFile.Name()
}

// sets up the test server with the mock datastore and returns the mock datastore
func setupTestServer(t *testing.T) *mock.Store {
	enqueuer := new(mock.MDMAppleStore)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	enqueuer.EnqueueDeviceLockCommandFunc = func(ctx context.Context, host *fleet.Host, cmd *mdm.Command, pin string) error {
		return nil
	}

	enqueuer.EnqueueDeviceWipeCommandFunc = func(ctx context.Context, host *fleet.Host, cmd *mdm.Command) error {
		return nil
	}

	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{
		MDMStorage:       enqueuer,
		MDMPusher:        mockPusher{},
		License:          license,
		NoCacheDatastore: true,
	})

	return ds
}

// sets up common data store mocks that are needed for the tests.
func setupDSMocks(ds *mock.Store, hostByUUID map[string]*fleet.Host, hostsByID map[uint]*fleet.Host) {
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		h, ok := hostByUUID[identifier]
		if !ok {
			return nil, &notFoundError{}
		}
		return h, nil
	}
	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*fleet.HostBattery, error) {
		return nil, nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.GetHostMDMAppleProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMAppleProfile, error) {
		return nil, nil
	}
	ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMWindowsProfile, error) {
		return nil, nil
	}
	ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDMMacOSSetup, error) {
		return nil, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		h, ok := hostsByID[hostID]
		if !ok {
			return nil, &notFoundError{}
		}

		return h, nil
	}
	ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
		return nil, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		h, ok := hostsByID[hostID]
		if !ok {
			return nil, &notFoundError{}
		}

		return h.MDMInfo, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
}

// sets up the various app configs for the tests. These app configs reflect the various
// states of the MDM configuration.
func setupAppConigs() (*fleet.AppConfig, *fleet.AppConfig, *fleet.AppConfig, *fleet.AppConfig) {
	appCfgAllMDM := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true, WindowsEnabledAndConfigured: true}}
	appCfgWinMDM := &fleet.AppConfig{MDM: fleet.MDM{WindowsEnabledAndConfigured: true}}
	appCfgMacMDM := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
	appCfgNoMDM := &fleet.AppConfig{MDM: fleet.MDM{}}

	return appCfgAllMDM, appCfgWinMDM, appCfgMacMDM, appCfgNoMDM
}

func runTestCases(t *testing.T, ds *mock.Store, actionType string, successfulOutput func(ident string) string, cases []struct {
	appCfg  *fleet.AppConfig
	desc    string
	flags   []string
	wantErr string
},
) {
	for _, c := range cases {
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return c.appCfg, nil
		}
		buf, err := runAppNoChecks(append([]string{"mdm", actionType}, c.flags...))
		if c.wantErr != "" {
			require.Error(t, err, c.desc)
			require.ErrorContains(t, err, c.wantErr, c.desc)
		} else {
			require.NoError(t, err, c.desc)
			require.Contains(t, buf.String(), successfulOutput(c.flags[1]), c.desc)
		}
	}
}
