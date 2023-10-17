package main

import (
	"context"
	"fmt"
	"os"
	"testing"

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
	enqueuer := new(mock.Storage)
	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{MDMStorage: enqueuer, MDMPusher: mockPusher{}})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		switch identifier {
		case "no-such-host":
			return nil, &notFoundError{}
		case "no-mdm-host":
			return &fleet.Host{ID: 1, UUID: identifier}, nil
		case "no-fleet-mdm-host":
			return &fleet.Host{ID: 2, UUID: identifier, MDM: fleet.MDMHostData{Name: fleet.WellKnownMDMJamf, EnrollmentStatus: ptr.String("On (manual)")}}, nil
		case "fleet-mdm-pending-host":
			return &fleet.Host{ID: 3, UUID: identifier, MDM: fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("Pending")}}, nil
		default:
			return &fleet.Host{ID: 4, Platform: "darwin", UUID: identifier, MDM: fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")}}, nil
		}
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
		return []*fleet.Host{
			{
				ID:       4,
				UUID:     uuids[0],
				Platform: "darwin",
				MDMInfo:  &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet},
				MDM:      fleet.MDMHostData{Name: fleet.WellKnownMDMFleet, EnrollmentStatus: ptr.String("On (manual)")},
			},
		}, nil
	}
	enqueuer.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
		return map[string]error{}, nil
	}

	_, err := runAppNoChecks([]string{"mdm", "run-command"})
	require.Error(t, err)
	require.ErrorContains(t, err, `Required flags "hosts, payload" not set`)

	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "abc"})
	require.Error(t, err)
	require.ErrorContains(t, err, `Required flag "payload" not set`)

	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "abc", "--payload", "no-such-file"})
	require.Error(t, err)
	require.ErrorContains(t, err, `open no-such-file: no such file or directory`)

	// pass a yaml file instead of xml
	yamlFilePath := writeTmpYml(t, `invalid`)
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "valid", "--payload", yamlFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `The payload isn't valid XML.`)

	// pass valid xml plist that doesn't match the MDM command schema
	mcFilePath := writeTmpMobileconfig(t, "Mobileconfig")
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "valid-host", "--payload", mcFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file:`)

	// host not found
	cmdFilePath := writeTmpMDMCmd(t, "FooBar")
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "no-such-host", "--payload", cmdFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `The host doesn't exist.`)

	// host not in mdm
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "no-mdm-host", "--payload", cmdFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `Can't run the MDM command because the host doesn't have MDM turned on.`)

	// host not in fleet mdm
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "no-fleet-mdm-host", "--payload", cmdFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `Can't run the MDM command because the host doesn't have MDM turned on.`)

	// host in fleet mdm but pending
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "fleet-mdm-pending-host", "--payload", cmdFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `Can't run the MDM command because the host doesn't have MDM turned on.`)

	// host enrolled in fleet mdm
	buf, err := runAppNoChecks([]string{"mdm", "run-command", "--hosts", "valid-host", "--payload", cmdFilePath})
	require.NoError(t, err)
	require.Contains(t, buf.String(), `The hosts will run the command the next time it checks into Fleet.`)
	require.Contains(t, buf.String(), `fleetctl get mdm-command-results --id=`)

	// try to run a fleet premium command
	cmdFilePath = writeTmpMDMCmd(t, "EraseDevice")
	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "valid-host", "--payload", cmdFilePath})
	require.Error(t, err)
	require.ErrorContains(t, err, `missing or invalid license`)

	// try to run an empty plist as a command
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.xml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
</plist>`))
	require.NoError(t, err)

	_, err = runAppNoChecks([]string{"mdm", "run-command", "--hosts", "valid-host", "--payload", tmpFile.Name()})
	require.Error(t, err)
	require.ErrorContains(t, err, `The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file.`)
}

func writeTmpMDMCmd(t *testing.T, commandName string) string {
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

func writeTmpMobileconfig(t *testing.T, name string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.mobileconfig")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(string(mobileconfigForTest(name, uuid.New().String())))
	require.NoError(t, err)
	return tmpFile.Name()
}
