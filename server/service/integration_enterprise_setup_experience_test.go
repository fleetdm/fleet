package service

// Setup experience tests for the enterprise (premium) suite.
//
// Belongs here: Linux and Windows setup experience software installs (with and without Fleet
// Desktop), payload-free packages and their retries, and the orbit setup experience status
// endpoint.
//
// Does not belong here: Apple setup experience, which needs MDM
// (integration_mdm_setup_experience_test.go), and installs requested outside setup experience
// (integration_enterprise_software_installers_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestOrbitSetupExperienceStatusChecksAuthBeforeMDM() {
	t := s.T()

	// macOS host should fail with MDM not enabled and configured.
	h1 := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	var orbitRes fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *h1.OrbitNodeKey}, http.StatusBadRequest, &orbitRes,
	)

	// Linux host should not fail with MDM not enabled and configured.
	h2 := createOrbitEnrolledHost(t, "ubuntu", "h2", s.ds)
	orbitRes = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *h2.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.Empty(t, orbitRes.Results)
}

func (s *integrationEnterpriseTestSuite) TestSetupExperienceLinuxWithSoftware() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Get "Setup experience" items.
	var respGetSetupExperience getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{}, http.StatusOK, &respGetSetupExperience,
		"platform", "linux",
		"team_id", fmt.Sprint(team.ID),
	)
	require.Empty(t, respGetSetupExperience.SoftwareTitles)

	// Add a deb package to the team.
	debPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for vim.deb",
		Filename:      "vim.deb",
		Title:         "vim",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, debPackage, http.StatusOK, "")
	debVimTitleID := getSoftwareTitleID(t, s.ds, "vim", "deb_packages")
	// Add an rpm package to the team.
	rpmPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for ruby.rpm",
		Filename:      "ruby.rpm",
		Title:         "ruby",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, rpmPackage, http.StatusOK, "")
	rubyVimTitleID := getSoftwareTitleID(t, s.ds, "ruby", "rpm_packages")
	// Add a tar.gz package to the team.
	tarGzPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install script for test.tar.gz",
		UninstallScript: "uninstall script for test.tar.gz",
		Filename:        "test.tar.gz",
		Title:           "test",
		TeamID:          &team.ID,
	}
	s.uploadSoftwareInstaller(t, tarGzPackage, http.StatusOK, "")
	// (tar.gz include the extension on their title.)
	tarGzPackageTitleID := getSoftwareTitleID(t, s.ds, "test.tar.gz", "tgz_packages")

	// Configure the deb, rpm, and tar.gz packages to run as part of the setup experience for Linux hosts.
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   team.ID,
		TitleIDs: []uint{debVimTitleID, rubyVimTitleID, tarGzPackageTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		fmt.Sprintf(`{"platform": "linux", "team_id": %d, "team_name": "%s", "fleet_id": %d, "fleet_name": "%s"}`, team.ID, team.Name, team.ID, team.Name), 0)

	// Get "Setup experience" items.
	respGetSetupExperience = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software",
		getSetupExperienceSoftwareRequest{},
		http.StatusOK,
		&respGetSetupExperience,
		"platform", "linux",
		"team_id", fmt.Sprint(team.ID),
		"order_key", "id",
		"order_direction", "asc",
	)
	require.Len(t, respGetSetupExperience.SoftwareTitles, 3)
	require.Equal(t, "vim", respGetSetupExperience.SoftwareTitles[0].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.Equal(t, "ruby", respGetSetupExperience.SoftwareTitles[1].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
	require.Equal(t, "test.tar.gz", respGetSetupExperience.SoftwareTitles[2].Name) // (tar.gz include the extension on their title.)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[2].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[2].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[2].SoftwarePackage.InstallDuringSetup)

	createHost := func(hostPlatform, hostPlatformLike string) *fleet.Host {
		name := t.Name() + "-" + hostPlatform
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Minute),
			OsqueryHostID:   new(name),
			NodeKey:         new(name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.local", name),
			HardwareSerial:  uuid.New().String(),
			Platform:        hostPlatform,
			PlatformLike:    hostPlatformLike,
			TeamID:          &team.ID,
		})
		require.NoError(t, err)
		orbitKey := setOrbitEnrollment(t, host, s.ds)
		host.OrbitNodeKey = &orbitKey
		err = s.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, "fleet-desktop-token-"+hostPlatform)
		require.NoError(t, err)
		return host
	}

	user1 := test.NewUser(t, s.ds, "Alice", "alice@example.com", true)
	teamPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "foobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	ubuntuHost := createHost("ubuntu", "debian")
	fedoraHost := createHost("rhel", "rhel")

	t.Run("ubuntu-success", func(t *testing.T) {
		// Get status of the "Setup experience" for the Ubuntu host (nothing yet).
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.Empty(t, getDeviceStatusResponse.Results.Software)

		// Trigger "Setup experience" for the Ubuntu host.
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *ubuntuHost.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should not return policies because the host
		// is running the setup experience).
		s.lq.On("QueriesForHost", ubuntuHost.ID).Return(map[string]string{fmt.Sprintf("%d", ubuntuHost.ID): "SELECT 1 FROM osquery;"}, nil)
		req := getDistributedQueriesRequest{NodeKey: *ubuntuHost.NodeKey}
		var dqResp getDistributedQueriesResponse
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.NotContains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", teamPolicy.ID))

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		ubuntuHostUUID, err := fleet.HostUUIDForSetupExperience(ubuntuHost)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		executionIDs := make(map[string]string) // installer name -> install execution ID
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.Empty(t, executionIDs["vim"]) // hasn't run yet due to alphanumeric ordering
		require.NotEmpty(t, executionIDs["test.tar.gz"])

		// Record a result for test.tar.gz.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*ubuntuHost.OrbitNodeKey,
				executionIDs["test.tar.gz"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "running", getDeviceStatusResponse.Results.Software[1].Status)

		// get execution ID for vim
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}

		// Record a result for vim
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*ubuntuHost.OrbitNodeKey,
				executionIDs["vim"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should now return policies because the host is done with the setup experience).
		req = getDistributedQueriesRequest{NodeKey: *ubuntuHost.NodeKey}
		dqResp = getDistributedQueriesResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", teamPolicy.ID))
	})

	t.Run("fedora-failure", func(t *testing.T) {
		// Get status of the "Setup experience" for the Fedora host (nothing yet).
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-rhel/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.Empty(t, getDeviceStatusResponse.Results.Software)

		// Trigger "Setup experience" for the Fedora host.
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *fedoraHost.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Fedora host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-rhel/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "ruby", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should not return policies because the host
		// is running the setup experience).
		s.lq.On("QueriesForHost", fedoraHost.ID).Return(map[string]string{fmt.Sprintf("%d", fedoraHost.ID): "SELECT 1 FROM osquery;"}, nil)
		req := getDistributedQueriesRequest{NodeKey: *fedoraHost.NodeKey}
		var dqResp getDistributedQueriesResponse
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.NotContains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", teamPolicy.ID))

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		fedoraHostUUID, err := fleet.HostUUIDForSetupExperience(fedoraHost)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, fedoraHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		executionIDs := make(map[string]string) // installer name -> install execution ID
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, executionIDs["ruby"])
		require.Empty(t, executionIDs["test.tar.gz"]) // hasn't run yet due to alphanumeric ordering

		// Record a result for ruby.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*fedoraHost.OrbitNodeKey,
				executionIDs["ruby"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Fedora host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-rhel/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "ruby", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "running", getDeviceStatusResponse.Results.Software[1].Status)

		// get exec ID for test.tar.gz
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, fedoraHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}

		// Record a result for test.tar.gz.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 1,
					"install_script_output": "failed"
				}`,
				*fedoraHost.OrbitNodeKey,
				executionIDs["test.tar.gz"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Fedora host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-rhel/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "ruby", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "failure", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should now return policies because the host is done with the setup experience).
		req = getDistributedQueriesRequest{NodeKey: *fedoraHost.NodeKey}
		dqResp = getDistributedQueriesResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", teamPolicy.ID))
	})

	t.Run("ubuntu-canceled", func(t *testing.T) {
		// Trigger "Setup experience" for the Ubuntu host (should clear any items from a previous setup experience).
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *ubuntuHost.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Ubuntu host.
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[1].Status)

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		ubuntuHostUUID, err := fleet.HostUUIDForSetupExperience(ubuntuHost)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		executionIDs := make(map[string]string) // installer name -> install execution ID
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.Empty(t, executionIDs["vim"]) // hasn't run yet due to alphanumeric ordering
		require.NotEmpty(t, executionIDs["test.tar.gz"])

		// Cancel the software install for test.tar.gz.
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", ubuntuHost.ID, executionIDs["test.tar.gz"]),
			nil, http.StatusNoContent)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.Equal(t, fleet.SetupExperienceStatusFailure, getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.Equal(t, fleet.SetupExperienceStatusPending, getDeviceStatusResponse.Results.Software[1].Status)

		// Get execution ID for vim
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}

		// Record a result for vim.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*ubuntuHost.OrbitNodeKey,
				executionIDs["vim"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.Equal(t, fleet.SetupExperienceStatusFailure, getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, getDeviceStatusResponse.Results.Software[1].Status)
	})

	t.Run("ubuntu-software-edited-during-se", func(t *testing.T) {
		// Trigger "Setup experience" for the Ubuntu host (should clear any items from a previous setup experience).
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *ubuntuHost.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Ubuntu host.
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[1].Status)

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		ubuntuHostUUID, err := fleet.HostUUIDForSetupExperience(ubuntuHost)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		executionIDs := make(map[string]string) // installer name -> install execution ID
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.Empty(t, executionIDs["vim"]) // hasn't run yet due to alphanumeric ordering
		require.NotEmpty(t, executionIDs["test.tar.gz"])

		// Record a result for test.tar.gz.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*ubuntuHost.OrbitNodeKey,
				executionIDs["test.tar.gz"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.Equal(t, fleet.SetupExperienceStatusRunning, getDeviceStatusResponse.Results.Software[1].Status)

		// Modify the vim installer. The in-flight setup experience install for vim must not be cancelled.
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			SelfService:       new(true),
			InstallScript:     new("some updated install script"),
			PreInstallQuery:   new("some new pre install query"),
			PostInstallScript: new("some new post install script"),
			Filename:          "vim.deb",
			TitleID:           debVimTitleID,
			TeamID:            &team.ID,
		}, http.StatusOK, "")

		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.Equal(t, fleet.SetupExperienceStatusRunning, getDeviceStatusResponse.Results.Software[1].Status)

		// Get execution ID for vim
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, executionIDs["vim"])
		require.NotEmpty(t, executionIDs["test.tar.gz"])

		// Record a result for vim. The setup experience should pick up the result and mark vim as success.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*ubuntuHost.OrbitNodeKey,
				executionIDs["vim"],
			),
		), http.StatusNoContent)

		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "test.tar.gz", getDeviceStatusResponse.Results.Software[0].Name)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "vim", getDeviceStatusResponse.Results.Software[1].Name)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, getDeviceStatusResponse.Results.Software[1].Status)
	})

	// Transfer the Ubuntu host to "No team".
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{ubuntuHost.ID}}, http.StatusOK)

	// Add a deb package to "No team".
	debEmacsPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for emacs.deb",
		Filename:      "emacs.deb",
		Title:         "emacs",
		TeamID:        nil,
	}
	s.uploadSoftwareInstaller(t, debEmacsPackage, http.StatusOK, "")
	debEmacsTitleID := getSoftwareTitleID(t, s.ds, "emacs", "deb_packages")

	// Configure the deb package to run as part of the setup experience for Linux hosts in "No team".
	swInstallResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   0,
		TitleIDs: []uint{debEmacsTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		`{"platform": "linux", "team_id": 0, "team_name": "", "fleet_id": 0, "fleet_name": ""}`, 0)

	// Get "Setup experience" items for "No team".
	respGetSetupExperience = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software",
		getSetupExperienceSoftwareRequest{},
		http.StatusOK,
		&respGetSetupExperience,
		"platform", "linux",
		"team_id", "0",
		"order_key", "id",
		"order_direction", "asc",
	)
	require.Len(t, respGetSetupExperience.SoftwareTitles, 1)
	require.Equal(t, "emacs", respGetSetupExperience.SoftwareTitles[0].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)

	t.Run("ubuntu-no-team", func(t *testing.T) {
		// Trigger "Setup experience" for the Ubuntu host.
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *ubuntuHost.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Ubuntu host.
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 1)
		require.Equal(t, "emacs", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		ubuntuHostUUID, err := fleet.HostUUIDForSetupExperience(ubuntuHost)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.NotNil(t, results[0].HostSoftwareInstallsExecutionID)
		emacsExecutionID := *results[0].HostSoftwareInstallsExecutionID
		require.NotEmpty(t, emacsExecutionID)

		// Record a result for vim.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*ubuntuHost.OrbitNodeKey,
				emacsExecutionID,
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Ubuntu host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-ubuntu/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 1)
		require.Len(t, getDeviceStatusResponse.Results.Software, 1)
		require.Equal(t, "emacs", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
	})
}

func (s *integrationEnterpriseTestSuite) TestSetupExperienceLinuxWithSoftwareWithoutDesktop() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Add a deb package to the team.
	debPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for vim.deb",
		Filename:      "vim.deb",
		Title:         "vim",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, debPackage, http.StatusOK, "")
	debVimTitleID := getSoftwareTitleID(t, s.ds, "vim", "deb_packages")
	// Add a tar.gz package to the team.
	tarGzPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install script for test.tar.gz",
		UninstallScript: "uninstall script for test.tar.gz",
		Filename:        "test.tar.gz",
		Title:           "test",
		TeamID:          &team.ID,
	}
	s.uploadSoftwareInstaller(t, tarGzPackage, http.StatusOK, "")
	// (tar.gz include the extension on their title.)
	tarGzPackageTitleID := getSoftwareTitleID(t, s.ds, "test.tar.gz", "tgz_packages")

	// Configure the deb and tar.gz packages to run as part of the setup experience for Linux hosts.
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   team.ID,
		TitleIDs: []uint{debVimTitleID, tarGzPackageTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	createHost := func(hostPlatform, hostPlatformLike string) *fleet.Host {
		name := t.Name() + "-" + hostPlatform
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Minute),
			OsqueryHostID:   new(name),
			NodeKey:         new(name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.local", name),
			HardwareSerial:  uuid.New().String(),
			Platform:        hostPlatform,
			PlatformLike:    hostPlatformLike,
			TeamID:          &team.ID,
		})
		require.NoError(t, err)
		orbitKey := setOrbitEnrollment(t, host, s.ds)
		host.OrbitNodeKey = &orbitKey
		err = s.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, "fleet-desktop-token-"+hostPlatform)
		require.NoError(t, err)
		return host
	}

	ubuntuHost := createHost("ubuntu", "debian")

	// Get status of the "Setup experience" for the Ubuntu host (nothing yet).
	var orbitRes fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *ubuntuHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.Empty(t, orbitRes.Results.Software)

	// Trigger "Setup experience" for the Ubuntu host.
	var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
	s.DoJSON(
		"POST",
		"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
			OrbitNodeKey: *ubuntuHost.OrbitNodeKey,
		}, http.StatusOK, &orbitInitResponse,
	)
	require.NoError(t, orbitInitResponse.Err)
	require.True(t, orbitInitResponse.Result.Enabled)

	// Get status of the "Setup experience" now that it has been triggered.
	orbitRes = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *ubuntuHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.NotNil(t, orbitRes.Results)
	require.Len(t, orbitRes.Results.Software, 2)
	sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
		return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
	})
	require.Equal(t, "test.tar.gz", orbitRes.Results.Software[0].Name)
	require.EqualValues(t, "pending", orbitRes.Results.Software[0].Status)
	require.Equal(t, "vim", orbitRes.Results.Software[1].Name)
	require.EqualValues(t, "pending", orbitRes.Results.Software[1].Status)

	// The setup_experience/status endpoint doesn't return the various IDs for executions,
	// so pull it out manually
	ubuntuHostUUID, err := fleet.HostUUIDForSetupExperience(ubuntuHost)
	require.NoError(t, err)
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	executionIDs := make(map[string]string) // installer name -> install execution ID
	for _, result := range results {
		if result.HostSoftwareInstallsExecutionID != nil {
			executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
		}
	}
	require.Empty(t, executionIDs["vim"]) // hasn't run yet due to alphanumeric ordering
	require.NotEmpty(t, executionIDs["test.tar.gz"])

	// Record a result for test.tar.gz.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
			*ubuntuHost.OrbitNodeKey,
			executionIDs["test.tar.gz"],
		),
	), http.StatusNoContent)

	// Again get status of the "Setup experience" for the Ubuntu host.
	orbitRes = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *ubuntuHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.NotNil(t, orbitRes.Results)
	require.NotNil(t, orbitRes.Results)
	require.Len(t, orbitRes.Results.Software, 2)
	sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
		return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
	})
	require.Equal(t, "test.tar.gz", orbitRes.Results.Software[0].Name)
	require.EqualValues(t, "success", orbitRes.Results.Software[0].Status)
	require.Equal(t, "vim", orbitRes.Results.Software[1].Name)
	require.EqualValues(t, "running", orbitRes.Results.Software[1].Status)

	// Get execution ID for vim
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, ubuntuHostUUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		if result.HostSoftwareInstallsExecutionID != nil {
			executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
		}
	}

	// Record a result for vim.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
			*ubuntuHost.OrbitNodeKey,
			executionIDs["vim"],
		),
	), http.StatusNoContent)

	// One last time get status of the "Setup experience" for the Ubuntu host.
	orbitRes = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *ubuntuHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.NotNil(t, orbitRes.Results)
	require.NotNil(t, orbitRes.Results)
	require.Len(t, orbitRes.Results.Software, 2)
	sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
		return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
	})
	require.Equal(t, "test.tar.gz", orbitRes.Results.Software[0].Name)
	require.EqualValues(t, "success", orbitRes.Results.Software[0].Status)
	require.Equal(t, "vim", orbitRes.Results.Software[1].Name)
	require.EqualValues(t, "success", orbitRes.Results.Software[1].Status)
}

func (s *integrationEnterpriseTestSuite) TestSetupExperienceWindowsWithSoftware() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Get "Setup experience" items.
	var respGetSetupExperience getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{}, http.StatusOK, &respGetSetupExperience,
		"platform", "windows",
		"team_id", fmt.Sprint(team.ID),
	)
	require.Empty(t, respGetSetupExperience.SoftwareTitles)

	// Add a msi package to the team.
	msiPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for fleet-osquery.msi",
		Filename:      "fleet-osquery.msi",
		Title:         "Fleet osquery",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, msiPackage, http.StatusOK, "")
	msiPackageTitleID := getSoftwareTitleID(t, s.ds, "Fleet osquery", "programs")
	// Add a exe installer to the team.
	exePackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install script for hello-world-installer.exe",
		UninstallScript: "uninstall script for hello-world-installer.exe", // required for .exe
		Filename:        "hello-world-installer.exe",
		Title:           "Hello world",
		TeamID:          &team.ID,
	}
	s.uploadSoftwareInstaller(t, exePackage, http.StatusOK, "")
	exePackageTitleID := getSoftwareTitleID(t, s.ds, "Hello world", "programs")

	// Configure the msi and exe packages to run as part of the setup experience for Windows hosts.
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "windows",
		TeamID:   team.ID,
		TitleIDs: []uint{msiPackageTitleID, exePackageTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		fmt.Sprintf(`{"platform": "windows", "team_id": %d, "team_name": "%s", "fleet_id": %d, "fleet_name": "%s"}`, team.ID, team.Name, team.ID, team.Name), 0)

	// Add a deb package to the Linux setup experience to test
	// that only msi and exe are queued on Windows hosts.
	debPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for vim.deb",
		Filename:      "vim.deb",
		Title:         "vim",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, debPackage, http.StatusOK, "")
	debVimTitleID := getSoftwareTitleID(t, s.ds, "vim", "deb_packages")
	swInstallResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   team.ID,
		TitleIDs: []uint{debVimTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	// Get "Setup experience" items.
	respGetSetupExperience = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software",
		getSetupExperienceSoftwareRequest{},
		http.StatusOK,
		&respGetSetupExperience,
		"platform", "windows",
		"team_id", fmt.Sprint(team.ID),
		"order_key", "id",
		"order_direction", "asc",
	)
	require.Len(t, respGetSetupExperience.SoftwareTitles, 2)
	require.Equal(t, "Fleet osquery", respGetSetupExperience.SoftwareTitles[0].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.Equal(t, "Hello world", respGetSetupExperience.SoftwareTitles[1].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)

	i := 0
	createHost := func(team *fleet.Team, hostPlatform, hostPlatformLike string) *fleet.Host {
		name := t.Name() + "-" + hostPlatform + fmt.Sprint(i)
		var teamID *uint
		if team != nil {
			teamID = &team.ID
		}
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Minute),
			OsqueryHostID:   new(name),
			NodeKey:         new(name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.local", name),
			HardwareSerial:  uuid.New().String(),
			Platform:        hostPlatform,
			PlatformLike:    hostPlatformLike,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		orbitKey := setOrbitEnrollment(t, host, s.ds)
		host.OrbitNodeKey = &orbitKey
		err = s.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, "fleet-desktop-token-"+fmt.Sprint(i))
		require.NoError(t, err)
		i++
		return host
	}

	user1 := test.NewUser(t, s.ds, "Alice", "alice@example.com", true)
	teamPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "foobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	windowsHost1 := createHost(team, "windows", "windows")

	t.Run("windows-success", func(t *testing.T) {
		// Get status of the "Setup experience" for the Windows host (nothing yet).
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-0/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.Empty(t, getDeviceStatusResponse.Results.Software)

		// Trigger "Setup experience" for the Windows host.
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *windowsHost1.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Windows host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-0/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "Hello world", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should not return policies because the host
		// is running the setup experience).
		s.lq.On("QueriesForHost", windowsHost1.ID).Return(map[string]string{fmt.Sprintf("%d", windowsHost1.ID): "SELECT 1 FROM osquery;"}, nil)
		req := getDistributedQueriesRequest{NodeKey: *windowsHost1.NodeKey}
		var dqResp getDistributedQueriesResponse
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.NotContains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", teamPolicy.ID))

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		windowsHostUUID, err := fleet.HostUUIDForSetupExperience(windowsHost1)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, windowsHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		executionIDs := make(map[string]string) // installer name -> install execution ID
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, executionIDs["Fleet osquery"])
		require.Empty(t, executionIDs["Hello world"]) // hasn't run yet due to alphanumeric ordering

		// Record a result for Fleet osquery.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*windowsHost1.OrbitNodeKey,
				executionIDs["Fleet osquery"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Windows host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-0/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "Hello world", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "running", getDeviceStatusResponse.Results.Software[1].Status)

		// Get execution ID for Hello world
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, windowsHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, executionIDs["Fleet osquery"])
		require.NotEmpty(t, executionIDs["Hello world"])

		// Record a result for Hello world.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*windowsHost1.OrbitNodeKey,
				executionIDs["Hello world"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Windows host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-0/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "Hello world", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should now return policies because the host is done with the setup experience).
		req = getDistributedQueriesRequest{NodeKey: *windowsHost1.NodeKey}
		dqResp = getDistributedQueriesResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", teamPolicy.ID))

		// Get status of the "Setup experience" via orbit (orbit will use it to mark the setup experience as done).
		var orbitRes fleet.GetOrbitSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
			fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *windowsHost1.OrbitNodeKey}, http.StatusOK, &orbitRes,
		)
		require.NotNil(t, orbitRes.Results)
		require.Len(t, orbitRes.Results.Software, 2)
		sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
			return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", orbitRes.Results.Software[0].Name)
		require.EqualValues(t, "success", orbitRes.Results.Software[0].Status)
		require.Equal(t, "Hello world", orbitRes.Results.Software[1].Name)
		require.EqualValues(t, "success", orbitRes.Results.Software[1].Status)
	})

	// Add a msi package to "No team".
	msiPackageNoTeam := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for fleet-osquery.msi",
		Filename:      "fleet-osquery.msi",
		Title:         "Fleet osquery",
		TeamID:        nil,
	}
	s.uploadSoftwareInstaller(t, msiPackageNoTeam, http.StatusOK, "")
	msiPackageTitleID = getSoftwareTitleID(t, s.ds, "Fleet osquery", "programs")
	// Add a exe installer to the team.
	exePackageNoTeam := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install script for hello-world-installer.exe",
		UninstallScript: "uninstall script for hello-world-installer.exe", // required for .exe
		Filename:        "hello-world-installer.exe",
		Title:           "Hello world",
		TeamID:          nil,
	}
	s.uploadSoftwareInstaller(t, exePackageNoTeam, http.StatusOK, "")
	exePackageTitleID = getSoftwareTitleID(t, s.ds, "Hello world", "programs")

	// Configure the msi and exe packages to run as part of the setup experience for Windows hosts in "No team".
	swInstallResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "windows",
		TeamID:   uint(0),
		TitleIDs: []uint{msiPackageTitleID, exePackageTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		`{"platform": "windows", "team_id": 0, "team_name": "", "fleet_id": 0, "fleet_name": ""}`, 0)

	// Add a deb package to the Linux setup experience to test
	// that only msi and exe are queued on Windows hosts.
	debPackageNoTeam := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for vim.deb",
		Filename:      "vim.deb",
		Title:         "vim",
		TeamID:        nil,
	}
	s.uploadSoftwareInstaller(t, debPackageNoTeam, http.StatusOK, "")
	debVimTitleID = getSoftwareTitleID(t, s.ds, "vim", "deb_packages")
	swInstallResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   uint(0),
		TitleIDs: []uint{debVimTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	// Get "Setup experience" items for "No team".
	respGetSetupExperience = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software",
		getSetupExperienceSoftwareRequest{},
		http.StatusOK,
		&respGetSetupExperience,
		"platform", "windows",
		"team_id", "0",
		"order_key", "id",
		"order_direction", "asc",
	)
	require.Len(t, respGetSetupExperience.SoftwareTitles, 2)
	require.Equal(t, "Fleet osquery", respGetSetupExperience.SoftwareTitles[0].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[0].SoftwarePackage.InstallDuringSetup)
	require.Equal(t, "Hello world", respGetSetupExperience.SoftwareTitles[1].Name)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, respGetSetupExperience.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
	require.True(t, *respGetSetupExperience.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)

	windowsHost2 := createHost(nil, "windows", "windows")

	globalPolicy, err := s.ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:  "foobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	t.Run("windows-failure-no-team", func(t *testing.T) {
		// Get status of the "Setup experience" for the Windows host (nothing yet).
		var getDeviceStatusResponse getDeviceSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-1/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.Empty(t, getDeviceStatusResponse.Results.Software)

		// Trigger "Setup experience" for the Windows host.
		var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
		s.DoJSON(
			"POST",
			"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
				OrbitNodeKey: *windowsHost2.OrbitNodeKey,
			}, http.StatusOK, &orbitInitResponse,
		)
		require.NoError(t, orbitInitResponse.Err)
		require.True(t, orbitInitResponse.Result.Enabled)

		// Get status of the "Setup experience" for the Windows host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-1/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "Hello world", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "pending", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should not return policies because the host
		// is running the setup experience).
		s.lq.On("QueriesForHost", windowsHost2.ID).Return(map[string]string{fmt.Sprintf("%d", windowsHost2.ID): "SELECT 1 FROM osquery;"}, nil)
		req := getDistributedQueriesRequest{NodeKey: *windowsHost2.NodeKey}
		var dqResp getDistributedQueriesResponse
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.NotContains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", globalPolicy.ID))

		// The setup_experience/status endpoint doesn't return the various IDs for executions,
		// so pull it out manually
		windowsHostUUID, err := fleet.HostUUIDForSetupExperience(windowsHost2)
		require.NoError(t, err)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, windowsHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		executionIDs := make(map[string]string) // installer name -> install execution ID
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, executionIDs["Fleet osquery"])
		require.Empty(t, executionIDs["Hello world"]) // hasn't run yet due to alphanumeric ordering

		// Record a failing result for Fleet osquery.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 1,
					"install_script_output": "ok"
				}`,
				*windowsHost2.OrbitNodeKey,
				executionIDs["Fleet osquery"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Windows host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-1/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "failure", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "Hello world", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "running", getDeviceStatusResponse.Results.Software[1].Status)

		// Get execution ID for Hello world
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, windowsHostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, result := range results {
			if result.HostSoftwareInstallsExecutionID != nil {
				executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, executionIDs["Fleet osquery"])
		require.NotEmpty(t, executionIDs["Hello world"])

		// Record a result for Hello world.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
				*windowsHost2.OrbitNodeKey,
				executionIDs["Hello world"],
			),
		), http.StatusNoContent)

		// Get status of the "Setup experience" for the Windows host.
		getDeviceStatusResponse = getDeviceSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/v1/fleet/device/fleet-desktop-token-1/setup_experience/status",
			getDeviceSetupExperienceStatusRequest{}, http.StatusOK, &getDeviceStatusResponse,
		)
		require.NoError(t, getDeviceStatusResponse.Err)
		require.NotNil(t, getDeviceStatusResponse.Results)
		require.Len(t, getDeviceStatusResponse.Results.Software, 2)
		sort.Slice(getDeviceStatusResponse.Results.Software, func(i, j int) bool {
			return getDeviceStatusResponse.Results.Software[i].Name < getDeviceStatusResponse.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", getDeviceStatusResponse.Results.Software[0].Name)
		require.EqualValues(t, "failure", getDeviceStatusResponse.Results.Software[0].Status)
		require.Equal(t, "Hello world", getDeviceStatusResponse.Results.Software[1].Name)
		require.EqualValues(t, "success", getDeviceStatusResponse.Results.Software[1].Status)

		// Get distributed queries for the host (should now return policies because the host is done with the setup experience).
		req = getDistributedQueriesRequest{NodeKey: *windowsHost2.NodeKey}
		dqResp = getDistributedQueriesResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
		require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_policy_query_%d", globalPolicy.ID))

		// Get status of the "Setup experience" via orbit (orbit will use it to mark the setup experience as done).
		var orbitRes fleet.GetOrbitSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
			fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *windowsHost2.OrbitNodeKey}, http.StatusOK, &orbitRes,
		)
		require.NotNil(t, orbitRes.Results)
		require.Len(t, orbitRes.Results.Software, 2)
		sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
			return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
		})
		require.Equal(t, "Fleet osquery", orbitRes.Results.Software[0].Name)
		require.EqualValues(t, "failure", orbitRes.Results.Software[0].Status)
		require.Equal(t, "Hello world", orbitRes.Results.Software[1].Name)
		require.EqualValues(t, "success", orbitRes.Results.Software[1].Status)
	})
}

