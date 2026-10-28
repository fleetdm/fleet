package service

// Fleet-maintained app tests for the enterprise (premium) suite.
//
// Belongs here: the Fleet-maintained apps catalog and adding apps from it, upgrade codes, the
// auto-update cron, version pinning and rollback, label scoping of replaced installers, and
// patch policies.
//
// Does not belong here: Fleet-maintained apps referenced from a batch-set request
// (integration_enterprise_software_batch_test.go) and custom package uploads
// (integration_enterprise_software_installers_test.go).

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps/maintainedappstest"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestMaintainedApps() {
	t := s.T()
	ctx := context.Background()

	installerBytes := []byte("abc")

	// Mock server to serve the "installers"
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/badinstaller":
			_, _ = w.Write([]byte("badinstaller"))
		case "/timeout":
			time.Sleep(3 * time.Second)
			_, _ = w.Write([]byte("timeout"))
		default:
			_, _ = w.Write(installerBytes)
		}
	}))
	defer installerServer.Close()

	getSoftwareInstallerIDByMAppID := func(mappID uint) uint {
		var id uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, "SELECT id FROM software_installers WHERE fleet_maintained_app_id = ?", mappID)
		})

		return id
	}

	// Non-existent maintained app
	s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 1}, http.StatusNotFound)

	// Insert the list of maintained apps
	insertedApps := maintainedappstest.SyncApps(t, s.ds)
	expectedApps := insertedApps

	h := sha256.New()
	_, err := h.Write(installerBytes)
	require.NoError(t, err)
	spoofedSHA := hex.EncodeToString(h.Sum(nil))

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, ".json"), "/")

		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "1",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       installerServer.URL + "/installer.zip",
			InstallScriptRef:   "foobaz",
			UninstallScriptRef: "foobaz",
			SHA256:             spoofedSHA,
			DefaultCategories:  []string{"Productivity"},
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs: map[string]string{
				"foobaz": "Hello World!",
			},
		}

		switch slug {
		case "fail":
			w.WriteHeader(http.StatusInternalServerError)
			return

		case "notfound":
			w.WriteHeader(http.StatusNotFound)
			return

		case "badinstaller":
			manifest.Versions[0].InstallerURL = installerServer.URL + "/badinstaller"

		case "timeout":
			manifest.Versions[0].InstallerURL = installerServer.URL + "/timeout"
		}

		err := json.NewEncoder(w).Encode(manifest)
		assert.NoError(t, err)
	}))
	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)
	defer dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team 1")}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Check apps returned
	var listMAResp listFleetMaintainedAppsResponse
	s.DoJSON(http.MethodGet, "/api/latest/fleet/software/fleet_maintained_apps", listFleetMaintainedAppsRequest{}, http.StatusOK,
		&listMAResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, listMAResp.Err)
	require.False(t, listMAResp.Meta.HasPreviousResults)
	require.False(t, listMAResp.Meta.HasNextResults)
	require.Len(t, listMAResp.FleetMaintainedApps, len(expectedApps))

	// Count is the total number of installable platform entries: an app's macOS
	// and Windows entries are separately installable (one Add button each), so
	// they count separately even though the UI combines them into a single row.
	// The unfiltered list returns every platform row, so here the count equals the
	// number of rows returned.
	require.Equal(t, len(expectedApps), listMAResp.Count)

	sortFMAs := func(a, b fleet.MaintainedApp) int {
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.Slug, b.Slug)
	}
	slices.SortFunc(listMAResp.FleetMaintainedApps, sortFMAs)
	slices.SortFunc(expectedApps, sortFMAs)
	require.Equal(t, expectedApps, listMAResp.FleetMaintainedApps)

	var listMAResp2 listFleetMaintainedAppsResponse
	s.DoJSON(
		http.MethodGet,
		"/api/latest/fleet/software/fleet_maintained_apps",
		listFleetMaintainedAppsRequest{},
		http.StatusOK,
		&listMAResp2,
		"team_id", fmt.Sprint(team.ID),
		"per_page", "2",
		"page", "2",
	)
	require.NoError(t, listMAResp2.Err)
	require.True(t, listMAResp2.Meta.HasPreviousResults)
	require.True(t, listMAResp2.Meta.HasNextResults)
	require.Len(t, listMAResp2.FleetMaintainedApps, 2)
	require.Contains(t, listMAResp.FleetMaintainedApps, listMAResp2.FleetMaintainedApps[0])

	// The platform filter narrows the list to apps available on that platform.
	// Fewer apps ship a Windows installer than a macOS one, so the Windows-only
	// count is a non-empty strict subset of the full count.
	var listMAWin listFleetMaintainedAppsResponse
	s.DoJSON(
		http.MethodGet,
		"/api/latest/fleet/software/fleet_maintained_apps",
		listFleetMaintainedAppsRequest{},
		http.StatusOK,
		&listMAWin,
		"team_id", fmt.Sprint(team.ID),
		"platform", "windows",
	)
	require.NoError(t, listMAWin.Err)
	require.Positive(t, listMAWin.Count)
	require.Less(t, listMAWin.Count, listMAResp.Count)

	// Check individual app fetch
	var getMAResp getFleetMaintainedAppResponse
	s.DoJSON(http.MethodGet, fmt.Sprintf("/api/latest/fleet/software/fleet_maintained_apps/%d", listMAResp.FleetMaintainedApps[0].ID), getFleetMaintainedAppRequest{}, http.StatusOK, &getMAResp)
	// TODO this will change when actual install scripts are created.
	dbAppRecord, err := s.ds.GetMaintainedAppByID(ctx, listMAResp.FleetMaintainedApps[0].ID, nil)
	require.NoError(t, err)
	_, err = maintained_apps.Hydrate(ctx, dbAppRecord, "", nil, nil)
	require.NoError(t, err)
	dbAppResponse := fleet.MaintainedApp{
		ID:                    dbAppRecord.ID,
		Name:                  dbAppRecord.Name,
		Slug:                  dbAppRecord.Slug,
		Version:               dbAppRecord.Version,
		Platform:              dbAppRecord.Platform,
		InstallerURL:          dbAppRecord.InstallerURL,
		InstallScript:         dbAppRecord.InstallScript,
		UninstallScript:       dbAppRecord.UninstallScript,
		AutomaticInstallQuery: dbAppRecord.AutomaticInstallQuery,
		Categories:            []string{"Productivity"},
	}
	require.NotEmpty(t, getMAResp.FleetMaintainedApp.InstallerURL)
	require.NotEmpty(t, getMAResp.FleetMaintainedApp.InstallScript)
	require.NotEmpty(t, getMAResp.FleetMaintainedApp.UninstallScript)
	require.Equal(t, dbAppResponse, *getMAResp.FleetMaintainedApp)

	// Try adding ingested app with invalid secret
	reqInvalidSecret := &addFleetMaintainedAppRequest{
		AppID:             1,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo $FLEET_SECRET_INVALID1",
		PostInstallScript: "echo done $FLEET_SECRET_INVALID2",
		UninstallScript:   "echo $FLEET_SECRET_INVALID3",
	}
	respBadSecret := s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", reqInvalidSecret, http.StatusUnprocessableEntity)
	errNames, errReasons := extractServerErrorNameReasons(respBadSecret.Body)
	assert.ElementsMatch(t, []string{"install script", "post-install script", "uninstall script"}, errNames)
	assert.Len(t, errReasons, 3)
	for _, reason := range errReasons {
		assert.Contains(t, reason, "$FLEET_SECRET_INVALID")
	}

	reqInvalidSecret = &addFleetMaintainedAppRequest{
		AppID:             1,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done $FLEET_SECRET_INVALID2",
		UninstallScript:   "echo",
	}
	respBadSecret = s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", reqInvalidSecret, http.StatusUnprocessableEntity)
	errNames, errReasons = extractServerErrorNameReasons(respBadSecret.Body)
	assert.ElementsMatch(t, []string{"post-install script"}, errNames)
	require.Len(t, errReasons, 1)
	assert.Contains(t, errReasons[0], "$FLEET_SECRET_INVALID2")

	reqInvalidSecret = &addFleetMaintainedAppRequest{
		AppID:             1,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
		UninstallScript:   "echo $FLEET_SECRET_INVALID3",
	}
	respBadSecret = s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", reqInvalidSecret, http.StatusUnprocessableEntity)
	errNames, errReasons = extractServerErrorNameReasons(respBadSecret.Body)
	assert.ElementsMatch(t, []string{"uninstall script"}, errNames)
	require.Len(t, errReasons, 1)
	assert.Contains(t, errReasons[0], "$FLEET_SECRET_INVALID3")

	// Add with unsupported shebang in install script fails validation
	reqBadShebang := &addFleetMaintainedAppRequest{
		AppID:         1,
		TeamID:        &team.ID,
		InstallScript: "#!/usr/bin/perl\nprint 'hello'",
	}
	respBadShebang := s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", reqBadShebang, http.StatusBadRequest)
	errMsg := extractServerErrorText(respBadShebang.Body)
	require.Contains(t, errMsg, "Interpreter not supported")

	// Add an ingested app to the team
	var addMAResp addFleetMaintainedAppResponse
	req := &addFleetMaintainedAppRequest{
		AppID:             1,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "echo done",
	}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.NoError(t, addMAResp.Err)

	// Verify the installer has is_active = 1 for this FMA
	var activeInstallerCount, onlyInstallerID uint
	var isActive bool
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, q, &activeInstallerCount,
			`SELECT COUNT(*) FROM software_installers WHERE fleet_maintained_app_id = ? AND global_or_team_id = ? AND is_active = 1`,
			req.AppID, team.ID)
		if err != nil {
			return err
		}

		err = sqlx.GetContext(ctx, q, &onlyInstallerID,
			`SELECT id FROM software_installers WHERE fleet_maintained_app_id = ? AND global_or_team_id = ?`,
			req.AppID, team.ID)
		if err != nil {
			return err
		}

		return sqlx.GetContext(ctx, q, &isActive,
			`SELECT is_active FROM software_installers WHERE id = ?`,
			onlyInstallerID)
	})
	require.True(t, isActive, "installer must have is_active = 1")
	require.Equal(t, uint(1), activeInstallerCount, "single-add API must set is_active = 1 for the installer")

	s.DoJSON(http.MethodGet, "/api/latest/fleet/software/fleet_maintained_apps", listFleetMaintainedAppsRequest{}, http.StatusOK,
		&listMAResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, listMAResp.Err)
	require.False(t, listMAResp.Meta.HasPreviousResults)
	require.Len(t, listMAResp.FleetMaintainedApps, len(expectedApps))

	// Validate software installer fields
	mapp, err := s.ds.GetMaintainedAppByID(ctx, 1, &team.ID)
	require.NoError(t, err)
	_, err = maintained_apps.Hydrate(ctx, mapp, "", nil, nil)
	require.NoError(t, err)
	i, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), getSoftwareInstallerIDByMAppID(1))
	require.NoError(t, err)
	require.Equal(t, mapp.TitleID, i.TitleID)
	require.Equal(t, new(uint(1)), i.FleetMaintainedAppID)
	require.Equal(t, mapp.SHA256, i.StorageID)
	require.Equal(t, "darwin", i.Platform)
	require.NotEmpty(t, i.InstallScriptContentID)
	require.Equal(t, req.PreInstallQuery, i.PreInstallQuery)
	install, err := s.ds.GetAnyScriptContents(ctx, i.InstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, mapp.InstallScript, string(install))
	require.NotNil(t, i.PostInstallScriptContentID)
	postinstall, err := s.ds.GetAnyScriptContents(ctx, *i.PostInstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, req.PostInstallScript, string(postinstall))

	// The maintained app should now be in software titles
	var resp listSoftwareTitlesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", fmt.Sprintf("%d", team.ID),
	)

	require.Equal(t, 1, resp.Count)
	title := resp.SoftwareTitles[0]
	require.NotNil(t, title.BundleIdentifier)
	require.Equal(t, new(mapp.UniqueIdentifier), title.BundleIdentifier)
	require.Equal(t, mapp.Version, title.SoftwarePackage.Version)
	require.Equal(t, "installer.zip", title.SoftwarePackage.Name)
	require.Equal(t, new(req.SelfService), title.SoftwarePackage.SelfService)

	// Check activity
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeAddedSoftware{}.ActivityName(),
		fmt.Sprintf(`{"software_title": "%[1]s", "software_package": "installer.zip", "team_name": "%s", "team_id": %d, "fleet_name": "%s", "fleet_id": %d, "self_service": true, "software_title_id": %d}`, mapp.Name, team.Name, team.ID, team.Name, team.ID, title.ID),
		0,
	)

	// Edit DB to point some apps at bad-installer/timeout slugs
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx, "UPDATE fleet_maintained_apps SET slug = ? WHERE id = 2", "badinstaller")
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, "UPDATE fleet_maintained_apps SET slug = ? WHERE id = 3", "timeout")
		return err
	})

	// Should return an error; SHAs don't match up
	r := s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 2}, http.StatusInternalServerError)
	require.Contains(t, extractServerErrorText(r.Body), "mismatch in maintained app SHA256 hash")

	// Should timeout
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT", "1s")
	r = s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 3}, http.StatusGatewayTimeout)
	dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT")
	require.Contains(t, extractServerErrorText(r.Body), fleet.AddMaintainedAppTimeoutErrMsg)

	// Add a maintained app to no team

	req = &addFleetMaintainedAppRequest{
		AppID:             4,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "echo done",
	}

	addMAResp = addFleetMaintainedAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.NoError(t, addMAResp.Err)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", "0",
	)

	mapp, err = s.ds.GetMaintainedAppByID(ctx, 4, new(uint(0)))
	require.NoError(t, err)
	_, err = maintained_apps.Hydrate(ctx, mapp, "", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	title = resp.SoftwareTitles[0]
	require.Equal(t, title.ID, *mapp.TitleID)
	require.Equal(t, mapp.Version, title.SoftwarePackage.Version)
	require.Equal(t, "installer.zip", title.SoftwarePackage.Name)
	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", title.ID), nil, http.StatusOK, &titleResponse, "team_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.SoftwarePackage)
	require.Equal(t, []string{"🖥️ Productivity"}, titleResponse.SoftwareTitle.SoftwarePackage.Categories)

	i, err = s.ds.GetSoftwareInstallerMetadataByID(context.Background(), getSoftwareInstallerIDByMAppID(4))
	require.NoError(t, err)
	require.Equal(t, new(uint(4)), i.FleetMaintainedAppID)
	require.Equal(t, mapp.SHA256, i.StorageID)
	require.NotEmpty(t, i.InstallScriptContentID)
	require.Equal(t, req.PreInstallQuery, i.PreInstallQuery)
	install, err = s.ds.GetAnyScriptContents(ctx, i.InstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, mapp.InstallScript, string(install))
	require.NotNil(t, i.PostInstallScriptContentID)
	postinstall, err = s.ds.GetAnyScriptContents(ctx, *i.PostInstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, req.PostInstallScript, string(postinstall))

	// Add some categories and validate them
	cat1, err := s.ds.NewSoftwareCategory(ctx, 0, "test_category_1")
	require.NoError(t, err)

	cat2, err := s.ds.NewSoftwareCategory(ctx, 0, "test_category_2")
	require.NoError(t, err)

	updatePayload := &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       title.ID,
		InstallerID:   i.InstallerID,
		InstallScript: new(mapp.InstallScript),
		Version:       title.SoftwarePackage.Version,
		SelfService:   new(true),
		Categories:    []string{cat1.Name, cat2.Name},
	}
	s.updateSoftwareInstaller(t, updatePayload, http.StatusOK, "")

	titleResponse = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", title.ID), nil, http.StatusOK, &titleResponse, "team_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.SoftwarePackage)
	require.Len(t, titleResponse.SoftwareTitle.SoftwarePackage.Categories, 2)
	require.Contains(t, titleResponse.SoftwareTitle.SoftwarePackage.Categories, cat1.Name)
	require.Contains(t, titleResponse.SoftwareTitle.SoftwarePackage.Categories, cat2.Name)

	// ===========================================================================================
	// Adding an automatically installed FMA
	// ===========================================================================================

	// Add another FMA
	req = &addFleetMaintainedAppRequest{
		AppID:             5,
		SelfService:       false,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
		TeamID:            new(uint(0)),
	}

	addMAResp = addFleetMaintainedAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.NoError(t, addMAResp.Err)
	require.NotEmpty(t, addMAResp.SoftwareTitleID)

	// Add the automatic install policy
	tpParams := fleet.TeamPolicyRequest{
		Name:            "[Install software]",
		Query:           "select * from osquery;",
		Description:     "Some description",
		Platform:        "darwin",
		SoftwareTitleID: &addMAResp.SoftwareTitleID,
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)

	// List software titles; we should see the policy on the software title object

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "2",
		"order_key", "id",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", "0",
	)

	require.Len(t, resp.SoftwareTitles, 2)
	// most recently added FMA should have 1 automatic install policy
	st := resp.SoftwareTitles[0] // sorted by ID above
	require.NotNil(t, st.SoftwarePackage)
	require.Len(t, st.SoftwarePackage.AutomaticInstallPolicies, 1)
	gotPolicy := st.SoftwarePackage.AutomaticInstallPolicies[0]
	require.Equal(t, tpResp.Policy.Name, gotPolicy.Name)
	require.Equal(t, tpResp.Policy.ID, gotPolicy.ID)

	// check that we created an activity for the policy creation
	noTeamID := int64(0)
	wantAct := fleet.ActivityTypeCreatedPolicy{
		ID:       gotPolicy.ID,
		Name:     gotPolicy.Name,
		TeamID:   &noTeamID,
		TeamName: nil,
	}
	s.lastActivityMatches(wantAct.ActivityName(), fmt.Sprintf(`{"fleet_id":0, "policy_id":%d, "policy_name":"[Install software]", "team_id":0}`, gotPolicy.ID), 0)

	// First FMA added doesn't have automatic install policies
	st = resp.SoftwareTitles[1] // sorted by ID above
	require.NotNil(t, st.SoftwarePackage)
	require.Empty(t, st.SoftwarePackage.AutomaticInstallPolicies)

	// Get the specific app that we set to be installed automatically
	var titleResp getSoftwareTitleResponse
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addMAResp.SoftwareTitleID),
		getSoftwareTitleRequest{},
		http.StatusOK, &titleResp,
		"team_id", "0",
	)
	require.NotNil(t, titleResp.SoftwareTitle)
	swTitle := titleResp.SoftwareTitle
	require.NotNil(t, swTitle.SoftwarePackage)
	require.Len(t, swTitle.SoftwarePackage.AutomaticInstallPolicies, 1)
	gotPolicy = swTitle.SoftwarePackage.AutomaticInstallPolicies[0]
	require.Equal(t, tpResp.Policy.Name, gotPolicy.Name)
	require.Equal(t, tpResp.Policy.ID, gotPolicy.ID)

	// Policy should appear in the list of policies
	var listPolResp fleet.ListTeamPoliciesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/teams/0/policies",
		fleet.ListTeamPoliciesRequest{},
		http.StatusOK, &listPolResp,
		"page", "0",
	)

	require.Len(t, listPolResp.Policies, 1)
	policies := listPolResp.Policies
	require.Equal(t, tpResp.Policy.Name, policies[0].Name)
	require.Equal(t, tpResp.Policy.ID, policies[0].ID)
	require.Equal(t, tpResp.Policy.Description, policies[0].Description)
	require.Equal(t, tpResp.Policy.Query, policies[0].Query)
	require.Equal(t, "darwin", policies[0].Platform)
	require.False(t, policies[0].Critical)
	require.NotNil(t, policies[0].InstallSoftware)
	require.Equal(t, tpResp.Policy.InstallSoftware.Name, policies[0].InstallSoftware.Name)
	require.Equal(t, tpResp.Policy.InstallSoftware.SoftwareTitleID, policies[0].InstallSoftware.SoftwareTitleID)

	// ===========================================================================================
	// Adding label-scoped FMA
	// ===========================================================================================

	// Add some labels
	var newLabelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:     t.Name() + "1",
		Platform: "darwin",
		Query:    "SELECT 1",
	}, http.StatusOK, &newLabelResp)
	lbl1 := newLabelResp.Label
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:     t.Name() + "2",
		Platform: "darwin",
		Query:    "SELECT 1",
	}, http.StatusOK, &newLabelResp)
	lbl2 := newLabelResp.Label

	// Add another FMA
	req = &addFleetMaintainedAppRequest{
		AppID:             6,
		SelfService:       false,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
		TeamID:            new(uint(0)),
		LabelsIncludeAny:  []string{lbl1.Name, lbl2.Name},
	}

	addMAResp = addFleetMaintainedAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.NoError(t, addMAResp.Err)
	require.NotEmpty(t, addMAResp.SoftwareTitleID)

	// Get software title details
	titleResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addMAResp.SoftwareTitleID),
		getSoftwareTitleRequest{},
		http.StatusOK, &titleResp,
		"team_id", "0",
	)

	require.NotNil(t, titleResp.SoftwareTitle)
	swTitle = titleResp.SoftwareTitle
	require.NotNil(t, swTitle.SoftwarePackage)
	require.Empty(t, swTitle.SoftwarePackage.LabelsExcludeAny)
	require.Empty(t, swTitle.SoftwarePackage.LabelsIncludeAll)
	require.Len(t, swTitle.SoftwarePackage.LabelsIncludeAny, 2)
	gotNames := make(map[string]struct{})
	for _, lbl := range swTitle.SoftwarePackage.LabelsIncludeAny {
		gotNames[lbl.LabelName] = struct{}{}
	}
	require.Contains(t, gotNames, lbl1.Name)
	require.Contains(t, gotNames, lbl2.Name)

	// Can't set non-existent label
	req = &addFleetMaintainedAppRequest{
		AppID:             7,
		SelfService:       false,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
		TeamID:            new(uint(0)),
		LabelsIncludeAny:  []string{"no-such-label"},
	}
	addMAResp = addFleetMaintainedAppResponse{}
	r = s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), `Couldn't update. Label "no-such-label" doesn't exist. Please remove the label from the software.`)

	// Can't set both labels_include_any and labels_exclude_any
	req.LabelsIncludeAny = []string{lbl1.Name, lbl2.Name}
	req.LabelsExcludeAny = []string{lbl1.Name}
	addMAResp = addFleetMaintainedAppResponse{}
	r = s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), `Only one of "labels_include_all", "labels_include_any" or "labels_exclude_any" can be included`)
}

