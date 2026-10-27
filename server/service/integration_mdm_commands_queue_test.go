package service

// MDM command queueing tests for the MDM suite.
//
// Belongs here: enqueueing, running, and listing MDM commands (Apple and Windows), commands with
// secrets, command results, invalid command UUIDs, and cleanup of stale Apple commands.
//
// Does not belong here: lock, unlock, wipe, and cancelling commands (integration_mdm_commands_test.go),
// upcoming activities (integration_mdm_upcoming_activities_test.go), and Android custom commands
// (integration_mdm_android_enrollment_test.go).

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestEnqueueMDMCommand() {
	ctx := context.Background()
	t := s.T()
	s.setSkipWorkerJobs(t)

	// list commands should return all the commands we sent
	var listCmdResp0 listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp0)
	require.Empty(t, listCmdResp0.Results)

	// Create host enrolled via osquery, but not enrolled in MDM.
	unenrolledHost := createHostAndDeviceToken(t, s.ds, "unused")

	// Create device enrolled in MDM but not enrolled via osquery.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	h, err := s.ds.HostByIdentifier(ctx, mdmDevice.UUID)
	require.NoError(t, err)
	h.Hostname = "test-host"
	err = s.ds.UpdateHost(ctx, h)
	require.NoError(t, err)

	base64Cmd := func(rawCmd string) string {
		return base64.RawStdEncoding.EncodeToString([]byte(rawCmd))
	}

	newRawCmd := func(cmdUUID string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>ManagedOnly</key>
        <false/>
        <key>RequestType</key>
        <string>ProfileList</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)
	}

	// call with unknown host UUID
	uuid1 := uuid.New().String()
	s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			// explicitly use standard encoding to make sure it also works
			// see #11384
			Command:   base64.StdEncoding.EncodeToString([]byte(newRawCmd(uuid1))),
			DeviceIDs: []string{"no-such-host"},
		}, http.StatusNotFound)

	// get command results returns 404, that command does not exist
	var getMDMCmdResp getMDMCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusNotFound, &getMDMCmdResp, "command_uuid", uuid1)

	// list commands returns empty set
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Empty(t, listCmdResp.Results)

	// call with unenrolled host UUID
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(newRawCmd(uuid.New().String())),
			DeviceIDs: []string{unenrolledHost.UUID},
		}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one of the hosts is not enrolled in MDM")

	// create a new Host to get the UUID on the DB
	linuxHost := createOrbitEnrolledHost(t, "linux", "h1", s.ds)
	windowsHost := createOrbitEnrolledHost(t, "windows", "h2", s.ds)
	// call with unenrolled host UUID
	res = s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(newRawCmd(uuid.New().String())),
			DeviceIDs: []string{linuxHost.UUID, windowsHost.UUID},
		}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one of the hosts is not enrolled in MDM or is not an elegible device")

	// call with payload that is not a valid, plist-encoded MDM command
	res = s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(string(mobileconfigForTest("test config profile", uuid.New().String()))),
			DeviceIDs: []string{mdmDevice.UUID},
		}, http.StatusUnsupportedMediaType)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "unable to decode plist command")

	// call with enrolled host UUID
	uuid2 := uuid.New().String()
	rawCmd := newRawCmd(uuid2)
	var resp enqueueMDMAppleCommandResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(rawCmd),
			DeviceIDs: []string{mdmDevice.UUID},
		}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.CommandUUID)
	require.Contains(t, rawCmd, resp.CommandUUID)
	require.Equal(t, "darwin", resp.Platform)
	require.Empty(t, resp.FailedUUIDs)
	require.Equal(t, "ProfileList", resp.RequestType)

	// the command exists but no results yet
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", uuid2)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Pending",
		RequestType: "ProfileList",
		Result:      []byte{},
		Payload:     []byte(rawCmd),
		Hostname:    "test-host",
	}, getMDMCmdResp.Results[0])

	// simulate a result and call again
	err = s.mdmStorage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: mdmDevice.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		Raw:         []byte(rawCmd),
	})
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", uuid2)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Result:      []byte(rawCmd),
		Payload:     []byte(rawCmd),
		Hostname:    "test-host",
	}, getMDMCmdResp.Results[0])

	// list commands returns that command
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	results, err := json.Marshal(listCmdResp.Results)
	require.NoError(t, err)
	t.Logf("GET /api/latest/fleet/mdm/apple/commands response:\n%s", results)

	// filter to the expected command.
	// there may be other commands due to the bootstrap packages uploaded in prior tests.
	// TODO: decouple these tests so we don't have to do this -- perhaps delete bootstrap packages?
	var profileListCommands []*fleet.MDMAppleCommand
	for _, result := range listCmdResp.Results {
		if result.RequestType == "ProfileList" {
			profileListCommands = append(profileListCommands, result)
		}
	}

	require.Len(t, profileListCommands, 1)
	require.NotZero(t, profileListCommands[0].UpdatedAt)
	profileListCommands[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMAppleCommand{
		DeviceID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Hostname:    "test-host",
	}, profileListCommands[0])
}

