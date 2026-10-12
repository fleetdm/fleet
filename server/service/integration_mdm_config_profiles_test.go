package service

// Configuration profile tests that run on the base MDM suite.
//
// Belongs here: profile error handling and removal of failed profiles, and profile metadata (name,
// description, self-service, hidden) set through single and batch endpoints.
//
// Does not belong here: the bulk of configuration profile tests (integration_mdm_profiles_test.go),
// DDM declarations (integration_mdm_ddm_test.go), and disk encryption profiles
// (integration_mdm_disk_encryption_test.go).

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestDontIgnoreAnyProfileErrors() {
	t := s.T()
	ctx := context.Background()

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.MDM.MacOSSetup = fleet.MacOSSetup{
		EnableEndUserAuthentication: false,
	}
	err = s.ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// Create a host and a couple of profiles
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}

	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// The profiles should be associated with the host we made + the standard fleet configs
	profs, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, profs, 4)

	// Acknowledge the profiles so we can mark them as verified
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetdConfigPayloadIdentifier:      {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetCARootConfigPayloadIdentifier: {Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
	}))

	// Check that the profile is marked as verified when fetching the host
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.EqualValues(t, fleet.MDMDeliveryVerified, *hm.Status)
	}

	// remove the profiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// On the host side, return errors for the two profile removal actions
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "RemoveProfile" {
			var errChain []mdm.ErrorChain
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

			if fullCmd.Command.RemoveProfile.Identifier == "I1" {
				errChain = append(errChain, mdm.ErrorChain{ErrorCode: 89, ErrorDomain: "MDMClientError", USEnglishDescription: "Profile with identifier 'I1' not found."})
			} else {
				errChain = append(errChain, mdm.ErrorChain{ErrorCode: 96, ErrorDomain: "MDMClientError", USEnglishDescription: "Cannot replace profile 'I2' because it was not installed by the MDM server."})
			}
			cmd, err = mdmDevice.Err(cmd.CommandUUID, errChain)
			require.NoError(t, err)
			continue
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// get that host - it should report "failed" for the profiles and include the error message detail.
	expectedErrs := map[string]string{
		"N1": "Failed to remove: MDMClientError (89): Profile with identifier 'I1' not found.\n",
		"N2": "Failed to remove: MDMClientError (96): Cannot replace profile 'I2' because it was not installed by the MDM server.\n",
	}
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		if wantErr, ok := expectedErrs[hm.Name]; ok {
			require.EqualValues(t, fleet.MDMDeliveryFailed, *hm.Status)
			require.Equal(t, wantErr, hm.Detail)
			continue
		}
		require.EqualValues(t, fleet.MDMDeliveryVerified, *hm.Status)
	}
}

