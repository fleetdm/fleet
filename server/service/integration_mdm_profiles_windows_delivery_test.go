package service

// Windows configuration profile tests for the MDM suite.
//
// Belongs here: Windows profile delivery, status, retries and resends, Windows profiles in fleet
// specs, and Fleet variables in Windows profiles.
//
// Does not belong here: Windows SCEP profiles (integration_mdm_profiles_certs_test.go), profile CRUD
// shared across platforms (integration_mdm_profiles_crud_test.go), batch endpoints
// (integration_mdm_profiles_batch_test.go), and user-scoped Windows profiles
// (integration_mdm_windows_userscope_test.go).

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestWindowsProfileRetries() {
	t := s.T()
	ctx := context.Background()

	testProfiles := []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}})},
		{Name: "N2", Contents: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L2", Data: "D2"}, {Verb: "Add", LocURI: "L3", Data: "D3"}})},
	}

	h, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	expectedProfileStatuses := map[string]fleet.MDMDeliveryStatus{
		"N1": fleet.MDMDeliveryVerified,
		"N2": fleet.MDMDeliveryVerified,
	}
	checkProfilesStatus := func(t *testing.T) {
		storedProfs, err := s.ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, storedProfs, len(expectedProfileStatuses))
		for _, p := range storedProfs {
			want, ok := expectedProfileStatuses[p.Name]
			require.True(t, ok, "unexpected profile: %s", p.Name)
			require.Equal(t, want, *p.Status, "expected status %s but got %s for profile: %s", want, *p.Status, p.Name)
		}
	}

	expectedRetryCounts := map[string]uint{
		"N1": 0,
		"N2": 0,
	}
	checkRetryCounts := func(t *testing.T) {
		counts, err := s.ds.GetHostMDMProfilesRetryCounts(ctx, h)
		require.NoError(t, err)
		require.Len(t, counts, len(expectedRetryCounts))
		for _, c := range counts {
			want, ok := expectedRetryCounts[c.ProfileName]
			require.True(t, ok, "unexpected profile: %s", c.ProfileName)
			require.Equal(t, want, c.Retries, "expected retry count %d but got %d for profile: %s", want, c.Retries, c.ProfileName)
		}
	}

	// A profile with retries left reads as "pending" while Fleet tries again, so it must not also carry the failed
	// attempt's error. The device's error comes back only once the failure is terminal.
	profileDetail := func(t *testing.T, name string) string {
		storedProfs, err := s.ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
		require.NoError(t, err)
		for _, p := range storedProfs {
			if p.Name == name {
				return p.Detail
			}
		}
		t.Fatalf("profile %s not found for host %s", name, h.UUID)
		return ""
	}
	requireNoDetailWhileRetrying := func(t *testing.T, name string) {
		require.Empty(t, profileDetail(t, name), "profile %s is waiting on a retry and must not surface an error", name)
	}
	requireDetailOnceFailed := func(t *testing.T, name string) {
		require.NotEmpty(t, profileDetail(t, name), "profile %s ran out of retries and must surface the device error", name)
	}

	verifyCommands := func(wantProfileInstalls int, status string) {
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		// profile installs + 2 protocol commands acks
		require.Len(t, cmds, wantProfileInstalls+2)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		atomicCmds := 0
		for _, c := range cmds {
			if c.Verb == "Atomic" {
				atomicCmds++
			}

			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  ptr.String(c.Cmd.CmdID.Value),
				Cmd:     ptr.String(c.Verb),
				Data:    ptr.String(status),
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		require.Equal(t, wantProfileInstalls, atomicCmds)
		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
	}

	t.Run("does not retry after successful delivery", func(t *testing.T) {
		t.Cleanup(func() {
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				stmt := `DELETE FROM host_mdm_windows_profiles WHERE host_uuid = ?`
				_, err := q.ExecContext(ctx, stmt, h.UUID)
				return err
			})
		})

		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)

		verifyCommands(len(testProfiles), syncml.CmdStatusOK)
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryVerified
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t) // all profiles verified
		checkRetryCounts(t)    // no retries
	})

	retriesBeforeFailure := servermdm.MaxWindowsProfileRetries
	t.Run(fmt.Sprintf("retries %d times before marking as failed", retriesBeforeFailure), func(t *testing.T) {
		// A real Windows device acks every command nested inside the <Atomic>, not just the wrapper, and Fleet builds
		// a failed profile's detail out of those nested statuses. The fixture has to send them for the detail
		// assertions below to mean anything.
		appendNestedStatuses := func(c fleet.ProtoCmdOperation, msgID, status string) {
			nested := append(append([]fleet.SyncMLCmd{}, c.Cmd.ReplaceCommands...), c.Cmd.AddCommands...)
			for _, n := range nested {
				cmdRef, verb := n.CmdID.Value, n.XMLName.Local
				mdmDevice.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &cmdRef,
					Cmd:     &verb,
					Data:    &status,
					Items:   nil,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
			}
		}

		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			mysqltest.DumpTable(t, q, "host_mdm_windows_profiles")
			return nil
		})

		for i := range retriesBeforeFailure {
			s.awaitTriggerProfileSchedule(t)

			cmds, err := mdmDevice.StartManagementSession()
			require.NoError(t, err)
			msgID, err := mdmDevice.GetCurrentMsgID()
			require.NoError(t, err)

			// Reply to initial status commands
			for _, c := range cmds {
				if c.Verb == "Status" {
					mdmDevice.AppendResponse(fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdStatus},
						MsgRef:  &msgID,
						CmdRef:  ptr.String(c.Cmd.CmdID.Value),
						Cmd:     ptr.String(c.Verb),
						Data:    ptr.String(syncml.CmdStatusOK),
						Items:   nil,
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
					})
				}
			}
			cmds, err = mdmDevice.SendResponse()
			require.NoError(t, err)

			atomicCmds := 0
			for _, c := range cmds {
				if c.Verb == "Atomic" {
					atomicCmds++
				}

				if c.Verb == "Status" {
					mdmDevice.AppendResponse(fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdStatus},
						MsgRef:  &msgID,
						CmdRef:  ptr.String(c.Cmd.CmdID.Value),
						Cmd:     ptr.String(c.Verb),
						Data:    ptr.String(syncml.CmdStatusOK),
						Items:   nil,
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
					})
					continue
				}

				status := syncml.CmdStatusOK
				if len(c.Cmd.ReplaceCommands) > 0 && c.Cmd.ReplaceCommands[0].GetTargetURI() == "L1" {
					status = syncml.CmdStatusAtomicFailed
				}

				mdmDevice.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  ptr.String(c.Cmd.CmdID.Value),
					Cmd:     ptr.String(c.Verb),
					Data:    ptr.String(status),
					Items:   nil,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
				appendNestedStatuses(c, msgID, status)
			}
			// The first pass delivers both profiles. N2 is acknowledged and never resent, so every later retry
			// carries N1 alone.
			wantAtomicCmds := 1
			if i == 0 {
				wantAtomicCmds = 2
			}
			require.Equal(t, wantAtomicCmds, atomicCmds)
			cmds, err = mdmDevice.SendResponse()
			require.NoError(t, err)
			// the ack of the message should be the only returned command
			require.Len(t, cmds, 1)

			expectedProfileStatuses["N1"] = fleet.MDMDeliveryPending
			expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
			checkProfilesStatus(t)
			expectedRetryCounts["N1"] = uint(i + 1) //nolint:gosec // dismiss G115
			checkRetryCounts(t)
			requireNoDetailWhileRetrying(t, "N1")
		}

		// Final run to mark it as failed
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)

		atomicCmds := 0
		for _, c := range cmds {
			if c.Verb == "Atomic" {
				atomicCmds++
			}

			if c.Verb == "Status" {
				mdmDevice.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  ptr.String(c.Cmd.CmdID.Value),
					Cmd:     ptr.String(c.Verb),
					Data:    ptr.String(syncml.CmdStatusOK),
					Items:   nil,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
				continue
			}

			status := syncml.CmdStatusOK
			if len(c.Cmd.ReplaceCommands) > 0 && c.Cmd.ReplaceCommands[0].GetTargetURI() == "L1" {
				status = syncml.CmdStatusAtomicFailed
			}

			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  ptr.String(c.Cmd.CmdID.Value),
				Cmd:     ptr.String(c.Verb),
				Data:    ptr.String(status),
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
			appendNestedStatuses(c, msgID, status)
		}
		require.Equal(t, 1, atomicCmds)
		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)

		expectedProfileStatuses["N1"] = fleet.MDMDeliveryFailed
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		expectedRetryCounts["N1"] = servermdm.MaxWindowsProfileRetries
		checkRetryCounts(t)
		requireDetailOnceFailed(t, "N1")
	})
}