func (s *integrationEnterpriseTestSuite) TestUpgradeCodesFromMaintainedApps() {
	// Specifically test that `upgrade_code` is correctly associated with host software that has
	// first been added via FMA. For a more robust handling of possible error scenarios when adding
	// Maintained Apps, see `TestMaintainedApps`

	t := s.T()
	ctx := context.Background()

	warpUpgradeCode := "{1BF42825-7B65-4CA9-AFFF-B7B5E1CE27B4}"

	ac, err := s.ds.AppConfig(ctx)
	s.Require().NoError(err)
	ac.Features.EnableSoftwareInventory = true
	err = s.ds.SaveAppConfig(context.Background(), ac)
	s.Require().NoError(err)
	time.Sleep(2 * time.Second) // Wait for the app config cache to clear

	installerBytes := []byte("abc")
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/badinstaller":
			_, _ = w.Write([]byte("badinstaller"))
		case "/timeout":
			time.Sleep(3 * time.Second)
			_, _ = w.Write([]byte("timeout"))
		default:
			_, _ = w.Write(installerBytes)
		}
	}))
	defer installerServer.Close()

	// Mock server to serve manifest with no_check
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "25.9.558.0",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       installerServer.URL + "/installer.msi",
			InstallScriptRef:   "foobaz",
			UninstallScriptRef: "foobaz",
			SHA256:             "no_check",
			UpgradeCode:        warpUpgradeCode,
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs: map[string]string{
				"foobaz": "Hello World!",
			},
		}

		err := json.NewEncoder(w).Encode(manifest)
		assert.NoError(t, err)
	}))
	t.Cleanup(manifestServer.Close)

	mockTransport := &mockRoundTripper{
		mockServer:  manifestServer.URL,
		origBaseURL: "https://maintained-apps.fleetdm.com",
		next:        http.DefaultTransport,
	}
	http.DefaultTransport = mockTransport

	// Insert the list of maintained apps
	maintainedappstest.SyncApps(t, s.ds)

	// verify WARP is in `fleet_maintained_apps` table but not in `software_installers`
	var warpFmaId uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &warpFmaId, "SELECT id FROM fleet_maintained_apps WHERE name = 'Cloudflare One' AND platform = 'windows'")
	})
	require.NotNil(t, warpFmaId)

	var count uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM software_installers")
	})
	require.Zero(t, count)

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team 1")}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Add WARP for Windows
	var warpAppId uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &warpAppId, "SELECT id FROM fleet_maintained_apps WHERE name = 'Cloudflare One' and platform = 'windows'")
	})

	var addMAresp addFleetMaintainedAppResponse
	req := &addFleetMaintainedAppRequest{
		AppID:             warpAppId,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "echo done",
	}

	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAresp)
	require.NoError(t, addMAresp.Err)

	// Verify WARP is now in `software_installers`, a `software_tiles` row has been created, and they
	// are associated
	var warpInstaller fleet.SoftwareInstaller
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &warpInstaller, "SELECT id, title_id, upgrade_code, filename FROM software_installers WHERE upgrade_code = ?", warpUpgradeCode)
	})
	require.NotNil(t, warpInstaller)
	require.NotNil(t, warpInstaller.Name)

	var lSTResp listSoftwareTitlesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &lSTResp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	title := lSTResp.SoftwareTitles[0]
	require.Equal(t, *warpInstaller.TitleID, title.ID)
	require.Equal(t, warpUpgradeCode, *title.UpgradeCode)

	// Create a Windows host on the team
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "windows",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// mock osquery ingesting matching software from the host
	ac, err = s.ds.AppConfig(context.Background())
	require.NoError(t, err)

	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, ac, &ac.Features, osquery_utils.Integrations{}, nil)

	rows := []map[string]string{
		{
			"name":         "Cloudflare WARP",
			"version":      "25.9.558.0",
			"source":       "programs",
			"vendor":       "Cloudflare, Inc.",
			"upgrade_code": warpUpgradeCode,
		},
	}

	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		&fleet.Host{ID: host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// confirm software row now in the database and `upgrade_code`s match for that row, the associated
	// `software_titles` row, and the associated `software_installers` row
	var swTitleId uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &swTitleId, "SELECT title_id FROM software WHERE upgrade_code = ?", warpUpgradeCode)
	})
	require.Equal(t, title.ID, swTitleId)
	require.Equal(t, *warpInstaller.TitleID, swTitleId)

	// GET host software endpoint, confirm upgrade_code is present
	var hSWRes getHostSoftwareResponse
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID),
		getHostSoftwareRequest{},
		http.StatusOK, &hSWRes,
	)
	// should only be the single software and software title in the database
	require.Equal(t, 1, hSWRes.Count)
	require.Len(t, hSWRes.Software, 1)
	sw0 := hSWRes.Software[0]
	require.Equal(t, warpUpgradeCode, *sw0.UpgradeCode)

	// nuke the title so we can try this with a batch upload
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM software_titles WHERE upgrade_code = ?", warpUpgradeCode)
		return err
	})

	// Use the batch endpoint (GitOps) to add the Windows FMA
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{Slug: new("cloudflare-warp/windows"), SelfService: true},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)

	// Verify the upgrade code was correctly set on the software installer
	var installerUpgradeCode *string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerUpgradeCode, "SELECT upgrade_code FROM software_installers WHERE title_id = ?", packages[0].TitleID)
	})
	require.NotNil(t, installerUpgradeCode)
	require.Equal(t, warpUpgradeCode, *installerUpgradeCode)

	// Also verify the software title has the upgrade code
	var afterBatch listSoftwareTitlesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &afterBatch,
		"per_page", "1",
		"available_for_install", "true",
		"team_id", "0",
	)
	require.Len(t, afterBatch.SoftwareTitles, 1)
	require.Equal(t, warpUpgradeCode, *afterBatch.SoftwareTitles[0].UpgradeCode)
}

