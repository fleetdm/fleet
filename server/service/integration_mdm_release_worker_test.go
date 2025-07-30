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

	expectMDMCommandsOfType := func(t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient, commandType string, count int) {
		// Get the first command
		cmd, err := mdmDevice.Idle()

		for range count {
			require.NoError(t, err)
			require.NotNil(t, cmd)
			require.Equal(t, commandType, cmd.Command.RequestType)
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		}

		// We do not expect any other commands
		require.Nil(t, cmd)
		require.NoError(t, err)
	}

	expectDeviceConfiguredSent := func(t *testing.T, shouldBeSent bool) {
		expectedCount := 0
		if shouldBeSent == true {
			expectedCount = 1
		}
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var count int
			err := sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM nano_commands WHERE request_type = 'DeviceConfigured'")
			require.NoError(t, err)
			require.EqualValues(t, expectedCount, count)
			return nil
		})
	}

	// Helper function to set the queued job not_before to current time to ensure it can be picked up without waiting.
	speedUpQueuedAppleMdmJob := func(t *testing.T) {
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE jobs SET not_before = ? WHERE name = 'apple_mdm' AND state = 'queued'", time.Now().Add(-1*time.Second))
			require.NoError(t, err)
			return nil
		})
	}

	enrollAppleDevice := func(t *testing.T, device godep.Device) *mdmtest.TestAppleMDMClient {
		depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
		mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
		mdmDevice.SerialNumber = device.SerialNumber
		err := mdmDevice.Enroll()
		require.NoError(t, err)

		cmd, err := mdmDevice.Idle()
		require.Nil(t, cmd) // check no command is enqueued.
		require.NoError(t, err)

		return mdmDevice
	}

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

		// enroll the host
		mdmDevice := enrollAppleDevice(t, device)

		// Run worker to start device release (NOTE: Should not release yet)
		s.runWorker()
		speedUpQueuedAppleMdmJob(t)

		// Get install enterprise application command and acknowledge it
		expectMDMCommandsOfType(t, mdmDevice, "InstallEnterpriseApplication", 1)

		s.runWorker() // Run after install enterprise command to install profiles. (Should requeue until we trigger profile schedule)

		// Verify device was not released yet
		expectDeviceConfiguredSent(t, false)

		// Trigger profiles scheduler to set which profiles should be installed on the host.
		s.awaitTriggerProfileSchedule(t)
		speedUpQueuedAppleMdmJob(t)

		// Verify install profiles three times due to the two default fleet profiles and our custom one added.
		expectMDMCommandsOfType(t, mdmDevice, "InstallProfile", 3)

		s.runWorker() // release device

		// See DeviceConfigured is in Database and next command for mdm device
		expectDeviceConfiguredSent(t, true)
		expectMDMCommandsOfType(t, mdmDevice, "DeviceConfigured", 1)
	})
}