func (s *integrationMDMTestSuite) TestEnqueueMDMCommandWithSecret() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	_, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// list commands should return all the commands we sent
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp, "host_identifier", mdmClient.UUID)
	require.Empty(t, listCmdResp.Results)

	base64Cmd := func(rawCmd string) string {
		return base64.RawStdEncoding.EncodeToString([]byte(rawCmd))
	}

	newRawCmd := func(cmdUUID string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>ManagedOnly</key>
        <false/>
        <key>RequestType</key>
        <string>ProfileList</string>
		<key>SecretValue</key>
		<string>$FLEET_SECRET_VALUE</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)
	}

	// Load secret(s)
	secretValue := "*abc123*"
	req := fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_VALUE",
				Value: secretValue,
			},
		},
	}
	secretResp := fleet.CreateSecretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	// call with enrolled host UUID
	uuid2 := uuid.New().String()
	rawCmd := newRawCmd(uuid2)
	var resp enqueueMDMAppleCommandResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(rawCmd),
			DeviceIDs: []string{mdmClient.UUID},
		}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.CommandUUID)
	require.Contains(t, rawCmd, resp.CommandUUID)
	require.Equal(t, "darwin", resp.Platform)
	require.Empty(t, resp.FailedUUIDs)
	require.Equal(t, "ProfileList", resp.RequestType)

	// 1 command queued up
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp, "host_identifier", mdmClient.UUID)
	require.Len(t, listCmdResp.Results, 1)

	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Contains(t, string(cmd.Raw), secretValue)
	assert.NotContains(t, string(cmd.Raw), "FLEET_SECRET_VALUE")
}

