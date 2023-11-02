package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mock "github.com/fleetdm/fleet/v4/server/mock/nanomdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
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
	for _, h := range []*fleet.Host{macEnrolled, winEnrolled, macUnenrolled, winUnenrolled, linuxUnenrolled, macEnrolled2, winEnrolled2, macNonFleetEnrolled, winNonFleetEnrolled, macPending, winPending} {
		hostByUUID[h.UUID] = h
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
			enqueuer := new(mock.Storage)
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
			ds.GetHostMDMProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMAppleProfile, error) {
				return nil, nil
			}
			ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDMMacOSSetup, error) {
				return nil, nil
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
			ds.MDMWindowsInsertCommandForHostsFunc = func(ctx context.Context, deviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
				return nil
			}
			ds.GetMDMWindowsBitLockerStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostMDMDiskEncryption, error) {
				return &fleet.HostMDMDiskEncryption{}, nil
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