// TestWindowsProfileResend verifies that a Windows profile is resent when its contents have been modified.
func (s *integrationMDMTestSuite) TestWindowsProfileResend() {
	t := s.T()
	ctx := context.Background()

	testProfiles := []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}})},
		{Name: "N2", Contents: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L2", Data: "D2"}, {Verb: "Replace", LocURI: "L3", Data: "D3"}})},
	}

	h, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	expectedProfileStatuses := map[string]fleet.MDMDeliveryStatus{
		"N1": fleet.MDMDeliveryVerifying,
		"N2": fleet.MDMDeliveryVerifying,
	}
	checkProfilesStatus := func(t *testing.T) {
		storedProfs, err := s.ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, storedProfs, len(expectedProfileStatuses))
		for _, p := range storedProfs {
			want, ok := expectedProfileStatuses[p.Name]
			require.True(t, ok, "unexpected profile: %s", p.Name)
			require.Equal(t, want, *p.Status, "expected status %s but got %s for profile: %s", want, *p.Status, p.Name)
		}
	}

	// verifyCommands checks the number of commands sent to the device.
	// nRemovals is optional. When a profile is replaced, <Delete> commands
	// are generated for the old version in addition to install commands.
	verifyCommands := func(wantProfileInstalls int, status string, nRemovals ...int) {
		nRemove := 0
		if len(nRemovals) > 0 {
			nRemove = nRemovals[0]
		}
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		// profile installs + delete commands + 2 protocol commands acks
		require.Len(t, cmds, wantProfileInstalls+nRemove+2)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		atomicCmds := 0
		for _, c := range cmds {
			if c.Verb == "Atomic" {
				atomicCmds++
			}
			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  ptr.String(c.Cmd.CmdID.Value),
				Cmd:     ptr.String(c.Verb),
				Data:    ptr.String(status),
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		require.Equal(t, wantProfileInstalls+nRemove, atomicCmds)
		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
	}

	// cleanupWindowsHostState removes host-profile rows and queued commands
	// left behind by two-phase removal, so subsequent subtests start clean.
	cleanupWindowsHostState := func() {
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}},
			http.StatusNoContent)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(context.Background(), `DELETE FROM host_mdm_windows_profiles WHERE host_uuid = ?`, h.UUID); err != nil {
				return err
			}
			_, err := q.ExecContext(context.Background(), `
				DELETE wmcq FROM windows_mdm_command_queue wmcq
				JOIN mdm_windows_enrollments mwe ON mwe.id = wmcq.enrollment_id
				WHERE mwe.host_uuid = ?`, h.UUID)
			return err
		})
	}

	t.Run("do not resend if nothing changed", func(t *testing.T) {
		t.Cleanup(cleanupWindowsHostState)

		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// profiles to install + 2 boilerplate <Status>
		verifyCommands(len(testProfiles), syncml.CmdStatusOK)
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryVerified
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t) // all profiles verified

		// trigger a profile sync and confirm that no profiles were sent
		verifyCommands(0, syncml.CmdStatusOK)

		// Upload the same profiles again
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that no profiles were sent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("resend if contents changed", func(t *testing.T) {
		t.Cleanup(cleanupWindowsHostState)

		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// profiles to install + 2 boilerplate <Status>
		verifyCommands(len(testProfiles), syncml.CmdStatusOK)
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryVerified
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t) // all profiles verified

		// trigger a profile sync and confirm that no profiles were sent
		verifyCommands(0, syncml.CmdStatusOK)

		// Change one profile and upload
		copiedTestProfiles := make([]fleet.MDMProfileBatchPayload, len(testProfiles))
		copy(copiedTestProfiles, testProfiles)
		copiedTestProfiles[0].Contents = syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1-Modified"}})
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: copiedTestProfiles}, http.StatusNoContent)

		// Confirm that one install profile was re-sent with updated content.
		// When a profile's content changes (same name, different checksum), the
		// profile is updated in place (not deleted+re-created), so only the
		// new install command is sent -- no <Delete> needed.
		verifyCommands(1, syncml.CmdStatusOK)
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryVerified
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t) // all profiles verified
	})
}

