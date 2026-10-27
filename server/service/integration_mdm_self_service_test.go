package service

// Self-service software tests for the MDM suite.
//
// Belongs here: software categories and installing all self-service software from Fleet Desktop.
//
// Does not belong here: VPP app management (integration_mdm_vpp_test.go) and self-service
// configuration profiles (integration_mdm_profiles_crud_test.go).

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSoftwareCategories() {
	t := s.T()
	ctx := context.Background()

	// Enroll a host
	token := "good_token"
	host := createOrbitEnrolledHost(t, "ubuntu", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		Version:       "1:2.5.1",
		SelfService:   true,
	}

	// Add some categories
	cat1, err := s.ds.NewSoftwareCategory(ctx, 0, "test_category_1")
	require.NoError(t, err)

	cat2, err := s.ds.NewSoftwareCategory(ctx, 0, "test_category_2")
	require.NoError(t, err)

	cat3, err := s.ds.NewSoftwareCategory(ctx, 0, "test_category_3")
	require.NoError(t, err)

	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")
	require.NotEmpty(t, titleID)

	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw := getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, "ruby", getDeviceSw.Software[0].Name)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Equal(t, payload.Filename, getDeviceSw.Software[0].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getDeviceSw.Software[0].SoftwarePackage.Version)
	require.Empty(t, getDeviceSw.Software[0].SoftwarePackage.Categories)

	updatePayload := &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		Filename:      "ruby.deb",
		InstallScript: new("install"),
		Version:       "1:2.5.1",
		SelfService:   new(true),
	}

	// Non-existent category is silently dropped
	updatePayload.Categories = []string{"not_found"}
	s.updateSoftwareInstaller(t, updatePayload, http.StatusOK, "")
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Empty(t, getDeviceSw.Software[0].SoftwarePackage.Categories)

	// Set some real categories
	updatePayload.Categories = []string{cat1.Name, cat2.Name, cat2.Name} // duplicate category shouldn't fail
	s.updateSoftwareInstaller(t, updatePayload, http.StatusOK, "")
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, "ruby", getDeviceSw.Software[0].Name)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Equal(t, payload.Filename, getDeviceSw.Software[0].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getDeviceSw.Software[0].SoftwarePackage.Version)
	require.Len(t, getDeviceSw.Software[0].SoftwarePackage.Categories, 2)
	require.Contains(t, getDeviceSw.Software[0].SoftwarePackage.Categories, cat1.Name)
	require.Contains(t, getDeviceSw.Software[0].SoftwarePackage.Categories, cat2.Name)

	// Enqueue an install. Changing categories should not affect the status of this install
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		host.ID, titleID), nil, http.StatusAccepted, &installResp)

	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "include_available_for_install", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, "ruby", getHostSw.Software[0].Name)
	require.NotNil(t, getHostSw.Software[0].SoftwarePackage)
	require.Equal(t, "ruby.deb", getHostSw.Software[0].SoftwarePackage.Name)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getHostSw.Software[0].SoftwarePackage.SelfService)

	// Switch to a different category (should remove the other 2)
	updatePayload.Categories = []string{cat3.Name}
	s.updateSoftwareInstaller(t, updatePayload, http.StatusOK, "")

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, "ruby", getDeviceSw.Software[0].Name)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Equal(t, payload.Filename, getDeviceSw.Software[0].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getDeviceSw.Software[0].SoftwarePackage.Version)
	require.Len(t, getDeviceSw.Software[0].SoftwarePackage.Categories, 1)
	require.Equal(t, cat3.Name, getDeviceSw.Software[0].SoftwarePackage.Categories[0])

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "include_available_for_install", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, "ruby", getHostSw.Software[0].Name)
	require.NotNil(t, getHostSw.Software[0].SoftwarePackage)
	require.Equal(t, "ruby.deb", getHostSw.Software[0].SoftwarePackage.Name)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status) // status is the same as above
	require.NotNil(t, getHostSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getHostSw.Software[0].SoftwarePackage.SelfService)

	// Now add a VPP app

	orgName := "Fleet Device Management Inc."
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Get the token
	var tokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokenResp)
	require.NoError(t, tokenResp.Err)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/fleets", tokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "fleet_id", "0")
	require.NoError(t, appResp.Err)
	require.NotEmpty(t, appResp.AppStoreApps)
	addedApp := appResp.AppStoreApps[0]

	// Add the app with cat2 and cat3
	var addedMacOSApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     nil,
		Platform:   addedApp.Platform,
		AppStoreID: addedApp.AdamID,
		Categories: []string{cat2.Name, cat3.Name, cat3.Name}, // duplicate category shouldn't fail
	}, http.StatusOK, &addedMacOSApp)

	// get the title ID
	listSWTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{},
		http.StatusOK, &listSWTitlesResp, "query", addedApp.Name, "fleet_id", "0")
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	vppAppTitleID := listSWTitlesResp.SoftwareTitles[0].ID

	// get the app
	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppAppTitleID), nil, http.StatusOK, &titleResponse, "fleet_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.AppStoreApp)
	require.Len(t, titleResponse.SoftwareTitle.AppStoreApp.Categories, 2)
	require.Contains(t, titleResponse.SoftwareTitle.AppStoreApp.Categories, cat2.Name)
	require.Contains(t, titleResponse.SoftwareTitle.AppStoreApp.Categories, cat3.Name)

	// update the app and change categories
	updateAppReq := &updateAppStoreAppRequest{TeamID: nil, SelfService: new(true)}
	updateAppReq.Categories = []string{cat1.Name, cat3.Name}
	var updateAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppAppTitleID), updateAppReq, http.StatusOK, &updateAppResp)
	require.Equal(t, updateAppResp.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.True(t, updateAppResp.AppStoreApp.SelfService)
	require.Equal(t, fleet.MacOSPlatform, updateAppResp.AppStoreApp.Platform)
	require.Contains(t, updateAppResp.AppStoreApp.Categories, cat1.Name)
	require.Contains(t, updateAppResp.AppStoreApp.Categories, cat3.Name)

	// do some gitops checks
	batchURL := "/api/latest/fleet/software/app_store_apps/batch"
	var batchAssociateResponse batchAssociateAppStoreAppsResponse
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: addedApp.AdamID, SelfService: true, Categories: []string{"Developer tools", "Communication", "Communication"}}, // duplicate category shouldn't fail
			},
		}, http.StatusOK, &batchAssociateResponse,
	)
	// the batch response returns the categories it uses and added
	require.ElementsMatch(t, []string{"🧰 Developer tools", "👬 Communication"}, batchAssociateResponse.Categories)

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppAppTitleID), nil, http.StatusOK, &titleResponse, "fleet_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.AppStoreApp)
	require.ElementsMatch(t, []string{"🧰 Developer tools", "👬 Communication"}, titleResponse.SoftwareTitle.AppStoreApp.Categories)

	// test GitOps with Security and Utilities categories
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: addedApp.AdamID, SelfService: true, Categories: []string{"Security", "Utilities"}},
			},
		}, http.StatusOK, &batchAssociateResponse,
	)

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppAppTitleID), nil, http.StatusOK, &titleResponse, "fleet_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.AppStoreApp)
	require.ElementsMatch(t, []string{"🔐 Security", "🛠️ Utilities"}, titleResponse.SoftwareTitle.AppStoreApp.Categories)

	// empty out categories via gitops
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: addedApp.AdamID, SelfService: true},
			},
		}, http.StatusOK, &batchAssociateResponse,
	)

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppAppTitleID), nil, http.StatusOK, &titleResponse, "fleet_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.AppStoreApp)
	require.Empty(t, titleResponse.SoftwareTitle.AppStoreApp.Categories)
}