func (s *integrationEnterpriseTestSuite) TestSetupExperienceWindowsWithSoftwareWithoutDesktop() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Add a msi package to the team.
	msiPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for fleet-osquery.msi",
		Filename:      "fleet-osquery.msi",
		Title:         "Fleet osquery",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, msiPackage, http.StatusOK, "")
	msiPackageTitleID := getSoftwareTitleID(t, s.ds, "Fleet osquery", "programs")
	// Add a exe installer to the team.
	exePackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install script for hello-world-installer.exe",
		UninstallScript: "uninstall script for hello-world-installer.exe", // required for .exe
		Filename:        "hello-world-installer.exe",
		Title:           "Hello world",
		TeamID:          &team.ID,
	}
	s.uploadSoftwareInstaller(t, exePackage, http.StatusOK, "")
	exePackageTitleID := getSoftwareTitleID(t, s.ds, "Hello world", "programs")
	// Add a deb package to the team.
	debPackage := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for vim.deb",
		Filename:      "vim.deb",
		Title:         "vim",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, debPackage, http.StatusOK, "")
	debVimTitleID := getSoftwareTitleID(t, s.ds, "vim", "deb_packages")

	// Configure the deb package to run as part of the setup experience for Linux hosts.
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   team.ID,
		TitleIDs: []uint{debVimTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	// Configure the msi and exe packages to run as part of the setup experience for Windows hosts.
	swInstallResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "windows",
		TeamID:   team.ID,
		TitleIDs: []uint{msiPackageTitleID, exePackageTitleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	createHost := func(hostPlatform, hostPlatformLike string) *fleet.Host {
		name := t.Name() + "-" + hostPlatform
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Minute),
			OsqueryHostID:   new(name),
			NodeKey:         new(name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.local", name),
			HardwareSerial:  uuid.New().String(),
			Platform:        hostPlatform,
			PlatformLike:    hostPlatformLike,
			TeamID:          &team.ID,
		})
		require.NoError(t, err)
		orbitKey := setOrbitEnrollment(t, host, s.ds)
		host.OrbitNodeKey = &orbitKey
		err = s.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, "fleet-desktop-token-"+hostPlatform)
		require.NoError(t, err)
		return host
	}

	windowsHost := createHost("windows", "windows")

	// Trigger "Setup experience" for the Windows host.
	var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
	s.DoJSON(
		"POST",
		"/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
			OrbitNodeKey: *windowsHost.OrbitNodeKey,
		}, http.StatusOK, &orbitInitResponse,
	)
	require.NoError(t, orbitInitResponse.Err)
	require.True(t, orbitInitResponse.Result.Enabled)

	// Get status of the "Setup experience" now that it has been triggered.
	var orbitRes fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *windowsHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.NotNil(t, orbitRes.Results)
	require.Len(t, orbitRes.Results.Software, 2)
	sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
		return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
	})
	require.Equal(t, "Fleet osquery", orbitRes.Results.Software[0].Name)
	require.EqualValues(t, "pending", orbitRes.Results.Software[0].Status)
	require.Equal(t, "Hello world", orbitRes.Results.Software[1].Name)
	require.EqualValues(t, "pending", orbitRes.Results.Software[1].Status)

	// The setup_experience/status endpoint doesn't return the various IDs for executions,
	// so pull it out manually
	windowsHostUUID, err := fleet.HostUUIDForSetupExperience(windowsHost)
	require.NoError(t, err)
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, windowsHostUUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	executionIDs := make(map[string]string) // installer name -> install execution ID
	for _, result := range results {
		if result.HostSoftwareInstallsExecutionID != nil {
			executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
		}
	}

	require.NotEmpty(t, executionIDs["Fleet osquery"]) // has started running because it's first alphabetically
	require.Empty(t, executionIDs["Hello world"])      // not running yet

	// Record a result for Fleet osquery.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
			*windowsHost.OrbitNodeKey,
			executionIDs["Fleet osquery"],
		),
	), http.StatusNoContent)

	// Again get status of the "Setup experience" for the Windows host.
	orbitRes = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *windowsHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.NotNil(t, orbitRes.Results)
	require.NotNil(t, orbitRes.Results)
	require.Len(t, orbitRes.Results.Software, 2)

	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, windowsHostUUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		if result.HostSoftwareInstallsExecutionID != nil {
			executionIDs[result.Name] = *result.HostSoftwareInstallsExecutionID
		}
	}

	require.Equal(t, "Fleet osquery", orbitRes.Results.Software[0].Name)
	require.EqualValues(t, "success", orbitRes.Results.Software[0].Status)
	require.Equal(t, "Hello world", orbitRes.Results.Software[1].Name)
	require.EqualValues(t, "running", orbitRes.Results.Software[1].Status)

	// Record a result for Hello world.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`,
			*windowsHost.OrbitNodeKey,
			executionIDs["Hello world"],
		),
	), http.StatusNoContent)

	// One last time get status of the "Setup experience" for the Windows host.
	orbitRes = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		fleet.GetOrbitSetupExperienceStatusRequest{OrbitNodeKey: *windowsHost.OrbitNodeKey}, http.StatusOK, &orbitRes,
	)
	require.NotNil(t, orbitRes.Results)
	require.NotNil(t, orbitRes.Results)
	require.Len(t, orbitRes.Results.Software, 2)
	sort.Slice(orbitRes.Results.Software, func(i, j int) bool {
		return orbitRes.Results.Software[i].Name < orbitRes.Results.Software[j].Name
	})
	require.Equal(t, "Fleet osquery", orbitRes.Results.Software[0].Name)
	require.EqualValues(t, "success", orbitRes.Results.Software[0].Status)
	require.Equal(t, "Hello world", orbitRes.Results.Software[1].Name)
	require.EqualValues(t, "success", orbitRes.Results.Software[1].Status)
}