func (s *integrationEnterpriseTestSuite) TestFMAAutoUpdateCron() {
	t := s.T()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Each cached version carries its own patch query with the version baked in, as real FMA manifests do.
	const warpPatchQueryFmt = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Cloudflare WARP' AND version_compare(version, '%s') < 0);"
	warpQueryV1 := fmt.Sprintf(warpPatchQueryFmt, "1.0")
	warpQueryV2 := fmt.Sprintf(warpPatchQueryFmt, "2.0")

	// Mock the FMA manifest + installer CDN via the shared helper. The state is
	// mutable: bumping warp.version/installerBytes (+ ComputeSHA) below simulates a
	// newly published upstream version on the next cron run.
	const slug = "cloudflare-warp/windows"
	// Non-ASCII on purpose: the migration hashes script_contents in MySQL against
	// hashes taken over the manifest text, which only agree if the bytes survive.
	const manifestScript = "#!/bin/sh\n# café — naïve ✓ 你好\ninstaller -pkg \"$TMPDIR/cloudflare-warp.msi\" -target /\n"
	warp := &fmaTestState{
		version: "1.0", installerBytes: []byte("abc"), installerPath: "/cloudflare-warp.msi",
		patchQuery: warpQueryV1, installScript: manifestScript,
	}
	startFMAServers(t, s.ds, map[string]*fmaTestState{"/" + slug + ".json": warp})

	// --- helpers ---
	setManifest := func(version string, b []byte) {
		warp.version = version
		warp.installerBytes = b
		warp.patchQuery = fmt.Sprintf(warpPatchQueryFmt, version)
		warp.ComputeSHA(b)
	}
	runCron := func() {
		require.NoError(t, eeservice.AutoUpdateFleetMaintainedApps(ctx, s.ds, s.softwareInstallStore, logger))
	}
	activeTitle := func(teamID uint) fleet.SoftwareTitleListResult {
		var resp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp,
			"per_page", "1", "order_key", "name", "order_direction", "desc",
			"available_for_install", "true", "team_id", fmt.Sprintf("%d", teamID))
		require.Equal(t, 1, resp.Count)
		return resp.SoftwareTitles[0]
	}
	activeSelfService := func(teamID, titleID uint) bool {
		var ss bool
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &ss,
				`SELECT self_service FROM software_installers WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1`,
				teamID, titleID)
		})
		return ss
	}
	activeScripts := func(teamID, titleID uint) (install, uninstall string) {
		var row struct {
			Install   string `db:"install_script"`
			Uninstall string `db:"uninstall_script"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row, `
				SELECT inst.contents AS install_script, COALESCE(uninst.contents, '') AS uninstall_script
				FROM software_installers si
				LEFT JOIN script_contents inst ON inst.id = si.install_script_content_id
				LEFT JOIN script_contents uninst ON uninst.id = si.uninstall_script_content_id
				WHERE si.global_or_team_id = ? AND si.title_id = ? AND si.is_active = 1`, teamID, titleID)
		})
		return row.Install, row.Uninstall
	}
	// LENGTH is bytes, CHAR_LENGTH is characters. Comparing bytes against Go catches
	// a transcode that reading the text back through the same connection would hide.
	activeScriptBytes := func(teamID, titleID uint) (bytes int, chars int) {
		var row struct {
			Bytes int `db:"stored_bytes"`
			Chars int `db:"stored_chars"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row, `
				SELECT LENGTH(sc.contents) AS stored_bytes, CHAR_LENGTH(sc.contents) AS stored_chars
				FROM software_installers si
				JOIN script_contents sc ON sc.id = si.install_script_content_id
				WHERE si.global_or_team_id = ? AND si.title_id = ? AND si.is_active = 1`, teamID, titleID)
		})
		return row.Bytes, row.Chars
	}
	activeScriptEditedFlags := func(teamID, titleID uint) (install bool, uninstall bool) {
		var row struct {
			Install   bool `db:"install_script_edited"`
			Uninstall bool `db:"uninstall_script_edited"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row, `
				SELECT install_script_edited, uninstall_script_edited
				FROM software_installers
				WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1`, teamID, titleID)
		})
		return row.Install, row.Uninstall
	}
	setPin := func(teamID, titleID uint, pin string) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO software_title_team_pins (team_id, title_id, fleet_maintained_app_id, pinned_version)
				 SELECT ?, ?, fleet_maintained_app_id, ? FROM software_installers
				 WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1 AND fleet_maintained_app_id IS NOT NULL
				 ON DUPLICATE KEY UPDATE pinned_version = VALUES(pinned_version)`, teamID, titleID, pin, teamID, titleID)
			return err
		})
	}
	clearPin := func(teamID, titleID uint) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM software_title_team_pins WHERE team_id = ? AND title_id = ?`, teamID, titleID)
			return err
		})
	}

	// --- setup: team with cloudflare-warp v1.0, self-service on ---
	var teamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{
		Name: new("team_" + t.Name()),
	}, http.StatusOK, &teamResp)
	team := *teamResp.Team

	var batchResp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new(slug), SelfService: true}}, TeamName: team.Name},
		http.StatusAccepted, &batchResp, "team_name", team.Name, "team_id", fmt.Sprint(team.ID))
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchResp.RequestUUID)

	title := activeTitle(team.ID)
	require.Equal(t, "1.0", title.SoftwarePackage.Version)
	titleID := title.ID
	require.True(t, activeSelfService(team.ID, titleID))

	// A patch policy generated from the active v1.0 installer; the cron must keep its query on the active version.
	var patchPolicy fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Type: new("patch"), PatchSoftwareTitleID: &titleID,
	}, http.StatusOK, &patchPolicy)
	patchPolicyQuery := func() string {
		var resp fleet.GetTeamPolicyByIDResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", team.ID, patchPolicy.Policy.ID), nil, http.StatusOK, &resp)
		return resp.Policy.Query
	}
	require.Equal(t, warpQueryV1, patchPolicyQuery())

	// The manifest script has to survive the add byte for byte, or the backfill
	// migration can't recognize it as one Fleet published.
	gotInstall, _ := activeScripts(team.ID, titleID)
	require.Equal(t, manifestScript, gotInstall)
	storedBytes, storedChars := activeScriptBytes(team.ID, titleID)
	require.Equal(t, len(manifestScript), storedBytes)
	require.Greater(t, storedBytes, storedChars, "fixture lost its multi-byte characters")

	// === Section A: unpinned advances to the newly published version ===
	setManifest("2.0", []byte("def"))
	runCron()
	title = activeTitle(team.ID)
	require.Equal(t, "2.0", title.SoftwarePackage.Version)
	require.Len(t, title.SoftwarePackage.FleetMaintainedVersions, 2)
	// Per-team config is carried forward onto the new version.
	require.True(t, activeSelfService(team.ID, titleID), "self-service must survive the auto-update")
	// The cron flip must rewrite the patch policy query to the newly active 2.0 version.
	require.Equal(t, warpQueryV2, patchPolicyQuery())

	// The cached bytes are real and installable: a host install serves v2.0 bytes.
	host := createOrbitEnrolledHost(t, "windows", "orbit-autoupdate", s.ds)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)
	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID, `
			SELECT software_installer_id FROM host_software_installs
			WHERE host_id = ? AND software_installer_id IS NOT NULL AND install_script_exit_code IS NULL
			ORDER BY created_at DESC LIMIT 1`, host.ID)
	})
	r := s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
		InstallerID: installerID, OrbitNodeKey: *host.OrbitNodeKey,
	}, http.StatusOK)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	r.Body.Close()
	require.Equal(t, []byte("def"), body, "auto-updated installer should serve v2.0 bytes")

	// === Section B: a literal pin blocks auto-advance ===
	setPin(team.ID, titleID, "2.0")
	setManifest("3.0", []byte("ghi"))
	runCron()
	require.Equal(t, "2.0", activeTitle(team.ID).SoftwarePackage.Version, "literal pin must not advance")

	// === Section C: caret pin advances within major, never across ===
	clearPin(team.ID, titleID)
	runCron() // unpinned: advance to the published 3.0
	require.Equal(t, "3.0", activeTitle(team.ID).SoftwarePackage.Version)

	setPin(team.ID, titleID, "^3")
	setManifest("3.1", []byte("j31"))
	runCron()
	require.Equal(t, "3.1", activeTitle(team.ID).SoftwarePackage.Version, "caret pin advances within its major")

	setManifest("4.0", []byte("k40"))
	runCron()
	require.Equal(t, "3.1", activeTitle(team.ID).SoftwarePackage.Version, "caret pin must not cross major")

	// === Section D: a re-run with nothing new is a no-op ===
	runCron()
	require.Equal(t, "3.1", activeTitle(team.ID).SoftwarePackage.Version)

	// === Section E: admin-customized scripts are carried forward on auto-update ===
	clearPin(team.ID, titleID)
	const customInstall = "echo custom-install"
	const customUninstall = "echo custom-uninstall"
	var batchRespE batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
			{Slug: new(slug), SelfService: true, InstallScript: customInstall, UninstallScript: customUninstall},
		}, TeamName: team.Name},
		http.StatusAccepted, &batchRespE, "team_name", team.Name, "team_id", fmt.Sprint(team.ID))
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchRespE.RequestUUID)

	gotInstall, gotUninstall := activeScripts(team.ID, titleID)
	require.Equal(t, customInstall, gotInstall)
	require.Equal(t, customUninstall, gotUninstall)

	// Scripts spelled out in the batch payload are flagged as edited, so they survive
	// a newly published version rather than reverting to the manifest's.
	editedInstall, editedUninstall := activeScriptEditedFlags(team.ID, titleID)
	require.True(t, editedInstall)
	require.True(t, editedUninstall)

	setManifest("5.0", []byte("v50"))
	runCron()
	require.Equal(t, "5.0", activeTitle(team.ID).SoftwarePackage.Version)
	gotInstall, gotUninstall = activeScripts(team.ID, titleID)
	require.Equal(t, customInstall, gotInstall, "custom install script must survive auto-update")
	require.Equal(t, customUninstall, gotUninstall, "custom uninstall script must survive auto-update")
	editedInstall, editedUninstall = activeScriptEditedFlags(team.ID, titleID)
	require.True(t, editedInstall, "the flags carry onto the new version")
	require.True(t, editedUninstall, "the flags carry onto the new version")

	// === Section F: dropping the scripts from the batch payload hands the app back to Fleet ===
	var batchRespF batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
			{Slug: new(slug), SelfService: true},
		}, TeamName: team.Name},
		http.StatusAccepted, &batchRespF, "team_name", team.Name, "team_id", fmt.Sprint(team.ID))
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchRespF.RequestUUID)
	editedInstall, editedUninstall = activeScriptEditedFlags(team.ID, titleID)
	require.False(t, editedInstall)
	require.False(t, editedUninstall)

	// The manifest scripts now win on the next published version.
	setManifest("6.0", []byte("v60"))
	runCron()
	require.Equal(t, "6.0", activeTitle(team.ID).SoftwarePackage.Version)
	gotInstall, gotUninstall = activeScripts(team.ID, titleID)
	require.NotEqual(t, customInstall, gotInstall, "an unedited install script follows the manifest")
	require.NotEqual(t, customUninstall, gotUninstall, "an unedited uninstall script follows the manifest")

	// === Section G: the update endpoint writes the flags, one script at a time ===
	const patchedInstall = "echo patched-install"
	activeInstallerID := func(teamID, titleID uint) uint {
		var id uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `
				SELECT id FROM software_installers
				WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1`, teamID, titleID)
		})
		return id
	}
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		TeamID:        &team.ID,
		InstallerID:   activeInstallerID(team.ID, titleID),
		InstallScript: new(patchedInstall),
		SelfService:   new(true),
	}, http.StatusOK, "")

	// Read back from MySQL, not from the payload the service handed the datastore.
	editedInstall, editedUninstall = activeScriptEditedFlags(team.ID, titleID)
	require.True(t, editedInstall, "the update endpoint marks the script it replaced")
	require.False(t, editedUninstall, "the untouched script stays unmarked")

	// The install script it just wrote survives the next version, the uninstall doesn't.
	gotInstall, gotUninstall = activeScripts(team.ID, titleID)
	require.Equal(t, patchedInstall, gotInstall)
	setManifest("7.0", []byte("v70"))
	runCron()
	require.Equal(t, "7.0", activeTitle(team.ID).SoftwarePackage.Version)
	patchedUninstall := gotUninstall
	gotInstall, gotUninstall = activeScripts(team.ID, titleID)
	require.Equal(t, patchedInstall, gotInstall, "a script the update endpoint marked survives auto-update")
	require.Equal(t, patchedUninstall, gotUninstall, "the unmarked script follows the manifest")
	editedInstall, editedUninstall = activeScriptEditedFlags(team.ID, titleID)
	require.True(t, editedInstall)
	require.False(t, editedUninstall)

	// A manifest can change a script or a query without publishing a new version.
	// There is nothing to download then, and the version can't be cached a second
	// time, so the row that is already there is refreshed in place.
	cachedRows := func() (count int) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count,
				`SELECT COUNT(*) FROM software_installers WHERE global_or_team_id = ? AND title_id = ?`, team.ID, titleID)
		})
		return count
	}
	activePatchQuery := func() (query string) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &query,
				`SELECT patch_query FROM software_installers WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1`,
				team.ID, titleID)
		})
		return query
	}
	rowsBefore := cachedRows()
	installerBefore := activeInstallerID(team.ID, titleID)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)

	warp.installScript = "#!/bin/sh\necho refreshed\n"
	warp.patchQuery = fmt.Sprintf(warpPatchQueryFmt, "7.0.1")
	runCron()

	require.Equal(t, "7.0", activeTitle(team.ID).SoftwarePackage.Version, "no version was published")
	require.Equal(t, rowsBefore, cachedRows(), "the version can't be cached twice")
	require.Equal(t, installerBefore, activeInstallerID(team.ID, titleID), "refreshed in place")
	gotInstall, gotUninstall = activeScripts(team.ID, titleID)
	require.Equal(t, patchedInstall, gotInstall, "the edited script is still left alone")
	require.Equal(t, warp.installScript, gotUninstall, "the unedited script follows the manifest")
	require.Equal(t, warp.patchQuery, activePatchQuery(), "the patch query follows the manifest too")

	// The queued install would otherwise have run a script it was never queued against.
	var canceled bool
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &canceled, `SELECT canceled FROM host_software_installs
			WHERE host_id = ? ORDER BY created_at DESC LIMIT 1`, host.ID)
	})
	require.True(t, canceled)

	// Nothing changed upstream, so a second pass leaves the scripts alone and the
	// install queued against them keeps running.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)
	runCron()
	gotInstall, gotUninstall = activeScripts(team.ID, titleID)
	require.Equal(t, patchedInstall, gotInstall)
	require.Equal(t, warp.installScript, gotUninstall)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &canceled, `SELECT canceled FROM host_software_installs
			WHERE host_id = ? ORDER BY created_at DESC LIMIT 1`, host.ID)
	})
	require.False(t, canceled)
}

func (s *integrationEnterpriseTestSuite) TestFMAAutoUpdateCronTimeBudget() {
	t := s.T()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Four Fleet-maintained apps behind a CDN that is slow but reachable, so each installer download eats into the run.
	const downloadDelay = 400 * time.Millisecond
	slugs := []string{"cloudflare-warp/windows", "zoom/windows", "1password/windows", "notion/windows"}
	states := make(map[string]*fmaTestState, len(slugs))
	for i, slug := range slugs {
		states["/"+slug+".json"] = &fmaTestState{
			version:        "1.0",
			installerBytes: []byte(fmt.Sprintf("v1-%d", i)),
			installerPath:  fmt.Sprintf("/fma-%d.msi", i),
			installerDelay: downloadDelay,
		}
	}
	startFMAServers(t, s.ds, states)

	activeVersions := func(teamID uint) []string {
		var versions []string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &versions,
				`SELECT version FROM software_installers WHERE global_or_team_id = ? AND is_active = 1 ORDER BY title_id`, teamID)
		})
		return versions
	}

	var teamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{
		Name: new("team_" + t.Name()),
	}, http.StatusOK, &teamResp)
	team := *teamResp.Team

	software := make([]*fleet.SoftwareInstallerPayload, 0, len(slugs))
	for _, slug := range slugs {
		software = append(software, &fleet.SoftwareInstallerPayload{Slug: new(slug)})
	}
	var batchResp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: software, TeamName: team.Name},
		http.StatusAccepted, &batchResp, "team_name", team.Name, "team_id", fmt.Sprint(team.ID))
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchResp.RequestUUID)
	require.Equal(t, []string{"1.0", "1.0", "1.0", "1.0"}, activeVersions(team.ID))

	// Every app publishes a new version at once, so each app in the run needs its own slow download.
	for i, slug := range slugs {
		state := &fmaTestState{
			version:        "2.0",
			installerBytes: []byte(fmt.Sprintf("v2-%d", i)),
			installerPath:  fmt.Sprintf("/fma-%d.msi", i),
			installerDelay: downloadDelay,
		}
		state.ComputeSHA(state.installerBytes)
		states["/"+slug+".json"] = state
	}

	// A run whose budget covers about one download must give up when the budget is gone instead of walking the whole list.
	budgetedCtx, cancel := context.WithTimeout(ctx, downloadDelay+downloadDelay/2)
	defer cancel()
	err := eeservice.AutoUpdateFleetMaintainedApps(budgetedCtx, s.ds, s.softwareInstallStore, logger)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, activeVersions(team.ID), "1.0", "the run stopped early, so at least one app stays on the old version")

	// The cancellation wedges nothing: the next run, with room to finish, advances every app.
	require.NoError(t, eeservice.AutoUpdateFleetMaintainedApps(ctx, s.ds, s.softwareInstallStore, logger))
	require.Equal(t, []string{"2.0", "2.0", "2.0", "2.0"}, activeVersions(team.ID))
}

func (s *integrationEnterpriseTestSuite) TestFMAVersionRollback() {
	t := s.T()
	ctx := context.Background()

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

	// Mock installer server — routes by path to serve per-FMA bytes.
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadMu.Lock()
		defer downloadMu.Unlock()
		switch r.URL.Path {
		case "/cloudflare-warp.msi":
			downloadedSlugs["cloudflare-warp/windows"] = true
			_, _ = w.Write(warpState.installerBytes)
		case "/zoom.msi":
			downloadedSlugs["zoom/windows"] = true
			_, _ = w.Write(zoomState.installerBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer installerServer.Close()

	// Non-existent maintained app
	s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 1}, http.StatusNotFound)

	// Insert the list of maintained apps
	maintainedappstest.SyncApps(t, s.ds)

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
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	}))
	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)
	defer dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	// -------------------------------------------------------------------------
	// Sub-test helper: fetch the active software title for a team by FMA name.
	// -------------------------------------------------------------------------
	getActiveTitleForTeam := func(teamID uint) fleet.SoftwareTitleListResult {
		var resp listSoftwareTitlesResponse
		s.DoJSON(
			"GET", "/api/latest/fleet/software/titles",
			listSoftwareTitlesRequest{},
			http.StatusOK, &resp,
			"per_page", "1",
			"order_key", "name",
			"order_direction", "desc",
			"available_for_install", "true",
			"team_id", fmt.Sprintf("%d", teamID),
		)
		require.Equal(t, 1, resp.Count)
		return resp.SoftwareTitles[0]
	}

	// -------------------------------------------------------------------------
	// Shared helpers used across sub-tests.
	// -------------------------------------------------------------------------

	// newTeam creates a new team with the given name and returns it.
	newTeam := func(name string) fleet.Team {
		var resp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{
			Name: new(name),
		}, http.StatusOK, &resp)
		return *resp.Team
	}

	// resetFMAState resets an fmaTestState to the given version and installer bytes.
	resetFMAState := func(state *fmaTestState, version string, installerBytes []byte) {
		state.version = version
		state.installerBytes = installerBytes
		state.sha256 = computeSHA(installerBytes)
	}

	// batchSet issues a batch-set request for the given team and software slice,
	// waits for completion, and returns the resulting packages.
	batchSet := func(team fleet.Team, software []*fleet.SoftwareInstallerPayload) []fleet.SoftwarePackageResponse {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: software, TeamName: team.Name},
			http.StatusAccepted, &resp,
			"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
		)
		return waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)
	}

	// newHostInTeam creates an orbit-enrolled host and assigns it to the given team.
	newHostInTeam := func(platform, name string, team fleet.Team) *fleet.Host {
		host := createOrbitEnrolledHost(t, platform, name, s.ds)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
		return host
	}

	// fmaAppID returns the fleet_maintained_apps.id for the given slug.
	fmaAppID := func(slug string) uint {
		var id uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, "SELECT id FROM fleet_maintained_apps WHERE slug = ?", slug)
		})
		require.NotZero(t, id)
		return id
	}

	// =========================================================================
	// Section 1 (existing): Batch-set / GitOps flow for cloudflare-warp/windows
	// =========================================================================

	team := newTeam("team_" + t.Name())

	// Add an ingested app to the team
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{Slug: new("cloudflare-warp/windows"), SelfService: true},
	}

	packages := batchSet(team, softwareToInstall)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)

	title := getActiveTitleForTeam(team.ID)
	require.Equal(t, "1.0", title.SoftwarePackage.Version)
	require.Equal(t, "cloudflare-warp.msi", title.SoftwarePackage.Name)

	// With only one version, fleet_maintained_versions should list it
	require.Len(t, title.SoftwarePackage.FleetMaintainedVersions, 1)
	require.Equal(t, "1.0", title.SoftwarePackage.FleetMaintainedVersions[0].Version)

	// Now add a second version
	resetFMAState(warpState, "2.0", []byte("def"))

	packages = batchSet(team, softwareToInstall)
	require.Len(t, packages, 2)

	title = getActiveTitleForTeam(team.ID)
	require.Equal(t, "2.0", title.SoftwarePackage.Version)
	require.Equal(t, "cloudflare-warp.msi", title.SoftwarePackage.Name)

	// fleet_maintained_versions should list both versions (newest first)
	require.Len(t, title.SoftwarePackage.FleetMaintainedVersions, 2)
	require.Equal(t, "2.0", title.SoftwarePackage.FleetMaintainedVersions[0].Version)
	require.Equal(t, "1.0", title.SoftwarePackage.FleetMaintainedVersions[1].Version)

	// We should have the previous version cached in the DB
	installers, err := s.ds.GetSoftwareInstallers(ctx, team.ID)
	s.Require().NoError(err)
	s.Len(installers, 2)

	// Now add a third version — should evict v1.0, keeping only v3.0 and v2.0
	resetFMAState(warpState, "3.0", []byte("ghi"))

	packages = batchSet(team, softwareToInstall)
	require.Len(t, packages, 2)

	title = getActiveTitleForTeam(team.ID)
	require.Equal(t, "3.0", title.SoftwarePackage.Version)

	// Only 2 versions should remain after eviction (max 2 cached)
	require.Len(t, title.SoftwarePackage.FleetMaintainedVersions, 2)
	require.Equal(t, "3.0", title.SoftwarePackage.FleetMaintainedVersions[0].Version)
	require.Equal(t, "2.0", title.SoftwarePackage.FleetMaintainedVersions[1].Version)

	// DB should also only have 2 installers
	installers, err = s.ds.GetSoftwareInstallers(ctx, team.ID)
	s.Require().NoError(err)
	s.Len(installers, 2)

	// ---- Test version rollback via fleet_maintained_app_version ----
	// Pin to version "2.0" in the batch request (simulating GitOps yaml with version specified).
	// The active installer should switch to the cached v2.0 installer.
	packages = batchSet(team, []*fleet.SoftwareInstallerPayload{
		{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "2.0"},
	})
	require.Len(t, packages, 2)

	// The active version shown in the title should now be "2.0"
	title = getActiveTitleForTeam(team.ID)
	require.Equal(t, "2.0", title.SoftwarePackage.Version)

	// Both versions should still be listed
	require.Len(t, title.SoftwarePackage.FleetMaintainedVersions, 2)

	// Create a host in the team, trigger an install, and verify the downloaded bytes match v2.0
	hostInTeam := newHostInTeam("windows", "orbit-host-rollback", team)

	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, title.ID), installSoftwareRequest{}, http.StatusAccepted)

	// Get the queued installer ID from the pending install record
	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID, `
			SELECT software_installer_id FROM host_software_installs
			WHERE host_id = ? AND software_installer_id IS NOT NULL AND install_script_exit_code IS NULL
			ORDER BY created_at DESC LIMIT 1
		`, hostInTeam.ID)
	})

	// Download the installer via orbit — should get the v2.0 bytes ("def")
	r := s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
		InstallerID:  installerID,
		OrbitNodeKey: *hostInTeam.OrbitNodeKey,
	}, http.StatusOK)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	r.Body.Close()
	// v2.0 installer bytes were []byte("def")
	require.Equal(t, []byte("def"), body, "downloaded installer should match v2.0 bytes")

	downloadMu.Lock()
	delete(downloadedSlugs, "cloudflare-warp/windows")
	downloadMu.Unlock()

	// ---- Test switching active version back to 3.0 ----
	// Pin to version "3.0" — the active installer should switch to the cached v3.0 installer.
	packages = batchSet(team, []*fleet.SoftwareInstallerPayload{
		{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "3.0"},
	})
	require.Len(t, packages, 2)

	// The active version shown in the title should now be "3.0"
	title = getActiveTitleForTeam(team.ID)
	require.Equal(t, "3.0", title.SoftwarePackage.Version)

	// Both versions should still be listed
	require.Len(t, title.SoftwarePackage.FleetMaintainedVersions, 2)

	// Trigger an install and verify the downloaded bytes match v3.0
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, title.ID), installSoftwareRequest{}, http.StatusAccepted)

	// Get the queued installer ID from the pending install record
	var installerIDv3 uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerIDv3, `
			SELECT software_installer_id FROM host_software_installs
			WHERE host_id = ? AND software_installer_id IS NOT NULL AND install_script_exit_code IS NULL
			ORDER BY created_at DESC LIMIT 1
		`, hostInTeam.ID)
	})

	// Download the installer via orbit — should get the v3.0 bytes ("ghi")
	r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
		InstallerID:  installerIDv3,
		OrbitNodeKey: *hostInTeam.OrbitNodeKey,
	}, http.StatusOK)
	body, err = io.ReadAll(r.Body)
	require.NoError(t, err)
	r.Body.Close()
	// v3.0 installer bytes were []byte("ghi")
	require.Equal(t, []byte("ghi"), body, "downloaded installer should match v3.0 bytes")

	// We have version 3.0 cached, so we should not have downloaded the bytes
	downloadMu.Lock()
	warpDownloaded := downloadedSlugs["cloudflare-warp/windows"]
	downloadMu.Unlock()
	s.False(warpDownloaded)

	// ---- Test rollback to an evicted version (v1.0) ----
	// v1.0 was evicted when v3.0 was added (max 2 cached: v3.0 + v2.0).
	// Per spec, pinning to an evicted version should fail with an error —
	// Fleet should NOT re-download it.
	downloadMu.Lock()
	delete(downloadedSlugs, "cloudflare-warp/windows")
	downloadMu.Unlock()

	var evictedVersionBatchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{
			Software: []*fleet.SoftwareInstallerPayload{
				{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "1.0"},
			},
			TeamName: team.Name,
		},
		http.StatusAccepted, &evictedVersionBatchResponse, "team_name", team.Name, "team_id", fmt.Sprint(team.ID))
	evictedVersionMessage := waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team.Name, evictedVersionBatchResponse.RequestUUID)
	require.Contains(t, evictedVersionMessage, "specified version is not available")

	// Should NOT have hit the installer server — no re-download attempt
	downloadMu.Lock()
	warpDownloaded = downloadedSlugs["cloudflare-warp/windows"]
	downloadMu.Unlock()
	require.False(t, warpDownloaded, "should not attempt to re-download an evicted version")

	// The active version should still be "3.0" (unchanged)
	title = getActiveTitleForTeam(team.ID)
	require.Equal(t, "3.0", title.SoftwarePackage.Version, "active version should remain 3.0 after failed rollback to evicted version")

	// ---- Test bumping a pin to a newly-published version ----
	// A version published upstream but not yet cached must be downloadable in a
	// single apply when pinned to directly. Previously this failed with
	// "specified version is not available" because a literal pin only resolved
	// against the already-cached set.
	resetFMAState(warpState, "4.0", []byte("jkl")) // publish 4.0 upstream; not yet cached
	downloadMu.Lock()
	delete(downloadedSlugs, "cloudflare-warp/windows")
	downloadMu.Unlock()

	packages = batchSet(team, []*fleet.SoftwareInstallerPayload{
		{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "4.0"},
	})
	require.Len(t, packages, 2)

	// 4.0 is downloaded, cached, and becomes the active version in one apply.
	downloadMu.Lock()
	warpDownloaded = downloadedSlugs["cloudflare-warp/windows"]
	downloadMu.Unlock()
	require.True(t, warpDownloaded, "a newly-published pinned version must be downloaded")

	title = getActiveTitleForTeam(team.ID)
	require.Equal(t, "4.0", title.SoftwarePackage.Version)
	require.Equal(t, "4.0", title.SoftwarePackage.FleetMaintainedVersions[0].Version)

	// Attempt to add a custom package that will map to the same software title. Should fail
	// (this tests the "custom installer vs existing FMA" direction).
	installerContent := "installerbytes"
	installerFile, err := fleet.NewTempFileReader(strings.NewReader(installerContent), t.TempDir)
	require.NoError(t, err)
	defer installerFile.Close()

	user, err := s.ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)

	customPayload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:          &team.ID,
		Filename:        "cloudflare-warp-installer.msi",
		InstallerFile:   installerFile,
		Version:         "99.0.0",
		StorageID:       computeSHA([]byte(installerContent)),
		SelfService:     true,
		Title:           "Cloudflare One Client", // windows installer, so Fleet does software title matching on this field
		InstallScript:   "install",
		UninstallScript: "uninstall",
		Source:          "programs",
		Platform:        "windows",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		UserID:          user.ID,
	}

	// We call the datastore method directly here because
	// the service layer would attempt to extract metadata from the
	// "installer", but that's just a fake file in this case so it'd fail.
	// That logic isn't what we're trying to test here.
	_, _, err = s.ds.MatchOrCreateSoftwareInstaller(ctx, customPayload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf(fleet.SoftwareAlreadyHasFleetMaintainedAppMessage, customPayload.Title, team.Name))

	// =========================================================================
	// Section 2: UI single-add flow
	//
	// The single-add endpoint (POST /fleet_maintained_apps) is used when a
	// Fleet admin clicks "Add" in the UI, as opposed to the GitOps batch-set
	// flow.  This verifies that:
	//   a) The endpoint correctly installs v1.0 from scratch.
	//   b) A subsequent batch-set upgrade to v2.0 caches both versions.
	//   c) Rolling back to the UI-added v1.0 via batch-set works correctly.
	// =========================================================================
	t.Run("ui_single_add_flow", func(t *testing.T) {
		// Reset warp state to v1.0 for this sub-test.
		resetFMAState(warpState, "1.0", []byte("abc"))

		// Create a fresh team so this sub-test is isolated from Section 1.
		uiTeam := newTeam("team_ui_" + t.Name())

		// Look up the cloudflare-warp/windows FMA ID that was inserted by SyncApps.
		warpAppID := fmaAppID("cloudflare-warp/windows")

		// Add the FMA via the single-add (UI) endpoint.
		var addMAResp addFleetMaintainedAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{
			AppID:       warpAppID,
			TeamID:      &uiTeam.ID,
			SelfService: true,
		}, http.StatusOK, &addMAResp)
		require.NoError(t, addMAResp.Err)

		// Verify v1.0 was installed and is the only (active) version.
		uiTitle := getActiveTitleForTeam(uiTeam.ID)
		require.Equal(t, "1.0", uiTitle.SoftwarePackage.Version)
		require.Len(t, uiTitle.SoftwarePackage.FleetMaintainedVersions, 1)
		require.Equal(t, "1.0", uiTitle.SoftwarePackage.FleetMaintainedVersions[0].Version)

		uiInstallers, err := s.ds.GetSoftwareInstallers(ctx, uiTeam.ID)
		require.NoError(t, err)
		require.Len(t, uiInstallers, 1)

		// Now bump to v2.0 via the batch-set (GitOps) flow — this is how an
		// upgrade would arrive after the initial UI-add.
		resetFMAState(warpState, "2.0", []byte("def"))

		pkgs := batchSet(uiTeam, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true},
		})
		require.Len(t, pkgs, 2, "both v1.0 (UI-added) and v2.0 should be cached")

		uiTitle = getActiveTitleForTeam(uiTeam.ID)
		require.Equal(t, "2.0", uiTitle.SoftwarePackage.Version)
		require.Len(t, uiTitle.SoftwarePackage.FleetMaintainedVersions, 2)

		uiInstallers, err = s.ds.GetSoftwareInstallers(ctx, uiTeam.ID)
		require.NoError(t, err)
		require.Len(t, uiInstallers, 2)

		// Roll back to the original UI-added v1.0 via batch-set.
		pkgs = batchSet(uiTeam, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "1.0"},
		})
		require.Len(t, pkgs, 2, "both versions should still be cached after rollback")

		// Active version should now be v1.0 (the one originally added via the UI).
		uiTitle = getActiveTitleForTeam(uiTeam.ID)
		require.Equal(t, "1.0", uiTitle.SoftwarePackage.Version,
			"active version should roll back to v1.0 (the version added via UI)")
		require.Len(t, uiTitle.SoftwarePackage.FleetMaintainedVersions, 2,
			"both versions should remain cached after rolling back")
		// Pinning picks which cached version is active without rewriting which one is
		// newest, so clearing the pin later still resolves Latest to v2.0.
		require.Equal(t, "2.0", uiTitle.SoftwarePackage.FleetMaintainedVersions[0].Version,
			"the pinned older version must not become the newest download")
	})

	// =========================================================================
	// Section 3: FMA fails when a custom installer already exists for the same
	// software title on the team (reverse of the conflict test in Section 1).
	// =========================================================================
	t.Run("fma_conflicts_with_existing_custom_installer", func(t *testing.T) {
		// Reset warp state back to v1.0.
		resetFMAState(warpState, "1.0", []byte("abc"))

		conflictTeam := newTeam("team_conflict_" + t.Name())

		user, err := s.ds.UserByEmail(ctx, "admin1@example.com")
		require.NoError(t, err)

		// Add a custom (non-FMA) installer for "Zoom Workplace (X64)" — the same
		// unique_identifier that the zoom/windows FMA would use.
		customInstallerFile, err := fleet.NewTempFileReader(strings.NewReader("fake-zoom-installer"), t.TempDir)
		require.NoError(t, err)
		defer customInstallerFile.Close()

		// We call the datastore directly because the service layer tries to parse
		// the installer binary (which is fake here).
		_, _, err = s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			TeamID:          &conflictTeam.ID,
			Filename:        "zoom-custom.msi",
			InstallerFile:   customInstallerFile,
			Version:         "1.0.0",
			StorageID:       computeSHA([]byte("fake-zoom-installer")),
			SelfService:     false,
			Title:           "Zoom Workplace (X64)", // matches zoom/windows FMA unique_identifier
			InstallScript:   "install",
			UninstallScript: "uninstall",
			Source:          "programs",
			Platform:        "windows",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			UserID:          user.ID,
		})
		require.NoError(t, err, "custom installer should be created without error")

		// Now try to add the zoom/windows FMA via the single-add (UI) endpoint.
		// It should fail because a custom installer already occupies the same
		// software title on this team.
		zoomAppID := fmaAppID("zoom/windows")

		conflictResp := s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps",
			&addFleetMaintainedAppRequest{
				AppID:  zoomAppID,
				TeamID: &conflictTeam.ID,
			},
			http.StatusConflict,
		)
		errMsg := extractServerErrorText(conflictResp.Body)
		require.Contains(t, errMsg, "already has a software package",
			"error should mention the conflict with the existing custom installer")

		// Confirm the FMA was NOT added — only the original custom installer exists.
		conflictInstallers, err := s.ds.GetSoftwareInstallers(ctx, conflictTeam.ID)
		require.NoError(t, err)
		require.Len(t, conflictInstallers, 1, "FMA should not have been added when a custom installer already exists")
	})

	// =========================================================================
	// Section 4: More than one FMA with cached versions on the same team.
	//
	// Both cloudflare-warp/windows and zoom/windows are added to the same team.
	// Each goes through v1.0 → v2.0 → v3.0 (evicting v1.0 for each), then both
	// are independently rolled back to v2.0.  Verifies that eviction and rollback
	// logic is per-FMA, not cross-FMA.
	// =========================================================================
	t.Run("multiple_fmas_cached_versions", func(t *testing.T) {
		// Reset both FMA states to v1.0.
		resetFMAState(warpState, "1.0", []byte("warp-v1"))
		resetFMAState(zoomState, "1.0", []byte("zoom-v1"))

		multiTeam := newTeam("team_multi_" + t.Name())

		bothFMAs := []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true},
			{Slug: new("zoom/windows"), SelfService: true},
		}

		// ---- v1.0 for both FMAs ----
		pkgs := batchSet(multiTeam, bothFMAs)
		require.Len(t, pkgs, 2, "both FMAs at v1.0")

		multiInstallers, err := s.ds.GetSoftwareInstallers(ctx, multiTeam.ID)
		require.NoError(t, err)
		require.Len(t, multiInstallers, 2)

		// ---- v2.0 for both FMAs ----
		resetFMAState(warpState, "2.0", []byte("warp-v2"))
		resetFMAState(zoomState, "2.0", []byte("zoom-v2"))

		pkgs = batchSet(multiTeam, bothFMAs)
		require.Len(t, pkgs, 4, "two cached versions for each of the two FMAs")

		multiInstallers, err = s.ds.GetSoftwareInstallers(ctx, multiTeam.ID)
		require.NoError(t, err)
		require.Len(t, multiInstallers, 4, "4 total installers: 2 per FMA")

		// ---- v3.0 for both FMAs — should evict v1.0 for each ----
		resetFMAState(warpState, "3.0", []byte("warp-v3"))
		resetFMAState(zoomState, "3.0", []byte("zoom-v3"))

		pkgs = batchSet(multiTeam, bothFMAs)
		// After eviction each FMA keeps 2 versions (v3.0 + v2.0), so 4 total.
		require.Len(t, pkgs, 4, "after eviction: 2 cached versions per FMA")

		multiInstallers, err = s.ds.GetSoftwareInstallers(ctx, multiTeam.ID)
		require.NoError(t, err)
		require.Len(t, multiInstallers, 4, "4 total installers after eviction of v1.0 for each FMA")

		// ---- Roll back cloudflare-warp to v2.0, leave zoom at v3.0 ----
		pkgs = batchSet(multiTeam, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "2.0"},
			{Slug: new("zoom/windows"), SelfService: true}, // no rollback — stays at latest
		})
		require.Len(t, pkgs, 4)

		// Collect active versions by title name for easy assertion.
		var multiTitlesResp listSoftwareTitlesResponse
		s.DoJSON(
			"GET", "/api/latest/fleet/software/titles",
			listSoftwareTitlesRequest{},
			http.StatusOK, &multiTitlesResp,
			"available_for_install", "true",
			"team_id", fmt.Sprintf("%d", multiTeam.ID),
		)
		require.Equal(t, 2, multiTitlesResp.Count)

		activeVersions := map[string]string{}
		for _, tl := range multiTitlesResp.SoftwareTitles {
			activeVersions[tl.Name] = tl.SoftwarePackage.Version
		}
		require.Equal(t, "2.0", activeVersions["Cloudflare One Client"],
			"warp should be rolled back to v2.0")
		require.Equal(t, "3.0", activeVersions["Zoom Workplace (X64)"],
			"zoom should remain at v3.0 (not rolled back)")

		// Zoom rollback to v2.0 should also succeed independently.
		pkgs = batchSet(multiTeam, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "2.0"},
			{Slug: new("zoom/windows"), SelfService: true, RollbackVersion: "2.0"},
		})
		require.Len(t, pkgs, 4)

		multiTitlesResp = listSoftwareTitlesResponse{}
		s.DoJSON(
			"GET", "/api/latest/fleet/software/titles",
			listSoftwareTitlesRequest{},
			http.StatusOK, &multiTitlesResp,
			"available_for_install", "true",
			"team_id", fmt.Sprintf("%d", multiTeam.ID),
		)
		require.Equal(t, 2, multiTitlesResp.Count)

		activeVersions = map[string]string{}
		for _, tl := range multiTitlesResp.SoftwareTitles {
			activeVersions[tl.Name] = tl.SoftwarePackage.Version
		}
		require.Equal(t, "2.0", activeVersions["Cloudflare One Client"], "warp should still be at v2.0")
		require.Equal(t, "2.0", activeVersions["Zoom Workplace (X64)"], "zoom should now be rolled back to v2.0")
	})

	// =========================================================================
	// Section 5: Team isolation — cached versions on one team must not affect
	// another team that holds the same FMA.
	//
	// Team A advances cloudflare-warp through v1.0 → v2.0 → v3.0 (evicts v1.0).
	// Team B advances only to v2.0 and should still have v1.0 cached afterward.
	// Rolling back Team A to v2.0 must not disturb Team B.
	// =========================================================================
	t.Run("team_isolation", func(t *testing.T) {
		// Reset warp state to v1.0.
		resetFMAState(warpState, "1.0", []byte("iso-warp-v1"))

		teamA := newTeam("team_iso_a_" + t.Name())
		teamB := newTeam("team_iso_b_" + t.Name())

		warpPayload := []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true},
		}

		// ---- Add v1.0 to both teams ----
		batchSet(teamA, warpPayload)
		batchSet(teamB, warpPayload)

		// Both teams should have exactly 1 installer (v1.0).
		instA, err := s.ds.GetSoftwareInstallers(ctx, teamA.ID)
		require.NoError(t, err)
		require.Len(t, instA, 1)

		instB, err := s.ds.GetSoftwareInstallers(ctx, teamB.ID)
		require.NoError(t, err)
		require.Len(t, instB, 1)

		// ---- Advance both teams to v2.0 ----
		resetFMAState(warpState, "2.0", []byte("iso-warp-v2"))

		batchSet(teamA, warpPayload)
		batchSet(teamB, warpPayload)

		// Both teams should have 2 cached installers (v1.0 + v2.0).
		instA, err = s.ds.GetSoftwareInstallers(ctx, teamA.ID)
		require.NoError(t, err)
		require.Len(t, instA, 2)

		instB, err = s.ds.GetSoftwareInstallers(ctx, teamB.ID)
		require.NoError(t, err)
		require.Len(t, instB, 2)

		// ---- Advance Team A to v3.0 (evicts v1.0 on Team A only) ----
		resetFMAState(warpState, "3.0", []byte("iso-warp-v3"))

		batchSet(teamA, warpPayload)

		// Team A: v1.0 evicted → only v2.0 + v3.0 remain.
		instA, err = s.ds.GetSoftwareInstallers(ctx, teamA.ID)
		require.NoError(t, err)
		require.Len(t, instA, 2, "Team A should still have 2 installers after v1.0 eviction")

		titleA := getActiveTitleForTeam(teamA.ID)
		require.Equal(t, "3.0", titleA.SoftwarePackage.Version, "Team A active version should be v3.0")
		require.Len(t, titleA.SoftwarePackage.FleetMaintainedVersions, 2)
		require.Equal(t, "3.0", titleA.SoftwarePackage.FleetMaintainedVersions[0].Version)
		require.Equal(t, "2.0", titleA.SoftwarePackage.FleetMaintainedVersions[1].Version)

		// Team B: must be completely unaffected — still has v1.0 + v2.0.
		instB, err = s.ds.GetSoftwareInstallers(ctx, teamB.ID)
		require.NoError(t, err)
		require.Len(t, instB, 2, "Team B should still have 2 installers; Team A's eviction must not affect Team B")

		titleB := getActiveTitleForTeam(teamB.ID)
		require.Equal(t, "2.0", titleB.SoftwarePackage.Version, "Team B active version should remain v2.0")
		require.Len(t, titleB.SoftwarePackage.FleetMaintainedVersions, 2)
		require.Equal(t, "2.0", titleB.SoftwarePackage.FleetMaintainedVersions[0].Version)
		require.Equal(t, "1.0", titleB.SoftwarePackage.FleetMaintainedVersions[1].Version,
			"Team B should still have v1.0 cached even though Team A evicted it")

		// ---- Roll back Team A to v2.0 — Team B must not be affected ----
		batchSet(teamA, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "2.0"},
		})

		titleA = getActiveTitleForTeam(teamA.ID)
		require.Equal(t, "2.0", titleA.SoftwarePackage.Version, "Team A should be rolled back to v2.0")

		// Team B still unaffected.
		titleB = getActiveTitleForTeam(teamB.ID)
		require.Equal(t, "2.0", titleB.SoftwarePackage.Version,
			"Team B active version should remain v2.0 after Team A rollback")
		require.Len(t, titleB.SoftwarePackage.FleetMaintainedVersions, 2,
			"Team B cached versions should be unchanged")

		// Verify Team B can still roll back to v1.0 (its eviction pool is independent).
		batchSet(teamB, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "1.0"},
		})

		titleB = getActiveTitleForTeam(teamB.ID)
		require.Equal(t, "1.0", titleB.SoftwarePackage.Version,
			"Team B should successfully roll back to v1.0, which is still in its cache")
	})

	// =========================================================================
	// Section 6: Deleting an FMA software title removes all cached versions,
	// not just the active one.
	//
	// An FMA is advanced through v1.0 → v2.0 → v3.0 so that two versions are
	// cached (v3.0 active, v2.0 inactive, v1.0 evicted).  After deleting the
	// software title the DB should contain zero software_installers rows for
	// that title+team.
	// =========================================================================
	t.Run("delete_fma_cleans_all_cached_versions", func(t *testing.T) {
		resetFMAState(warpState, "1.0", []byte("del-warp-v1"))

		delTeam := newTeam("team_del_" + t.Name())

		warpPayload := []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true},
		}

		batchSet(delTeam, warpPayload)

		// ---- v2.0 ----
		resetFMAState(warpState, "2.0", []byte("del-warp-v2"))

		batchSet(delTeam, warpPayload)

		// Confirm we have exactly 2 cached installers (v1.0 + v2.0).
		delInstallers, err := s.ds.GetSoftwareInstallers(ctx, delTeam.ID)
		require.NoError(t, err)
		require.Len(t, delInstallers, 2, "should have 2 cached FMA versions before delete")

		// Retrieve the title ID so we can call the delete endpoint.
		delTitle := getActiveTitleForTeam(delTeam.ID)
		require.Equal(t, "2.0", delTitle.SoftwarePackage.Version)

		// Delete via the API endpoint — this is the path exercised by the UI.
		s.Do("DELETE",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", delTitle.ID),
			nil, http.StatusNoContent,
			"team_id", fmt.Sprint(delTeam.ID),
		)

		// After deletion, zero software_installer rows should remain for this
		// title+team — both the active (v2.0) and the inactive cached (v1.0)
		// version must be gone.
		delInstallers, err = s.ds.GetSoftwareInstallers(ctx, delTeam.ID)
		require.NoError(t, err)
		require.Empty(t, delInstallers,
			"all cached FMA versions (active + inactive) should be deleted, not just the active one")

		// The software title should no longer appear in the available-for-install
		// list for the team.
		var titlesResp listSoftwareTitlesResponse
		s.DoJSON(
			"GET", "/api/latest/fleet/software/titles",
			listSoftwareTitlesRequest{},
			http.StatusOK, &titlesResp,
			"available_for_install", "true",
			"team_id", fmt.Sprintf("%d", delTeam.ID),
		)
		require.Zero(t, titlesResp.Count,
			"software title should no longer be available for install after deletion")
	})

	// =========================================================================
	// Section 7: Status summary resets to all-zeros when the active version
	// changes.
	//
	//   1. Add warp at v1.0, then advance to v2.0 (v2.0 active, v1.0 cached).
	//   2. Install v2.0 on a host and record a success → summary shows Installed: 1.
	//   3. Roll back to v1.0 via batch-set (v1.0 becomes the active installer).
	//   4. Assert the active installer's (v1.0) status summary is all zeros —
	//      the install history from v2.0 must not bleed over.
	// =========================================================================
	t.Run("status_summary_resets_on_version_switch", func(t *testing.T) {
		// Reset warp state to v1.0 for this sub-test.
		resetFMAState(warpState, "1.0", []byte("status-warp-v1"))

		statusTeam := newTeam("team_status_" + t.Name())

		warpPayload := []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true},
		}

		// ---- Step 1a: Add v1.0 ----
		batchSet(statusTeam, warpPayload)

		// ---- Step 1b: Advance to v2.0 ----
		resetFMAState(warpState, "2.0", []byte("status-warp-v2"))

		batchSet(statusTeam, warpPayload)

		statusTitle := getActiveTitleForTeam(statusTeam.ID)
		require.Equal(t, "2.0", statusTitle.SoftwarePackage.Version)
		titleID := statusTitle.ID

		// ---- Step 2: Install v2.0 on a host and record success ----
		statusHost := newHostInTeam("windows", "orbit-host-status-"+t.Name(), statusTeam)

		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", statusHost.ID, titleID),
			installSoftwareRequest{}, http.StatusAccepted)

		// Retrieve the execution UUID for the pending install.
		installUUID := getLatestSoftwareInstallExecID(t, s.ds, statusHost.ID)

		// Report a successful install via orbit.
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *statusHost.OrbitNodeKey, installUUID)), http.StatusNoContent)

		// Verify: v2.0 installer summary shows Installed: 1.
		var titleRespV2 getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil,
			http.StatusOK, &titleRespV2, "team_id", fmt.Sprint(statusTeam.ID))
		require.NotNil(t, titleRespV2.SoftwareTitle.SoftwarePackage.Status)
		require.Equal(t, uint(1), titleRespV2.SoftwareTitle.SoftwarePackage.Status.Installed,
			"v2.0 installer should show Installed: 1 after a successful install")

		// ---- Step 3: Roll back to v1.0 ----
		batchSet(statusTeam, []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: true, RollbackVersion: "1.0"},
		})

		statusTitle = getActiveTitleForTeam(statusTeam.ID)
		require.Equal(t, "1.0", statusTitle.SoftwarePackage.Version,
			"active version should be v1.0 after rollback")

		// ---- Step 4: Assert the active (v1.0) installer summary is all zeros ----
		// v1.0's software_installers row was never the target of any install
		// request, so its summary must be completely empty. The title ID is
		// stable across version switches, so we can query it directly.
		var titleRespV1 getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil,
			http.StatusOK, &titleRespV1, "team_id", fmt.Sprint(statusTeam.ID))
		require.NotNil(t, titleRespV1.SoftwareTitle.SoftwarePackage.Status)
		require.Equal(t, fleet.SoftwareInstallerStatusSummary{}, *titleRespV1.SoftwareTitle.SoftwarePackage.Status,
			"v1.0 installer status summary must be all zeros after switching from v2.0 — "+
				"install history from v2.0 must not carry over to the rolled-back version")
	})

	// =========================================================================
	// Section 8: Editing an FMA that has multiple cached versions must succeed.
	//
	// Regression test for the bug where SoftwareTitleByID counted ALL
	// software_installers rows for a title+team (active + inactive cached
	// versions) instead of only the active one.  That made
	// SoftwareInstallersCount > 1, which tripped the `!= 1` guard in
	// UpdateSoftwareInstaller and returned a false "no installers defined"
	// 400 error whenever an admin tried to edit an FMA that had any cached
	// inactive version.
	//
	// The fix is `AND si.is_active = TRUE` in the LEFT JOIN inside
	// SoftwareTitleByID so the count stays at exactly 1.
	// =========================================================================
	t.Run("edit_fma_with_cached_versions", func(t *testing.T) {
		resetFMAState(warpState, "1.0", []byte("edit-warp-v1"))

		editTeam := newTeam("team_edit_" + t.Name())

		warpPayload := []*fleet.SoftwareInstallerPayload{
			{Slug: new("cloudflare-warp/windows"), SelfService: false},
		}

		// ---- Add v1.0 ----
		batchSet(editTeam, warpPayload)

		// ---- Advance to v2.0 so there are now 2 cached rows (v1.0 inactive, v2.0 active) ----
		resetFMAState(warpState, "2.0", []byte("edit-warp-v2"))

		batchSet(editTeam, warpPayload)

		editTitle := getActiveTitleForTeam(editTeam.ID)
		require.Equal(t, "2.0", editTitle.SoftwarePackage.Version)

		// Confirm self_service starts as false before the edit.
		var titleRespBefore getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", editTitle.ID), nil,
			http.StatusOK, &titleRespBefore, "team_id", fmt.Sprint(editTeam.ID))
		require.False(t, titleRespBefore.SoftwareTitle.SoftwarePackage.SelfService,
			"self_service should be false before the edit")

		// Sanity-check: both cached versions are present in the DB.
		editInstallers, err := s.ds.GetSoftwareInstallers(ctx, editTeam.ID)
		require.NoError(t, err)
		require.Len(t, editInstallers, 2, "should have 2 cached FMA versions before attempting edit")

		// ---- Attempt a metadata-only edit (toggle self_service) ----
		// Without the fix, SoftwareTitleByID counts both installer rows and returns
		// SoftwareInstallersCount = 2, which makes UpdateSoftwareInstaller return a
		// 400 "no installers defined" error.  With the fix the count is 1 and the
		// edit succeeds.
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			TitleID:     editTitle.ID,
			TeamID:      &editTeam.ID,
			SelfService: new(true),
		}, http.StatusOK, "")

		// Confirm the edit actually took effect.
		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", editTitle.ID), nil,
			http.StatusOK, &titleResp, "team_id", fmt.Sprint(editTeam.ID))
		require.True(t, titleResp.SoftwareTitle.SoftwarePackage.SelfService,
			"self_service should be true after the edit")
	})

	// =========================================================================
	// Section 9: the manifest publishes a version lower than one already cached.
	// This happens when a maintained app is withdrawn upstream, as iMazing did
	// going from 3.5.5.0 back to 3.3.1.0. Fleet follows the manifest either way:
	// whether the withdrawn version is rebuilt with new bytes or comes back byte
	// for byte, it becomes active and is listed first.
	// =========================================================================
	cachedVersions := func(pkg *fleet.SoftwarePackageOrApp) []string {
		got := make([]string, 0, len(pkg.FleetMaintainedVersions))
		for _, v := range pkg.FleetMaintainedVersions {
			got = append(got, v.Version)
		}
		return got
	}

	// The active row itself, so a test can check which row is active and what bytes it
	// points at, not just that the version string reads correctly.
	activeInstaller := func(teamID uint, titleID uint) *fleet.SoftwareInstaller {
		meta, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &teamID, titleID, false)
		require.NoError(t, err)
		return meta
	}

	cronLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runCron := func() {
		require.NoError(t, eeservice.AutoUpdateFleetMaintainedApps(ctx, s.ds, s.softwareInstallStore, cronLogger))
	}

	t.Run("manifest_publishes_lower_version", func(t *testing.T) {
		lowerTeam := newTeam("team_lower_" + t.Name())
		applyManifest := func() []fleet.SoftwarePackageResponse {
			return batchSet(lowerTeam, []*fleet.SoftwareInstallerPayload{
				{Slug: new("cloudflare-warp/windows")},
			})
		}
		resetFMAState(warpState, "1.0", []byte("lower-1-0"))
		require.Len(t, applyManifest(), 1)
		pkg := getActiveTitleForTeam(lowerTeam.ID).SoftwarePackage
		require.Equal(t, "1.0", pkg.Version)
		require.Equal(t, []string{"1.0"}, cachedVersions(pkg))

		// 1.1 is published and cached alongside 1.0.
		resetFMAState(warpState, "1.1", []byte("lower-1-1"))
		require.Len(t, applyManifest(), 2)
		pkg = getActiveTitleForTeam(lowerTeam.ID).SoftwarePackage
		require.Equal(t, "1.1", pkg.Version)
		require.Equal(t, []string{"1.1", "1.0"}, cachedVersions(pkg))
		// Every apply below updates one of these two rows, so their ids must not change.
		v11ID := pkg.FleetMaintainedVersions[0].ID
		v10ID := pkg.FleetMaintainedVersions[1].ID

		// 1.1 is withdrawn and the manifest goes back to 1.0, rebuilt with new bytes.
		resetFMAState(warpState, "1.0", []byte("lower-1-0-rebuilt"))
		require.Len(t, applyManifest(), 2)
		title := getActiveTitleForTeam(lowerTeam.ID)
		pkg = title.SoftwarePackage
		require.Equal(t, "1.0", pkg.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(pkg))
		active := activeInstaller(lowerTeam.ID, title.ID)
		require.Equal(t, v10ID, active.InstallerID, "the cached 1.0 row is the active one")
		require.Equal(t, warpState.sha256, active.StorageID, "and carries the rebuilt bytes")

		// 1.1 is published again with the bytes Fleet already has cached, so there is
		// nothing to download.
		resetFMAState(warpState, "1.1", []byte("lower-1-1"))
		require.Len(t, applyManifest(), 2)
		title = getActiveTitleForTeam(lowerTeam.ID)
		pkg = title.SoftwarePackage
		require.Equal(t, "1.1", pkg.Version)
		require.Equal(t, []string{"1.1", "1.0"}, cachedVersions(pkg))
		active = activeInstaller(lowerTeam.ID, title.ID)
		require.Equal(t, v11ID, active.InstallerID)
		require.Equal(t, warpState.sha256, active.StorageID)

		// 1.1 is withdrawn again and 1.0 comes back byte for byte. Nothing is
		// downloaded, but the manifest points at 1.0, so it is active and newest.
		resetFMAState(warpState, "1.0", []byte("lower-1-0-rebuilt"))
		require.Len(t, applyManifest(), 2)
		pkg = getActiveTitleForTeam(lowerTeam.ID).SoftwarePackage
		require.Equal(t, "1.0", pkg.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(pkg))

		// 1.1 is published a third time, rebuilt with bytes Fleet has never seen. The
		// version is cached but its hash is not, so the row is updated rather than added.
		resetFMAState(warpState, "1.1", []byte("lower-1-1-rebuilt"))
		require.Len(t, applyManifest(), 2)
		title = getActiveTitleForTeam(lowerTeam.ID)
		pkg = title.SoftwarePackage
		require.Equal(t, "1.1", pkg.Version)
		require.Equal(t, []string{"1.1", "1.0"}, cachedVersions(pkg))
		active = activeInstaller(lowerTeam.ID, title.ID)
		require.Equal(t, v11ID, active.InstallerID, "the same 1.1 row, updated in place")
		require.Equal(t, warpState.sha256, active.StorageID, "now pointing at the rebuilt bytes")

		// A repeated apply with no manifest change must not disturb the order.
		require.Len(t, applyManifest(), 2)
		pkg = getActiveTitleForTeam(lowerTeam.ID).SoftwarePackage
		require.Equal(t, "1.1", pkg.Version)
		require.Equal(t, []string{"1.1", "1.0"}, cachedVersions(pkg))

		require.Equal(t, v11ID, pkg.FleetMaintainedVersions[0].ID, "rows are updated in place, never re-inserted")
		require.Equal(t, v10ID, pkg.FleetMaintainedVersions[1].ID)
	})

	// =========================================================================
	// Section 10: the cron and a GitOps apply treat the manifest the same way, whether
	// it goes back to a cached version or republishes one with different bytes.
	// =========================================================================
	t.Run("cron_matches_gitops_for_a_cached_version", func(t *testing.T) {
		cronTeam := newTeam("team_cron_" + t.Name())
		applyManifest := func() []fleet.SoftwarePackageResponse {
			return batchSet(cronTeam, []*fleet.SoftwareInstallerPayload{
				{Slug: new("cloudflare-warp/windows")},
			})
		}

		resetFMAState(warpState, "1.0", []byte("cron-1-0"))
		require.Len(t, applyManifest(), 1)
		resetFMAState(warpState, "1.1", []byte("cron-1-1"))
		require.Len(t, applyManifest(), 2)
		title := getActiveTitleForTeam(cronTeam.ID)
		require.Equal(t, "1.1", title.SoftwarePackage.Version)
		require.Equal(t, []string{"1.1", "1.0"}, cachedVersions(title.SoftwarePackage))

		// 1.1 is withdrawn and the manifest serves 1.0 again, byte for byte. Nothing needs
		// downloading, but 1.0 is what the manifest publishes now, so the cron marks it
		// current and promotes to it.
		resetFMAState(warpState, "1.0", []byte("cron-1-0"))
		runCron()
		title = getActiveTitleForTeam(cronTeam.ID)
		require.Equal(t, "1.0", title.SoftwarePackage.Version, "the cron follows the manifest back down")
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(title.SoftwarePackage))
		active := activeInstaller(cronTeam.ID, title.ID)
		v10ID := title.SoftwarePackage.FleetMaintainedVersions[0].ID
		require.Equal(t, v10ID, active.InstallerID,
			"the row the cron promoted is the one the manifest publishes")
		storageBeforeRebuild := active.StorageID
		require.Equal(t, warpState.sha256, storageBeforeRebuild)

		// A second pass with the same manifest changes nothing, so the version list keeps
		// reporting when Fleet actually downloaded each version.
		runCron()
		title = getActiveTitleForTeam(cronTeam.ID)
		require.Equal(t, "1.0", title.SoftwarePackage.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(title.SoftwarePackage))

		// 1.0 is rebuilt with bytes Fleet has never seen, so the cron downloads it again and
		// refreshes the cached row in place instead of leaving stale bytes behind.
		resetFMAState(warpState, "1.0", []byte("cron-1-0-rebuilt"))
		runCron()
		title = getActiveTitleForTeam(cronTeam.ID)
		require.Equal(t, "1.0", title.SoftwarePackage.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(title.SoftwarePackage))
		active = activeInstaller(cronTeam.ID, title.ID)
		require.NotEqual(t, storageBeforeRebuild, active.StorageID, "the cron takes the rebuilt bytes")
		require.Equal(t, warpState.sha256, active.StorageID)
		require.Equal(t, v10ID, active.InstallerID, "refreshed in place, no new row")

		// A GitOps apply of the same manifest is then a no-op, so the two agree.
		require.Len(t, applyManifest(), 2)
		title = getActiveTitleForTeam(cronTeam.ID)
		require.Equal(t, "1.0", title.SoftwarePackage.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(title.SoftwarePackage))
		gitopsActive := activeInstaller(cronTeam.ID, title.ID)
		require.Equal(t, active.InstallerID, gitopsActive.InstallerID)
		require.Equal(t, active.StorageID, gitopsActive.StorageID)
	})

	// =========================================================================
	// Section 11: a caret pin still lets the manifest move between cached versions
	// of that major, and the cron leaves a pinned team alone once the version is
	// cached.
	// =========================================================================
	t.Run("caret_pin_with_an_already_cached_version", func(t *testing.T) {
		pinTeam := newTeam("team_caretpin_" + t.Name())
		applyManifest := func(pin string) []fleet.SoftwarePackageResponse {
			sw := &fleet.SoftwareInstallerPayload{Slug: new("cloudflare-warp/windows")}
			sw.RollbackVersion = pin
			return batchSet(pinTeam, []*fleet.SoftwareInstallerPayload{sw})
		}

		resetFMAState(warpState, "1.0", []byte("pin-1-0"))
		require.Len(t, applyManifest(""), 1)
		resetFMAState(warpState, "1.1", []byte("pin-1-1"))
		require.Len(t, applyManifest(""), 2)

		// Pin the major both cached versions share: the newest download of that major wins.
		require.Len(t, applyManifest("^1"), 2)
		pkg := getActiveTitleForTeam(pinTeam.ID).SoftwarePackage
		require.Equal(t, "1.1", pkg.Version)
		require.Equal(t, []string{"1.1", "1.0"}, cachedVersions(pkg))

		// The manifest goes back to the other cached version of the same major while the
		// caret pin is in force, so the pin allows it and 1.0 becomes newest and active.
		resetFMAState(warpState, "1.0", []byte("pin-1-0"))
		require.Len(t, applyManifest("^1"), 2)
		pkg = getActiveTitleForTeam(pinTeam.ID).SoftwarePackage
		require.Equal(t, "1.0", pkg.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(pkg))

		// A cron pass with the pin in place and the version already cached changes nothing.
		runCron()
		pkg = getActiveTitleForTeam(pinTeam.ID).SoftwarePackage
		require.Equal(t, "1.0", pkg.Version)
		require.Equal(t, []string{"1.0", "1.1"}, cachedVersions(pkg))
	})
}

func (s *integrationEnterpriseTestSuite) TestPatchPolicies() {
	t := s.T()
	ctx := context.Background()

	// team tests and no team tests
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Team 1" + t.Name()})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Team 2" + t.Name()})
	require.NoError(t, err)

	// mock an installer being uploaded through FMA
	updateInstallerFMAID := func(fmaID, teamID, titleID uint) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err = q.ExecContext(ctx, `UPDATE software_installers SET fleet_maintained_app_id = ? WHERE global_or_team_id = ? AND title_id = ?`, fmaID, teamID, titleID)
			require.NoError(t, err)
			return nil
		})
	}

	// resetFMAState resets an fmaTestState to the given version and installer bytes.
	resetFMAState := func(state *fmaTestState, version string, installerBytes []byte, patchQuery string) {
		state.version = version
		state.installerBytes = installerBytes
		state.ComputeSHA(installerBytes)
		state.patchQuery = patchQuery
	}

	checkPolicy := func(policy *fleet.Policy, name, version string, titleID uint) {
		require.NotNil(t, policy.PatchSoftware)
		require.Equal(t, titleID, policy.PatchSoftware.SoftwareTitleID)
		require.Equal(t, fleet.PolicyTypePatch, policy.Type)
		require.Equal(t, fmt.Sprintf(`SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Zoom Workplace (X64)' AND version_compare(version, '%s') < 0);`, version), policy.Query)
		require.Equal(t, name, policy.Name)
	}

	checkPolicyInstaller := func(policyID uint, version string) {
		var row struct {
			Version  string `db:"version"`
			IsActive bool   `db:"is_active"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row,
				`SELECT si.version, si.is_active
				 FROM policies p
				 JOIN software_installers si ON si.id = p.software_installer_id
				 WHERE p.id = ?`,
				policyID,
			)
		})
		require.Equal(t, version, row.Version)
		require.True(t, row.IsActive)
	}

	createHostPolicyResults := func(host *fleet.Host, policy *fleet.Policy) {
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			map[uint]*bool{
				policy.ID: new(false),
			},
		), http.StatusOK, &distributedResp)
		err = s.ds.UpdateHostPolicyCounts(ctx)
		require.NoError(t, err)
	}

	t.Run("no_team_patch_policy", func(t *testing.T) {
		// add a regular "No team" policy
		tpParams := fleet.TeamPolicyRequest{
			Name:  "noTeamPolicy1",
			Query: "SELECT 1;",
		}
		r := fleet.TeamPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/fleets/0/policies", tpParams, http.StatusOK, &r)
		require.NotNil(t, r.Policy.TeamID)

		// try to add a patch policy with a query, should fail
		params := map[string]any{
			"name":                    "test-2",
			"query":                   "SELECT 1 FROM;",
			"description":             "um",
			"type":                    "patch",
			"patch_software_title_id": 4484,
		}
		res := s.Do("POST", "/api/latest/fleet/fleets/0/policies", params, http.StatusBadRequest)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `If the "type" is "patch", the "query" field is not supported.`)
		res.Body.Close()

		// Upload some software installer (not fma)
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "some install script",
			Filename:      "dummy_installer.pkg",
			TeamID:        nil,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
		titleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")

		// try to add a patch policy that matches to an installer, but not an FMA
		params = map[string]any{
			"name":                    "test-3",
			"query":                   "",
			"description":             "um",
			"type":                    "patch",
			"patch_software_title_id": titleID,
		}
		res = s.Do("POST", "/api/latest/fleet/fleets/0/policies", params, http.StatusBadRequest)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Software installer for Fleet maintained app with title ID %d does not exist for team ID 0", titleID))
		res.Body.Close()

		// add a fleet maintained app and associate the installer with it
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
			VALUES ('DummyApp', 'dummy/darwin', 'darwin', 'com.example.dummy')`)
			require.NoError(t, err)
			id, err := res.LastInsertId()
			require.NoError(t, err)

			_, err = q.ExecContext(ctx, `UPDATE software_installers SET fleet_maintained_app_id = ? WHERE title_id = ?`, id, titleID)
			require.NoError(t, err)
			return nil
		})

		// add the same patch policy, but this time it belongs to an FMA
		policyResp := fleet.TeamPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/fleets/0/policies", fleet.TeamPolicyRequest{
			Name:                 "",
			Query:                "",
			Description:          "",
			Type:                 new("patch"),
			PatchSoftwareTitleID: &titleID,
		}, http.StatusOK, &policyResp)
		require.Equal(t, "macOS - DummyApp up to date", policyResp.Policy.Name)
		require.Equal(t, "Outdated software might introduce security vulnerabilities or compatibility issues.", policyResp.Policy.Description)
		require.Equal(t, "Install the latest version from self-service.", *policyResp.Policy.Resolution)
		require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.example.dummy' AND version_compare(bundle_short_version, '1.0.0') < 0);", policyResp.Policy.Query)
		require.NotNil(t, policyResp.Policy.PatchSoftware)
		require.Equal(t, titleID, policyResp.Policy.PatchSoftware.SoftwareTitleID)
		policyID := policyResp.Policy.ID

		// attempt to add the same policy again
		params = map[string]any{
			"name":                    "test-5",
			"query":                   "",
			"description":             "um",
			"type":                    "patch",
			"patch_software_title_id": titleID,
		}
		res = s.Do("POST", "/api/latest/fleet/fleets/0/policies", params, http.StatusConflict)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `Couldn't add. Specified "patch_software_title_id" already has a policy with "type" set to "patch".`)
		res.Body.Close()

		// attempt to update patch policy with query
		params = map[string]any{
			"query": "blablabla",
		}
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/0/policies/%d", policyID), params, http.StatusBadRequest)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `"query" can't be updated`)
		res.Body.Close()

		// attempt to update patch policy with platform
		params = map[string]any{
			"platform": "windows,linux",
		}
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/0/policies/%d", policyID), params, http.StatusBadRequest)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `"platform" can't be updated`)
		res.Body.Close()

		// update the patch policy successfully
		params = map[string]any{
			"name":        "test-6-new-name",
			"description": "new description",
		}
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/0/policies/%d", policyID), params, http.StatusOK)
		res.Body.Close()

		// list policies
		var listPolResp fleet.ListTeamPoliciesResponse
		s.DoJSON("GET", "/api/latest/fleet/fleets/0/policies", fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 2)
		require.NotNil(t, listPolResp.Policies[1].PatchSoftware)
		require.Equal(t, titleID, listPolResp.Policies[1].PatchSoftware.SoftwareTitleID)

		// get policy by id
		getPolicyResp := fleet.GetTeamPolicyByIDResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/0/policies/%d", policyID), nil, http.StatusOK, &getPolicyResp)
		require.NotNil(t, getPolicyResp.Policy.PatchSoftware)
		require.Equal(t, titleID, getPolicyResp.Policy.PatchSoftware.SoftwareTitleID)
	})

	t.Run("global_patch_policy", func(t *testing.T) {
		// attempt to add a global patch policy
		params := map[string]any{
			"name":                    "global-policy",
			"query":                   "UNAFFECTED;",
			"description":             "patch policies only work for teams",
			"type":                    "patch",
			"patch_software_title_id": "1", // should not matter
		}
		res := s.Do("POST", "/api/latest/fleet/policies", params, http.StatusOK)

		resBody, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		res.Body.Close()

		policyResp := fleet.GlobalPolicyResponse{}
		err = json.Unmarshal(resBody, &policyResp)
		require.NoError(t, err)
		policyID := policyResp.Policy.ID

		getPolicyResp := fleet.GetPolicyByIDResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", policyID), nil, http.StatusOK, &getPolicyResp)
		require.NotNil(t, getPolicyResp.Policy)
		require.Equal(t, "dynamic", getPolicyResp.Policy.Type)
		require.Empty(t, getPolicyResp.Policy.PatchSoftware)
		require.Empty(t, getPolicyResp.Policy.PatchSoftwareTitleID)

		// Should not be able to apply global patch policy
		spec := &fleet.PolicySpec{
			Name:                   "team patch policy",
			Query:                  "SELECT 1",
			Type:                   fleet.PolicyTypePatch,
			FleetMaintainedAppSlug: "zoom/windows",
		}
		applyResp := fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusBadRequest, &applyResp,
		)
	})

	t.Run("patch_policy_lifecycle", func(t *testing.T) {
		// Test 1: delete software installer when there is an associatd patch policy

		resp := teamResponse{}
		s.DoJSON("POST", "/api/latest/fleet/fleets", &createTeamRequest{
			Name: new("team_1"),
		}, http.StatusOK, &resp)
		teamID := resp.Team.ID

		// upload a software installer
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "some install script",
			Filename:      "dummy_installer.pkg",
			TeamID:        &teamID,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
		titleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
		fmaID := getFleetMaintainedAppID(t, s.ds, "dummy/darwin")

		// add a fleet maintained app and associate the installer with it
		updateInstallerFMAID(fmaID, teamID, titleID)

		// add a patch policy
		policyResp := fleet.TeamPolicyResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", teamID), fleet.TeamPolicyRequest{
			Type:                 new("patch"),
			PatchSoftwareTitleID: &titleID,
		}, http.StatusOK, &policyResp)

		// attempt to delete installer
		res := s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusConflict, "team_id", strconv.Itoa(int(teamID))) //nolint:gosec // dismiss G115
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `Couldn’t delete. This software has a patch policy. Please remove the patch policy and try again.`)
		res.Body.Close()

		// remove patch policy and delete installer
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/delete", teamID), fleet.DeleteTeamPoliciesRequest{
			TeamID: teamID,
			IDs:    []uint{policyResp.Policy.ID},
		}, http.StatusOK, &fleet.DeleteTeamPoliciesResponse{})

		// can delete installer successfully
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", strconv.Itoa(int(teamID))) //nolint:gosec // dismiss G115

		// Test 2: Updating an FMA installer with a new file fails
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
		updateInstallerFMAID(fmaID, teamID, titleID)

		dummyBytes, err := os.ReadFile("testdata/software-installers/dummy_installer.pkg")
		require.NoError(t, err)
		body, headers := generateMultipartRequest(t, "software", "dummy_installer.pkg", dummyBytes, s.token, map[string][]string{"team_id": {strconv.Itoa(int(teamID))}}) //nolint:gosec // dismiss G115
		res = s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusBadRequest, headers)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `Couldn't update. The package can't be changed for Fleet-maintained apps.`)
		res.Body.Close()
	})

	t.Run("batch team patch policy", func(t *testing.T) {
		// initialize FMA server
		states := make(map[string]*fmaTestState, 1)
		states["/zoom/windows.json"] = &fmaTestState{
			version:        "1.0",
			installerBytes: []byte("xyz"),
			installerPath:  "/zoom.msi",
		}
		startFMAServers(t, s.ds, states)

		// Add some hosts
		teamHosts := s.createHosts(t, "windows", "windows")
		require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHosts[0].ID})))
		require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHosts[1].ID})))

		getActiveTitleForTeam := func(teamID uint, titleName string) fleet.SoftwareTitleListResult {
			var resp listSoftwareTitlesResponse
			s.DoJSON(
				"GET", "/api/latest/fleet/software/titles",
				listSoftwareTitlesRequest{},
				http.StatusOK, &resp,
				"per_page", "1",
				"order_key", "name",
				"order_direction", "desc",
				"available_for_install", "true",
				"team_id", fmt.Sprintf("%d", teamID),
				"query", titleName,
			)
			require.Equal(t, 1, resp.Count)
			return resp.SoftwareTitles[0]
		}

		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
			http.StatusAccepted, &resp,
			"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

		spec := &fleet.PolicySpec{
			Name:                   "team patch policy",
			Query:                  "SELECT 1",
			Team:                   team.Name,
			Type:                   fleet.PolicyTypePatch,
			FleetMaintainedAppSlug: "zoom/windows",
		}

		applyResp := fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusOK, &applyResp,
		)

		title := getActiveTitleForTeam(team.ID, "zoom")

		listPolResp := fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 1)
		checkPolicy(listPolResp.Policies[0], spec.Name, "1.0", title.ID)

		// now enable automation on the policy
		spec = &fleet.PolicySpec{
			Name:                   "team patch policy",
			Query:                  "SELECT 1",
			Team:                   team.Name,
			Type:                   fleet.PolicyTypePatch,
			FleetMaintainedAppSlug: "zoom/windows",
			SoftwareTitleID:        new(title.ID),
		}

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusOK, &applyResp,
		)

		title = getActiveTitleForTeam(team.ID, "zoom")

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 1)
		checkPolicy(listPolResp.Policies[0], spec.Name, "1.0", title.ID)
		// This is only set if the automation is enable
		require.Equal(t, title.ID, listPolResp.Policies[0].InstallSoftware.SoftwareTitleID)
		checkPolicyInstaller(listPolResp.Policies[0].ID, "1.0")

		// Now disable the automation
		spec = &fleet.PolicySpec{
			Name:                   "team patch policy",
			Query:                  "SELECT 1",
			Team:                   team.Name,
			Type:                   fleet.PolicyTypePatch,
			FleetMaintainedAppSlug: "zoom/windows",
		}

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusOK, &applyResp,
		)

		title = getActiveTitleForTeam(team.ID, "zoom")

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 1)
		checkPolicy(listPolResp.Policies[0], spec.Name, "1.0", title.ID)
		require.Nil(t, listPolResp.Policies[0].InstallSoftware)

		// check policy membership
		createHostPolicyResults(teamHosts[0], listPolResp.Policies[0])
		listPolResp.Policies[0], err = s.ds.Policy(ctx, listPolResp.Policies[0].ID)
		require.NoError(t, err)
		require.Equal(t, uint(1), listPolResp.Policies[0].FailingHostCount)

		// Test 2: FMA Version is updated (query should use new version)
		resetFMAState(states["/zoom/windows.json"], "1.2", []byte("abc"), "")

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
			http.StatusAccepted, &resp,
			"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusOK, &applyResp,
		)
		title = getActiveTitleForTeam(team.ID, "zoom")

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 1)
		checkPolicy(listPolResp.Policies[0], spec.Name, "1.2", title.ID)

		// check policy membership
		listPolResp.Policies[0], err = s.ds.Policy(ctx, listPolResp.Policies[0].ID)
		require.NoError(t, err)
		require.Equal(t, uint(0), listPolResp.Policies[0].FailingHostCount)

		// Test 3: FMA Version is pinned back after update to a newer version (query should use old version)
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
				{Slug: new("zoom/windows"), SelfService: true, RollbackVersion: "1.0"},
			}, TeamName: team.Name},
			http.StatusAccepted, &resp,
			"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusOK, &applyResp,
		)
		title = getActiveTitleForTeam(team.ID, "zoom")

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 1)
		checkPolicy(listPolResp.Policies[0], spec.Name, "1.0", title.ID)

		// Test 4: FMA upgraded again with two versions already cached.
		spec = &fleet.PolicySpec{
			Name:                   "team patch policy",
			Query:                  "SELECT 1",
			Team:                   team.Name,
			Type:                   fleet.PolicyTypePatch,
			FleetMaintainedAppSlug: "zoom/windows",
			SoftwareTitleID:        new(title.ID),
		}

		resetFMAState(states["/zoom/windows.json"], "1.3", []byte("ghi"), "")

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
			http.StatusAccepted, &resp,
			"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{spec}},
			http.StatusOK, &applyResp,
		)
		title = getActiveTitleForTeam(team.ID, "zoom")

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		require.Len(t, listPolResp.Policies, 1)
		checkPolicy(listPolResp.Policies[0], spec.Name, "1.3", title.ID)
		checkPolicyInstaller(listPolResp.Policies[0].ID, "1.3")
	})

	t.Run("override and empty queries behave the same", func(t *testing.T) {
		// initialize FMA server
		states := make(map[string]*fmaTestState, 2)
		states["/zoom/windows.json"] = &fmaTestState{
			version:        "1.0",
			installerBytes: []byte("xyz"),
			installerPath:  "/zoom.msi",
		}
		states["/1password/darwin.json"] = &fmaTestState{
			version:        "1.0",
			installerBytes: []byte("xyz"),
			installerPath:  "/1password.pkg",
			patchQuery:     "SELECT 1; --custom query 1.0",
		}
		startFMAServers(t, s.ds, states)

		// Add installers at version 1.0
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}, {Slug: new("1password/darwin")}}, TeamName: team2.Name},
			http.StatusAccepted, &resp,
			"team_name", team2.Name, "team_id", fmt.Sprint(team2.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, resp.RequestUUID)

		checkPolicies := func(policies []*fleet.Policy, version string) {
			require.Len(t, policies, 2)
			require.Equal(t, fmt.Sprintf(`SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Zoom Workplace (X64)' AND version_compare(version, '%s') < 0);`, version), policies[0].Query)
			require.Equal(t, fmt.Sprintf(`SELECT 1; --custom query %s`, version), policies[1].Query)
		}

		specs := []*fleet.PolicySpec{
			{
				Name:                   "team patch policy",
				Query:                  "SELECT 1",
				Team:                   team2.Name,
				Type:                   fleet.PolicyTypePatch,
				FleetMaintainedAppSlug: "zoom/windows",
			},
			{
				Name:                   "team patch policy 2",
				Team:                   team2.Name,
				Type:                   fleet.PolicyTypePatch,
				FleetMaintainedAppSlug: "1password/darwin",
			},
		}

		// Apply patch policies
		applyResp := fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: specs},
			http.StatusOK, &applyResp,
		)

		listPolResp := fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team2.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		checkPolicies(listPolResp.Policies, "1.0")

		// Pin FMA versions to 1.2, queries should update
		resetFMAState(states["/zoom/windows.json"], "1.2", []byte("abc"), "")
		resetFMAState(states["/1password/darwin.json"], "1.2", []byte("abc"), "SELECT 1; --custom query 1.2")

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}, {Slug: new("1password/darwin")}}, TeamName: team2.Name},
			http.StatusAccepted, &resp,
			"team_name", team2.Name, "team_id", fmt.Sprint(team2.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, resp.RequestUUID)

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: specs},
			http.StatusOK, &applyResp,
		)

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team2.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		checkPolicies(listPolResp.Policies, "1.2")

		// Rollback FMA versions to 1.0, queries should use older versions
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
				{Slug: new("zoom/windows"), SelfService: true, RollbackVersion: "1.0"},
				{Slug: new("1password/darwin"), SelfService: true, RollbackVersion: "1.0"},
			}, TeamName: team2.Name},
			http.StatusAccepted, &resp,
			"team_name", team2.Name, "team_id", fmt.Sprint(team2.ID),
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, resp.RequestUUID)

		applyResp = fleet.ApplyPolicySpecsResponse{}
		s.DoJSON("POST", "/api/latest/fleet/spec/policies",
			fleet.ApplyPolicySpecsRequest{Specs: specs},
			http.StatusOK, &applyResp,
		)

		listPolResp = fleet.ListTeamPoliciesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team2.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
		checkPolicies(listPolResp.Policies, "1.0")
	})

	t.Run("notify_before_patching", func(t *testing.T) {
		resp := teamResponse{}
		s.DoJSON("POST", "/api/latest/fleet/fleets", &createTeamRequest{
			Name: new("notify_before_patching_team"),
		}, http.StatusOK, &resp)
		teamID := resp.Team.ID

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "some install script",
			Filename:      "dummy_installer.pkg",
			TeamID:        &teamID,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
		titleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")

		// Own Fleet-maintained app rather than the one an earlier subtest inserts, so this doesn't
		// depend on subtest order or on what the batch subtests leave behind.
		fma, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
			Name:             "DummyApp",
			Slug:             "notify-dummy/darwin",
			Platform:         "darwin",
			UniqueIdentifier: "com.example.dummy",
		})
		require.NoError(t, err)
		updateInstallerFMAID(fma.ID, teamID, titleID)

		policyPath := func(policyID uint) string {
			return fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", teamID, policyID)
		}

		// Creating with the flag forces continuous automations on without the request saying so.
		policyResp := fleet.TeamPolicyResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", teamID), fleet.TeamPolicyRequest{
			Type:                         new("patch"),
			PatchSoftwareTitleID:         &titleID,
			NotifyBeforePatching:         true,
			ContinuousAutomationsEnabled: true,
		}, http.StatusOK, &policyResp)
		policyID := policyResp.Policy.ID
		require.True(t, policyResp.Policy.NotifyBeforePatching)
		require.False(t, policyResp.Policy.PatchWhenClosed)
		require.True(t, policyResp.Policy.ContinuousAutomationsEnabled)

		// The flag survives a round trip through the wire, not just the create response.
		getPolicyResp := fleet.GetTeamPolicyByIDResponse{}
		s.DoJSON("GET", policyPath(policyID), nil, http.StatusOK, &getPolicyResp)
		require.True(t, getPolicyResp.Policy.NotifyBeforePatching)

		// The title reports the flag and shows Fleet's managed app open query read-only, which is
		// what the End user experience dropdown reads back on load.
		titleResp := getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
			"fleet_id", fmt.Sprint(teamID))
		require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
		require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage.PatchPolicy)
		require.True(t, titleResp.SoftwareTitle.SoftwarePackage.PatchPolicy.NotifyBeforePatching)

		// A modify payload carries only the delta, so the stored flag still conflicts.
		res := s.Do("PATCH", policyPath(policyID), map[string]any{"patch_when_closed": true}, http.StatusBadRequest)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `Only one of "patch_when_closed" or "notify_before_patching" can be set to true`)
		res.Body.Close()

		// An explicit continuous_automations_enabled false is rejected while the flag is on.
		res = s.Do("PATCH", policyPath(policyID), map[string]any{"continuous_automations_enabled": false}, http.StatusBadRequest)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `If "notify_before_patching" is true, "continuous_automations_enabled" can't be set to false.`)
		res.Body.Close()

		// Omitting continuous automations on modify still auto-sets it.
		s.Do("PATCH", policyPath(policyID), map[string]any{"continuous_automations_enabled": false, "notify_before_patching": false}, http.StatusOK).Body.Close()
		s.Do("PATCH", policyPath(policyID), map[string]any{"notify_before_patching": true}, http.StatusOK).Body.Close()

		getPolicyResp = fleet.GetTeamPolicyByIDResponse{}
		s.DoJSON("GET", policyPath(policyID), nil, http.StatusOK, &getPolicyResp)
		require.True(t, getPolicyResp.Policy.NotifyBeforePatching)
		require.True(t, getPolicyResp.Policy.ContinuousAutomationsEnabled)

		// The flag is only for patch policies.
		res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", teamID), map[string]any{
			"name":                           "dynamic-notify",
			"query":                          "SELECT 1;",
			"notify_before_patching":         true,
			"continuous_automations_enabled": true,
		}, http.StatusBadRequest)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `"notify_before_patching" is only supported for patch policies`)
		res.Body.Close()
	})
}

