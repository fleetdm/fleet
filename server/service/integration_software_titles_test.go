package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSoftwareTitleDisplayNames() {
	t := s.T()

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("team_" + t.Name())}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Enroll a host
	token := "good_token"
	host := createOrbitEnrolledHost(t, "ubuntu", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	var addResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &team.ID,
		HostIDs: []uint{host.ID},
	}, http.StatusOK, &addResp)

	s.setVPPTokenForTeam(team.ID)

	// Add a custom package
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		SelfService:   false,
		TeamID:        &team.ID,
		Platform:      "linux",
		// additional fields below are pre-populated so we can re-use the payload later for the test assertions
		Title:     "ruby",
		Version:   "1:2.5.1",
		Source:    "deb_packages",
		StorageID: "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	_, titleID := checkSoftwareInstaller(t, s.ds, payload)

	// Display name exceeds max length
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		SelfService:       ptr.Bool(true),
		InstallScript:     ptr.String("some install script"),
		PreInstallQuery:   ptr.String("some pre install query"),
		PostInstallScript: ptr.String("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           titleID,
		TeamID:            &team.ID,
		DisplayName:       strings.Repeat("a", 256),
	}, http.StatusBadRequest, "The maximum display name length is 255 characters.")

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		SelfService:       ptr.Bool(true),
		InstallScript:     ptr.String("some install script"),
		PreInstallQuery:   ptr.String("some pre install query"),
		PostInstallScript: ptr.String("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           titleID,
		TeamID:            &team.ID,
		DisplayName:       "RubyUpdate1",
	}, http.StatusOK, "")

	activityData := fmt.Sprintf(`
	{
		"software_title": "ruby",
		"software_package": "ruby.deb",
		"software_icon_url": null,
		"team_name": "%s",
	    "team_id": %d,
		"self_service": true,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		team.Name, team.ID, titleID, "RubyUpdate1")
	s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

	// Entity has display name
	stResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("RubyUpdate1", stResp.SoftwareTitle.DisplayName)

	// List software titles has display name
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))

	s.Assert().Len(resp.SoftwareTitles, 1)
	s.Assert().Equal("RubyUpdate1", resp.SoftwareTitles[0].DisplayName)

	// My device self service has display name
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw := getDeviceSoftwareResponse{}
	err := json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, getDeviceSw.Software[0].Name, "ruby")
	s.Assert().Equal("RubyUpdate1", getDeviceSw.Software[0].DisplayName)

	// Set display name to be empty
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		SelfService:       ptr.Bool(true),
		InstallScript:     ptr.String("some install script"),
		PreInstallQuery:   ptr.String("some pre install query"),
		PostInstallScript: ptr.String("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           titleID,
		TeamID:            &team.ID,
		DisplayName:       "",
	}, http.StatusOK, "")

	// Entity display name is empty
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Empty(stResp.SoftwareTitle.DisplayName)

	// List software titles display name is empty
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Len(resp.SoftwareTitles, 1)
	s.Assert().Empty(resp.SoftwareTitles[0].DisplayName)

	// My device self service has display name
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, getDeviceSw.Software[0].Name, "ruby")
	s.Assert().Empty(getDeviceSw.Software[0].DisplayName)

}