func (s *integrationMDMTestSuite) TestInstallAllSelfServiceSoftware() {
	t := s.T()
	ctx := context.Background()
	installAllActivityName := fleet.ActivityTypeInstalledAllSelfServiceSoftware{}.ActivityName()

	installAll := func(token string, expectedStatus int, qp ...string) {
		s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install_all", token), nil, expectedStatus, qp...)
	}
	deviceInstall := func(token string, titleID uint) {
		s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token, titleID), nil, http.StatusAccepted)
	}
	countRows := func(query string, args ...any) int {
		var n int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n, query, args...)
		})
		return n
	}
	// queued software-install titles for a host, in insertion (queue) order
	queuedTitles := func(hostID uint) []string {
		var titles []string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &titles, `
				SELECT st.name
				FROM upcoming_activities ua
				JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				JOIN software_titles st ON st.id = siua.software_title_id
				WHERE ua.host_id = ? AND ua.activity_type = 'software_install'
				ORDER BY ua.id`, hostID)
		})
		return titles
	}
	// create a no-team or team self-service deb installer, returning its title id
	newInstaller := func(name string, teamID *uint, selfService bool, labels fleet.LabelIdentsWithScope, categoryIDs ...uint) uint {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("install-"+name), t.TempDir)
		require.NoError(t, err)
		_, titleID, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			StorageID:       name + "-storage",
			Filename:        name + ".deb",
			Title:           name,
			Extension:       "deb",
			Source:          "deb_packages",
			Platform:        "linux",
			Version:         "1.0",
			InstallScript:   "install",
			UninstallScript: "uninstall",
			InstallerFile:   tfr,
			SelfService:     selfService,
			UserID:          s.users["admin1@example.com"].ID,
			TeamID:          teamID,
			ValidatedLabels: &labels,
			CategoryIDs:     categoryIDs,
		})
		require.NoError(t, err)
		return titleID
	}
	// complete the host's currently-activated (head) install with the given exit code
	completeActivatedInstall := func(host *fleet.Host, exitCode int) {
		var uid string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT execution_id FROM host_software_installs WHERE host_id = ? AND status = 'pending_install' LIMIT 1`, host.ID)
		})
		require.NotEmpty(t, uid, "expected an activated install to complete")
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "install_uuid": %q, "install_script_exit_code": %d, "install_script_output": "done"}`, *host.OrbitNodeKey, uid, exitCode)),
			http.StatusNoContent)
	}

	// VPP and in-house installs are exercised by the runs below. A stateless mock
	// stands in for Apple's VPP API and a global VPP token is inserted. Apple MDM is
	// already enabled by the MDM suite.
	vppSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/disassociate"):
			_, _ = w.Write([]byte(`{"eventId":"d"}`))
		case strings.Contains(r.URL.Path, "associate"):
			_, _ = w.Write([]byte(`{"eventId":"evt"}`))
		case strings.Contains(r.URL.Path, "assets"):
			var assets []vpp.Asset
			if adamID := r.URL.Query().Get("adamId"); adamID != "" {
				assets = []vpp.Asset{{AdamID: adamID, PricingParam: "STDQ", AvailableCount: 12}}
			}
			_ = json.NewEncoder(w).Encode(map[string][]vpp.Asset{"assets": assets})
		default:
			_, _ = w.Write([]byte(`{"locationName":"Fleet","countryISO2ACode":"US"}`))
		}
	}))
	t.Cleanup(vppSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", vppSrv.URL, t)

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// MDM-connect an Apple host (required for VPP candidacy and because activation
	// of either Apple type enqueues an MDM command).
	newMDMHost := func(platform, suffix string, teamID *uint) (*fleet.Host, string) {
		host := createOrbitEnrolledHost(t, platform, suffix, s.ds)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(teamID, []uint{host.ID})))
		host.TeamID = teamID
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate, platform, enroll_team_id) VALUES (?, NULLIF(?, ''), 'test', ?, ?)`, host.UUID, host.HardwareSerial, host.Platform, host.TeamID)
			return err
		})
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, user_id, type, topic, push_magic, token_hex, token_update_tally) VALUES (?, ?, ?, 'Device', ?, ?, ?, 1)`,
				host.UUID, host.UUID, nil, host.UUID+".topic", host.UUID+".magic", host.UUID)
			if err != nil {
				return err
			}
			_, err = q.ExecContext(ctx, `INSERT INTO nano_seen_times (id, seen_time) VALUES (?, ?)`, host.UUID, time.Now())
			return err
		})
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://example.com", false, "Fleet", "", fleet.PersonalEnrollmentTypeNone))
		token := suffix + "-tok"
		createDeviceTokenForHost(t, s.ds, host.ID, token)
		return host, token
	}
	adamSeq := 0
	newVPPApp := func(name string, teamID *uint, labels *fleet.LabelIdentsWithScope) uint {
		adamSeq++
		app, err := s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
			Name:             name,
			BundleIdentifier: "com.example." + name,
			AdamID:           fmt.Sprint(adamSeq), Platform: fleet.MacOSPlatform,
			SelfService:     true,
			ValidatedLabels: labels,
		}, teamID, nil)
		require.NoError(t, err)
		return app.TitleID
	}

	t.Run("install statuses", func(t *testing.T) {
		host := createOrbitEnrolledHost(t, "ubuntu", "ia-st", s.ds)
		token := "ia-st-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, host.ID, token)
		t.Cleanup(func() {
			require.NoError(t, s.ds.DeleteHost(ctx, host.ID))
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `DELETE FROM software_installers WHERE global_or_team_id = 0`)
				return err
			})
		})

		// bad device token -> 401; nothing available to install -> 202 with no rows or roll-up
		s.DoRawNoAuth("POST", "/api/latest/fleet/device/not-a-token/software/install_all", nil, http.StatusUnauthorized)
		installAll(token, http.StatusAccepted)
		require.Zero(t, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?`, host.ID))
		s.lastActivityOfTypeDoesNotMatch(installAllActivityName, "", 0)

		// Which self-service installs qualify (statuses, inventory, label/category/team
		// scoping, alphabetical order) is covered in testGetSoftwareTitlesForInstallAll;
		// here we verify the endpoint queues those installs, records the activities, and
		// is idempotent. "installed" is a smoke check that an installed title stays out.
		newInstaller("available", nil, true, fleet.LabelIdentsWithScope{})
		newInstaller("also-available", nil, true, fleet.LabelIdentsWithScope{})
		installedID := newInstaller("installed", nil, true, fleet.LabelIdentsWithScope{})
		pendingID := newInstaller("pending", nil, true, fleet.LabelIdentsWithScope{})

		// the host's queue runs one install at a time, so finish each before the next
		deviceInstall(token, installedID)
		completeActivatedInstall(host, 0) // -> installed, must be skipped
		deviceInstall(token, pendingID)   // -> pending head, must be left untouched

		// install_all queues only the available titles, in name order, after the
		// already-pending title (which it leaves untouched and does not duplicate)
		installAll(token, http.StatusAccepted)
		require.Equal(t, []string{"pending", "also-available", "available"}, queuedTitles(host.ID))
		s.lastActivityOfTypeMatches(installAllActivityName, fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "self_service_category_id": null, "self_service_category_name": null, "software_titles_count": 2}`,
			host.ID, host.DisplayName(),
		), 0)

		// every queued install surfaces as a pending, self-service installed_software activity
		var upcoming listHostUpcomingActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &upcoming)
		require.Len(t, upcoming.Activities, 3)
		var actTitles []string
		for _, act := range upcoming.Activities {
			require.Equal(t, fleet.ActivityTypeInstalledSoftware{}.ActivityName(), act.Type)
			var d fleet.ActivityTypeInstalledSoftware
			require.NoError(t, json.Unmarshal(*act.Details, &d))
			require.True(t, d.SelfService)
			require.Equal(t, string(fleet.SoftwareInstallPending), d.Status)
			actTitles = append(actTitles, d.SoftwareTitle)
		}
		require.ElementsMatch(t, []string{"pending", "also-available", "available"}, actTitles)

		// idempotent: a second call adds nothing
		installAll(token, http.StatusAccepted)
		require.Len(t, queuedTitles(host.ID), 3)

		// the same MDM-enrolled darwin host installs custom packages (chrome, zoom) and a
		// VPP app (slack); install_all queues them intermixed, sorted by name (not grouped
		// by type). (In-house apps are iOS/iPadOS-only and the device endpoint is desktop-
		// only, so those installs are covered in the MDM suite, not here.)
		macTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "apple-statuses"})
		require.NoError(t, err)
		macHost, macTok := newMDMHost("darwin", "av-statuses", &macTeam.ID)
		newMacOSInstaller := func(name string) {
			tfr, err := fleet.NewTempFileReader(strings.NewReader("install-"+name), t.TempDir)
			require.NoError(t, err)
			_, _, err = s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				StorageID:       name + "-storage",
				Filename:        name + ".pkg",
				Title:           name,
				Extension:       "pkg",
				Source:          "apps",
				Platform:        "darwin",
				Version:         "1.0",
				InstallScript:   "install",
				UninstallScript: "uninstall",
				InstallerFile:   tfr,
				SelfService:     true,
				UserID:          s.users["admin1@example.com"].ID,
				TeamID:          &macTeam.ID,
				ValidatedLabels: &fleet.LabelIdentsWithScope{},
			})
			require.NoError(t, err)
		}
		newMacOSInstaller("chrome")          // custom package
		newMacOSInstaller("zoom")            // custom package
		newVPPApp("slack", &macTeam.ID, nil) // VPP app; sorts between the packages
		installAll(macTok, http.StatusAccepted)

		// the package and VPP installs queue intermixed, in name order
		var macHostUpcoming listHostUpcomingActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", macHost.ID), nil, http.StatusOK, &macHostUpcoming)
		var macQueued []string
		for _, act := range macHostUpcoming.Activities {
			var d struct {
				SoftwareTitle string `json:"software_title"`
			}
			require.NoError(t, json.Unmarshal(*act.Details, &d))
			macQueued = append(macQueued, d.SoftwareTitle)
		}
		require.Equal(t, []string{"chrome", "slack", "zoom"}, macQueued)
	})

	// With a search query on the self-service page, install_all should queue
	// only the titles whose name matches — matching what the user sees on
	// screen.
	t.Run("scopes to the query parameter when provided", func(t *testing.T) {
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		host := createOrbitEnrolledHost(t, "ubuntu", "ia-q", s.ds)
		token := "ia-q-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, host.ID, token)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		newInstaller("q-apple", &team.ID, true, fleet.LabelIdentsWithScope{})
		newInstaller("q-banana", &team.ID, true, fleet.LabelIdentsWithScope{})
		newInstaller("q-cherry", &team.ID, true, fleet.LabelIdentsWithScope{})

		installAll(token, http.StatusAccepted, "query", "ban")
		require.Equal(t, []string{"q-banana"}, queuedTitles(host.ID))

		// blank query behaves like no query — all three queue
		installAll(token, http.StatusAccepted, "query", "")
		require.ElementsMatch(t, []string{"q-banana", "q-apple", "q-cherry"}, queuedTitles(host.ID))
	})

	t.Run("coexists with existing and incoming activities", func(t *testing.T) {
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		host := createOrbitEnrolledHost(t, "ubuntu", "ia-act", s.ds)
		token := "ia-act-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, host.ID, token)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		aID := newInstaller("act-a", &team.ID, true, fleet.LabelIdentsWithScope{})
		newInstaller("act-b", &team.ID, true, fleet.LabelIdentsWithScope{})
		newInstaller("act-c", &team.ID, true, fleet.LabelIdentsWithScope{})

		// a manual install of A is already pending before install_all runs
		deviceInstall(token, aID)
		require.Equal(t, []string{"act-a"}, queuedTitles(host.ID))

		// install_all appends the remaining titles behind A without re-queueing A
		installAll(token, http.StatusAccepted)
		require.Equal(t, []string{"act-a", "act-b", "act-c"}, queuedTitles(host.ID))
		s.lastActivityOfTypeMatches(installAllActivityName, fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "self_service_category_id": null, "self_service_category_name": null, "software_titles_count": 2}`,
			host.ID, host.DisplayName(),
		), 0)

		// the queue drains in order
		for range 3 {
			completeActivatedInstall(host, 0)
		}
		require.Zero(t, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?`, host.ID))
		require.Equal(t, 3, countRows(`SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND status = 'installed'`, host.ID))

		// an install request arriving after install_all coexists and drains too
		dID := newInstaller("act-d", &team.ID, true, fleet.LabelIdentsWithScope{})
		deviceInstall(token, dID)
		completeActivatedInstall(host, 0)
		require.Equal(t, 4, countRows(`SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND status = 'installed'`, host.ID))

		// an install request racing install_all on the same host must not corrupt the
		// queue: exactly one activated head, every install request present
		host2 := createOrbitEnrolledHost(t, "ubuntu", "ia-act2", s.ds)
		token2 := "ia-act2-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, host2.ID, token2)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host2.ID})))

		client := s.server.Client()
		var wg sync.WaitGroup
		var instErr, allErr error
		var instStatus, allStatus int
		wg.Add(2)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("POST", s.server.URL+fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token2, aID), nil)
			if err != nil {
				instErr = err
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				instErr = err
				return
			}
			resp.Body.Close()
			instStatus = resp.StatusCode
		}()
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("POST", s.server.URL+fmt.Sprintf("/api/latest/fleet/device/%s/software/install_all", token2), nil)
			if err != nil {
				allErr = err
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				allErr = err
				return
			}
			resp.Body.Close()
			allStatus = resp.StatusCode
		}()
		wg.Wait()
		require.NoError(t, instErr)
		require.NoError(t, allErr)
		require.Equal(t, http.StatusAccepted, allStatus)
		// the single install may win (202) or find the title already queued by
		// install_all (400); either way the queue must stay consistent
		require.Contains(t, []int{http.StatusAccepted, http.StatusBadRequest}, instStatus)
		require.Equal(t, 1, countRows(`SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND status = 'pending_install'`, host2.ID))
		require.GreaterOrEqual(t, len(queuedTitles(host2.ID)), 4)

		// double-queue: if another endpoint queues a title behind an activated head
		// while install_all's title list (read before that) still includes it,
		// install_all queues a second copy. Its per-install reset only cancels
		// ACTIVATED installs, so the not-yet-activated concurrent install survives.
		// This reproduces that interleaving deterministically using install_all's exact
		// insert sequence (ResetNonPolicyInstallAttempts + InsertSoftwareInstallRequest).
		host3 := createOrbitEnrolledHost(t, "ubuntu", "ia-dq", s.ds)
		token3 := "ia-dq-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, host3.ID, token3)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host3.ID})))

		headID := newInstaller("dq-head", &team.ID, true, fleet.LabelIdentsWithScope{})
		fooID := newInstaller("dq-foo", &team.ID, true, fleet.LabelIdentsWithScope{})
		deviceInstall(token3, headID) // activated head
		deviceInstall(token3, fooID)  // queued behind the head, not yet activated

		var fooInstallerID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &fooInstallerID, `SELECT id FROM software_installers WHERE title_id = ? AND global_or_team_id = ?`, fooID, team.ID)
		})
		require.NoError(t, s.ds.ResetNonPolicyInstallAttempts(ctx, host3.ID, fooInstallerID))
		_, err = s.ds.InsertSoftwareInstallRequest(ctx, host3.ID, fooInstallerID, fleet.HostSoftwareInstallOptions{SelfService: true, WithRetries: true})
		require.NoError(t, err)

		require.Equal(t, 2, countRows(`
			SELECT COUNT(*) FROM upcoming_activities ua
			JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
			JOIN software_titles st ON st.id = siua.software_title_id
			WHERE ua.host_id = ? AND st.name = 'dq-foo'`, host3.ID))

		// install_all also queues VPP self-service app installs (MDM-enrolled darwin host)
		vppTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "apple-activities"})
		require.NoError(t, err)
		vppHost, vppTok := newMDMHost("darwin", "av-activities", &vppTeam.ID)
		newVPPApp("vpp-activities", &vppTeam.ID, nil)
		installAll(vppTok, http.StatusAccepted)
		require.Equal(t, 1, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activity_type = 'vpp_app_install'`, vppHost.ID))

		require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
	})

	t.Run("category label and team scoping", func(t *testing.T) {
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)

		// a no-team installer must never be queued for a team host
		newInstaller("global-app", nil, true, fleet.LabelIdentsWithScope{})
		t.Cleanup(func() {
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `DELETE FROM software_installers WHERE global_or_team_id = 0`)
				return err
			})
		})

		// hostA exercises team + label scoping
		hostA := createOrbitEnrolledHost(t, "ubuntu", "ia-sc-a", s.ds)
		tokenA := "ia-sc-a-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, hostA.ID, tokenA)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{hostA.ID})))

		lbl, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "-lbl", Query: "select 1"})
		require.NoError(t, err)
		require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, hostA, map[uint]*bool{lbl.ID: new(true)}, time.Now(), false))
		includeLbl := fleet.LabelIdentsWithScope{LabelScope: fleet.LabelScopeIncludeAny, ByName: map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}}}
		excludeLbl := fleet.LabelIdentsWithScope{LabelScope: fleet.LabelScopeExcludeAny, ByName: map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}}}

		newInstaller("team-app", &team.ID, true, fleet.LabelIdentsWithScope{})
		newInstaller("lbl-in", &team.ID, true, includeLbl)
		newInstaller("lbl-out", &team.ID, true, excludeLbl)

		// in scope: team-app and the include-any match; the exclude-any title and the
		// no-team installer are skipped
		installAll(tokenA, http.StatusAccepted)
		require.ElementsMatch(t, []string{"team-app", "lbl-in"}, queuedTitles(hostA.ID))

		// hostB exercises category scoping
		hostB := createOrbitEnrolledHost(t, "ubuntu", "ia-sc-b", s.ds)
		tokenB := "ia-sc-b-token" //nolint:gosec // G101: test value only
		createDeviceTokenForHost(t, s.ds, hostB.ID, tokenB)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{hostB.ID})))

		cat, err := s.ds.NewSoftwareCategory(ctx, team.ID, t.Name()+"-cat")
		require.NoError(t, err)
		newInstaller("cat-app", &team.ID, true, fleet.LabelIdentsWithScope{}, cat.ID)

		// scoped to the category -> only its title, and the roll-up carries the category
		installAll(tokenB, http.StatusAccepted, "category_id", fmt.Sprint(cat.ID))
		require.Equal(t, []string{"cat-app"}, queuedTitles(hostB.ID))
		s.lastActivityOfTypeMatches(installAllActivityName, fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "self_service_category_id": %d, "self_service_category_name": %q, "software_titles_count": 1}`,
			hostB.ID, hostB.DisplayName(), cat.ID, cat.Name,
		), 0)

		// nonexistent category, or a category on another fleet -> 400
		installAll(tokenB, http.StatusBadRequest, "category_id", "9999999")
		otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-other"})
		require.NoError(t, err)
		otherCat, err := s.ds.NewSoftwareCategory(ctx, otherTeam.ID, t.Name()+"-othercat")
		require.NoError(t, err)
		installAll(tokenB, http.StatusBadRequest, "category_id", fmt.Sprint(otherCat.ID))

		// VPP label scoping: include-any (host is a member) is queued, exclude-any is skipped
		vppHost, vppToken := newMDMHost("darwin", "sc-vpp", &team.ID)
		require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, vppHost, map[uint]*bool{lbl.ID: new(true)}, time.Now(), false))
		newVPPApp("vpp-in", &team.ID, &fleet.LabelIdentsWithScope{LabelScope: fleet.LabelScopeIncludeAny, ByName: map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}}})
		newVPPApp("vpp-out", &team.ID, &fleet.LabelIdentsWithScope{LabelScope: fleet.LabelScopeExcludeAny, ByName: map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}}})
		installAll(vppToken, http.StatusAccepted)
		require.Equal(t, 1, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activity_type = 'vpp_app_install'`, vppHost.ID))

		require.NoError(t, s.ds.DeleteTeam(ctx, otherTeam.ID))
		require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
	})

	t.Run("multiple hosts", func(t *testing.T) {
		const (
			numHosts      = 5
			numInstallers = 5
		)
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		for i := range numInstallers {
			newInstaller(fmt.Sprintf("mh-%d", i), &team.ID, true, fleet.LabelIdentsWithScope{})
		}
		type hostToken struct {
			host  *fleet.Host
			token string
		}
		hosts := make([]hostToken, numHosts)
		for i := range numHosts {
			h := createOrbitEnrolledHost(t, "ubuntu", fmt.Sprintf("mh-%d", i), s.ds)
			require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{h.ID})))
			tok := fmt.Sprintf("mh-tok-%d", i)
			createDeviceTokenForHost(t, s.ds, h.ID, tok)
			hosts[i] = hostToken{h, tok}
		}

		// every host fires install_all at once (require can't run off the test goroutine)
		client := s.server.Client()
		statuses := make([]int, numHosts)
		errs := make([]error, numHosts)
		var wg sync.WaitGroup
		for i := range hosts {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req, err := http.NewRequest("POST", s.server.URL+fmt.Sprintf("/api/latest/fleet/device/%s/software/install_all", hosts[i].token), nil)
				if err != nil {
					errs[i] = err
					return
				}
				resp, err := client.Do(req)
				if err != nil {
					errs[i] = err
					return
				}
				resp.Body.Close()
				statuses[i] = resp.StatusCode
			}(i)
		}
		wg.Wait()

		for i := range hosts {
			require.NoErrorf(t, errs[i], "host %d", i)
			require.Equalf(t, http.StatusAccepted, statuses[i], "host %d", i)
		}

		// each host queued exactly its installers, with no lost or cross-host rows
		for i := range hosts {
			require.Equalf(t, numInstallers, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activity_type = 'software_install'`, hosts[i].host.ID), "host %d", i)
		}

		// re-running queues nothing new, then drain every host's queue
		for i := range hosts {
			installAll(hosts[i].token, http.StatusAccepted)
			require.Equalf(t, numInstallers, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activity_type = 'software_install'`, hosts[i].host.ID), "host %d", i)
		}
		for i := range hosts {
			for range numInstallers {
				completeActivatedInstall(hosts[i].host, 0)
			}
			require.Zerof(t, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?`, hosts[i].host.ID), "host %d", i)
			require.Equalf(t, numInstallers, countRows(`SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND status = 'installed'`, hosts[i].host.ID), "host %d", i)
		}

		require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))

		// install_all also queues VPP self-service app installs (MDM-enrolled darwin host)
		vppTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "apple-multihost"})
		require.NoError(t, err)
		vppHost, vppTok := newMDMHost("darwin", "av-multihost", &vppTeam.ID)
		newVPPApp("vpp-multihost", &vppTeam.ID, nil)
		installAll(vppTok, http.StatusAccepted)
		require.Equal(t, 1, countRows(`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ? AND activity_type = 'vpp_app_install'`, vppHost.ID))
	})
}