func (s *integrationMDMTestSuite) TestWindowsProfileManagement() {
	t := s.T()
	ctx := context.Background()

	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	globalProfiles := []string{
		mysqltest.InsertWindowsProfileForTest(t, s.ds, 0),
		mysqltest.InsertWindowsProfileForTest(t, s.ds, 0),
		mysqltest.InsertWindowsProfileForTest(t, s.ds, 0),
	}

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)
	teamProfiles := []string{
		mysqltest.InsertWindowsProfileForTest(t, s.ds, tm.ID),
		mysqltest.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}

	// create a non-Windows host
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("non-windows-host"),
		NodeKey:       ptr.String("non-windows-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.windows", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a Windows host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            2,
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	verifyHostProfileStatus := func(cmds []fleet.ProtoCmdOperation, wantStatus string) {
		for _, cmd := range cmds {
			var gotProfile struct {
				Status  string `db:"status"`
				Retries int    `db:"retries"`
			}
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				stmt := `
				SELECT COALESCE(status, 'pending') as status, retries
				FROM host_mdm_windows_profiles
				WHERE command_uuid = ?`
				return sqlx.GetContext(context.Background(), q, &gotProfile, stmt, cmd.Cmd.CmdID.Value)
			})

			wantDeliveryStatus := fleet.WindowsResponseToDeliveryStatus(wantStatus)
			if gotProfile.Retries <= servermdm.MaxWindowsProfileRetries && wantDeliveryStatus == fleet.MDMDeliveryFailed {
				require.EqualValues(t, "pending", gotProfile.Status, "command_uuid", cmd.Cmd.CmdID.Value)
			} else {
				require.EqualValues(t, wantDeliveryStatus, gotProfile.Status, "command_uuid", cmd.Cmd.CmdID.Value)
			}
		}
	}

	verifyProfiles := func(device *mdmtest.TestWindowsMDMClient, n int, fail bool, nRemovals ...int) {
		mdmResponseStatus := syncml.CmdStatusOK
		if fail {
			mdmResponseStatus = syncml.CmdStatusAtomicFailed
		}
		nRemove := 0
		if len(nRemovals) > 0 {
			nRemove = nRemovals[0]
		}
		s.awaitTriggerProfileSchedule(t)
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)
		// 2 Status + n install profiles + nRemove delete commands
		require.Len(t, cmds, n+nRemove+2)

		var atomicInstallCmds []fleet.ProtoCmdOperation
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			cmdID := c.Cmd.CmdID
			status := syncml.CmdStatusOK
			if c.Verb == "Atomic" {
				if len(c.Cmd.ReplaceCommands) > 0 {
					// Install command (Atomic with Replace sub-commands)
					atomicInstallCmds = append(atomicInstallCmds, c)
					status = mdmResponseStatus
					for _, rc := range c.Cmd.ReplaceCommands {
						require.NotEmpty(t, rc.CmdID)
					}
				} else {
					// Delete command (Atomic with Delete sub-commands)
					status = syncml.CmdStatusOK
				}
			}
			device.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID.Value,
				Cmd:     ptr.String(c.Verb),
				Data:    &status,
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		// TODO: verify profile contents as well
		require.Len(t, atomicInstallCmds, n)

		// before we send the response, commands should be "pending"
		verifyHostProfileStatus(atomicInstallCmds, "")

		cmds, err = device.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)

		// verify that we updated status in the db
		verifyHostProfileStatus(atomicInstallCmds, mdmResponseStatus)
	}

	// drainCommands triggers the profile schedule, then acks every command the device is handed with 200 OK and returns them.
	// Several subtests below need the device brought to a clean state before exercising the behaviour they actually assert on.
	drainCommands := func(device *mdmtest.TestWindowsMDMClient) map[string]fleet.ProtoCmdOperation {
		s.awaitTriggerProfileSchedule(t)
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			cmdID := c.Cmd.CmdID
			status := syncml.CmdStatusOK
			device.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID.Value,
				Cmd:     new(c.Verb),
				Data:    &status,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		_, err = device.SendResponse()
		require.NoError(t, err)
		return cmds
	}

	checkHostsProfilesMatch := func(host *fleet.Host, wantUUIDs []string) {
		var gotUUIDs []string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT profile_uuid FROM host_mdm_windows_profiles WHERE host_uuid = ? AND operation_type = ?`
			return sqlx.SelectContext(context.Background(), q, &gotUUIDs, stmt, host.UUID, fleet.MDMOperationTypeInstall)
		})
		require.ElementsMatch(t, wantUUIDs, gotUUIDs)
	}

	checkHostDetails := func(t *testing.T, host *fleet.Host, wantProfs []string, wantStatus fleet.MDMDeliveryStatus) {
		var gotHostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &gotHostResp)
		require.NotNil(t, gotHostResp.Host.MDM.Profiles)
		var gotProfs []string
		require.Len(t, *gotHostResp.Host.MDM.Profiles, len(wantProfs))
		for _, p := range *gotHostResp.Host.MDM.Profiles {
			gotProfs = append(gotProfs, strings.Replace(p.Name, "name-", "", 1))
			require.NotNil(t, p.Status)
			require.EqualValues(t, wantStatus, *p.Status, "profile", p.Name)
			require.Equal(t, "windows", p.Platform)
			// Fleet reserved profiles (e.g., OS updates) should be screened from the host details response
			require.NotContains(t, servermdm.ListFleetReservedWindowsProfileNames(), p.Name)
		}
		require.ElementsMatch(t, wantProfs, gotProfs)
	}

	checkHostsFilteredByOSSettingsStatus := func(t *testing.T, wantHosts []string, wantStatus fleet.MDMDeliveryStatus, teamID *uint, labels ...*fleet.Label) {
		// Filtering hosts by OS settings reads the maintained host_mdm_windows_profiles_status rollup, same as the profiles
		// summary. This test simulates device reports by writing host_mdm_windows_profiles directly, bypassing the write paths
		// that maintain the rollup, so reconcile it before reading.
		require.NoError(t, s.ds.ReconcileWindowsProfilesStatus(t.Context()))

		var teamFilter string
		if teamID != nil {
			teamFilter = fmt.Sprintf("&team_id=%d", *teamID)
		}
		var gotHostsResp listHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?os_settings=%s%s", wantStatus, teamFilter), nil, http.StatusOK, &gotHostsResp)
		require.NotNil(t, gotHostsResp.Hosts)
		var gotHosts []string
		for _, h := range gotHostsResp.Hosts {
			gotHosts = append(gotHosts, h.Hostname)
		}
		require.ElementsMatch(t, wantHosts, gotHosts)

		var countHostsResp countHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/count?os_settings=%s%s", wantStatus, teamFilter), nil, http.StatusOK, &countHostsResp)
		require.Equal(t, len(wantHosts), countHostsResp.Count)

		for _, l := range labels {
			gotHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d/hosts?os_settings=%s%s", l.ID, wantStatus, teamFilter), nil, http.StatusOK, &gotHostsResp)
			require.NotNil(t, gotHostsResp.Hosts)
			gotHosts = []string{}
			for _, h := range gotHostsResp.Hosts {
				gotHosts = append(gotHosts, h.Hostname)
			}
			require.ElementsMatch(t, wantHosts, gotHosts, "label", l.Name)

			countHostsResp = countHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/count?label_id=%d&os_settings=%s%s", l.ID, wantStatus, teamFilter), nil, http.StatusOK, &countHostsResp)
		}
	}

	getProfileUUID := func(t *testing.T, profName string, teamID *uint) string {
		var profUUID string
		mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			var globalOrTeamID uint
			if teamID != nil {
				globalOrTeamID = *teamID
			}
			return sqlx.GetContext(ctx, tx, &profUUID, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, globalOrTeamID, profName)
		})
		require.NotNil(t, profUUID)
		return profUUID
	}

	checkHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, wantStatus fleet.MDMDeliveryStatus) {
		var gotStatus fleet.MDMDeliveryStatus
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT status FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_uuid = ?`
			err := sqlx.GetContext(context.Background(), q, &gotStatus, stmt, hostUUID, profUUID)
			return err
		})
		require.Equal(t, wantStatus, gotStatus)
	}

	// Create a host and then enroll to MDM.
	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// Trigger a profile sync (PENDING)
	s.awaitTriggerProfileSchedule(t)

	// can't resend windows configuration profiles as admin or from device endpoint while pending
	res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, globalProfiles[0]), nil, http.StatusConflict)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Configuration profiles with “pending” or “verifying” status can’t be resent")
	deviceToken := "windows-device-token"
	createDeviceTokenForHost(t, s.ds, host.ID, deviceToken)
	res = s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/configuration_profiles/%s/resend", deviceToken, globalProfiles[0]), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Configuration profiles with “pending” or “verifying” status can’t be resent")

	// trigger a profile sync
	verifyProfiles(mdmDevice, 3, false)
	checkHostsProfilesMatch(host, globalProfiles)
	checkHostDetails(t, host, globalProfiles, fleet.MDMDeliveryVerified)

	// create new label that includes host
	label := &fleet.Label{
		Name:  t.Name() + "foo",
		Query: "select * from foo;",
	}
	label, err = s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)
	require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, host, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	// simulate osquery reporting host mdm details (host_mdm.enrolled = 1 is condition for
	// hosts filtering by os settings status and generating mdm profiles summaries)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerified, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verified: 1,
	}, nil)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// make fleet add a Windows OS Updates profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }}`), http.StatusOK, &acResp)
	osUpdatesProf := getProfileUUID(t, servermdm.FleetWindowsOSUpdatesProfileName, nil)

	// os updates is sent via a profiles commands
	verifyProfiles(mdmDevice, 1, false)
	checkHostsProfilesMatch(host, append(globalProfiles, osUpdatesProf))
	// but is hidden from host details response
	checkHostDetails(t, host, globalProfiles, fleet.MDMDeliveryVerified)

	// os updates profile status doesn't matter for filtered hosts results or summaries
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryVerified)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerified, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verified: 1,
	}, nil)
	// force os updates profile to failed, doesn't impact filtered hosts results or summaries
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, osUpdatesProf)
		return err
	})
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerified, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verified: 1,
	}, nil)
	// force another profile to failed, does impact filtered hosts results and summaries
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, globalProfiles[0])
		return err
	})
	checkHostProfileStatus(t, host.UUID, globalProfiles[0], fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{}, fleet.MDMDeliveryVerified, nil, label)            // expect no hosts
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryFailed, nil, label) // expect host
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Failed:   1,
		Verified: 0,
	}, nil)

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{host.ID}))
	require.NoError(t, err)

	// trigger a profile sync, device gets the team profile (+ delete commands for old global + OS updates profiles).
	// Delete commands are individual <Delete> per LocURI: 3 global profiles × 1 LocURI + 1 OS updates × 6 LocURIs = 9.
	verifyProfiles(mdmDevice, 2, false, 9)
	checkHostsProfilesMatch(host, teamProfiles)
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerified)

	// set new team profiles (delete + addition)
	oldTeamProfile := teamProfiles[1]
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, oldTeamProfile)
		return err
	})
	// Also clean up the host-profile row for the deleted config profile
	// since the direct SQL deletion bypasses cancelWindowsHostInstallsForDeletedMDMProfiles.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), `DELETE FROM host_mdm_windows_profiles WHERE profile_uuid = ?`, oldTeamProfile)
		return err
	})
	teamProfiles = []string{
		teamProfiles[0],
		mysqltest.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}

	// trigger a profile sync, device gets the team profile
	verifyProfiles(mdmDevice, 1, false)

	// check that we deleted the old profile in the DB
	checkHostsProfilesMatch(host, teamProfiles)
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerified)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// set new team profiles (delete + addition)
	oldTeamProfile = teamProfiles[1]
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, oldTeamProfile)
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), `DELETE FROM host_mdm_windows_profiles WHERE profile_uuid = ?`, oldTeamProfile)
		return err
	})
	teamProfiles = []string{
		teamProfiles[0],
		mysqltest.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}
	// trigger a profile sync, this time fail the delivery
	verifyProfiles(mdmDevice, 1, true)

	// check that we deleted the old profile in the DB
	checkHostsProfilesMatch(host, teamProfiles)

	// a second sync gets the profile again, because of delivery retries.
	// Succeed that one
	verifyProfiles(mdmDevice, 1, false)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// make fleet add a Windows OS Updates profile
	tmResp := teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }}`), http.StatusOK, &tmResp)
	osUpdatesProf = getProfileUUID(t, servermdm.FleetWindowsOSUpdatesProfileName, &tm.ID)

	// os updates is sent via a profiles commands
	verifyProfiles(mdmDevice, 1, false)
	checkHostsProfilesMatch(host, append(teamProfiles, osUpdatesProf))
	// but is hidden from host details response
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerified)

	// os updates profile status doesn't matter for filtered hosts results or summaries
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryVerified)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerified, &tm.ID, label)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Verified: 1,
	}, nil)
	// force os updates profile to failed, doesn't impact filtered hosts results or summaries
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, osUpdatesProf)
		return err
	})
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerified, &tm.ID, label)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Verified: 1,
	}, nil)
	// force another profile to failed, does impact filtered hosts results and summaries
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[0])
		return err
	})
	checkHostProfileStatus(t, host.UUID, teamProfiles[0], fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{}, fleet.MDMDeliveryVerified, &tm.ID, label)            // expect no hosts
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryFailed, &tm.ID, label) // expect host
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Failed:   1,
		Verified: 0,
	}, nil)

	// Resend the failed profile. Should succeed
	s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, teamProfiles[0]), nil, http.StatusAccepted)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Pending: 1,
	}, nil)
	// Try resending, should fail since it's pending
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, teamProfiles[0]), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Configuration profiles with “pending” or “verifying” status can’t be resent")

	// Trigger a profile sync, device gets the resent profile
	verifyProfiles(mdmDevice, 1, false)

	// update to verifying - should not allow resending
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'verifying' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[0])
		return err
	})
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, teamProfiles[0]), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Configuration profiles with “pending” or “verifying” status can’t be resent")

	// trigger a profile sync, device doesn't get the profile since resend was not allowed
	verifyProfiles(mdmDevice, 0, false)

	// Update to verified, resending allowed again
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'verified' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[0])
		return err
	})
	s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, teamProfiles[0]), nil, http.StatusAccepted)

	// Trigger a profile sync, device gets the resent profile
	verifyProfiles(mdmDevice, 1, false)

	// add a macOS profile to the team
	mcUUID := "a" + uuid.NewString()
	prof := mcBytesForTest("name-"+mcUUID, "identifier-"+mcUUID, mcUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(context.Background(), stmt, mcUUID, tm.ID, "name-"+mcUUID, "identifier-"+mcUUID, prof, test.MakeTestBytes())
		return err
	})

	// trigger a profile sync, device doesn't get the macOS profile
	verifyProfiles(mdmDevice, 0, false)

	// can't resend a macOS profile to a Windows host
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Profile is not compatible with host platform")

	t.Run("team_transfer_removes_old_profiles_and_installs_new", func(t *testing.T) {
		// Test transferring a host between teams with different profiles.
		// Verifies that profiles from the old team are removed and profiles
		// from the new team are installed. This is the scenario that exercises
		// the remove query's desired-state subquery.

		// At this point, host is in tm with teamProfiles (2 profiles) verified.
		// Disable OS updates on tm to simplify profile counts.
		tmResp := teamResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": null, "grace_period_days": null} }}`), http.StatusOK, &tmResp)
		// Drain any OS updates removal commands
		drainCommands(mdmDevice)
		verifyProfiles(mdmDevice, 0, false) // drain remaining

		// Create a team with no profiles
		tmEmpty, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_empty_team"})
		require.NoError(t, err)

		// Transfer host to team with no profiles — should remove old team's profiles
		err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tmEmpty.ID, []uint{host.ID}))
		require.NoError(t, err)

		// Each profile has 1 LocURI, so 2 profiles = 2 delete commands
		verifyProfiles(mdmDevice, 0, false, 2)
		checkHostsProfilesMatch(host, nil)

		// Create a team with 3 different profiles
		tmOther, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_other_team"})
		require.NoError(t, err)
		otherTeamProfiles := []string{
			mysqltest.InsertWindowsProfileForTest(t, s.ds, tmOther.ID),
			mysqltest.InsertWindowsProfileForTest(t, s.ds, tmOther.ID),
			mysqltest.InsertWindowsProfileForTest(t, s.ds, tmOther.ID),
		}

		// Transfer host from empty team to team with 3 profiles — should install 3
		err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tmOther.ID, []uint{host.ID}))
		require.NoError(t, err)
		verifyProfiles(mdmDevice, 3, false)
		checkHostsProfilesMatch(host, otherTeamProfiles)

		// Transfer host back to original team — should remove 3 and install 2
		err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{host.ID}))
		require.NoError(t, err)
		verifyProfiles(mdmDevice, 2, false, 3)
		checkHostsProfilesMatch(host, teamProfiles)
	})

	t.Run("edit_profile_removes_locuri_sends_delete", func(t *testing.T) {
		// Editing a profile to remove a LocURI should send a <Delete> for the removed LocURI.
		// Create a profile with two LocURIs via batch-set, then edit it to have only one.
		// The device should receive a <Delete> for the removed LocURI plus an install for the updated profile.

		// First, clean up: use batch-set with only our new test profile to remove all
		// existing team profiles cleanly (this goes through the proper deletion path).
		twoLocURIProfile := `<Atomic><Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace><Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Experience/AllowCortana</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace></Atomic>`
		// Also disable OS updates to simplify
		tmResp := teamResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": null, "grace_period_days": null} }}`), http.StatusOK, &tmResp)
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: "edit-locuri-test", Contents: []byte(twoLocURIProfile)},
			}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

		// Trigger profile sync: device gets the new profile + delete commands for old profiles.
		// Drain all commands from the device without asserting exact counts (cleanup varies).
		drainCommands(mdmDevice)

		// Drain any remaining commands until the device is clean.
		verifyProfiles(mdmDevice, 0, false)

		// Now edit the profile to remove AllowCortana, keeping only AllowCamera.
		oneLocURIProfile := `<Atomic><Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace></Atomic>`
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: "edit-locuri-test", Contents: []byte(oneLocURIProfile)},
			}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

		// Trigger profile sync: device should get:
		// - 1 <Delete> for the removed AllowCortana LocURI (enqueued by the reconciler)
		// - 1 Atomic install for the updated profile (from reconciler checksum mismatch)
		// Total: 2 Status + 1 Delete + 1 Atomic = 4 commands
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		require.Len(t, cmds, 4, "expected 2 status + 1 delete + 1 install")

		var gotDelete, gotInstall bool
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			cmdID := c.Cmd.CmdID
			status := syncml.CmdStatusOK
			switch c.Verb {
			case "Delete":
				gotDelete = true
				require.Contains(t, c.Cmd.GetTargetURI(), "AllowCortana",
					"Delete should target the removed LocURI")
			case "Atomic":
				if len(c.Cmd.ReplaceCommands) > 0 {
					gotInstall = true
					// Verify the install only has AllowCamera, not AllowCortana
					var installURIs []string
					for _, rc := range c.Cmd.ReplaceCommands {
						installURIs = append(installURIs, rc.GetTargetURI())
					}
					require.Len(t, installURIs, 1)
					require.Contains(t, installURIs[0], "AllowCamera")
				}
			}
			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID.Value,
				Cmd:     ptr.String(c.Verb),
				Data:    &status,
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		require.True(t, gotDelete, "expected a Delete command for the removed LocURI")
		require.True(t, gotInstall, "expected an install command for the updated profile")

		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		require.Len(t, cmds, 1) // just the ack
	})

	t.Run("edit_profile_does_not_delete_locuri_used_by_another_profile", func(t *testing.T) {
		// If profile A has LocURIs X, Y and profile B has LocURI Y,
		// editing A to remove Y should NOT send a <Delete> for Y
		// because profile B still enforces it.

		// Set up two profiles that share a LocURI (AllowCamera).
		profileA := `<Atomic><Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace><Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Experience/AllowCortana</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace></Atomic>`
		profileB := `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>1</Data></Item></Replace>`
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: "shared-locuri-A", Contents: []byte(profileA)},
				{Name: "shared-locuri-B", Contents: []byte(profileB)},
			}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

		// Drain install commands.
		drainCommands(mdmDevice)
		verifyProfiles(mdmDevice, 0, false)

		// Edit profile A to remove AllowCamera (shared with B), keep only AllowCortana.
		profileAEdited := `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Experience/AllowCortana</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace>`
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: "shared-locuri-A", Contents: []byte(profileAEdited)},
				{Name: "shared-locuri-B", Contents: []byte(profileB)},
			}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

		// Trigger sync: should get an install for the updated A, but NO delete
		// for AllowCamera since profile B still uses it.
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)

		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			if c.Verb == "Delete" {
				require.NotContains(t, c.Cmd.GetTargetURI(), "AllowCamera",
					"should NOT delete AllowCamera because profile B still uses it")
			}
			cmdID := c.Cmd.CmdID
			status := syncml.CmdStatusOK
			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID.Value,
				Cmd:     ptr.String(c.Verb),
				Data:    &status,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		_, err = mdmDevice.SendResponse()
		require.NoError(t, err)
	})

	t.Run("delete_profile_fully_protected_removes_host_row", func(t *testing.T) {
		// If profile A and profile B enforce the same LocURI and A is deleted, there is no <Delete> to send because B still
		// enforces it. The host's row for A must still be cleaned up.
		sharedProfile := `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Privacy/AllowInputPersonalization</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>0</Data></Item></Replace>`
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: "fully-protected-A", Contents: []byte(sharedProfile)},
				{Name: "fully-protected-B", Contents: []byte(sharedProfile)},
			}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

		// Deliver both profiles so profile A has a host row to leak.
		drainCommands(mdmDevice)

		var profileAUUID string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &profileAUUID,
				`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ? AND team_id = ?`, "fully-protected-A", tm.ID)
		})
		hostProfileRowCount := func(profileUUID string) int {
			var count int
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(context.Background(), q, &count,
					`SELECT COUNT(*) FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_uuid = ?`, host.UUID, profileUUID)
			})
			return count
		}
		require.Equal(t, 1, hostProfileRowCount(profileAUUID))

		// Delete profile A by batch-setting only profile B, which keeps the shared LocURI enforced.
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: "fully-protected-B", Contents: []byte(sharedProfile)},
			}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

		// No <Delete> goes out, since profile B still enforces the LocURI.
		for _, c := range drainCommands(mdmDevice) {
			require.NotContains(t, c.Cmd.GetTargetURI(), "AllowInputPersonalization",
				"should NOT delete AllowInputPersonalization because profile B still enforces it")
		}

		// The host's row for the deleted profile must be gone even though no command was ever sent, so the deleted profile
		// stops showing up in the host's OS settings.
		require.Zero(t, hostProfileRowCount(profileAUUID))
		var gotHostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &gotHostResp)
		require.NotNil(t, gotHostResp.Host.MDM.Profiles)
		for _, p := range *gotHostResp.Host.MDM.Profiles {
			require.NotEqual(t, "fully-protected-A", p.Name, "deleted profile must not remain listed on the host")
		}
	})
}