// TestSetupExperiencePayloadFreePackageWithRetries tests that when a payload-free package
// (script package) fails during setup experience with retries enabled, only ONE activity
// is created for the final failure, not one activity per retry attempt.
//
// This test verifies the fix for https://github.com/fleetdm/fleet/issues/34818
func (s *integrationEnterpriseTestSuite) TestSetupExperiencePayloadFreePackageWithRetries() {
	t := s.T()
	ctx := context.Background()

	// Create a team for setup experience
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	// Create temporary script file for testing
	shContent := "#!/bin/bash\necho 'This will fail'\nexit 1\n"
	shFile, err := fleet.NewTempFileReader(strings.NewReader(shContent), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer shFile.Close()

	// Upload a script package (.sh file) as a software installer
	payload := &fleet.UploadSoftwareInstallerPayload{
		Filename:      "test-script.sh",
		TeamID:        &team.ID,
		InstallerFile: shFile,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	// Get the installer title ID we just created
	var titleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, &team.ID, payload.Filename)
	})
	require.NotZero(t, titleID)

	// Configure the script package to run as part of the setup experience
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "linux",
		TeamID:   team.ID,
		TitleIDs: []uint{titleID},
	}, http.StatusOK, &swInstallResp)
	require.NoError(t, swInstallResp.Err)

	// Create a Linux host for testing
	name := t.Name() + "-linux"
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   new(name),
		NodeKey:         new(name),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", name),
		HardwareSerial:  uuid.New().String(),
		Platform:        "ubuntu",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Manually create a pending software install for setup experience
	// This simulates what happens when setup experience starts
	var executionID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		execUUID := uuid.New().String()
		executionID = execUUID
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_software_installs (
				execution_id, host_id, software_installer_id, install_script_exit_code,
				pre_install_query_output, post_install_script_exit_code, post_install_script_output,
				user_id, self_service
			)
			SELECT ?, ?, id, NULL, '', NULL, '', NULL, 0
			FROM software_installers
			WHERE title_id = ? AND global_or_team_id = ?
		`, execUUID, host.ID, titleID, &team.ID)
		return err
	})
	require.NotEmpty(t, executionID)

	// Helper: count installed_software activities for this specific executionID
	countActivitiesForExec := func() int {
		acts := s.listActivities()
		cnt := 0
		for _, a := range acts {
			if a.Details == nil {
				continue
			}
			var d fleet.ActivityTypeInstalledSoftware
			if err := json.Unmarshal(*a.Details, &d); err != nil {
				continue
			}
			if d.InstallUUID == executionID {
				cnt++
			}
		}
		return cnt
	}

	// Before posting results, there should be no activity for this execution
	require.Equal(t, 0, countActivitiesForExec())

	// Simulate retry attempt #1 - FAILS with RetriesRemaining=2
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"install_script_exit_code": 1,
			"install_script_output": "Attempt 1 failed",
			"retries_remaining": 2
		}`, *host.OrbitNodeKey, executionID),
	), http.StatusNoContent)

	// Verify NO new activity was created for intermediate failure (retries_remaining=2)
	require.Equal(t, 0, countActivitiesForExec(), "Intermediate failure #1 should NOT create an activity")

	// Simulate retry attempt #2 - FAILS with RetriesRemaining=1
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"install_script_exit_code": 1,
			"install_script_output": "Attempt 2 failed",
			"retries_remaining": 1
		}`, *host.OrbitNodeKey, executionID),
	), http.StatusNoContent)

	// Verify still NO new activity was created (retries_remaining=1)
	require.Equal(t, 0, countActivitiesForExec(), "Intermediate failure #2 should NOT create an activity")

	// Simulate final attempt #3 - FAILS with RetriesRemaining=0 (no more retries)
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
		fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"install_script_exit_code": 1,
			"install_script_output": "Final attempt failed",
			"retries_remaining": 0
		}`, *host.OrbitNodeKey, executionID),
	), http.StatusNoContent)

	// Verify exactly ONE activity was created for the final failure
	require.Equal(t, 1, countActivitiesForExec(), "Final failure should create exactly ONE activity")

	// Find the software installation activity for this execution
	acts := s.listActivities()
	var installActivity *fleet.Activity
	for i := range acts {
		if acts[i].Details == nil {
			continue
		}
		var d fleet.ActivityTypeInstalledSoftware
		if json.Unmarshal(*acts[i].Details, &d) == nil && d.InstallUUID == executionID {
			installActivity = acts[i]
			break
		}
	}
	require.NotNil(t, installActivity, "Should have found an installed_software activity")

	var activityDetails fleet.ActivityTypeInstalledSoftware
	require.NoError(t, json.Unmarshal(*installActivity.Details, &activityDetails))
	require.Equal(t, host.ID, activityDetails.HostID)
	require.Equal(t, host.DisplayName(), activityDetails.HostDisplayName)
	require.Equal(t, executionID, activityDetails.InstallUUID)
	require.Equal(t, "failed_install", activityDetails.Status)
}