func (s *integrationMDMTestSuite) TestRemoveFailedProfiles() {
	t := s.T()

	teamName := t.Name()
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc " + teamName,
	}

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team.ID = createTeamResp.Team.ID

	// Ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)

	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	ident := uuid.NewString()

	// simulate TTL expiration
	err := s.keyValueStore.Delete(context.Background(), fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	mdmDeviceRespond := func(device *mdmtest.TestAppleMDMClient) {
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallProfile" {
				var fullCmd micromdm.CommandPayload
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

				if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), ident) {
					var errChain []mdm.ErrorChain
					errChain = append(errChain, mdm.ErrorChain{ErrorCode: -102, ErrorDomain: "CPProfile", USEnglishDescription: "The profile is either missing some required information, or contains information in an invalid format."})
					cmd, err = device.Err(cmd.CommandUUID, errChain)
					require.NoError(t, err)
					continue
				}
			}
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", ident),
		mobileconfigForTest("N2", "I2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	mdmDeviceRespond(mdmDevice)
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
	}))

	// Do additional trigger + command fetching cycles until max retries are exhausted.
	// Each cycle: cron re-enqueues the profile, device responds with error, HandleHostMDMProfileInstallResult increments retries.
	for range servermdm.MaxAppleProfileRetries - 1 {
		s.awaitTriggerProfileSchedule(t)
		mdmDeviceRespond(mdmDevice)
	}

	// One more cycle: cron re-enqueues, device responds with error. Since retries == MaxAppleProfileRetries,
	// HandleHostMDMProfileInstallResult marks the profile as failed.
	s.awaitTriggerProfileSchedule(t)
	mdmDeviceRespond(mdmDevice)

	// Verify the non-failing profiles via osquery
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetdConfigPayloadIdentifier:      {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "dn1", InstallDate: time.Now()},
		mobileconfig.FleetCARootConfigPayloadIdentifier: {Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, DisplayName: "dn2", InstallDate: time.Now()},
	}))

	// Check that the profile is marked as failed when fetching the host
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 4)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		if hm.Name == "N1" {
			require.EqualValues(t, fleet.MDMDeliveryFailed, *hm.Status)
			continue
		}

		require.EqualValues(t, fleet.MDMDeliveryVerified, *hm.Status)
	}

	// transfer host to a team without the failed profile

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{host.ID}}, http.StatusOK, &addHostsToTeamResponse{})
	s.awaitTriggerProfileSchedule(t)

	// confirm that we remove the failed profile
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3) // This would be 4 if we hadn't deleted the profile that failed to install.
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.NotEqual(t, "N1", hm.Name)
	}

	// Test case where the profile never makes it to the host at all
	host, _ = createHostThenEnrollMDM(s.ds, s.server.URL, t)
	ident = uuid.NewString()
	err = s.keyValueStore.Delete(context.Background(), fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	globalProfiles = [][]byte{
		mobileconfigForTest("N3", ident),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3)
	var profUUID string
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.EqualValues(t, fleet.MDMDeliveryPending, *hm.Status)
		if hm.Name == "N3" {
			profUUID = hm.ProfileUUID
		}
	}

	// delete the custom profile
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", profUUID), &deleteMDMAppleConfigProfileRequest{}, http.StatusOK)
	s.awaitTriggerProfileSchedule(t)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	// Since Fleet doesn't know for sure whether profile was installed or not, it sends a remove command just in case.
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.EqualValues(t, fleet.MDMDeliveryPending, *hm.Status)
		if hm.Name == "N3" {
			assert.Equal(t, fleet.MDMOperationTypeRemove, hm.OperationType)
		}
	}
}