func (s *integrationMDMTestSuite) TestApplyTeamsMDMWindowsProfiles() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	rawTeamSpec := func(mdmValue string) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{ "specs": [{ "name": %q, "mdm": %s }] }`, team.Name, mdmValue))
	}

	// set the windows custom settings fields
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{
			"windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["baz"]},
					{"path": "bar", "labels_exclude_any": ["x", "y"]}
				]
			}
		}
	`), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// check that they are returned by a GET /config
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"baz"}},
		{Path: "bar", LabelsExcludeAny: []string{"x", "y"}},
	}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch without specifying the windows custom settings fields and an unrelated
	// field, should not remove them
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`{ "enable_disk_encryption": true }`), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// check that they are returned by a GET /config
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"baz"}},
		{Path: "bar", LabelsExcludeAny: []string{"x", "y"}},
	}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, would remove
	// them but this is a dry-run
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{ "windows_settings": { "custom_settings": null } }
  `), http.StatusOK, &applyResp, "dry_run", "true")
	assert.Equal(t, map[string]uint{team.Name: team.ID}, applyResp.TeamIDsByName)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"baz"}},
		{Path: "bar", LabelsExcludeAny: []string{"x", "y"}},
	}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, removes them
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{ "windows_settings": { "custom_settings": null } }
  `), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// apply with invalid mix of labels fails
	res := s.Do("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{
			"windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["a"], "labels_include_all": ["b"]}
				]
			}
		}
	`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `For each profile, only one of "labels_include_all", "labels_include_any" or "labels" can be included.`)
}

func (s *integrationMDMTestSuite) TestWindowsProfilesWithFleetVariables() {
	t := s.T()
	ctx := t.Context()

	// Create a team for team-scoped tests
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test_windows_fleet_vars"})
	require.NoError(t, err)

	testCases := []struct {
		name            string
		profiles        []fleet.MDMProfileBatchPayload
		teamID          *uint
		wantStatus      int
		wantErrContains string
	}{
		{
			name: "HOST_UUID variable accepted for team",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "TestHostUUID",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "$FLEET_VAR_HOST_UUID"},
					}),
				},
			},
			teamID:     &tm.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "HOST_UUID variable with braces accepted",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "TestHostUUIDBraces",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "${FLEET_VAR_HOST_UUID}"},
					}),
				},
			},
			teamID:     &tm.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "unsupported variable rejected",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "TestUnsupported",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/Serial", Data: "$FLEET_VAR_HOST_FAKE"},
					}),
				},
			},
			teamID:          &tm.ID,
			wantStatus:      http.StatusUnprocessableEntity,
			wantErrContains: "Fleet variable $FLEET_VAR_HOST_FAKE is not supported in Windows profiles",
		},
		{
			name: "mixed supported and unsupported variables rejected",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "TestMixed",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "$FLEET_VAR_HOST_UUID"},
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/Email", Data: "$FLEET_VAR_BOGUS"},
					}),
				},
			},
			teamID:          &tm.ID,
			wantStatus:      http.StatusUnprocessableEntity,
			wantErrContains: "Fleet variable $FLEET_VAR_BOGUS is not supported in Windows profiles",
		},
		{
			name: "HOST_UUID variable accepted globally",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "GlobalHostUUID",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "$FLEET_VAR_HOST_UUID"},
					}),
				},
			},
			teamID:     nil, // global profile
			wantStatus: http.StatusNoContent,
		},
		{
			name: "batch with regular and variable profiles accepted",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "RegularProfile",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/Policy/Config/System/AllowLocation", Data: "1"},
					}),
				},
				{
					Name: "ProfileWithVar",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "$FLEET_VAR_HOST_UUID"},
					}),
				},
			},
			teamID:     &tm.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "multiple HOST_UUID variables in single profile accepted",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "MultipleHostUUID",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "$FLEET_VAR_HOST_UUID"},
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/BackupID", Data: "${FLEET_VAR_HOST_UUID}"},
					}),
				},
			},
			teamID:     &tm.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "unknown Fleet variable rejected",
			profiles: []fleet.MDMProfileBatchPayload{
				{
					Name: "UnknownVar",
					Contents: syncml.ForTestWithData([]syncml.TestCommand{
						{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/Policy/Config/System/SomeValue", Data: "${FLEET_VAR_UNKNOWN_VAR}"},
					}),
				},
			},
			teamID:          &tm.ID,
			wantStatus:      http.StatusUnprocessableEntity,
			wantErrContains: "Fleet variable $FLEET_VAR_UNKNOWN_VAR is not supported in Windows profiles",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response

			// Execute request with or without team_id
			if tc.teamID != nil {
				resp = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
					batchSetMDMProfilesRequest{Profiles: tc.profiles},
					tc.wantStatus,
					"team_id", fmt.Sprint(*tc.teamID))
			} else {
				resp = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
					batchSetMDMProfilesRequest{Profiles: tc.profiles},
					tc.wantStatus)
			}

			// Check error message if expected
			if tc.wantErrContains != "" {
				errMsg := extractServerErrorText(resp.Body)
				require.Contains(t, errMsg, tc.wantErrContains)
			}
		})
	}
}

func (s *integrationMDMTestSuite) TestWindowsProfilesFleetVariableSubstitution() {
	t := s.T()
	ctx := context.Background()

	// Create a team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team"})
	require.NoError(t, err)

	// Create and enroll three Windows hosts (two global, one in team)
	hostGlobal1, device1 := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	hostGlobal2, device2 := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	hostTeam, deviceTeam := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// Add the team host to the team
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hostTeam.ID}))
	require.NoError(t, err)

	// Create profiles with HOST_UUID variable for global and team
	globalProfile := syncml.ForTestWithData([]syncml.TestCommand{
		{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/UserSCEP_/SCEP/HostID", Data: "Device ID: $FLEET_VAR_HOST_UUID"},
	})
	teamProfile := syncml.ForTestWithData([]syncml.TestCommand{
		{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/TeamDevice/ID", Data: "Team Device: ${FLEET_VAR_HOST_UUID}"},
	})

	// Upload global profile
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "GlobalProfileWithVar", Contents: globalProfile},
		}},
		http.StatusNoContent)

	// Upload team profile
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "TeamProfileWithVar", Contents: teamProfile},
		}},
		http.StatusNoContent,
		"team_id", fmt.Sprint(tm.ID))

	// Helper to verify profile contains substituted UUID
	verifyProfileSubstitution := func(device *mdmtest.TestWindowsMDMClient, expectedUUID string, expectedData string) {
		s.awaitTriggerProfileSchedule(t)
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)

		// Find the Atomic command containing the profile
		var foundProfile bool
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)

		for _, cmd := range cmds {
			// Send status response for each command
			device.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmd.Cmd.CmdID.Value,
				Cmd:     ptr.String(cmd.Verb),
				Data:    ptr.String(syncml.CmdStatusOK),
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})

			if cmd.Verb == "Atomic" {
				// Check if the command contains our expected data with UUID substituted
				for _, replaceCmd := range cmd.Cmd.ReplaceCommands {
					for _, item := range replaceCmd.Items {
						if item.Data != nil && item.Data.Content != "" {
							if strings.Contains(item.Data.Content, expectedData) {
								// Verify the UUID was substituted correctly
								require.Contains(t, item.Data.Content, expectedUUID)
								require.NotContains(t, item.Data.Content, "$FLEET_VAR_HOST_UUID")
								require.NotContains(t, item.Data.Content, "${FLEET_VAR_HOST_UUID}")
								foundProfile = true
							}
						}
					}
				}
			}
		}
		require.True(t, foundProfile, "Expected profile with UUID substitution not found")

		// Send the response to complete the session
		cmds, err = device.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
	}

	// Verify global hosts receive profile with their UUID substituted
	verifyProfileSubstitution(device1, hostGlobal1.UUID, "Device ID: "+hostGlobal1.UUID)
	verifyProfileSubstitution(device2, hostGlobal2.UUID, "Device ID: "+hostGlobal2.UUID)

	// Verify team host receives team profile with UUID substituted
	verifyProfileSubstitution(deviceTeam, hostTeam.UUID, "Team Device: "+hostTeam.UUID)

	// Check that profile statuses are updated correctly in the database
	checkHostProfileStatus := func(hostUUID string, profileName string, expectedStatus fleet.MDMDeliveryStatus) {
		profiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, hostUUID)
		require.NoError(t, err)

		// Find the specific profile by name
		var foundProfile *fleet.HostMDMWindowsProfile
		for _, p := range profiles {
			if p.Name == profileName {
				foundProfile = &p
				break
			}
		}
		require.NotNil(t, foundProfile, "Profile %s not found for host %s", profileName, hostUUID)
		require.NotNil(t, foundProfile.Status, "Profile %s status is nil for host %s", profileName, hostUUID)
		assert.Equal(t, expectedStatus, *foundProfile.Status, "Profile %s has unexpected status for host %s", profileName, hostUUID)
	}

	checkHostProfileStatus(hostGlobal1.UUID, "GlobalProfileWithVar", fleet.MDMDeliveryVerified)
	checkHostProfileStatus(hostGlobal2.UUID, "GlobalProfileWithVar", fleet.MDMDeliveryVerified)
	checkHostProfileStatus(hostTeam.UUID, "TeamProfileWithVar", fleet.MDMDeliveryVerified)

	// Now let's check profile verification
	// Also create and test a host without Fleet variables to ensure normal verification still works
	hostNoVars, deviceNoVars := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// Create a profile without variables
	profileNoVars := syncml.ForTestWithData([]syncml.TestCommand{
		{Verb: "Replace", LocURI: "./Device/Vendor/MSFT/DMClient/Provider/ProviderID/Static/Value", Data: "Static Value: NoSubstitution"},
	})

	// Upload profile without variables
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "ProfileNoVars", Contents: profileNoVars},
		}},
		http.StatusNoContent)

	// Let the host get the profile
	s.awaitTriggerProfileSchedule(t)
	cmds, err := deviceNoVars.StartManagementSession()
	require.NoError(t, err)
	msgID, err := deviceNoVars.GetCurrentMsgID()
	require.NoError(t, err)
	for _, cmd := range cmds {
		deviceNoVars.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &cmd.Cmd.CmdID.Value,
			Cmd:     ptr.String(cmd.Verb),
			Data:    ptr.String(syncml.CmdStatusOK),
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
	}
	cmds, err = deviceNoVars.SendResponse()
	require.NoError(t, err)
	require.Len(t, cmds, 1) // ack

	checkHostProfileStatus(hostNoVars.UUID, "ProfileNoVars", fleet.MDMDeliveryVerified)

	// To ensure any verification failures result in retry (pending) status instead of staying as verifying,
	// we need to be outside the grace period. The grace period check is:
	// hostDetailUpdatedAt.Before(profileEarliestInstallDate.Add(1 hour))
	//
	// We need to make sure the host checked in recently (detail_updated_at = now)
	// but the profiles are old (created more than 1 hour ago)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// Set profile timestamps to 2 hours ago
		// IMPORTANT: uploaded_at is used as EarliestInstallDate in verification logic
		_, err := q.ExecContext(ctx,
			`UPDATE mdm_windows_configuration_profiles
			 SET created_at = DATE_SUB(NOW(), INTERVAL 2 HOUR),
			     uploaded_at = DATE_SUB(NOW(), INTERVAL 2 HOUR)
			 WHERE name IN (?, ?, ?)`,
			"GlobalProfileWithVar", "TeamProfileWithVar", "ProfileNoVars")
		if err != nil {
			return err
		}
		// Also update the host profile associations to have old timestamps (only created_at)
		_, err = q.ExecContext(ctx,
			`UPDATE host_mdm_windows_profiles
			 SET created_at = DATE_SUB(NOW(), INTERVAL 2 HOUR)
			 WHERE host_uuid IN (?, ?, ?, ?)`,
			hostGlobal1.UUID, hostGlobal2.UUID, hostTeam.UUID, hostNoVars.UUID)
		if err != nil {
			return err
		}
		// Set host detail_updated_at to now (recent check-in)
		_, err = q.ExecContext(ctx, `UPDATE hosts SET detail_updated_at = NOW() WHERE id IN (?, ?, ?, ?)`,
			hostGlobal1.ID, hostGlobal2.ID, hostTeam.ID, hostNoVars.ID)
		return err
	})

	// Normal profile should be verified successfully
	checkHostProfileStatus(hostNoVars.UUID, "ProfileNoVars", fleet.MDMDeliveryVerified)

	// Team host has TeamProfileWithVar which now correctly verifies with Fleet variables
	// The fix has been implemented and the profile should be verified successfully
	checkHostProfileStatus(hostTeam.UUID, "TeamProfileWithVar", fleet.MDMDeliveryVerified)

	// Hit the host details API and check the status in the mdm.profiles section
	// Verify team host
	var hostRespTeam getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostTeam.ID), getHostRequest{}, http.StatusOK, &hostRespTeam)
	require.NotNil(t, hostRespTeam.Host.MDM.Profiles)
	require.Len(t, *hostRespTeam.Host.MDM.Profiles, 1)
	require.Equal(t, "TeamProfileWithVar", (*hostRespTeam.Host.MDM.Profiles)[0].Name)
	require.EqualValues(t, fleet.MDMDeliveryVerified, *(*hostRespTeam.Host.MDM.Profiles)[0].Status,
		"Profile should be verified in host details API for team host")

	// Verify no-vars host
	var hostRespNoVars getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostNoVars.ID), getHostRequest{}, http.StatusOK, &hostRespNoVars)
	require.NotNil(t, hostRespNoVars.Host.MDM.Profiles)
	require.Len(t, *hostRespNoVars.Host.MDM.Profiles, 1)
	require.Equal(t, "ProfileNoVars", (*hostRespNoVars.Host.MDM.Profiles)[0].Name)
	require.EqualValues(t, fleet.MDMDeliveryVerified, *(*hostRespNoVars.Host.MDM.Profiles)[0].Status,
		"Profile should be verified in host details API for no-vars host")
}

func (s *integrationMDMTestSuite) TestWindowsProfileRetry() {
	t := s.T()
	ctx := t.Context()

	// Create a host and enroll it in MDM
	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	t.Run("Command gets retried with Replace after 418", func(t *testing.T) {
		profilePayload := syncml.ForTestWithData([]syncml.TestCommand{
			{Verb: "Add", LocURI: "./Device/Vendor/MSFT/Policy/Config/System/AllowLocation", Data: "1"},
		})
		profilePayloadNonAtomic := syncml.ForTestWithDataNonAtomic([]syncml.TestCommand{
			{Verb: "Add", LocURI: "./Device/Vendor/MSFT/Policy/Config/System/AtomicLocation", Data: "1"},
		})

		profileName := "RetryProfile"
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: profileName, Contents: profilePayload},
				{Name: profileName + "NonAtomic", Contents: profilePayloadNonAtomic},
			}},
			http.StatusNoContent)

		expectRetry := func(profileName string, expectedRetries int) {
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				var retryCount int
				err := sqlx.GetContext(t.Context(), q, &retryCount,
					`SELECT retries FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_name = ?`,
					host.UUID, profileName)
				require.NoError(t, err)
				require.Equal(t, expectedRetries, retryCount, "Unexpected retry count for profile %s", profileName)
				return nil
			})
		}

		// Trigger profile schedule
		s.awaitTriggerProfileSchedule(t)

		// Get initial host profile
		profiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
		require.NoError(t, err)
		var initialProfile fleet.HostMDMWindowsProfile
		for _, p := range profiles {
			if p.Name == profileName {
				initialProfile = p
				break
			}
		}
		require.NotNil(t, initialProfile)

		require.Equal(t, fleet.MDMDeliveryPending, *initialProfile.Status)

		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, cmd := range cmds {
			if cmd.Verb == "Status" {
				continue
			}

			var syncCmd fleet.SyncMLCmd
			if cmd.Verb == "Atomic" {
				syncCmd = fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &cmd.Cmd.CmdID.Value,
					Cmd:     ptr.String(cmd.Verb),
					Data:    ptr.String(syncml.CmdStatusAtomicFailed),
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				}
			} else {
				syncCmd = fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &cmd.Cmd.CmdID.Value,
					Cmd:     ptr.String(cmd.Verb),
					Data:    ptr.String(syncml.CmdStatusAlreadyExists),
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				}
			}
			mdmDevice.AppendResponse(syncCmd)

			for _, addCmd := range cmd.Cmd.AddCommands {
				for range addCmd.Items {
					itemCmd := fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdStatus},
						MsgRef:  &msgID,
						CmdRef:  &addCmd.CmdID.Value,
						Cmd:     ptr.String(fleet.CmdStatus),
						// 418 triggers Replace resend logic
						Data:  ptr.String(syncml.CmdStatusAlreadyExists),
						CmdID: fleet.CmdID{Value: uuid.NewString()},
					}
					mdmDevice.AppendResponse(itemCmd)
				}
			}
		}
		cmds, err = mdmDevice.SendResponse() // we have atomic replace and normal replace (resend after 418 attempt in this cmd list here)
		require.NoError(t, err)
		require.Len(t, cmds, 3) // status + atomic replace + replace

		// After initial 418 resend: pending, empty detail, retries = 0.
		profiles, err = s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
		require.NoError(t, err)
		var updatedProfile fleet.HostMDMWindowsProfile
		for _, p := range profiles {
			if p.Name == profileName {
				updatedProfile = p
				break
			}
		}
		require.NotNil(t, updatedProfile)
		require.Equal(t, fleet.MDMDeliveryPending, *updatedProfile.Status)
		require.Empty(t, updatedProfile.Detail)
		expectRetry(profileName, 0)

		// Second session: fail Atomic to trigger normal retry (status NULL, retries++ -> 1).
		cmds, err = mdmDevice.StartManagementSession()
		require.NoError(t, err)
		msgID, err = mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, cmd := range cmds {
			if cmd.Verb == "Status" {
				continue
			}
			syncCmd := fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmd.Cmd.CmdID.Value,
				Cmd:     ptr.String(cmd.Verb),
				Data:    ptr.String(syncml.CmdStatusAtomicFailed),
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			}
			mdmDevice.AppendResponse(syncCmd)
		}
		_, err = mdmDevice.SendResponse()
		require.NoError(t, err)

		// Verify raw DB status is NULL and retries = 1.
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var status sql.NullString
			var retries int
			err := sqlx.GetContext(t.Context(), q, &status,
				`SELECT status FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_name = ?`,
				host.UUID, profileName)
			require.NoError(t, err)
			require.False(t, status.Valid, "status should be NULL")
			err = sqlx.GetContext(t.Context(), q, &retries,
				`SELECT retries FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_name = ?`,
				host.UUID, profileName)
			require.NoError(t, err)
			require.Equal(t, 1, retries)
			return nil
		})

		// Third session: Add 418 again to requeue Replace; retries decremented back to 0.
		s.awaitTriggerProfileSchedule(t)

		cmds, err = mdmDevice.StartManagementSession()
		require.NoError(t, err)
		msgID, err = mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, cmd := range cmds {
			if cmd.Verb == "Status" {
				continue
			}
			syncCmd := fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmd.Cmd.CmdID.Value,
				Cmd:     ptr.String(cmd.Verb),
				Data:    ptr.String(syncml.CmdStatusAtomicFailed),
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			}
			mdmDevice.AppendResponse(syncCmd)
			for _, addCmd := range cmd.Cmd.AddCommands {
				for range addCmd.Items {
					mdmDevice.AppendResponse(fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdStatus},
						MsgRef:  &msgID,
						CmdRef:  &addCmd.CmdID.Value,
						Cmd:     ptr.String(fleet.CmdStatus),
						Data:    ptr.String(syncml.CmdStatusAlreadyExists), // 418
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
					})
				}
			}
		}
		newCmds, err := mdmDevice.SendResponse()
		require.NoError(t, err)
		require.Len(t, newCmds, 2) // status + atomic replace

		// Pending and retries back to 0.
		profiles, err = s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
		require.NoError(t, err)
		for _, p := range profiles {
			if p.Name == profileName {
				updatedProfile = p
				break
			}
		}
		require.NotNil(t, updatedProfile)
		require.Equal(t, fleet.MDMDeliveryPending, *updatedProfile.Status)
		require.Empty(t, updatedProfile.Detail)
		expectRetry(profileName, 1)

		// Fourth session: Replace succeeds (Atomic OK + item 200) → verifying.
		s.awaitTriggerProfileSchedule(t)
		cmds, err = mdmDevice.StartManagementSession()
		require.NoError(t, err)
		msgID, err = mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, cmd := range cmds {
			syncCmd := fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmd.Cmd.CmdID.Value,
				Cmd:     ptr.String(cmd.Verb),
				Data:    ptr.String(syncml.CmdStatusOK),
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			}
			mdmDevice.AppendResponse(syncCmd)
			for _, repCmd := range cmd.Cmd.ReplaceCommands {
				for range repCmd.Items {
					mdmDevice.AppendResponse(fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdStatus},
						MsgRef:  &msgID,
						CmdRef:  &cmd.Cmd.CmdID.Value,
						Cmd:     ptr.String(fleet.CmdStatus),
						Data:    ptr.String(syncml.CmdStatusOK), // 200
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
					})
				}
			}
		}
		_, err = mdmDevice.SendResponse()
		require.NoError(t, err)

		profiles, err = s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
		require.NoError(t, err)
		for _, p := range profiles {
			if p.Name == profileName {
				updatedProfile = p
				break
			}
		}
		require.NotNil(t, updatedProfile)
		require.Equal(t, fleet.MDMDeliveryVerified, *updatedProfile.Status)
		require.Empty(t, updatedProfile.Detail)
		expectRetry(profileName, 1)
	})

	t.Run("No resend on non-retryable error", func(t *testing.T) {
		profilePayload2 := syncml.ForTestWithData([]syncml.TestCommand{
			{Verb: "Add", LocURI: "./Device/Vendor/MSFT/Policy/Config/System/AllowCamera", Data: "1"},
		})

		profileName2 := "NonRetryProfile"
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: profileName2, Contents: profilePayload2},
			}},
			http.StatusNoContent)

		// Trigger profile schedule
		s.awaitTriggerProfileSchedule(t)

		// Get initial host profile
		profiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
		require.NoError(t, err)
		var initialProfile2 fleet.HostMDMWindowsProfile
		for _, p := range profiles {
			if p.Name == profileName2 {
				initialProfile2 = p
				break
			}
		}
		require.NotNil(t, initialProfile2)
		require.Equal(t, fleet.MDMDeliveryPending, *initialProfile2.Status)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		for _, cmd := range cmds {
			if cmd.Verb == "Status" {
				continue
			}
			syncCmd := fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmd.Cmd.CmdID.Value,
				Cmd:     ptr.String(cmd.Verb),
				Data:    ptr.String(syncml.CmdStatusAtomicFailed), // Generic failure, should not retry
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			}
			mdmDevice.AppendResponse(syncCmd)
			for _, addCmd := range cmd.Cmd.AddCommands {
				for range addCmd.Items {
					itemCmd := fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdStatus},
						MsgRef:  &msgID,
						CmdRef:  &addCmd.CmdID.Value,
						Cmd:     ptr.String(fleet.CmdStatus),
						// 500 generic failure
						Data:  ptr.String(syncml.CmdStatusBadRequest),
						CmdID: fleet.CmdID{Value: uuid.NewString()},
					}
					mdmDevice.AppendResponse(itemCmd)
				}
			}
		}

		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		require.Len(t, cmds, 1) // only ack returned
	})

	t.Run("Other hosts can not get all commands", func(t *testing.T) {
		// Let's insert a command for the original host, with some random raw_command data
		commandUUID := uuid.NewString()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			commandData := `<Add><!-- CmdID generated by Fleet --><CmdID>` + commandUUID + `</CmdID><Item><Target><LocURI>./BogusLocURI</LocURI></Target><Meta><Type xmlns="syncml:metinf">text/plain</Type><Format xmlns="syncml:metinf">bool</Format></Meta><Data>true</Data></Item></Add>`
			_, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, '')`, commandUUID, commandData)
			require.NoError(t, err)
			return nil
		})

		// Create another host and enroll in MDM
		host2, mdmDevice2 := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
		// Consume host2's own enroll-time profile delivery
		s.recordWindowsHostStatus(host2, mdmDevice2)

		// Start connection, and then reply with a missing CmdRef
		cmds, err := mdmDevice2.StartManagementSession()
		require.NoError(t, err)
		msgID, err := mdmDevice2.GetCurrentMsgID()
		require.NoError(t, err)
		for _, cmd := range cmds {
			if cmd.Verb == "Status" {
				continue
			}
		}
		syncCmd := fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			Cmd:     ptr.String("Add"),
			CmdRef:  ptr.String(""),
			Data:    ptr.String(syncml.CmdStatusAlreadyExists),
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		}
		mdmDevice2.AppendResponse(syncCmd)
		cmds, err = mdmDevice2.SendResponse()
		require.NoError(t, err)
		require.Len(t, cmds, 1) // Only 1 status message

		for _, cmd := range cmds {
			if cmd.Verb != "Status" {
				t.Errorf("Expected only Status command, got %s", cmd.Verb)
			}
		}
	})
}
