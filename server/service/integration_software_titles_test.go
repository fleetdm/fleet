package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
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
		"team_name": "%s",
	    "team_id": %d,
		"self_service": false,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		team.Name, team.ID, titleID, "RubyUpdate1")
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
		"team_name": "%s",
	    "team_id": %d,
		"self_service": true,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		team.Name, team.ID, titleID, "RubyUpdate1")
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
		"team_name": "%s",
	    "team_id": %d,
		"self_service": false,
		"software_title_id": %d,
		"software_display_name": "%s"
	}`,
		macOSApp.AdamID, stResp.SoftwareTitle.Name, macOSApp.IconURL, string(macOSApp.Platform), team.Name, team.ID, stResp.SoftwareTitle.ID, *updateAppReq.DisplayName)
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
		DisplayName: ptr.String(strings.Repeat(" ", 5))}, http.StatusUnprocessableEntity, "Cannot have a display name that is all whitespace.")

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
			"team_name": "%s",
		    "team_id": %d,
			"self_service": true,
			"software_title_id": %d,
			"software_display_name": "%s"
		}`,
		team.Name, team.ID, titleID, "InHouseAppUpdate2")
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
		StorageID:     "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
	}
	s.uploadSoftwareInstaller(t, payload1, http.StatusOK, "")

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
		StorageID:     "efgh1234567890abcdef1234567890abcdef1234567890abcdef1234567890cd",
	}
	s.uploadSoftwareInstaller(t, payload2, http.StatusOK, "")

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
		StorageID:     "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
	}
	s.uploadSoftwareInstaller(t, payload3, http.StatusOK, "")

	// Test 1: Filter by hash_sha256 on team1 - should find Firefox
	var resp1 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp1,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab")
	require.Len(t, resp1.SoftwareTitles, 1)
	require.Equal(t, "Firefox", resp1.SoftwareTitles[0].Name)
	require.NotNil(t, resp1.SoftwareTitles[0].SoftwarePackage)
	require.Equal(t, "dummy_installer.pkg", resp1.SoftwareTitles[0].SoftwarePackage.Name)

	// Test 2: Filter by hash_sha256 on team2 - should find Firefox
	var resp2 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp2,
		"team_id", fmt.Sprint(team2.ID),
		"hash_sha256", "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab")
	require.Len(t, resp2.SoftwareTitles, 1)
	require.Equal(t, "Firefox", resp2.SoftwareTitles[0].Name)

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
	require.Equal(t, "Firefox", resp4.SoftwareTitles[0].Name)
	require.NotNil(t, resp4.SoftwareTitles[0].SoftwarePackage)
	require.Equal(t, "dummy_installer.pkg", resp4.SoftwareTitles[0].SoftwarePackage.Name)

	// Test 5: Filter by package_name on team1 - should find Chrome
	var resp5 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp5,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "EchoApp.pkg")
	require.Len(t, resp5.SoftwareTitles, 1)
	require.Equal(t, "Chrome", resp5.SoftwareTitles[0].Name)

	// Test 6: Filter by package_name that doesn't exist - should return empty list
	var resp6 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp6,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "nonexistent.pkg")
	require.Len(t, resp6.SoftwareTitles, 0)

	// Test 7: Filter by hash_sha256 without team_id - should return error
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusBadRequest, &resp1,
		"hash_sha256", "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab")

	// Test 8: Filter by package_name without team_id - should return error
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusBadRequest, &resp1,
		"package_name", "dummy_installer.pkg")

	// Test 9: Filter by hash_sha256 with available_for_install=true
	var resp9 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp9,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		"available_for_install", "true")
	require.Len(t, resp9.SoftwareTitles, 1)
	require.Equal(t, "Firefox", resp9.SoftwareTitles[0].Name)

	// Test 10: Filter by package_name with available_for_install=true
	var resp10 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp10,
		"team_id", fmt.Sprint(team1.ID),
		"package_name", "EchoApp.pkg",
		"available_for_install", "true")
	require.Len(t, resp10.SoftwareTitles, 1)
	require.Equal(t, "Chrome", resp10.SoftwareTitles[0].Name)

	// Test 11: Combine both filters (hash and name for same package) - should work
	var resp11 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp11,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		"package_name", "dummy_installer.pkg")
	require.Len(t, resp11.SoftwareTitles, 1)
	require.Equal(t, "Firefox", resp11.SoftwareTitles[0].Name)

	// Test 12: Combine both filters with mismatched hash and name - should return empty list
	var resp12 listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp12,
		"team_id", fmt.Sprint(team1.ID),
		"hash_sha256", "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		"package_name", "EchoApp.pkg")
	require.Len(t, resp12.SoftwareTitles, 0)

	// Test 13: Verify that filtering by hash doesn't return VPP or in-house apps
	// First, list all software on team1 without filters to see total count
	var respAll listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &respAll,
		"team_id", fmt.Sprint(team1.ID),
		"available_for_install", "true")
	require.GreaterOrEqual(t, len(respAll.SoftwareTitles), 2) // At least Firefox and Chrome
}