func getFleetMaintainedAppID(t *testing.T, ds *mysql.Datastore, slug string) uint {
	var id uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM fleet_maintained_apps WHERE slug = ?`, slug)
	})
	return id
}

func (s *integrationEnterpriseTestSuite) TestPinMajorVersion() {
	t := s.T()

	resetFMAState := func(state *fmaTestState, version string, installerBytes []byte, patchQuery string) {
		state.version = version
		state.installerBytes = installerBytes
		state.ComputeSHA(installerBytes)
		state.patchQuery = patchQuery
	}

	t.Run("pin major version", func(t *testing.T) {
		teamName := ""
		states := make(map[string]*fmaTestState, 1)
		states["/zoom/windows.json"] = &fmaTestState{
			version:        "1.0",
			installerBytes: []byte("xyz"),
			installerPath:  "/zoom.msi",
		}
		startFMAServers(t, s.ds, states)

		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, resp.RequestUUID)

		var listTitlesResp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listTitlesResp, "team_id", "0")
		require.Len(t, listTitlesResp.SoftwareTitles, 1)
		require.Equal(t, "zoom.msi", listTitlesResp.SoftwareTitles[0].SoftwarePackage.Name)
		require.Equal(t, "1.0", listTitlesResp.SoftwareTitles[0].SoftwarePackage.Version)
		titleID := listTitlesResp.SoftwareTitles[0].ID

		// pin version to ^1, call batch, should get 1.0
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^1"}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, resp.RequestUUID)

		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")
		require.NotNil(t, titleResp.SoftwareTitle)
		require.Equal(t, "1.0", titleResp.SoftwareTitle.SoftwarePackage.Version)
		require.WithinDuration(t, time.Now(), titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].UploadedAt, 5*time.Second)
		installer10UploadedAt := titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].UploadedAt

		// update manifest to 1.1, FleetMaintainedVersions should have 1.0, 1.1
		// installer version should be 1.1, while still pinned to ^1
		resetFMAState(states["/zoom/windows.json"], "1.1", []byte("abc"), "")

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^1"}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, resp.RequestUUID)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")
		require.NotNil(t, titleResp.SoftwareTitle)
		require.Len(t, titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions, 2)
		require.Equal(t, "1.1", titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].Version)
		require.Equal(t, "1.0", titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[1].Version)
		require.Equal(t, "1.1", titleResp.SoftwareTitle.SoftwarePackage.Version)
		require.WithinDuration(t, time.Now(), titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].UploadedAt, 5*time.Second)
		installer11UploadedAt := titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].UploadedAt
		require.WithinDuration(t, installer10UploadedAt, titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[1].UploadedAt, 5*time.Second)

		// pin version to 1.0? then pin to ^1, installer should be 1.1. if fleet keeps enough versions we need to test this.
		// maybe unnecessary test, can remove to speed it up

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "1.0"}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, resp.RequestUUID)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")
		require.NotNil(t, titleResp.SoftwareTitle)
		require.Equal(t, "1.0", titleResp.SoftwareTitle.SoftwarePackage.Version)

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^1"}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, resp.RequestUUID)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")
		require.NotNil(t, titleResp.SoftwareTitle)
		require.Equal(t, "1.1", titleResp.SoftwareTitle.SoftwarePackage.Version)

		// update manifesst to 2.0, installer should be 1.1
		resetFMAState(states["/zoom/windows.json"], "2.0", []byte("abc"), "")

		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^1"}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, resp.RequestUUID)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")
		require.NotNil(t, titleResp.SoftwareTitle)
		require.Len(t, titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions, 2)
		require.Equal(t, "1.1", titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].Version)
		require.Equal(t, "1.0", titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[1].Version)
		require.Equal(t, "1.1", titleResp.SoftwareTitle.SoftwarePackage.Version)
		require.WithinDuration(t, installer11UploadedAt, titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[0].UploadedAt, 5*time.Second)
		require.WithinDuration(t, installer10UploadedAt, titleResp.SoftwareTitle.SoftwarePackage.FleetMaintainedVersions[1].UploadedAt, 5*time.Second)
	})

	// Test pinning ^1 for the first time when only 2.0 is available from manifest
	t.Run("add installer first time with version pin", func(t *testing.T) {
		teamName := ""
		states := make(map[string]*fmaTestState, 1)
		// use a different installer that isn't available yet
		states["/1password/darwin.json"] = &fmaTestState{
			version:        "2.0",
			installerBytes: []byte("xyz"),
			installerPath:  "/1password.pkg",
		}
		startFMAServers(t, s.ds, states)

		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("1password/darwin"), RollbackVersion: "^1"}}, TeamName: teamName},
			http.StatusAccepted, &resp,
			"team_name", teamName, "team_id", "0",
		)
		message := waitBatchSetSoftwareInstallersFailed(t, &s.withServer, teamName, resp.RequestUUID)
		require.Contains(t, message, "specified major version is not available")
	})
}

// After an FMA replacement, Library tab and device self-service must honor the active
// installer's label scope so list and install agree.
func (s *integrationEnterpriseTestSuite) TestFMAReplacedInstallerLabelScopeListAndInstallAgree() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name(), Description: t.Name()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID)) })

	host := createOrbitEnrolledHost(t, "darwin", "fma_scope_host", s.ds)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
	host.TeamID = &team.ID
	deviceToken := uuid.NewString()
	createDeviceTokenForHost(t, s.ds, host.ID, deviceToken)

	excludeLabel, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "_excl", Query: "select 1"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddLabelsToHost(ctx, host.ID, []uint{excludeLabel.ID}))
	host.LabelUpdatedAt = time.Now()
	require.NoError(t, s.ds.UpdateHost(ctx, host))

	// Old installer (FMA v1): no labels.
	fma, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             t.Name() + " App",
		Slug:             "fma-scope-test/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.example." + t.Name(),
	})
	require.NoError(t, err)

	adminUserID := uint(1) // admin user from suite setup
	tfr, err := fleet.NewTempFileReader(strings.NewReader("v1"), t.TempDir)
	require.NoError(t, err)
	oldInstallerID, titleID, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:        "exit 0",
		InstallerFile:        tfr,
		StorageID:            "fma_scope_v1",
		FleetMaintainedAppID: &fma.ID,
		Filename:             "fma_scope.pkg",
		Title:                t.Name() + " App",
		Version:              "1.0",
		Source:               "apps",
		Platform:             string(fleet.MacOSPlatform),
		TeamID:               &team.ID,
		UserID:               adminUserID,
		SelfService:          true,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Successful install against old installer.
	installUUID, err := s.ds.InsertSoftwareInstallRequest(ctx, host.ID, oldInstallerID, fleet.HostSoftwareInstallOptions{SelfService: true})
	require.NoError(t, err)
	_, err = s.ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           installUUID,
		InstallScriptExitCode: new(int(0)),
	}, nil)
	require.NoError(t, err)

	// Simulate FMA replacement.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET is_active = FALSE WHERE id = ?", oldInstallerID)
		return err
	})

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("v2"), t.TempDir)
	require.NoError(t, err)
	newInstallerID, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:        "exit 0",
		InstallerFile:        tfr2,
		StorageID:            "fma_scope_v2",
		FleetMaintainedAppID: new(fma.ID),
		Filename:             "fma_scope.pkg",
		Title:                t.Name() + " App",
		Version:              "2.0",
		Source:               "apps",
		Platform:             string(fleet.MacOSPlatform),
		TeamID:               &team.ID,
		UserID:               adminUserID,
		SelfService:          true,
		ValidatedLabels: &fleet.LabelIdentsWithScope{
			ByName: map[string]fleet.LabelIdent{
				excludeLabel.Name: {LabelName: excludeLabel.Name, LabelID: excludeLabel.ID},
			},
			LabelScope: fleet.LabelScopeExcludeAny,
		},
	})
	require.NoError(t, err)
	require.NotEqual(t, oldInstallerID, newInstallerID)

	// Library tab must not surface the title.
	var hostSw getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &hostSw, "available_for_install", "true")
	for _, sw := range hostSw.Software {
		require.NotEqual(t, titleID, sw.ID, "title must not appear in Library tab when active installer excludes the host")
	}

	// Self-service must not surface the title.
	var deviceSw getDeviceSoftwareResponse
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+deviceToken+"/software?self_service=true", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&deviceSw))
	for _, sw := range deviceSw.Software {
		require.NotEqual(t, titleID, sw.ID, "title must not appear in device self-service when active installer excludes the host")
	}

	// Install endpoint rejects with label-scope error.
	resp := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(resp.Body), "Couldn't install. Host isn't member of the labels defined for this software title.")
}

func (s *integrationEnterpriseTestSuite) TestFleetMaintainedAppVersionPin() {
	t := s.T()
	ctx := context.Background()

	// Each cached zoom version ships its own patch query with the version baked in, as real FMA manifests do.
	const zoomPatchQueryFmt = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Zoom' AND version_compare(version, '%s') < 0);"
	zoomQueryV1 := fmt.Sprintf(zoomPatchQueryFmt, "1.0")
	zoomQueryV2 := fmt.Sprintf(zoomPatchQueryFmt, "2.0")

	// Real FMA mock servers for zoom/windows with a mutable version, so we can cache multiple versions and drive
	// the GitOps (batch) path end to end. patchQuery makes the app patchable so a patch policy can be created.
	zoom := &fmaTestState{
		version:        "1.0",
		installerBytes: []byte("zoom-1.0"),
		installerPath:  "/zoom.msi",
		patchQuery:     zoomQueryV1,
	}
	// Google Chrome ships 4-component versions (e.g. 149.0.7827.156), which are not valid semver. It rides along
	// on every batch apply below pinned to "^149", exercising the GitOps major match against a non-semver
	// version — that used to fail the whole apply with "Invalid Semantic Version".
	chrome := &fmaTestState{
		version:        "149.0.7827.156",
		installerBytes: []byte("chrome-149"),
		installerPath:  "/googlechrome.msi",
		patchQuery:     "SELECT 1 FROM osquery_info;",
	}
	startFMAServers(t, s.ds, map[string]*fmaTestState{
		"/zoom/windows.json":          zoom,
		"/google-chrome/windows.json": chrome,
	})

	var teamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{
		Name: new("fma_pin_" + t.Name()),
	}, http.StatusOK, &teamResp)
	team := *teamResp.Team

	batchSet := func(software []*fleet.SoftwareInstallerPayload) []fleet.SoftwarePackageResponse {
		software = append(software, &fleet.SoftwareInstallerPayload{Slug: new("google-chrome/windows"), RollbackVersion: "^149"})
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: software, TeamName: team.Name},
			http.StatusAccepted, &resp, "team_name", team.Name)
		return waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)
	}
	bumpVersion := func(version string, bytes []byte) {
		zoom.version = version
		zoom.installerBytes = bytes
		zoom.patchQuery = fmt.Sprintf(zoomPatchQueryFmt, version)
		zoom.ComputeSHA(bytes)
	}

	// Cache v1.0, then bump the manifest and cache v2.0 (GitOps applies, no pin).
	pkgs := batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}})
	var titleID, chromeTitleID uint
	for _, pkg := range pkgs {
		require.NotNil(t, pkg.TitleID)
		switch pkg.Slug {
		case "zoom/windows":
			titleID = *pkg.TitleID
		case "google-chrome/windows":
			chromeTitleID = *pkg.TitleID
		}
	}
	require.NotZero(t, titleID)
	require.NotZero(t, chromeTitleID)

	bumpVersion("2.0", []byte("zoom-2.0"))
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}})

	getPkg := func() *fleet.SoftwareInstaller {
		var resp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, resp.SoftwareTitle)
		require.NotNil(t, resp.SoftwareTitle.SoftwarePackage)
		return resp.SoftwareTitle.SoftwarePackage
	}
	policyInstallerID := func(policyID uint) uint {
		var id *uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, "SELECT software_installer_id FROM policies WHERE id = ?", policyID)
		})
		require.NotNilf(t, id, "policy %d has no software_installer_id", policyID)
		return *id
	}
	// Patch policies are title-scoped (patch_software_title_id), not installer-scoped, so they don't carry a
	// software_installer_id and aren't re-pointed on a version flip — they stay attached to the title.
	patchPolicyTitleID := func(policyID uint) uint {
		var id *uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, "SELECT patch_software_title_id FROM policies WHERE id = ?", policyID)
		})
		require.NotNilf(t, id, "patch policy %d has no patch_software_title_id", policyID)
		return *id
	}
	patchPolicyQuery := func(policyID uint) string {
		var resp fleet.GetTeamPolicyByIDResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", team.ID, policyID), nil, http.StatusOK, &resp)
		return resp.Policy.Query
	}

	// Two cached versions (1.0, 2.0); the no-version GitOps applies above left the title on Latest (active = newest,
	// no pin).
	p := getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Nil(t, p.PinnedVersion)
	// Chrome's ^149 caret resolved against its 4-component version on the applies above.
	var chromeResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", chromeTitleID), nil, http.StatusOK, &chromeResp, "team_id", fmt.Sprint(team.ID))
	require.Equal(t, "149.0.7827.156", chromeResp.SoftwareTitle.SoftwarePackage.Version)
	require.Equal(t, new("^149"), chromeResp.SoftwareTitle.SoftwarePackage.PinnedVersion)

	// Dependencies on the title: an install-automation policy (carries software_installer_id, re-pointed to the
	// active installer on a flip) and a patch policy (title-scoped via patch_software_title_id, never re-pointed).
	var installPol fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Name: "install zoom", Query: "SELECT 1 FROM osquery_info;", SoftwareTitleID: &titleID,
	}, http.StatusOK, &installPol)
	var patchPol fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Type: new("patch"), PatchSoftwareTitleID: &titleID,
	}, http.StatusOK, &patchPol)

	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	require.Equal(t, titleID, patchPolicyTitleID(patchPol.Policy.ID))
	require.Equal(t, zoomQueryV2, patchPolicyQuery(patchPol.Policy.ID))

	patchVersion := func(version string) {
		body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id": {fmt.Sprint(team.ID)},
			"version": {version},
		})
		resp := s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)
		var patchResp getSoftwareInstallerResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&patchResp))
		resp.Body.Close()
		// the response must reflect the newly-pinned active installer, not the one pinned away from
		require.NotNil(t, patchResp.SoftwareInstaller)
		require.Equal(t, getPkg().Version, patchResp.SoftwareInstaller.Version)
	}
	// requireLastPinActivity asserts the latest activity is an edited_software with want as pinned_version.
	// Marshaling want renders a pin as "1.0"/"^2" and a cleared pin (nil) as null, so no branching is needed.
	requireLastPinActivity := func(want *string) {
		pinnedVersion, err := json.Marshal(want)
		require.NoError(t, err)
		s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), fmt.Sprintf(
			`{"team_id": %d, "fleet_id": %d, "team_name": %q, "fleet_name": %q, "self_service": false, `+
				`"software_title": "Zoom Workplace (X64)", "software_package": "zoom.msi", "software_icon_url": null, `+
				`"software_title_id": %d, "software_display_name": "", "pinned_version": %s}`,
			team.ID, team.ID, team.Name, team.Name, titleID, pinnedVersion), 0)
	}

	// --- UI (PATCH) pins, on the static 1.0/2.0 cache ---

	// Rollback to an older cached version. The install policy is re-pointed to the now-active installer; the patch
	// policy is title-scoped and stays put.
	patchVersion("1.0")
	requireLastPinActivity(new("1.0"))
	p = getPkg()
	require.Equal(t, "1.0", p.Version)
	require.Equal(t, new("1.0"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	require.Equal(t, titleID, patchPolicyTitleID(patchPol.Policy.ID))
	require.Equal(t, zoomQueryV1, patchPolicyQuery(patchPol.Policy.ID))

	// Caret resolves to the newest cached minor in that major.
	patchVersion("^2")
	requireLastPinActivity(new("^2"))
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Equal(t, new("^2"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	require.Equal(t, zoomQueryV2, patchPolicyQuery(patchPol.Policy.ID))

	// A caret with no cached installer in that major keeps the newest cached version instead of erroring.
	patchVersion("^9")
	requireLastPinActivity(new("^9"))
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Equal(t, new("^9"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))

	// Empty clears the pin row, back to Latest.
	patchVersion("")
	requireLastPinActivity(nil)
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Nil(t, p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))

	// --- GitOps pins persist the pin expression as written ---

	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "1.0"}})
	p = getPkg()
	require.Equal(t, "1.0", p.Version)
	require.Equal(t, new("1.0"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	require.Equal(t, titleID, patchPolicyTitleID(patchPol.Policy.ID)) // patch policy stays title-scoped across a GitOps flip

	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^2"}})
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Equal(t, new("^2"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))

	// Last write wins: a GitOps apply with no version resets a prior UI pin back to Latest.
	patchVersion("1.0")
	requireLastPinActivity(new("1.0"))
	p = getPkg()
	require.Equal(t, "1.0", p.Version)
	require.Equal(t, new("1.0"), p.PinnedVersion)

	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}})
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Nil(t, p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))

	// --- New versions released over time (each GitOps run can bring a newer manifest version) ---

	// A caret pin tracks a newly released minor in the pinned major: releasing 2.5 under "^2" moves the active
	// version forward to 2.5.
	patchVersion("^2")
	requireLastPinActivity(new("^2"))
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Equal(t, new("^2"), p.PinnedVersion)

	bumpVersion("2.5", []byte("zoom-2.5"))
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^2"}})
	p = getPkg()
	require.Equal(t, "2.5", p.Version)
	require.Equal(t, new("^2"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	// The cache caps at 2 versions per title, so caching 2.5 evicts the oldest (1.0).
	require.Len(t, p.FleetMaintainedVersions, 2)
	require.ElementsMatch(t, []string{"2.0", "2.5"},
		[]string{p.FleetMaintainedVersions[0].Version, p.FleetMaintainedVersions[1].Version},
		"caching a new minor evicts the oldest cached version")

	// A caret pin ignores a newly released major: releasing 3.0 under "^2" leaves the active version on the newest
	// cached in-major version, and 3.0 is never cached.
	bumpVersion("3.0", []byte("zoom-3.0"))
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "^2"}})
	p = getPkg()
	require.Equal(t, "2.5", p.Version)
	require.Equal(t, new("^2"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	require.Len(t, p.FleetMaintainedVersions, 2)
	require.ElementsMatch(t, []string{"2.0", "2.5"},
		[]string{p.FleetMaintainedVersions[0].Version, p.FleetMaintainedVersions[1].Version},
		"a new major must not be cached under a caret pin")

	// A literal pin holds across a new release: releasing 4.0 under a "2.0" pin keeps the active version on 2.0, and
	// 4.0 is never cached (the literal pin fetches its exact cached version, not the latest manifest).
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "2.0"}})
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Equal(t, new("2.0"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID)) // flip 2.5 -> 2.0 re-points the install policy

	bumpVersion("4.0", []byte("zoom-4.0"))
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "2.0"}})
	p = getPkg()
	require.Equal(t, "2.0", p.Version)
	require.Equal(t, new("2.0"), p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))
	require.Len(t, p.FleetMaintainedVersions, 2)
	require.ElementsMatch(t, []string{"2.0", "2.5"},
		[]string{p.FleetMaintainedVersions[0].Version, p.FleetMaintainedVersions[1].Version},
		"a literal pin must not cache a newer release")

	// Clearing the pin lets Latest track the new release.
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}})
	p = getPkg()
	require.Equal(t, "4.0", p.Version)
	require.Nil(t, p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))

	// Newest is determined by semantic version, not string order: 10.0 is newer than 4.0 even though it sorts
	// earlier lexicographically.
	bumpVersion("10.0", []byte("zoom-10.0"))
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}})
	p = getPkg()
	require.Equal(t, "10.0", p.Version)

	// Pin the older 4.0, then clear to Latest and confirm it resolves to 10.0, not the string-larger "4.0".
	patchVersion("4.0")
	requireLastPinActivity(new("4.0"))
	p = getPkg()
	require.Equal(t, "4.0", p.Version)
	require.Equal(t, new("4.0"), p.PinnedVersion)

	patchVersion("")
	requireLastPinActivity(nil)
	p = getPkg()
	require.Equal(t, "10.0", p.Version)
	require.Nil(t, p.PinnedVersion)
	require.Equal(t, p.InstallerID, policyInstallerID(installPol.Policy.ID))

	// --- Validation ---

	// "version" can't be changed in the same PATCH as another field.
	body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
		"team_id":        {fmt.Sprint(team.ID)},
		"version":        {"2.0"},
		"install_script": {"echo changed"},
	})
	s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusBadRequest, headers)

	// "version" is only valid for a Fleet-maintained app, not a custom package.
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "echo install",
		Filename:      "ruby.deb",
		TeamID:        &team.ID,
	}, http.StatusOK, "")
	customTitleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")
	body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
		"team_id": {fmt.Sprint(team.ID)},
		"version": {"1.0"},
	})
	s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", customTitleID), body.Bytes(), http.StatusBadRequest, headers)

	// --- Deleting an FMA clears its pin row ---

	// The pin row is keyed on (team, title); the title row outlives the installer
	// rows, so without explicit cleanup a stale pin would resurface on re-add.
	pinRowCount := func() int {
		var n int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n,
				"SELECT COUNT(*) FROM software_title_team_pins WHERE team_id = ? AND title_id = ?", team.ID, titleID)
		})
		return n
	}
	patchVersion("10.0")
	require.Equal(t, new("10.0"), getPkg().PinnedVersion)
	require.Equal(t, 1, pinRowCount())

	// The policies attached above block a delete, so remove them first.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team.ID),
		fleet.DeleteTeamPoliciesRequest{IDs: []uint{installPol.Policy.ID, patchPol.Policy.ID}},
		http.StatusOK, &fleet.DeleteTeamPoliciesResponse{})

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", fmt.Sprint(team.ID))
	require.Equal(t, 0, pinRowCount(), "deleting an FMA must delete its pin row")

	// --- GitOps removal also clears the pin ---

	// Re-add zoom (deleted above) and pin it via GitOps.
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}})
	batchSet([]*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows"), RollbackVersion: "10.0"}})
	require.Equal(t, 1, pinRowCount())

	// Dropping zoom from the config (chrome stays) clears its pin via the
	// title_id-NOT-IN removal path.
	batchSet([]*fleet.SoftwareInstallerPayload{})
	require.Equal(t, 0, pinRowCount(), "removing an FMA via GitOps must delete its pin row")

	// An empty config (remove all software) clears every pin for the team,
	// including chrome's "^149", via the delete-all path.
	teamPinCount := func() int {
		var n int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n, "SELECT COUNT(*) FROM software_title_team_pins WHERE team_id = ?", team.ID)
		})
		return n
	}
	require.NotZero(t, teamPinCount()) // chrome's "^149" pin is present
	var emptyResp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{}, TeamName: team.Name},
		http.StatusAccepted, &emptyResp, "team_name", team.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, emptyResp.RequestUUID)
	require.Zero(t, teamPinCount(), "removing all software via GitOps must delete all team pins")
}

// The x64 and ARM64 builds of a Windows Fleet-maintained app share a title in one fleet, and each host
// gets the build for its CPU.
func (s *integrationEnterpriseTestSuite) TestFleetMaintainedAppArchSiblings() {
	t := s.T()
	ctx := t.Context()

	startFMAServers(t, s.ds, map[string]*fmaTestState{
		"/google-chrome/windows.json":       {version: "150.0", installerBytes: []byte("chrome-x64"), installerPath: "/chrome-x64.msi"},
		"/google-chrome-arm64/windows.json": {version: "150.0", installerBytes: []byte("chrome-arm64"), installerPath: "/chrome-arm64.msi"},
	})

	var teamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new(t.Name())}, http.StatusOK, &teamResp)
	team := teamResp.Team

	fmaID := func(slug string) uint {
		var id uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, "SELECT id FROM fleet_maintained_apps WHERE slug = ?", slug)
		})
		require.NotZero(t, id, slug)
		return id
	}
	x64AppID, arm64AppID := fmaID("google-chrome/windows"), fmaID("google-chrome-arm64/windows")

	var x64Resp, arm64Resp addFleetMaintainedAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps",
		&addFleetMaintainedAppRequest{AppID: x64AppID, TeamID: &team.ID, SelfService: true}, http.StatusOK, &x64Resp)
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps",
		&addFleetMaintainedAppRequest{AppID: arm64AppID, TeamID: &team.ID, SelfService: true}, http.StatusOK, &arm64Resp)
	titleID := x64Resp.SoftwareTitleID
	require.NotZero(t, titleID)
	require.Equal(t, titleID, arm64Resp.SoftwareTitleID)

	var titleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.Packages, 2)
	installerByApp := make(map[uint]uint, 2)
	for _, pkg := range titleResp.SoftwareTitle.Packages {
		require.NotNil(t, pkg.FleetMaintainedAppID)
		require.Len(t, pkg.FleetMaintainedVersions, 1)
		installerByApp[*pkg.FleetMaintainedAppID] = pkg.InstallerID
	}
	x64InstallerID, arm64InstallerID := installerByApp[x64AppID], installerByApp[arm64AppID]
	require.NotZero(t, x64InstallerID)
	require.NotZero(t, arm64InstallerID)

	// Edits name the build.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{TitleID: titleID, TeamID: &team.ID, SelfService: new(false)},
		http.StatusBadRequest, "installer_id is required")
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{TitleID: titleID, TeamID: &team.ID, InstallerID: arm64InstallerID, SelfService: new(false)},
		http.StatusOK, "")

	queuedInstaller := func(hostID uint) uint {
		var id uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `
				SELECT siua.software_installer_id FROM upcoming_activities ua
				JOIN software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
				WHERE ua.host_id = ?`, hostID)
		})
		return id
	}
	for _, c := range []struct {
		cpuType string
		want    uint
	}{
		{"ARM", arm64InstallerID},
		{"x86_64", x64InstallerID},
		{"", x64InstallerID},
	} {
		host := createOrbitEnrolledHost(t, "windows", "-"+c.cpuType, s.ds)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE hosts SET cpu_type = ? WHERE id = ?", c.cpuType, host.ID)
			return err
		})
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)
		require.Equal(t, c.want, queuedInstaller(host.ID), "cpu_type %q", c.cpuType)
	}

	// One patch policy covers both builds.
	var patchPolicy fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Type: new("patch"), PatchSoftwareTitleID: &titleID,
	}, http.StatusOK, &patchPolicy)
	patchPolicyQuery := func() string {
		var resp fleet.GetTeamPolicyByIDResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", team.ID, patchPolicy.Policy.ID), nil, http.StatusOK, &resp)
		return resp.Policy.Query
	}
	require.Contains(t, patchPolicyQuery(), "cpu_type LIKE 'ARM%'")

	// Deleting the whole title is refused up front rather than after removing one build.
	teamParam := strconv.Itoa(int(team.ID)) //nolint:gosec // dismiss G115
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusConflict, "team_id", teamParam)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.Packages, 2)

	// One build can go; the patch policy then covers the other alone.
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
		"team_id", teamParam, "installer_id", fmt.Sprint(arm64InstallerID))
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.Packages, 1)
	require.Equal(t, x64InstallerID, titleResp.SoftwareTitle.Packages[0].InstallerID)
	require.NotContains(t, patchPolicyQuery(), "cpu_type")
}
