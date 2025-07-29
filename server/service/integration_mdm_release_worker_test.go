package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestReleaseWorker() {
	t := s.T()
	ctx := context.Background()
	mysql.TruncateTables(t, s.ds, "nano_commands") // We truncate this table beforehand to avoid persistence from other tests.

	device := godep.Device{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"}

	profileAssignmentReqs := []profileAssignmentReq{}

	s.setSkipWorkerJobs(t)
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			profileAssignmentReqs = append(profileAssignmentReqs, prof)
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// query all hosts
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Empty(t, listHostsRes.Hosts) // no hosts yet

	// trigger a profile sync
	s.runDEPSchedule()

	// all devices should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)

	t.Run("waits for config profiles installation before automatic release", func(t *testing.T) {
		config := mobileconfigForTest("N1", "I1")
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{config}}, http.StatusNoContent)
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			mysql.DumpTable(t, q, "mdm_apple_configuration_profiles")
			return nil
		})
		// enroll the host
		depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
		mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
		mdmDevice.SerialNumber = device.SerialNumber
		err := mdmDevice.Enroll()
		require.NoError(t, err)

		cmd, err := mdmDevice.Idle()
		require.Nil(t, cmd) // No command as no command is enqueued.
		require.NoError(t, err)

		// Run worker to start device release (NOTE: Should not release yet)
		s.runWorker()
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			q.ExecContext(ctx, "UPDATE jobs SET not_before = ? WHERE name = 'apple_mdm' AND state = 'queued'", time.Now())
			return nil
		})
		time.Sleep(1 * time.Second)

		// Get install enterprise application command and acknowledge it
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, cmd)

		time.Sleep(1 * time.Second)

		s.runWorker() // Run after install enterprise command to install profiles. (Should requeue until we trigger profile schedule)

		// Verify device was not released yet
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var count int
			sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM nano_commands WHERE request_type = 'DeviceConfigured'")
			require.EqualValues(t, 0, count)
			return nil
		})

		// Trigger profiles scheduler to set which profiles should be installed on the host.
		s.awaitTriggerProfileSchedule(t)
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			q.ExecContext(ctx, "UPDATE jobs SET not_before = ? WHERE name = 'apple_mdm' AND state = 'queued'", time.Now())
			return nil
		})

		// Verify install profiles three times due to the two default fleet profiles and our custom one added.
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "InstallProfile", cmd.Command.RequestType)

		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NotNil(t, cmd)
		require.Equal(t, "InstallProfile", cmd.Command.RequestType)
		require.NoError(t, err)

		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NotNil(t, cmd)
		require.Equal(t, "InstallProfile", cmd.Command.RequestType)
		require.NoError(t, err)

		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID) // No other commands now.
		require.Nil(t, cmd)
		require.NoError(t, err)

		time.Sleep(1 * time.Second) // Wait for the acks to come in.

		s.runWorker()               // release device
		time.Sleep(1 * time.Second) // Ensure we wait just a bit for state to update.

		// See DeviceConfigured is in Database and next command for mdm device
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var count int
			sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM nano_commands WHERE request_type = 'DeviceConfigured'")
			require.EqualValues(t, 1, count)
			return nil
		})
		cmd, err = mdmDevice.Idle()
		require.NotNil(t, cmd)
		require.Equal(t, "DeviceConfigured", cmd.Command.RequestType)
		require.NoError(t, err)
	})
}