func (s *integrationMDMTestSuite) TestConfigProfileNameAndDescription() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamFields := func(extra map[string][]string) map[string][]string {
		fields := map[string][]string{"team_id": {fmt.Sprint(team.ID)}}
		maps.Copy(fields, extra)
		return fields
	}

	create := func(fileName string, content []byte, extra map[string][]string, wantStatus int) string {
		body, headers := generateNewProfileMultipartRequest(t, fileName, content, s.token, teamFields(extra))
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)
		defer res.Body.Close()
		if wantStatus != http.StatusOK {
			return ""
		}
		var resp newMDMConfigProfileResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		return resp.ProfileUUID
	}
	update := func(profileUUID, fileName string, content []byte, fields map[string][]string, wantStatus int) {
		body, headers := generateMultipartRequest(t, "profile", fileName, content, s.token, fields)
		res := s.DoRawWithHeaders("PATCH", "/api/latest/fleet/configuration_profiles/"+profileUUID, body.Bytes(), wantStatus, headers)
		res.Body.Close()
	}
	get := func(profileUUID string) *fleet.MDMConfigProfilePayload {
		var resp getMDMConfigProfileResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/"+profileUUID, getMDMConfigProfileRequest{}, http.StatusOK, &resp)
		return resp.MDMConfigProfilePayload
	}
	list := func() map[string]*fleet.MDMConfigProfilePayload {
		var resp listMDMConfigProfilesResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))
		byUUID := make(map[string]*fleet.MDMConfigProfilePayload, len(resp.Profiles))
		for _, p := range resp.Profiles {
			byUUID[p.ProfileUUID] = p
		}
		return byUUID
	}

	// create: explicit name and description on every type, derived
	// name otherwise
	appleUUID := create("apple.mobileconfig", mobileconfigForTest("Payload Apple", "com.test.named.apple"),
		map[string][]string{"name": {"Custom Apple"}, "description": {"Blocks the camera"}}, http.StatusOK)
	appleDerivedUUID := create("apple2.mobileconfig", mobileconfigForTest("Payload Apple 2", "com.test.derived.apple"), nil, http.StatusOK)
	winUUID := create("firewall.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Firewall/Test"),
		map[string][]string{"name": {"Custom Windows"}, "description": {"Firewall rules"}}, http.StatusOK)
	winDerivedUUID := create("bitlocker-report.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Bitlocker/Test"), nil, http.StatusOK)
	declUUID := create("decl.json", declBytesForTest("com.test.named.decl", "named"),
		map[string][]string{"name": {"Custom Declaration"}, "description": {"A declaration"}}, http.StatusOK)

	apple := get(appleUUID)
	require.Equal(t, "Custom Apple", apple.Name)
	require.Equal(t, "Blocks the camera", apple.Description)
	require.Equal(t, "Payload Apple", apple.PayloadDisplayName)

	appleDerived := get(appleDerivedUUID)
	require.Equal(t, "Payload Apple 2", appleDerived.Name)
	require.Empty(t, appleDerived.Description)
	require.Equal(t, "Payload Apple 2", appleDerived.PayloadDisplayName)

	win := get(winUUID)
	require.Equal(t, "Custom Windows", win.Name)
	require.Equal(t, "Firewall rules", win.Description)
	require.Empty(t, win.PayloadDisplayName)
	require.Equal(t, "bitlocker-report", get(winDerivedUUID).Name)

	decl := get(declUUID)
	require.Equal(t, "Custom Declaration", decl.Name)
	require.Equal(t, "A declaration", decl.Description)
	require.Empty(t, decl.PayloadDisplayName)

	listed := list()
	require.Len(t, listed, 5)
	listedApple, listedAppleDerived, listedWin, listedDecl := listed[appleUUID], listed[appleDerivedUUID], listed[winUUID], listed[declUUID]
	require.NotNil(t, listedApple)
	require.NotNil(t, listedAppleDerived)
	require.NotNil(t, listedWin)
	require.NotNil(t, listedDecl)
	require.Equal(t, "Blocks the camera", listedApple.Description)
	require.Equal(t, "Payload Apple", listedApple.PayloadDisplayName)
	require.Equal(t, "Payload Apple 2", listedAppleDerived.PayloadDisplayName)
	require.Equal(t, "Firewall rules", listedWin.Description)
	require.Empty(t, listedWin.PayloadDisplayName)
	require.Equal(t, "A declaration", listedDecl.Description)

	// create: validation
	create("blank.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Blank/Test"),
		map[string][]string{"name": {"   "}}, http.StatusUnprocessableEntity)
	// Windows validation keeps its existing 400s for long and reserved names
	create("long.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Long/Test"),
		map[string][]string{"name": {strings.Repeat("n", fleet.MaxProfileNameLength+1)}}, http.StatusBadRequest)
	create("longdesc.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/LongDesc/Test"),
		map[string][]string{"description": {strings.Repeat("d", fleet.MDMProfileMaxDescriptionLen+1)}}, http.StatusUnprocessableEntity)
	create("reserved.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Reserved/Test"),
		map[string][]string{"name": {servermdm.FleetdConfigProfileName}}, http.StatusBadRequest)
	create("secret.mobileconfig", mobileconfigForTest("Payload Secret", "com.test.secret.apple"),
		map[string][]string{"name": {"Wi-Fi $FLEET_SECRET_PSK"}}, http.StatusUnprocessableEntity)
	// same name as the Apple profile, on another platform, in the same team
	create("dup.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Dup/Test"),
		map[string][]string{"name": {"Custom Apple"}}, http.StatusConflict)

	// update: description only, nothing else moves
	update(appleUUID, "", nil, map[string][]string{"description": {"Blocks the camera for real"}}, http.StatusOK)
	updated := get(appleUUID)
	require.Equal(t, "Custom Apple", updated.Name)
	require.Equal(t, "Blocks the camera for real", updated.Description)
	require.Equal(t, apple.UploadedAt, updated.UploadedAt)
	require.Equal(t, apple.Checksum, updated.Checksum)

	// update: an empty description clears it
	update(appleUUID, "", nil, map[string][]string{"description": {""}}, http.StatusOK)
	require.Empty(t, get(appleUUID).Description)

	// update: a replacement file with another PayloadDisplayName keeps the name
	update(appleUUID, "apple.mobileconfig", mobileconfigForTest("Renamed Payload", "com.test.named.apple"), nil, http.StatusOK)
	updated = get(appleUUID)
	require.Equal(t, "Custom Apple", updated.Name)
	require.Equal(t, "Renamed Payload", updated.PayloadDisplayName)
	require.NotEqual(t, apple.Checksum, updated.Checksum)

	// update: rename a Windows profile without new content
	update(winUUID, "", nil, map[string][]string{"name": {"Renamed Windows"}}, http.StatusOK)
	renamedWin := get(winUUID)
	require.Equal(t, "Renamed Windows", renamedWin.Name)
	require.Equal(t, "Firewall rules", renamedWin.Description)
	require.Equal(t, win.UploadedAt, renamedWin.UploadedAt)

	// update: a replacement Windows file doesn't rename from the file name
	update(winUUID, "something-else.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Firewall/Test2"), nil, http.StatusOK)
	require.Equal(t, "Renamed Windows", get(winUUID).Name)

	// update: declarations can be renamed too
	update(declUUID, "", nil, map[string][]string{"name": {"Renamed Declaration"}, "description": {""}}, http.StatusOK)
	renamedDecl := get(declUUID)
	require.Equal(t, "Renamed Declaration", renamedDecl.Name)
	require.Empty(t, renamedDecl.Description)

	// update: collisions and limits
	update(winUUID, "", nil, map[string][]string{"name": {"Custom Apple"}}, http.StatusConflict)
	update(declUUID, "", nil, map[string][]string{"name": {"Custom Apple"}}, http.StatusConflict)
	update(appleUUID, "", nil, map[string][]string{"name": {" "}}, http.StatusUnprocessableEntity)
	update(appleUUID, "", nil, map[string][]string{"name": {servermdm.FleetdConfigProfileName}}, http.StatusUnprocessableEntity)

	// batch: name and description are stored, display_name still works
	// as an alias
	batch := func(profiles []fleet.BatchModifyMDMConfigProfilePayload, wantStatus int) {
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: profiles,
		}, wantStatus, "team_id", fmt.Sprint(team.ID))
	}
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		// padded like a hand-written YAML value; stored trimmed
		{Profile: mobileconfigForTest("Payload Batch", "com.test.batch.apple"), Name: "  Batch Apple ", Description: " batch apple\n"},
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), DisplayName: "Batch Windows", Description: "batch windows"},
		{Profile: declBytesForTest("com.test.batch.decl", "batch"), Name: "Batch Declaration"},
	}, http.StatusNoContent)
	listed = list()
	require.Len(t, listed, 3)
	byName := make(map[string]*fleet.MDMConfigProfilePayload, len(listed))
	for _, p := range listed {
		byName[p.Name] = p
	}
	batchApple, batchWin, batchDecl := byName["Batch Apple"], byName["Batch Windows"], byName["Batch Declaration"]
	require.NotNil(t, batchApple)
	require.NotNil(t, batchWin)
	require.NotNil(t, batchDecl)
	require.Equal(t, "batch apple", batchApple.Description)
	require.Equal(t, "Payload Batch", batchApple.PayloadDisplayName)
	require.Equal(t, "batch windows", batchWin.Description)
	require.Empty(t, batchDecl.Description)

	// batch: name and display_name must agree when both are given
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), Name: "A", DisplayName: "B"},
	}, http.StatusUnprocessableEntity)
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), Name: "Same", DisplayName: "Same"},
	}, http.StatusNoContent)
	require.Len(t, list(), 1)

	// a name of only whitespace is rejected rather than derived
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), Name: "   "},
	}, http.StatusUnprocessableEntity)

	// display_name alone doesn't rename a .mobileconfig, but it still has to
	// agree with name when both are given
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: mobileconfigForTest("Payload Alias", "com.test.batch.alias"), DisplayName: "Ignored"},
	}, http.StatusNoContent)
	listed = list()
	require.Len(t, listed, 1)
	for _, p := range listed {
		require.Equal(t, "Payload Alias", p.Name)
	}
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: mobileconfigForTest("Payload Alias", "com.test.batch.alias"), Name: "A", DisplayName: "B"},
	}, http.StatusUnprocessableEntity)

	// a PayloadDisplayName is held to the same length limit as a given name
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: mobileconfigForTest(strings.Repeat("p", fleet.MaxProfileNameLength+1), "com.test.batch.long")},
	}, http.StatusUnprocessableEntity)
}