func (s *integrationMDMTestSuite) TestListMDMCommands() {
	t := s.T()

	// No host identifier when using command status
	res := s.DoRaw("GET", "/api/latest/fleet/mdm/commands?command_status=ran", nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"host_identifier" must be specified when filtering by "command_status".`)

	// Invalid command_status
	res = s.DoRaw("GET", "/api/latest/fleet/mdm/commands?host_identifier=some-uuid&command_status=invalid-status", nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `command_status only accepts the following values:`)

	// Command status when host identifier resolves to Windows host
	ctx := context.Background()
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-win-host-name",
		OsqueryHostID: new("1337"),
		NodeKey:       new("1337"),
		UUID:          "test-win-host-uuid",
		Platform:      "windows",
	})
	require.NoError(t, err)
	res = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/commands?host_identifier=%s&command_status=ran", h.UUID), nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"command_status" filter is only available for macOS, iOS, iPadOS, and Android hosts`)

	// per_page above the documented maximum is rejected with a clear message.
	res = s.DoRaw("GET", "/api/latest/fleet/mdm/commands?per_page=1001", nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Please set a per_page limit of 1000 or less")

	// per_page at the cap is accepted.
	s.DoRaw("GET", "/api/latest/fleet/mdm/commands?per_page=1000", nil, http.StatusOK)

	// page above the cap is rejected so the inner LIMIT (page*per_page+per_page+1)
	// stays bounded.
	res = s.DoRaw("GET", "/api/latest/fleet/mdm/commands?page=101", nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Please set page to 100 or less")

	// page at the cap is accepted.
	s.DoRaw("GET", "/api/latest/fleet/mdm/commands?page=100", nil, http.StatusOK)

	// order_key not in the allowlist is rejected by the secure list-options
	// helper (defense against SQL injection via crafted ORDER BY). The
	// helper's InvalidOrderKeyError implements the validation-error
	// interface, so the response is a 422. Both the unscoped path and
	// the host-scoped path enforce the same allowlist.
	res = s.DoRaw("GET", "/api/latest/fleet/mdm/commands?order_key=team_id", nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "team_id")

	res = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/commands?host_identifier=%s&order_key=team_id", h.UUID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "team_id")

	// order_key=hostname is supported on both paths.
	s.DoRaw("GET", "/api/latest/fleet/mdm/commands?order_key=hostname", nil, http.StatusOK)
	s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/commands?host_identifier=%s&order_key=hostname", h.UUID), nil, http.StatusOK)
}

func (s *integrationMDMTestSuite) TestMDMWindowsCommandResults() {
	ctx := context.Background()
	t := s.T()

	h, err := s.ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-win-host-name",
		OsqueryHostID: new("1337"),
		NodeKey:       new("1337"),
		UUID:          "test-win-host-uuid",
		Platform:      "windows",
	})
	require.NoError(t, err)

	dev := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "test-device-id",
		MDMHardwareID:          "test-hardware-id",
		MDMDeviceState:         "ds",
		MDMDeviceType:          "dt",
		MDMDeviceName:          "dn",
		MDMEnrollType:          "et",
		MDMEnrollUserID:        "euid",
		MDMEnrollProtoVersion:  "epv",
		MDMEnrollClientVersion: "ecv",
		MDMNotInOOBE:           false,
		HostUUID:               h.UUID,
	}

	require.NoError(t, s.ds.MDMWindowsInsertEnrolledDevice(ctx, dev))
	var enrollmentID uint

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &enrollmentID, `SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, dev.MDMDeviceID)
	})

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE mdm_windows_enrollments SET host_uuid = ? WHERE id = ?`, dev.HostUUID, enrollmentID)
		return err
	})

	rawCmd := "some-command"
	cmdUUID := "some-uuid"
	cmdTarget := "some-target-loc-uri"

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`, cmdUUID, rawCmd, cmdTarget)
		return err
	})

	var responseID int64
	rawResponse := []byte("some-response")
	var rawResponseGz bytes.Buffer
	gzw := gzip.NewWriter(&rawResponseGz)
	_, gzErr := gzw.Write(rawResponse)
	require.NoError(t, gzErr)
	require.NoError(t, gzw.Close())
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response_gz) VALUES (?, ?)`, enrollmentID, rawResponseGz.Bytes())
		if err != nil {
			return err
		}
		responseID, err = res.LastInsertId()
		return err
	})

	rawResult := []byte("some-result")
	statusCode := "200"
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, response_id, status_code) VALUES (?, ?, ?, ?, ?)`, enrollmentID, cmdUUID, rawResult, responseID, statusCode)
		return err
	})

	var resp getMDMCommandResultsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", cmdUUID), nil, http.StatusOK, &resp)
	require.Len(t, resp.Results, 1)
	require.Equal(t, dev.HostUUID, resp.Results[0].HostUUID)
	require.Equal(t, cmdUUID, resp.Results[0].CommandUUID)
	require.Equal(t, rawResponse, resp.Results[0].Result)
	require.Equal(t, cmdTarget, resp.Results[0].RequestType)
	require.Equal(t, statusCode, resp.Results[0].Status)
	require.Equal(t, h.Hostname, resp.Results[0].Hostname)

	resp = getMDMCommandResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", uuid.New().String()), nil, http.StatusNotFound, &resp)
	require.Empty(t, resp.Results)
}

func (s *integrationMDMTestSuite) TestRunMDMCommands() {
	t := s.T()

	// create a Windows host enrolled in MDM
	enrolledWindows, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create an unenrolled Windows host
	unenrolledWindows := createOrbitEnrolledHost(t, "windows", "h2", s.ds)

	// create an enrolled and unenrolled macOS host
	enrolledMac, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	unenrolledMac := createOrbitEnrolledHost(t, "darwin", "h4", s.ds)

	macRawCmd := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>ShutDownDevice</string>
    </dict>
    <key>CommandUUID</key>
    <string>0001_ShutDownDevice</string>
</dict>
</plist>`

	winRawCmd := `
<Exec>
	<CmdID>11</CmdID>
	<Item>
		<Target>
			<LocURI>./SetValues</LocURI>
		</Target>
		<Meta>
			<Format xmlns="syncml:metinf">chr</Format>
			<Type xmlns="syncml:metinf">text/plain</Type>
		</Meta>
		<Data>NamedValuesList=MinPasswordLength,8;</Data>
	</Item>
</Exec>
`

	var runResp runMDMCommandResponse

	// no host provided
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command: base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
	}, http.StatusNotFound, &runResp)

	// mix of mdm and non-mdm hosts
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID, unenrolledMac.UUID},
	}, http.StatusPreconditionFailed, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID, unenrolledWindows.UUID},
	}, http.StatusPreconditionFailed, &runResp)

	// mix of windows and macos hosts
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID, enrolledWindows.UUID},
	}, http.StatusUnprocessableEntity, &runResp)

	// windows only, invalid command
	res := s.Do("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "You can run only <Exec> command type")

	// macOS only, invalid command
	res = s.Do("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID},
	}, http.StatusUnsupportedMediaType)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "unable to decode plist command")

	// valid windows
	runResp = runMDMCommandResponse{}
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID},
	}, http.StatusOK, &runResp)
	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, "windows", runResp.Platform)
	require.Equal(t, "./SetValues", runResp.RequestType)
	s.lastActivityMatches(fleet.ActivityTypeRanCustomMDMCommand{}.ActivityName(), fmt.Sprintf(`{
		"host_id": %d,
		"host_display_name": %q,
		"host_uuid": %q,
		"command_uuid": %q,
		"request_type": "./SetValues",
		"platform": "windows"
	}`, enrolledWindows.ID, enrolledWindows.DisplayName(), enrolledWindows.UUID, runResp.CommandUUID), 0)

	// valid macOS
	runResp = runMDMCommandResponse{}
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID},
	}, http.StatusOK, &runResp)
	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, "darwin", runResp.Platform)
	require.Equal(t, "ShutDownDevice", runResp.RequestType)
	s.lastActivityMatches(fleet.ActivityTypeRanCustomMDMCommand{}.ActivityName(), fmt.Sprintf(`{
		"host_id": %d,
		"host_display_name": %q,
		"host_uuid": %q,
		"command_uuid": %q,
		"request_type": "ShutDownDevice",
		"platform": "darwin"
	}`, enrolledMac.ID, enrolledMac.DisplayName(), enrolledMac.UUID, runResp.CommandUUID), 0)
}

