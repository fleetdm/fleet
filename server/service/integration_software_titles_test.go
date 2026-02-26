package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSoftwareTitleDisplayNames() {
	t := s.T()
	ctx := context.Background()

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

	// =========================================
	//           CUSTOM PACKAGES
	// =========================================

	// Add a custom package
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		SelfService:   false,
		TeamID:        &team.ID,
		Platform:      "linux",
		// additional fields below are pre-populated so we can re-use the payload later for the test assertions
		Title:            "ruby",
		Version:          "1:2.5.1",
		Source:           "deb_packages",
		StorageID:        "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
		AutomaticInstall: true,
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
		DisplayName:       ptr.String(strings.Repeat("a", 256)),
	}, http.StatusBadRequest, "The maximum display name length is 255 characters.")

	// Display name can't be all whitespace
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		SelfService:       ptr.Bool(true),
		InstallScript:     ptr.String("some install script"),
		PreInstallQuery:   ptr.String("some pre install query"),
		PostInstallScript: ptr.String("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           titleID,
		TeamID:            &team.ID,
		DisplayName:       ptr.String(strings.Repeat(" ", 5)),
	}, http.StatusUnprocessableEntity, "Cannot have a display name that is all whitespace.")

	// Should update the display name even if no other fields are passed
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String("RubyUpdate1"),
	}, http.StatusOK, "")

	activityData := fmt.Sprintf(`
	{
		"software_title": "ruby",
		"software_package": "ruby.deb",
		"software_icon_url": null,
		"fleet_name": "%s",
		"team_name": "%s",
	    "fleet_id": %d,
	    "team_id": %d,
		"self_service": false,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		team.Name, team.Name, team.ID, team.ID, titleID, "RubyUpdate1")
	s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

	// Entity has display name
	stResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("RubyUpdate1", stResp.SoftwareTitle.DisplayName)
	s.Assert().Len(stResp.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies, 1)

	// Auto install policy should have the display name
	var getPolicyResp getPolicyByIDResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", stResp.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies[0].ID), getPolicyByIDRequest{}, http.StatusOK, &getPolicyResp)
	s.Assert().NotNil(getPolicyResp.Policy)
	s.Assert().Equal("RubyUpdate1", getPolicyResp.Policy.InstallSoftware.DisplayName)

	// List software titles has display name
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))

	s.Assert().Len(resp.SoftwareTitles, 1)
	s.Assert().Equal("RubyUpdate1", resp.SoftwareTitles[0].DisplayName)

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:           titleID,
		TeamID:            &team.ID,
		PostInstallScript: ptr.String("updated post install script"),
	}, http.StatusOK, "")

	// Entity has display name
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("RubyUpdate1", stResp.SoftwareTitle.DisplayName)
	s.Assert().Len(stResp.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies, 1)

	// set self service to true
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String("RubyUpdate1"),
		SelfService: ptr.Bool(true),
		Categories:  []string{"Developer tools", "Browsers"},
	}, http.StatusOK, "")

	activityData = fmt.Sprintf(`
	{
		"software_title": "ruby",
		"software_package": "ruby.deb",
		"software_icon_url": null,
		"fleet_name": "%s",
		"team_name": "%s",
	    "fleet_id": %d,
	    "team_id": %d,
		"self_service": true,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		team.Name, team.Name, team.ID, team.ID, titleID, "RubyUpdate1")
	s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

	// My device self service has display name
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw := getDeviceSoftwareResponse{}
	err := json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, getDeviceSw.Software[0].Name, "ruby")
	s.Assert().Equal("RubyUpdate1", getDeviceSw.Software[0].DisplayName)

	// Display name shows up in host software library
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	s.Assert().Len(getHostSw.Software, 1)
	s.Assert().Equal("RubyUpdate1", getHostSw.Software[0].DisplayName)

	// Set display name to be empty
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript:     ptr.String("some install script"),
		PreInstallQuery:   ptr.String("some pre install query"),
		PostInstallScript: ptr.String("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           titleID,
		TeamID:            &team.ID,
		DisplayName:       ptr.String(""),
	}, http.StatusOK, "")

	// Entity display name is empty
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Empty(stResp.SoftwareTitle.DisplayName)
	// PATCH semantics, so we shouldn't overwrite self service
	s.Assert().True(stResp.SoftwareTitle.SoftwarePackage.SelfService)
	s.Assert().ElementsMatch([]string{"Developer tools", "Browsers"}, stResp.SoftwareTitle.SoftwarePackage.Categories)

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

	// =========================================
	//           APP STORE APPS
	// =========================================

	// create MDM enrolled macOS host
	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.runWorker()

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &team.ID,
		HostIDs: []uint{mdmHost.ID},
	}, http.StatusOK, &addResp)

	macOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1/512x512.png",
		LatestVersion:    "1.0.0",
	}

	// Create a label
	clr := createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:    "foo",
			HostIDs: []uint{mdmHost.ID},
		},
	}, http.StatusOK, &clr)

	lbl1Name := clr.Label.Name

	clr = createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name: "bar",
		},
	}, http.StatusOK, &clr)

	lbl2Name := clr.Label.Name

	var addAppResp addAppStoreAppResponse
	addAppReq := &addAppStoreAppRequest{
		TeamID:           &team.ID,
		AppStoreID:       macOSApp.AdamID,
		SelfService:      true,
		LabelsIncludeAny: []string{lbl1Name, lbl2Name},
		AutomaticInstall: true,
	}

	// Now add it for real
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusOK, &addAppResp)

	macOSTitleID := addAppResp.TitleID

	// Attempt to set name to be all whitespace, should fail
	updateAppReq := &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), DisplayName: ptr.String(strings.Repeat(" ", 5))}
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusUnprocessableEntity)
	s.Assert().Contains(extractServerErrorText(res.Body), "Cannot have a display name that is all whitespace.")

	// This display name edit should succeed
	updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), DisplayName: ptr.String("MacOSAppStoreAppUpdated1")}
	var updateAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusOK, &updateAppResp)
	s.Assert().Equal(*updateAppReq.DisplayName, updateAppResp.AppStoreApp.DisplayName)

	// Entity has display name
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal(*updateAppReq.DisplayName, stResp.SoftwareTitle.DisplayName)

	// Auto install policy has display name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", stResp.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies[0].ID), getPolicyByIDRequest{}, http.StatusOK, &getPolicyResp)
	s.Assert().NotNil(getPolicyResp.Policy)
	s.Assert().Equal(*updateAppReq.DisplayName, getPolicyResp.Policy.InstallSoftware.DisplayName)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID), "query", macOSApp.Name)
	for _, a := range resp.SoftwareTitles {
		if a.ID == macOSTitleID {
			s.Assert().Equal(*updateAppReq.DisplayName, a.DisplayName)
		}
	}

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	s.Assert().Len(getHostSw.Software, 1)
	s.Assert().Equal(*updateAppReq.DisplayName, getHostSw.Software[0].DisplayName)

	// Activity has display name
	activityData = fmt.Sprintf(`
	{
		"app_store_id": "%s",
		"software_title": "%s",
		"software_icon_url": "%s",
		"platform": "%s",
		"fleet_name": "%s",
		"team_name": "%s",
	    "fleet_id": %d,
	    "team_id": %d,
		"self_service": false,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		macOSApp.AdamID, stResp.SoftwareTitle.Name, macOSApp.IconURL, string(macOSApp.Platform), team.Name, team.Name, team.ID, team.ID, stResp.SoftwareTitle.ID, *updateAppReq.DisplayName)
	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), activityData, 0)

	updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), DisplayName: ptr.String("MacOSAppStoreAppUpdated2")}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusOK, &updateAppResp)

	s.Assert().Equal(*updateAppReq.DisplayName, updateAppResp.AppStoreApp.DisplayName)

	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal(*updateAppReq.DisplayName, stResp.SoftwareTitle.DisplayName)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID), "query", macOSApp.Name)
	for _, a := range resp.SoftwareTitles {
		if a.ID == macOSTitleID {
			s.Assert().Equal(*updateAppReq.DisplayName, a.DisplayName)
		}
	}

	existingDisplayName := *updateAppReq.DisplayName

	// Omitting the field is a no-op
	updateAppReq = &updateAppStoreAppRequest{
		TeamID:      &team.ID,
		SelfService: ptr.Bool(true),
		Categories:  []string{"Developer tools", "Browsers"},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusOK, &updateAppResp)

	s.Assert().Equal(existingDisplayName, updateAppResp.AppStoreApp.DisplayName)

	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal(existingDisplayName, stResp.SoftwareTitle.DisplayName)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID), "query", macOSApp.Name)
	for _, a := range resp.SoftwareTitles {
		if a.ID == macOSTitleID {
			s.Assert().Equal(existingDisplayName, a.DisplayName)
		}
	}

	updateAppReq = &updateAppStoreAppRequest{
		TeamID:      &team.ID,
		DisplayName: ptr.String(""),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusOK, &updateAppResp)

	s.Assert().Equal(*updateAppReq.DisplayName, updateAppResp.AppStoreApp.DisplayName)

	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Empty(stResp.SoftwareTitle.DisplayName)
	// PATCH semantics, so we shouldn't overwrite self service or categories or labels
	s.Assert().True(stResp.SoftwareTitle.AppStoreApp.SelfService)
	s.Assert().ElementsMatch([]string{"Developer tools", "Browsers"}, stResp.SoftwareTitle.AppStoreApp.Categories)
	s.Assert().ElementsMatch([]string{lbl1Name, lbl2Name}, func() []string {
		var ret []string
		for _, l := range stResp.SoftwareTitle.AppStoreApp.LabelsIncludeAny {
			ret = append(ret, l.LabelName)
		}
		return ret
	}())

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID), "query", macOSApp.Name)
	for _, a := range resp.SoftwareTitles {
		if a.ID == macOSTitleID {
			s.Assert().Empty(a.DisplayName)
			// PATCH semantics, so we shouldn't overwrite self service
			s.Assert().True(*a.AppStoreApp.SelfService)
		}
	}

	// =========================================
	//           IN HOUSE APPS
	// =========================================

	// Upload in-house app for iOS, with the label as "exclude any"
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa", TeamID: &team.ID}, http.StatusOK, "")

	// Get title ID
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleID, "SELECT title_id FROM in_house_apps WHERE filename = 'ipa_test.ipa'")
	})

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String(strings.Repeat(" ", 5)),
	}, http.StatusUnprocessableEntity, "Cannot have a display name that is all whitespace.")

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String("InHouseAppUpdate"),
	}, http.StatusOK, "")

	// Entity has display name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("InHouseAppUpdate", stResp.SoftwareTitle.DisplayName)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))

	for _, t := range resp.SoftwareTitles {
		if t.ID == titleID {
			s.Assert().Equal("InHouseAppUpdate", t.DisplayName)
		}
	}

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String("InHouseAppUpdate2"),
		SelfService: ptr.Bool(true),
		Categories:  []string{"Developer tools", "Browsers"},
	}, http.StatusOK, "")

	// Entity has display name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("InHouseAppUpdate2", stResp.SoftwareTitle.DisplayName)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))

	for _, t := range resp.SoftwareTitles {
		if t.ID == titleID {
			s.Assert().Equal("InHouseAppUpdate2", t.DisplayName)
		}
	}

	activityData = fmt.Sprintf(`
		{
			"software_title": "ipa_test",
			"software_package": "ipa_test.ipa",
			"software_icon_url": null,
			"fleet_name": "%s",
			"team_name": "%s",
		    "fleet_id": %d,
		    "team_id": %d,
			"self_service": true,
			"software_title_id": %d,
			"software_display_name": "%s"
		}`,
		team.Name, team.Name, team.ID, team.ID, titleID, "InHouseAppUpdate2")
	s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

	// Omitting the field is a no-op
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: titleID,
		TeamID:  &team.ID,
	}, http.StatusOK, "")

	// Entity has display name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Equal("InHouseAppUpdate2", stResp.SoftwareTitle.DisplayName)
	// PATCH semantics, so we shouldn't overwrite self service or categories
	s.Assert().True(stResp.SoftwareTitle.SoftwarePackage.SelfService)
	s.Assert().ElementsMatch([]string{"Developer tools", "Browsers"}, stResp.SoftwareTitle.SoftwarePackage.Categories)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))

	for _, t := range resp.SoftwareTitles {
		if t.ID == titleID {
			s.Assert().Equal("InHouseAppUpdate2", t.DisplayName)
			// PATCH semantics, so we shouldn't overwrite self service
			s.Assert().True(*t.SoftwarePackage.SelfService)
		}
	}

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String(""),
	}, http.StatusOK, "")

	// Entity has display name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team.ID))
	s.Assert().Empty(stResp.SoftwareTitle.DisplayName)

	// List software titles has display name
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))

	for _, t := range resp.SoftwareTitles {
		if t.ID == titleID {
			s.Assert().Empty(t.DisplayName)
		}
	}
}

func (s *integrationMDMTestSuite) TestSoftwareTitleCustomIconsPermissions() {
	t := s.T()
	ctx := context.Background()

	user, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// add a team maintainer
	teamMaintainerUser := fleet.User{
		Name:       "test team user",
		Email:      "user1+team@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *tm,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, teamMaintainerUser.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, &teamMaintainerUser)
	require.NoError(t, err)

	// set an installer
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	_, titleID, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "hello",
		InstallerFile:    tfr1,
		StorageID:        "storage1",
		Filename:         "foo.pkg",
		Title:            "foo",
		Version:          "0.0.3",
		Source:           "apps",
		TeamID:           &tm.ID,
		UserID:           user.ID,
		BundleIdentifier: "foo.bundle.id",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// no custom icon set yet, this returns 404
	s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		nil, http.StatusNotFound)

	// get a custom icon
	iconBytes, err := os.ReadFile("testdata/icons/valid-icon.png")
	require.NoError(t, err)

	// set the custom icon, as global admin
	body, headers := generateMultipartRequest(t, "icon", "icon.png", iconBytes, s.token, nil)
	s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		body.Bytes(), http.StatusOK, headers)

	// do the next steps as team maintainer
	s.setTokenForTest(t, teamMaintainerUser.Email, test.GoodPassword)

	// list software titles on "No team" to confirm we're the team maintainer (doesn't have access)
	var listTitlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles?team_id=", listSoftwareTitlesRequest{}, http.StatusForbidden, &listTitlesResp)

	// get the custom icon
	res := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		nil, http.StatusOK)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, iconBytes, b)

	// get another custom icon and set it as team maintainer
	otherIconBytes, err := os.ReadFile("testdata/icons/other-icon.png")
	require.NoError(t, err)

	body, headers = generateMultipartRequest(t, "icon", "icon.png", otherIconBytes, s.token, nil)
	s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		body.Bytes(), http.StatusOK, headers)

	// delete the custom icon as team maintainer
	var delIconResp deleteSoftwareTitleIconResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		nil, http.StatusOK, &delIconResp)

	// get the custom icon, it is back to not found
	s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		nil, http.StatusNotFound)
}

func (s *integrationMDMTestSuite) TestListSoftwareTitlesByHashAndName() {
	t := s.T()

	// Create two teams
	var team1Resp, team2Resp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("team1_" + t.Name())}}, http.StatusOK, &team1Resp)
	team1 := team1Resp.Team
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("team2_" + t.Name())}}, http.StatusOK, &team2Resp)
	team2 := team2Resp.Team

	// Upload a software installer to team1
	payload1 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install firefox",
		Filename:      "dummy_installer.pkg",
		SelfService:   true,
		TeamID:        &team1.ID,
		Platform:      "darwin",
		Title:         "Firefox",
		Version:       "120.0",
		Source:        "apps",
	}
	s.uploadSoftwareInstaller(t, payload1, http.StatusOK, "")
	// Get the installer ID directly from the database
	var installer1ID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &installer1ID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			*payload1.TeamID, payload1.Filename)
	})
	require.NotZero(t, installer1ID)
	installer1, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), installer1ID)
	require.NoError(t, err)
	hash1 := installer1.StorageID
	// Get the actual title that was extracted from the package
	title1, err := s.ds.SoftwareTitleByID(context.Background(), *installer1.TitleID, nil, fleet.TeamFilter{})
	require.NoError(t, err)
	titleName := title1.Name

	// Upload a different software installer to team1 with different hash
	payload2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install chrome",
		Filename:      "EchoApp.pkg",
		SelfService:   false,
		TeamID:        &team1.ID,
		Platform:      "darwin",
		Title:         "Chrome",
		Version:       "120.0",
		Source:        "apps",
	}
	s.uploadSoftwareInstaller(t, payload2, http.StatusOK, "")
	// Get the installer ID and title for the second package
	var installer2ID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &installer2ID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			*payload2.TeamID, payload2.Filename)
	})
	require.NotZero(t, installer2ID)
	installer2, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), installer2ID)
	require.NoError(t, err)
	title2, err := s.ds.SoftwareTitleByID(context.Background(), *installer2.TitleID, nil, fleet.TeamFilter{})
	require.NoError(t, err)
	title2Name := title2.Name

	// Upload a software installer to team2 with same hash as payload1 (should be allowed)
	payload3 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install firefox",
		Filename:      "dummy_installer.pkg",
		SelfService:   true,
		TeamID:        &team2.ID,
		Platform:      "darwin",
		Title:         "Firefox",
		Version:       "120.0",
		Source:        "apps",
	}
	s.uploadSoftwareInstaller(t, payload3, http.StatusOK, "")

	// Test 1: Filter by hash_sha256 on team1 - should find Firefox
	var resp1 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp1,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", hash1)
	require.Len(t, resp1.SoftwareTitles, 1)
	require.Equal(t, titleName, resp1.SoftwareTitles[0].Name)
	require.NotNil(t, resp1.SoftwareTitles[0].SoftwarePackage)
	require.Equal(t, "dummy_installer.pkg", resp1.SoftwareTitles[0].SoftwarePackage.Name)

	// Test 2: Filter by hash_sha256 on team2 - should find Firefox
	var resp2 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp2,
		"team_id", fmt.Sprint(team2.ID),
		"hash_sha256", hash1)
	require.Len(t, resp2.SoftwareTitles, 1)
	require.Equal(t, titleName, resp2.SoftwareTitles[0].Name)

	// Test 3: Filter by hash_sha256 that doesn't exist - should return empty list
	var resp3 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp3,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", "nonexistent1234567890abcdef1234567890abcdef1234567890abcdef12345678")
	require.Len(t, resp3.SoftwareTitles, 0)

	// Test 4: Filter by package_name on team1 - should find Firefox
	var resp4 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp4,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "dummy_installer.pkg")
	require.Len(t, resp4.SoftwareTitles, 1)
	require.Equal(t, titleName, resp4.SoftwareTitles[0].Name)
	require.NotNil(t, resp4.SoftwareTitles[0].SoftwarePackage)
	require.Equal(t, "dummy_installer.pkg", resp4.SoftwareTitles[0].SoftwarePackage.Name)

	// Test 5: Filter by package_name on team1 - should find Chrome
	var resp5 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp5,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "EchoApp.pkg")
	require.Len(t, resp5.SoftwareTitles, 1)
	require.Equal(t, title2Name, resp5.SoftwareTitles[0].Name)

	// Test 6: Filter by package_name that doesn't exist - should return empty list
	var resp6 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp6,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "nonexistent.pkg")
	require.Len(t, resp6.SoftwareTitles, 0)

	// Test 7: Filter by hash_sha256 without team_id - should return error
	var resp7 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusUnprocessableEntity, &resp7,
		"hash_sha256", hash1)

	// Test 8: Filter by package_name without team_id - should return error
	var resp8 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusUnprocessableEntity, &resp8,
		"package_name", "dummy_installer.pkg")

	// Test 9: Filter by hash_sha256 with available_for_install=true
	var resp9 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp9,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", hash1,
		"available_for_install", "true")
	require.Len(t, resp9.SoftwareTitles, 1)
	require.Equal(t, titleName, resp9.SoftwareTitles[0].Name)

	// Test 10: Filter by package_name with available_for_install=true
	var resp10 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp10,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "EchoApp.pkg",
		"available_for_install", "true")
	require.Len(t, resp10.SoftwareTitles, 1)
	require.Equal(t, title2Name, resp10.SoftwareTitles[0].Name)

	// Test 11: Combine both filters (hash and name for same package) - should work
	var resp11 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp11,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", hash1,
		"package_name", "dummy_installer.pkg")
	require.Len(t, resp11.SoftwareTitles, 1)
	require.Equal(t, titleName, resp11.SoftwareTitles[0].Name)

	// Test 12: Combine both filters with mismatched hash and name - should return empty list
	var resp12 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp12,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", hash1,
		"package_name", "EchoApp.pkg")
	require.Len(t, resp12.SoftwareTitles, 0)

	// Test 13: Verify that filtering by hash doesn't return VPP or in-house apps
	var respAll listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &respAll,
		"team_id", fmt.Sprint(team1.ID),
		"available_for_install", "true")
	require.GreaterOrEqual(t, len(respAll.SoftwareTitles), 2) // At least the two packages we uploaded
}

func (s *integrationMDMTestSuite) TestListHostsSoftwareTitleIDFilter() {
	t := s.T()
	ctx := context.Background()

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("team_" + t.Name())}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("team_2_" + t.Name())}}, http.StatusOK, &newTeamResp)
	team2 := newTeamResp.Team

	// Enroll a host
	token := "good_token"
	host := createOrbitEnrolledHost(t, "ubuntu", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	var addResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &team.ID,
		HostIDs: []uint{host.ID},
	}, http.StatusOK, &addResp)

	// create some software for that host
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.1", Source: "deb_packages"},
	}
	_, err := s.ds.UpdateHostSoftware(ctx, host.ID, software)
	s.Require().NoError(err)

	s.Require().NoError(s.ds.SyncHostsSoftware(ctx, time.Now()))

	sw, _, err := s.ds.ListHostSoftware(ctx, host, fleet.HostSoftwareTitleListOptions{})
	s.Require().NoError(err)

	var titleID uint
	for _, s := range sw {
		if s.Name == "bar" {
			titleID = s.ID
		}
	}

	s.Require().NotZero(titleID)

	var listResp listHostsResponse
	s.DoJSON(
		"GET",
		"/api/latest/fleet/hosts",
		nil,
		http.StatusOK,
		&listResp,
		"team_id",
		fmt.Sprint(team.ID),
		"software_title_id",
		fmt.Sprint(titleID),
	)
	s.Require().Len(listResp.Hosts, 1)
	s.Assert().Equal(titleID, listResp.SoftwareTitle.ID)
	s.Assert().Equal("bar", listResp.SoftwareTitle.Name)

	// Use the other team ID, should still get a response with the name and title ID
	s.DoJSON(
		"GET",
		"/api/latest/fleet/hosts",
		nil,
		http.StatusOK,
		&listResp,
		"team_id",
		fmt.Sprint(team2.ID),
		"software_title_id",
		fmt.Sprint(titleID),
	)
	s.Require().Len(listResp.Hosts, 1)
	s.Assert().NotNil(listResp.SoftwareTitle)
	s.Assert().Equal(titleID, listResp.SoftwareTitle.ID)
	s.Assert().Equal("bar", listResp.SoftwareTitle.Name)
	v := reflect.ValueOf(*listResp.SoftwareTitle)
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name != "ID" && v.Type().Field(i).Name != "Name" {
			s.Assert().True(v.Field(i).IsZero())
		}
	}

	// Add a custom package and set a display name for the software title
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

	_, titleID = checkSoftwareInstaller(t, s.ds, payload)
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		DisplayName: ptr.String("My cool display name"),
	}, http.StatusOK, "")

	latestInstallUUID := func() string {
		var id string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT execution_id FROM upcoming_activities ORDER BY id DESC LIMIT 1`)
		})
		return id
	}

	resp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusAccepted, &resp)
	installUUID := latestInstallUUID()

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "success",
			"post_install_script_exit_code": 0,
			"post_install_script_output": "ok"
		}`, *host.OrbitNodeKey, installUUID)),
		http.StatusNoContent)

	software = append(software, fleet.Software{
		Name:    "ruby",
		Version: "1:2.5.1",
		Source:  "deb_packages",
	})
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	s.Require().NoError(err)
	s.Require().NoError(s.ds.SyncHostsSoftware(context.Background(), time.Now()))
	s.Require().NoError(s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	currToken := s.token
	t.Cleanup(func() {
		s.token = currToken
	})

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:                     ptr.String("Observer 1"),
		Email:                    ptr.String("observer@nurv.com"),
		Password:                 ptr.String(userRawPwd),
		Teams:                    ptr.T([]fleet.UserTeam{{Team: *team, Role: fleet.RoleObserver}}),
		AdminForcedPasswordReset: ptr.Bool(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	s.Assert().NotZero(createResp.User.ID)

	s.token = s.getTestToken(*params.Email, *params.Password)

	// Use the other team ID, should still get a response with the display name and title ID
	fmt.Println("before final call")
	s.DoJSON(
		"GET",
		"/api/latest/fleet/hosts",
		nil,
		http.StatusOK,
		&listResp,
		"team_id",
		fmt.Sprint(team2.ID),
		"software_title_id",
		fmt.Sprint(titleID),
	)
	s.Require().Len(listResp.Hosts, 1)
	s.Assert().NotNil(listResp.SoftwareTitle)
	s.Assert().Equal(titleID, listResp.SoftwareTitle.ID)
	s.Assert().Equal("My cool display name", listResp.SoftwareTitle.DisplayName)
}

func (s *integrationMDMTestSuite) TestGitopsInstallableSoftwareRetries() {
	t := s.T()

	newTeam := func(name string) fleet.Team {
		var resp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{
			TeamPayload: fleet.TeamPayload{Name: ptr.String(name)},
		}, http.StatusOK, &resp)
		return *resp.Team
	}

	// batchSet issues a batch-set request for the given team and software slice,
	// waits for completion, and returns the resulting packages.
	batchSet := func(team fleet.Team, software []*fleet.SoftwareInstallerPayload, shouldError bool) []fleet.SoftwarePackageResponse {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: software, TeamName: team.Name},
			http.StatusAccepted, &resp,
			"fleet_name", team.Name, "fleet_id", fmt.Sprint(team.ID),
		)
		if shouldError {
			waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team.Name, resp.RequestUUID)
			return nil
		}

		return waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)
	}

	// --- Shared per-FMA state for mock servers ---
	// Each FMA (warp, zoom) has independently mutable version/bytes/sha so the
	// manifest and installer servers can serve different content per slug.
	type fmaTestState struct {
		version        string
		installerBytes []byte
		sha256         string
	}

	computeSHA := func(b []byte) string {
		h := sha256.New()
		h.Write(b)
		return hex.EncodeToString(h.Sum(nil))
	}

	warpState := &fmaTestState{version: "1.0", installerBytes: []byte("abc")}
	warpState.sha256 = computeSHA(warpState.installerBytes)

	zoomState := &fmaTestState{version: "1.0", installerBytes: []byte("xyz")}
	zoomState.sha256 = computeSHA(zoomState.installerBytes)

	// downloadedSlugs tracks which FMA slugs were hit on the installer server
	// so individual tests can assert whether a download occurred.
	downloadedSlugs := map[string]bool{}
	var downloadMu sync.Mutex
	var hitCount int

	// Mock installer server — routes by path to serve per-FMA bytes.
	shouldError := false
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadMu.Lock()
		defer downloadMu.Unlock()
		hitCount++
		switch r.URL.Path {
		case "/cloudflare-warp.msi":

			downloadedSlugs["cloudflare-warp/windows"] = true
			_, _ = w.Write(warpState.installerBytes)
		case "/zoom.msi":
			if shouldError {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			downloadedSlugs["zoom/windows"] = true
			_, _ = w.Write(zoomState.installerBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer installerServer.Close()

	// Insert the list of maintained apps
	maintained_apps.SyncApps(t, s.ds)

	// Mock manifest server — routes by slug path and returns current per-FMA state.
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var state *fmaTestState
		var installerPath string
		switch r.URL.Path {
		case "/cloudflare-warp/windows.json":
			state = warpState
			installerPath = "/cloudflare-warp.msi"
		case "/zoom/windows.json":
			state = zoomState
			installerPath = "/zoom.msi"
		default:
			http.NotFound(w, r)
			return
		}

		versions := []*ma.FMAManifestApp{
			{
				Version:            state.version,
				Queries:            ma.FMAQueries{Exists: "SELECT 1 FROM osquery_info;"},
				InstallerURL:       installerServer.URL + installerPath,
				InstallScriptRef:   "foobaz",
				UninstallScriptRef: "foobaz",
				SHA256:             state.sha256,
				DefaultCategories:  []string{"Productivity"},
			},
		}
		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs:     map[string]string{"foobaz": "Hello World!"},
		}
		require.NoError(t, json.NewEncoder(w).Encode(manifest))
	}))
	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)
	defer dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	team := newTeam("team_" + t.Name())

	// Add an ingested app to the team
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{Slug: ptr.String("cloudflare-warp/windows"), SelfService: true},
	}

	packages := batchSet(team, softwareToInstall, false)
	require.Len(t, packages, 1)
	assert.Equal(t, 1, hitCount)
	hitCount = 0

	shouldError = true
	softwareToInstall = append(
		softwareToInstall,
		&fleet.SoftwareInstallerPayload{Slug: ptr.String("zoom/windows"), SelfService: true},
	)
	packages = batchSet(team, softwareToInstall, true)
	require.Len(t, packages, 0)
	assert.Equal(t, 3, hitCount)
}