func (s *integrationMDMTestSuite) TestConfigProfileSelfServiceAndHidden() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamID := fmt.Sprint(team.ID)

	create := func(fileName string, content []byte, fields map[string][]string, wantStatus int) string {
		fields["team_id"] = []string{teamID}
		body, headers := generateNewProfileMultipartRequest(t, fileName, content, s.token, fields)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)
		defer res.Body.Close()
		var resp newMDMConfigProfileResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		return resp.ProfileUUID
	}
	update := func(profileUUID string, fields map[string][]string, wantStatus int) {
		body, headers := generateMultipartRequest(t, "profile", "", nil, s.token, fields)
		s.DoRawWithHeaders("PATCH", "/api/latest/fleet/configuration_profiles/"+profileUUID, body.Bytes(), wantStatus, headers).Body.Close()
	}
	get := func(profileUUID string) *fleet.MDMConfigProfilePayload {
		var resp getMDMConfigProfileResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/"+profileUUID, getMDMConfigProfileRequest{}, http.StatusOK, &resp)
		return resp.MDMConfigProfilePayload
	}
	installOn := func(hostUUID, profileUUID, identifier string) {
		require.NoError(t, s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
			ProfileUUID: profileUUID, ProfileIdentifier: identifier, HostUUID: hostUUID, CommandUUID: uuid.NewString(),
			OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified, Checksum: []byte("csum"), Scope: fleet.PayloadScopeSystem,
		}}))
	}
	requireOptedIn := func(hostUUID, profileUUID string) {
		ok, err := s.ds.HasHostMDMProfileOptIn(ctx, hostUUID, profileUUID)
		require.NoError(t, err)
		require.True(t, ok)
	}
	yes := []string{"true"}

	ssUUID := create("ss.mobileconfig", mobileconfigForTest("SS", "com.test.ss"), map[string][]string{"self_service": yes}, http.StatusOK)
	winUUID := create("hidden.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Hidden/Test"), map[string][]string{"hidden": yes}, http.StatusOK)
	declUUID := create("hidden-decl.json", declBytesForTest("com.test.hidden.decl", "hidden"), map[string][]string{"hidden": yes}, http.StatusOK)
	require.True(t, get(ssUUID).SelfService)
	require.True(t, get(winUUID).Hidden)
	require.True(t, get(declUUID).Hidden)

	create("ss.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/SS/Test"), map[string][]string{"self_service": yes}, http.StatusUnprocessableEntity)
	create("ss.json", declBytesForTest("com.test.ss.decl", "ss"), map[string][]string{"self_service": yes}, http.StatusUnprocessableEntity)
	create("both.mobileconfig", mobileconfigForTest("Both", "com.test.both"), map[string][]string{"self_service": yes, "hidden": yes}, http.StatusUnprocessableEntity)
	create("bad.mobileconfig", mobileconfigForTest("Bad", "com.test.bad"), map[string][]string{"hidden": {"nope"}}, http.StatusBadRequest)
	// hidden is validated against the stored self_service
	update(ssUUID, map[string][]string{"hidden": yes}, http.StatusUnprocessableEntity)
	update(declUUID, map[string][]string{"self_service": yes}, http.StatusUnprocessableEntity)

	// flipping to self-service opts in hosts that have it, without moving uploaded_at
	forcedUUID := create("forced.mobileconfig", mobileconfigForTest("Forced", "com.test.forced"), map[string][]string{}, http.StatusOK)
	installOn("host-1", forcedUUID, "com.test.forced")
	before := get(forcedUUID)
	update(forcedUUID, map[string][]string{"self_service": yes}, http.StatusOK)
	after := get(forcedUUID)
	require.True(t, after.SelfService)
	require.Equal(t, before.UploadedAt, after.UploadedAt)
	requireOptedIn("host-1", forcedUUID)
	update(forcedUUID, map[string][]string{"description": {"kept"}}, http.StatusOK)
	require.True(t, get(forcedUUID).SelfService)

	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listResp, "team_id", teamID)
	for _, p := range listResp.Profiles {
		require.Equal(t, p.ProfileUUID == ssUUID || p.ProfileUUID == forcedUUID, p.SelfService, p.Name)
		require.Equal(t, p.ProfileUUID == winUUID || p.ProfileUUID == declUUID, p.Hidden, p.Name)
	}

	// batch: the flip seeds opt-ins too, and is declarative for the flags
	batch := func(profiles []fleet.MDMProfileBatchPayload, wantStatus int) {
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: profiles}, wantStatus, "team_id", teamID)
	}
	batch([]fleet.MDMProfileBatchPayload{{Name: "B", Contents: mobileconfigForTest("B", "com.test.b")}}, http.StatusNoContent)
	var bUUID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &bUUID, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = 'com.test.b'`)
	})
	installOn("host-2", bUUID, "com.test.b")
	batch([]fleet.MDMProfileBatchPayload{
		{Name: "B", Contents: mobileconfigForTest("B", "com.test.b"), SelfService: true},
		{Name: "W", Contents: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/W/Test"), Hidden: true},
		{Name: "A", Contents: []byte(`{"cameraDisabled": true}`), Hidden: true},
	}, http.StatusNoContent)
	require.True(t, get(bUUID).SelfService)
	requireOptedIn("host-2", bUUID)
	listResp = listMDMConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listResp, "team_id", teamID)
	require.Len(t, listResp.Profiles, 3)
	for _, p := range listResp.Profiles {
		require.Equal(t, p.Name == "W" || p.Name == "A", p.Hidden, p.Name)
	}
	batch([]fleet.MDMProfileBatchPayload{{Name: "W", Contents: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/W/Test"), SelfService: true}}, http.StatusUnprocessableEntity)
}

func (s *integrationMDMTestSuite) TestBatchModifyConfigProfilesSelfServiceAndHidden() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamID := fmt.Sprint(team.ID)

	// raw JSON so the request isn't limited to the fields the Go struct knows about
	batch := func(profiles []map[string]any, wantStatus int) {
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", map[string]any{"configuration_profiles": profiles}, wantStatus, "team_id", teamID)
	}
	batch([]map[string]any{
		{"name": "SS", "profile": mobileconfigForTest("SS", "com.test.batchmodify.ss"), "self_service": true},
		{"name": "W", "profile": syncMLForTest("./Device/Vendor/MSFT/Policy/Config/BatchModifyW/Test"), "hidden": true},
		{"name": "D", "profile": declBytesForTest("com.test.batchmodify.decl", "d"), "hidden": true},
		{"name": "Plain", "profile": mobileconfigForTest("Plain", "com.test.batchmodify.plain")},
	}, http.StatusNoContent)

	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listResp, "team_id", teamID)
	require.Len(t, listResp.Profiles, 4)
	for _, p := range listResp.Profiles {
		require.Equal(t, p.Name == "SS", p.SelfService, p.Name)
		require.Equal(t, p.Name == "W" || p.Name == "D", p.Hidden, p.Name)
	}

	// validation applies here too
	batch([]map[string]any{
		{"name": "W", "profile": syncMLForTest("./Device/Vendor/MSFT/Policy/Config/BatchModifyW/Test"), "self_service": true},
	}, http.StatusUnprocessableEntity)
}