func (s *integrationMDMTestSuite) TestInvalidCommandUUID() {
	t := s.T()
	_, device := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	cmd, err := device.Acknowledge("foo")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func (s *integrationMDMTestSuite) TestAppleMDMCommandCleanup() {
	t := s.T()
	ctx := t.Context()

	origPush := s.pushProvider.PushFunc
	t.Cleanup(func() { s.pushProvider.PushFunc = origPush })
	s.pushProvider.PushFunc = mockSuccessfulPush

	// a Mac receives a profile it must keep; an iPhone receives a refetch it
	// should shed (refetches are MDM commands only on iOS/iPadOS)
	macHost, mac := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// the profile schedule assigns, the Apple MDM worker sends; the
	// processing key is the per-host lock between the two
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+macHost.UUID))
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	iphoneHost, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:  mdmtest.RandSerialNumber(),
		UUID:            strings.ToUpper(uuid.NewString()),
		Platform:        string(fleet.IOSPlatform),
		DetailUpdatedAt: time.Now().Add(-2 * time.Hour),
	})
	require.NoError(t, err)
	iphone := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "iPhone14,6")
	iphone.UUID = iphoneHost.UUID
	iphone.SerialNumber = iphoneHost.HardwareSerial
	require.NoError(t, iphone.Enroll())
	require.NoError(t, apple_mdm.IOSiPadOSRefetch(ctx, s.ds, s.mdmCommander, s.logger, s.fleetSvc.NewActivity))

	// both devices answer everything they are served
	var profileCmds []string
	cmd, err := mac.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "InstallProfile" {
			profileCmds = append(profileCmds, cmd.CommandUUID)
		}
		cmd, err = mac.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotEmpty(t, profileCmds)

	var refetchCmds []string
	cmd, err = iphone.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if strings.HasPrefix(cmd.CommandUUID, fleet.RefetchBaseCommandUUIDPrefix) {
			refetchCmds = append(refetchCmds, cmd.CommandUUID)
		}
		switch cmd.Command.RequestType {
		case "DeviceInformation":
			cmd, err = iphone.AcknowledgeDeviceInformation(iphone.UUID, cmd.CommandUUID, "cleanup-iphone", "iPhone 14", "UTC")
		case "InstalledApplicationList":
			cmd, err = iphone.AcknowledgeInstalledApplicationList(iphone.UUID, cmd.CommandUUID, nil)
		case "CertificateList":
			cmd, err = iphone.AcknowledgeCertificateList(iphone.UUID, cmd.CommandUUID, nil)
		default:
			cmd, err = iphone.Acknowledge(cmd.CommandUUID)
		}
		require.NoError(t, err)
	}
	require.NotEmpty(t, refetchCmds)

	// every answer is now older than both windows; only the reference guard
	// stands between the profile commands and deletion
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE nano_command_results SET updated_at = NOW(6) - INTERVAL 31 DAY WHERE id IN (?, ?)`, macHost.UUID, iphoneHost.UUID)
		return err
	})

	_, stats, err := s.ds.CleanupNanoCommands(ctx, fleet.MDMAppleCommandCleanupOptions{
		ShortRetention:    24 * time.Hour,
		StandardRetention: 30 * 24 * time.Hour,
		MaxRowDeletions:   1000,
		MaxCmdDeletions:   1000,
	}, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.ShortPairsDeleted, len(refetchCmds))

	countRows := func(table, cmdUUID string) int {
		var n int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n, `SELECT COUNT(*) FROM `+table+` WHERE command_uuid = ?`, cmdUUID)
		})
		return n
	}
	for _, c := range refetchCmds {
		require.Zero(t, countRows("nano_enrollment_queue", c), c)
		require.Zero(t, countRows("nano_command_results", c), c)
		require.Zero(t, countRows("nano_commands", c), c)
	}
	for _, c := range profileCmds {
		require.Equal(t, 1, countRows("nano_enrollment_queue", c), c)
		require.Equal(t, 1, countRows("nano_command_results", c), c)
		require.Equal(t, 1, countRows("host_mdm_apple_profiles", c), "still the host's current profile command")
	}

	// the UI listing shows the profile commands and none of the refetches
	var listResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listResp)
	listed := map[string]struct{}{}
	for _, c := range listResp.Results {
		listed[c.CommandUUID] = struct{}{}
	}
	for _, c := range profileCmds {
		require.Contains(t, listed, c)
	}
	for _, c := range refetchCmds {
		require.NotContains(t, listed, c)
	}

	// the devices' next check-ins are served nothing: no command was re-queued
	cmd, err = mac.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)
	cmd, err = iphone.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)
}

func (s *integrationMDMTestSuite) TestAppleCustomCommandRejectsHostSecretVariables() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	require.NoError(t, mdmDevice.Enroll())

	rawCmd := func(cmdUUID, command string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
%s
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, command, cmdUUID)
	}
	commands := map[string]string{
		"RotateFileVaultKey": `        <key>RequestType</key>
        <string>RotateFileVaultKey</string>
        <key>KeyType</key>
        <string>personal</string>
        <key>FileVaultUnlock</key>
        <dict>
            <key>Password</key>
            <string>` + fleet.HostSecretPlaceholder(fleet.HostSecretFileVaultKey) + `</string>
        </dict>`,
		"VerifyRecoveryLock": `        <key>RequestType</key>
        <string>VerifyRecoveryLock</string>
        <key>Password</key>
        <string>${` + fleet.HostSecretPrefix + fleet.HostSecretRecoveryLockPassword + `}</string>`,
		"Settings": `        <key>RequestType</key>
        <string>Settings</string>
        <key>Settings</key>
        <array>
            <dict>
                <key>Item</key>
                <string>DeviceName</string>
                <key>DeviceName</key>
                <string>` + fleet.HostSecretPlaceholder(fleet.HostSecretMDMUnlockToken) + `</string>
            </dict>
        </array>`,
	}

	for name, command := range commands {
		for _, endpoint := range []string{"/api/latest/fleet/commands/run", "/api/latest/fleet/mdm/apple/enqueue"} {
			t.Run(name+" "+endpoint, func(t *testing.T) {
				cmdUUID := uuid.NewString()
				encoded := base64.StdEncoding.EncodeToString([]byte(rawCmd(cmdUUID, command)))
				var body any = &runMDMCommandRequest{Command: encoded, HostUUIDs: []string{mdmDevice.UUID}}
				if endpoint == "/api/latest/fleet/mdm/apple/enqueue" {
					body = enqueueMDMAppleCommandRequest{Command: encoded, DeviceIDs: []string{mdmDevice.UUID}}
				}
				res := s.Do("POST", endpoint, body, http.StatusBadRequest)
				assert.Contains(t, extractServerErrorText(res.Body), "is reserved")

				s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusNotFound, &getMDMCommandResultsResponse{}, "command_uuid", cmdUUID)
				cmd, err := mdmDevice.Idle()
				require.NoError(t, err)
				require.Nil(t, cmd)
			})
		}
	}
}
