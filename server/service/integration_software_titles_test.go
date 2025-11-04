package service

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

func (s *integrationMDMTestSuite) TestSoftwareTitleDisplayNames() {
	// Create a team
	t := s.T()
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("team_" + t.Name())}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

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
	// TODO: check activities
	// activityData := fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": null,
	// "team_id": null, "self_service": true, "software_title_id": %d, "labels_include_any": [{"id": %d, "name": %q}]}`,
	// 	titleID, labelResp.Label.ID, t.Name())
	// s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "software_title_display_names")
		return nil
	})

	stResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("RubyUpdate1", stResp.SoftwareTitle.DisplayName)

}
