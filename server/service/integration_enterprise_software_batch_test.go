package service

// Batch software tests for the enterprise (premium) suite.
//
// Belongs here: the batch-set software installers endpoint (dry runs, side effects, conditional
// downloads, hashes, categories, Fleet variables, policy associations, Fleet-maintained apps
// referenced from a batch) and applying software through fleet specs.
//
// Does not belong here: single-package uploads (integration_enterprise_software_installers_test.go),
// Fleet-maintained app catalog, versioning, and auto-update (integration_enterprise_software_fma_test.go),
// and batch scripts (integration_enterprise_scripts_test.go).

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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestApplyTeamsSoftwareConfig() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// apply with software
	// must not use applyTeamSpecsRequest and marshal it as JSON, as it will set
	// all keys to their zerovalue, and some are only valid with mdm enabled.
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"packages": []map[string]any{
						{
							"url":          "http://foo.com",
							"self_service": true,
							"install_script": map[string]string{
								"path": "./foo/install-script.sh",
							},
							"post_install_script": map[string]string{
								"path": "./foo/post-install-script.sh",
							},
							"pre_install_query": map[string]string{
								"path": "./foo/query.yaml",
							},
						},
						{
							"url": "http://bar.com",
							"install_script": map[string]string{
								"path": "./bar/install-script.sh",
							},
							"post_install_script": map[string]string{
								"path": "./bar/post-install-script.sh",
							},
							"pre_install_query": map[string]string{
								"path": "./bar/query.yaml",
							},
						},
					},
					"app_store_apps": []map[string]any{
						{
							"app_store_id": "1234",
						},
						{
							"app_store_id": "5678",
						},
					},
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	wantSoftwarePackages := []fleet.SoftwarePackageSpec{
		{
			URL:                "http://foo.com",
			SelfService:        true,
			InstallScript:      fleet.TeamSpecSoftwareAsset{Path: "./foo/install-script.sh"},
			PostInstallScript:  fleet.TeamSpecSoftwareAsset{Path: "./foo/post-install-script.sh"},
			PreInstallQuery:    fleet.TeamSpecSoftwareAsset{Path: "./foo/query.yaml"},
			InstallDuringSetup: optjson.Bool{Set: true},
		},
		{
			URL:                "http://bar.com",
			SelfService:        false,
			InstallScript:      fleet.TeamSpecSoftwareAsset{Path: "./bar/install-script.sh"},
			PostInstallScript:  fleet.TeamSpecSoftwareAsset{Path: "./bar/post-install-script.sh"},
			PreInstallQuery:    fleet.TeamSpecSoftwareAsset{Path: "./bar/query.yaml"},
			InstallDuringSetup: optjson.Bool{Set: true},
		},
	}
	wantAppStoreApps := []fleet.TeamSpecAppStoreApp{
		{
			AppStoreID:         "1234",
			InstallDuringSetup: optjson.Bool{Set: true},
		},
		{
			AppStoreID:         "5678",
			InstallDuringSetup: optjson.Bool{Set: true},
		},
	}

	// retrieving the team returns the software
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)

	// apply without custom software specified, should not replace existing software
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with explicitly empty custom software would clear the existing
	// software, but dry-run
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"packages": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)

	// apply with explicitly empty custom app store apps would clear the existing
	// software, but dry-run
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"app_store_apps": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with empty top-level software field, should not clear packages
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":     teamName,
				"software": nil,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with explicitly empty software packages clears the existing software, but not apps
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"packages": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.Packages.Value)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with explicitly empty software apps clears the existing apps
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"app_store_apps": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.Packages.Value)
	require.Empty(t, teamResp.Team.Config.Software.AppStoreApps.Value)

	// patch with an invalid array returns an error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":     teamName,
				"software": []any{"foo", 1},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.Packages.Value)

	// patch with an invalid array returns an error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":           teamName,
				"app_store_apps": []any{"foo", 1},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.AppStoreApps.Value)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallers() {
	t := s.T()
	ctx := context.Background()

	// non-existent team
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{}, http.StatusNotFound, "team_name", "foo")

	// create a team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// software with a bad URL
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{URL: "."},
	}
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusUnprocessableEntity, "team_name", tm.Name)

	// software with a too big URL
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{URL: "https://ftp.mozilla.org/" + strings.Repeat("a", 4000-23)},
	}
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusUnprocessableEntity, "team_name", tm.Name)

	// create an HTTP server to host the software installer
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ruby.deb" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		file, err := os.Open(filepath.Join("testdata", "software-installers", "ruby.deb"))
		assert.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err = io.Copy(w, file)
		assert.NoError(t, err)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// do a request with a URL that returns a 404.
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{URL: srv.URL + "/not_found.pkg"},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	message := waitBatchSetSoftwareInstallersFailed(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.NotEmpty(t, message)
	expectedNotFoundMessage := fmt.Sprintf("validation failed: software.url URL (\"%s/not_found.pkg\") returned \"Not Found\". Please make sure that URLs are reachable from your Fleet server.", srv.URL)
	require.Contains(t, message, expectedNotFoundMessage)

	// do a request with a valid URL
	rubyURL := srv.URL + "/ruby.deb"
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{URL: rubyURL},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)

	var installerHash string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerHash, "SELECT storage_id FROM software_installers WHERE title_id = ?", packages[0].TitleID)
	})

	require.NotEmpty(t, installerHash)

	softwareToInstallBadSecret := []*fleet.SoftwareInstallerPayload{
		{
			URL:           rubyURL,
			InstallScript: "echo $FLEET_SECRET_INVALID",
		},
	}
	resp := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusUnprocessableEntity, "team_name", tm.Name)
	errMsg := extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID")

	resp = s.Do("POST", "/api/latest/fleet/software/batch?dry_run=true", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusAccepted, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Empty(t, errMsg)

	softwareToInstallBadSecret[0].InstallScript = ""
	softwareToInstallBadSecret[0].PostInstallScript = "echo $FLEET_SECRET_ALSO_INVALID"
	resp = s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusUnprocessableEntity, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_ALSO_INVALID")

	resp = s.Do("POST", "/api/latest/fleet/software/batch?dry_run=true", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusAccepted, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Empty(t, errMsg)

	softwareToInstallBadSecret[0].PostInstallScript = ""
	softwareToInstallBadSecret[0].UninstallScript = "echo $FLEET_SECRET_THIRD_INVALID"
	resp = s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusUnprocessableEntity, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_THIRD_INVALID")

	resp = s.Do("POST", "/api/latest/fleet/software/batch?dry_run=true", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusAccepted, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Empty(t, errMsg)

	// batch upload with unsupported shebang in install script fails validation
	softwareToInstallBadShebang := []*fleet.SoftwareInstallerPayload{
		{
			URL:           rubyURL,
			InstallScript: "#!/usr/bin/perl\nprint 'hello'",
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadShebang}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	message = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Contains(t, message, "Interpreter not supported")

	// TODO(roberto): test with a variety of response codes

	// check the application status
	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, 1, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	// Check that the URL is set to software installers uploaded via batch.
	require.NotNil(t, titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)
	require.Equal(t, rubyURL, *titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)

	// check that platform is set when the installer is created
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var platform string
		if err := sqlx.GetContext(context.Background(), q, &platform, `SELECT platform FROM software_installers WHERE title_id= ? AND team_id = ?`, titlesResp.SoftwareTitles[0].ID, tm.ID); err != nil {
			return err
		}
		require.Equal(t, "linux", platform)
		return nil
	})

	// same payload doesn't modify anything
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	newTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, titlesResp, newTitlesResp)

	// setting self-service to true updates the software title metadata
	softwareToInstall[0].SelfService = true
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	newTitlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	titlesResp.SoftwareTitles[0].SoftwarePackage.SelfService = new(true)
	titlesResp.SoftwareTitles[0].Packages[0].SelfService = new(true)
	require.Equal(t, titlesResp, newTitlesResp)

	// empty payload cleans the software items
	softwareToInstall = []*fleet.SoftwareInstallerPayload{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Empty(t, packages)
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, 0, titlesResp.Count)
	require.Empty(t, titlesResp.SoftwareTitles)

	//////////////////////////
	// Do a request with a valid URL with no team
	//////////////////////////
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{URL: rubyURL, SHA256: installerHash},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)

	// check the application status on team 0
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	require.Equal(t, 1, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 1)

	// same payload doesn't modify anything
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)
	newTitlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	require.Equal(t, titlesResp, newTitlesResp)

	// setting self-service to true updates the software title metadata
	softwareToInstall[0].SelfService = true
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)
	newTitlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	titlesResp.SoftwareTitles[0].SoftwarePackage.SelfService = new(true)
	titlesResp.SoftwareTitles[0].Packages[0].SelfService = new(true)
	require.Equal(t, titlesResp, newTitlesResp)

	// create some labels A, B and C
	lblA, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "A"})
	require.NoError(t, err)
	lblB, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "B"})
	require.NoError(t, err)
	lblC, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "C"})
	require.NoError(t, err)

	// validate that providing more than 1 type of label
	// results in an error
	testCases := []struct {
		desc    string
		incAny  []string
		exclAny []string
		incAll  []string
	}{
		{
			desc:    "include_any_exclude_any",
			incAny:  []string{lblA.Name},
			exclAny: []string{lblB.Name},
		},
		{
			desc:   "include_any_include_all",
			incAny: []string{lblA.Name},
			incAll: []string{lblB.Name},
		},
		{
			desc:    "exclude_any_include_all",
			exclAny: []string{lblA.Name},
			incAll:  []string{lblB.Name},
		},
		{
			desc:    "all_types",
			incAny:  []string{lblA.Name},
			exclAny: []string{lblB.Name},
			incAll:  []string{lblC.Name},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			softwareToInstall = []*fleet.SoftwareInstallerPayload{
				{
					URL:              rubyURL,
					LabelsIncludeAny: tc.incAny,
					LabelsExcludeAny: tc.exclAny,
					LabelsIncludeAll: tc.incAll,
				},
			}
			res := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusBadRequest)
			assert.Contains(t, extractServerErrorText(res.Body), `Only one of "labels_include_all", "labels_include_any" or "labels_exclude_any" can be included.`)
		})
	}

	// providing a non-existing label results in an error
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{URL: rubyURL, LabelsIncludeAny: []string{"no-such-label"}},
	}
	res := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `Couldn't update. Label "no-such-label" doesn't exist. Please remove the label from the software.`)

	// valid installer scoped by label
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{URL: rubyURL, LabelsIncludeAny: []string{lblA.Name}},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)
	meta, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *packages[0].TitleID, false)
	require.NoError(t, err)
	require.Empty(t, meta.LabelsExcludeAny)
	require.Empty(t, meta.LabelsIncludeAll)
	require.Len(t, meta.LabelsIncludeAny, 1)
	require.Equal(t, lblA.ID, meta.LabelsIncludeAny[0].LabelID)
	require.Equal(t, lblA.Name, meta.LabelsIncludeAny[0].LabelName)

	// empty payload cleans the software items
	softwareToInstall = []*fleet.SoftwareInstallerPayload{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Empty(t, packages)
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	require.Equal(t, 0, titlesResp.Count)
	require.Empty(t, titlesResp.SoftwareTitles)

	//////////////////////////
	// Do a request with a fleet maintained app
	//////////////////////////
	oldTransport := http.DefaultTransport
	onePassMockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("mocked content")); err != nil {
			// Handle the error, e.g., log it or fail the test
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer onePassMockServer.Close()
	mockTransport := &mockRoundTripper{
		mockServer:  onePassMockServer.URL,
		origBaseURL: "https://downloads.1password.com",
		next:        http.DefaultTransport,
	}
	http.DefaultTransport = mockTransport

	// https://downloads.1password.com/mac/1Password-8.10.82-aarch64.zip
	maintained1, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "1Password",
		Slug:             "1password/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.1password.1password",
	})
	require.NoError(t, err)

	// basic fleet maintained app with install script override
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{Slug: &maintained1.Slug, InstallScript: "echo 'Hello world'"},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].URL)
	require.Nil(t, packages[0].TeamID)

	var softwareTitleResponse getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *packages[0].TitleID), nil, http.StatusOK, &softwareTitleResponse, "teamId", "0")
	require.Equal(t, "echo 'Hello world'", softwareTitleResponse.SoftwareTitle.SoftwarePackage.InstallScript)
	require.Contains(t, softwareTitleResponse.SoftwareTitle.SoftwarePackage.UninstallScript, "LOGGED_IN_USER") // should pass through FMA script

	// with a team
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)

	// with a label
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{Slug: &maintained1.Slug, LabelsIncludeAll: []string{lblA.Name}},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].URL)
	require.Nil(t, packages[0].TeamID)
	meta, err = s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *packages[0].TitleID, false)
	require.NoError(t, err)
	require.Empty(t, meta.LabelsExcludeAny)
	require.Empty(t, meta.LabelsIncludeAny)
	require.Len(t, meta.LabelsIncludeAll, 1)
	require.Equal(t, lblA.ID, meta.LabelsIncludeAll[0].LabelID)
	require.Equal(t, lblA.Name, meta.LabelsIncludeAll[0].LabelName)

	// maintained app with no_check for sha, latest for version
	maintained2, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Google Chrome",
		Slug:             "google-chrome/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.google.Chrome",
	})
	require.NoError(t, err)

	chromeBytes, err := os.ReadFile(filepath.Join("testdata", "software-installers", "dummy_installer.pkg"))
	require.NoError(t, err)
	h := sha256.New()
	_, err = h.Write(chromeBytes)
	require.NoError(t, err)
	chromeSHA := hex.EncodeToString(h.Sum(nil))
	chromeMockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(chromeBytes); err != nil {
			// Handle the error, e.g., log it or fail the test
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer chromeMockServer.Close()
	mockTransport = &mockRoundTripper{
		mockServer:  chromeMockServer.URL,
		origBaseURL: "https://dl.google.com",
		next:        http.DefaultTransport,
	}
	http.DefaultTransport = mockTransport

	// Mock server to serve manifest with no_check/latest
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "latest",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       "https://dl.google.com/chrome.pkg",
			InstallScriptRef:   "foobaz",
			UninstallScriptRef: "foobaz",
			SHA256:             "no_check",
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
	defer manifestServer.Close()
	mockTransport = &mockRoundTripper{
		mockServer:  manifestServer.URL,
		origBaseURL: "https://maintained-apps.fleetdm.com",
		next:        http.DefaultTransport,
	}
	http.DefaultTransport = mockTransport

	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{Slug: &maintained2.Slug, LabelsIncludeAll: []string{lblA.Name}},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].URL)
	// Given that the manifest returns "no_check" for Chrome hash, the SHA should be calculated from the downloaded file.
	require.Equal(t, chromeSHA, packages[0].HashSHA256)
	require.Nil(t, packages[0].TeamID)

	// Given that the manifest returns "latest", the version should be parsed from the downloaded file.
	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", *packages[0].TitleID), nil, http.StatusOK, &titleResponse, "team_id", "0")
	require.Equal(t, "1.0.0", titleResponse.SoftwareTitle.SoftwarePackage.Version)

	meta, err = s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *packages[0].TitleID, false)
	require.NoError(t, err)

	// Validate labels
	require.Empty(t, meta.LabelsExcludeAny)
	require.Empty(t, meta.LabelsIncludeAny)
	require.Len(t, meta.LabelsIncludeAll, 1)
	require.Equal(t, lblA.ID, meta.LabelsIncludeAll[0].LabelID)
	require.Equal(t, lblA.Name, meta.LabelsIncludeAll[0].LabelName)

	http.DefaultTransport = oldTransport
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersDryRunEmptyShortCircuit() {
	t := s.T()

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// No-team / global endpoint: the path the customer named in #42607.
	// Omit the team_name query arg to exercise the global flow.
	var globalResp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch?dry_run=true",
		batchSetSoftwareInstallersRequest{Software: nil},
		http.StatusAccepted, &globalResp)
	require.Empty(t, globalResp.RequestUUID, "global dry-run + nil software should return empty request_uuid")

	globalResp = batchSetSoftwareInstallersResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch?dry_run=true",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{}},
		http.StatusAccepted, &globalResp)
	require.Empty(t, globalResp.RequestUUID, "global dry-run + empty software should return empty request_uuid")

	// Team-scoped: dry-run with nil software should short-circuit and return an
	// empty request_uuid (no async round-trip, no Redis key written).
	var batchResp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch?dry_run=true",
		batchSetSoftwareInstallersRequest{Software: nil},
		http.StatusAccepted, &batchResp, "team_name", tm.Name)
	require.Empty(t, batchResp.RequestUUID, "dry-run + nil software should return empty request_uuid")

	// Same with an explicitly empty slice.
	batchResp = batchSetSoftwareInstallersResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch?dry_run=true",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{}},
		http.StatusAccepted, &batchResp, "team_name", tm.Name)
	require.Empty(t, batchResp.RequestUUID, "dry-run + empty software should return empty request_uuid")

	// Genuine-miss path on the result endpoint must still 404. The short-circuit
	// doesn't change that contract for unknown UUIDs.
	s.Do("GET", "/api/latest/fleet/software/batch/00000000-0000-0000-0000-000000000000",
		nil, http.StatusNotFound, "team_name", tm.Name)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersSideEffects() {
	t := s.T()

	// create a team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// create an HTTP server to host the software installer
	trailer := ""
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(filepath.Join("testdata", "software-installers", "ruby.deb"))
		assert.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err = io.Copy(w, file)
		assert.NoError(t, err)
		_, err = w.Write([]byte(trailer))
		assert.NoError(t, err)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// set up software to install
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{URL: srv.URL},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)
	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	uploadedAt := titleResponse.SoftwareTitle.SoftwarePackage.UploadedAt

	// create a host that doesn't have fleetd installed
	h, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + uuid.New().String()),
		NodeKey:         new(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "ubuntu",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tm.ID, []uint{h.ID}))
	require.NoError(t, err)
	h.TeamID = &tm.ID

	// host installs fleetd
	orbitKey := setOrbitEnrollment(t, h, s.ds)
	h.OrbitNodeKey = &orbitKey

	// create another host that doesn't have fleetd installed
	h2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + uuid.New().String()),
		NodeKey:         new(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sbar.local", t.Name()),
		Platform:        "ubuntu",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tm.ID, []uint{h2.ID}))
	require.NoError(t, err)
	h2.TeamID = &tm.ID

	// host installs fleetd
	orbitKey2 := setOrbitEnrollment(t, h2, s.ds)
	h2.OrbitNodeKey = &orbitKey2

	// install software
	installResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &installResp)

	// Get the install response, should be pending
	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSoftwareResp.Software[0].Status)

	// Switch self-service flag
	softwareToInstall[0].SelfService = true
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)
	newTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.True(t, *newTitlesResp.SoftwareTitles[0].SoftwarePackage.SelfService)

	// Install should still be pending
	afterSelfServiceHostResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterSelfServiceHostResp)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSoftwareResp.Software[0].Status)

	// update pre-install query
	withUpdatedPreinstallQuery := []*fleet.SoftwareInstallerPayload{
		{URL: srv.URL, PreInstallQuery: "SELECT * FROM os_version"},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: withUpdatedPreinstallQuery}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)
	titleResponse = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, "SELECT * FROM os_version", titleResponse.SoftwareTitle.SoftwarePackage.PreInstallQuery)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)

	// install should no longer be pending
	afterPreinstallHostResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterPreinstallHostResp, "include_available_for_install", "true")
	require.Nil(t, afterPreinstallHostResp.Software[0].Status)

	// install software fully
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &installResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	installUUID := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *h.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// ensure install count is updated
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uint(1), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)

	// install should show as complete
	hostResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &hostResp, "include_available_for_install", "true")
	require.Equal(t, fleet.SoftwareInstalled, *hostResp.Software[0].Status)

	// update install script
	withUpdatedInstallScript := []*fleet.SoftwareInstallerPayload{
		{URL: srv.URL, InstallScript: "apt install ruby"},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: withUpdatedInstallScript}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)

	// ensure install count is the same, and uploaded_at hasn't changed
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uint(1), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)
	require.Equal(t, uploadedAt, titleResponse.SoftwareTitle.SoftwarePackage.UploadedAt)

	// install should still show as complete
	hostResp = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &hostResp, "include_available_for_install", "true")
	require.Equal(t, fleet.SoftwareInstalled, *hostResp.Software[0].Status)

	trailer = " " // add a character to the response for the installer HTTP call to ensure the file hashes differently
	// update package
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: withUpdatedInstallScript}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)

	// ensure install count is zeroed and uploaded_at HAS changed
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)
	require.NotEqual(t, uploadedAt, titleResponse.SoftwareTitle.SoftwarePackage.UploadedAt)

	// install should be nulled out
	hostResp = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &hostResp, "include_available_for_install", "true")
	require.Nil(t, hostResp.Software[0].Status)

	// install details record should still show as installed
	installDetailsResp := getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &installDetailsResp)
	require.Equal(t, fleet.SoftwareInstalled, installDetailsResp.Results.Status)

	// queue another install before we delete the installer via batch
	pendingResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &pendingResp)

	// install should show as pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterPreinstallHostResp, "include_available_for_install", "true")
	require.Equal(t, fleet.SoftwareInstallPending, *afterPreinstallHostResp.Software[0].Status)
	installUUID = afterPreinstallHostResp.Software[0].SoftwarePackage.LastInstall.InstallUUID

	// queue an uninstall on another host
	uninstallResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h2.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &uninstallResp)

	// uninstall should show as pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &afterPreinstallHostResp, "include_available_for_install", "true")
	require.Equal(t, fleet.SoftwareUninstallPending, *afterPreinstallHostResp.Software[0].Status)

	// delete all installers
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{}}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Empty(t, packages)

	// software should no longer exist on either host
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Empty(t, afterPreinstallHostResp.Software)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Empty(t, afterPreinstallHostResp.Software)

	// pending install record should not exist
	installDetailsResp = getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusNotFound, &installDetailsResp)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersMSIReusePreservesUpgradeCode() {
	t := s.T()
	ctx := t.Context()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	tmB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-B"})
	require.NoError(t, err)

	const etag = `"fleet-osquery-msi-v1"`
	var (
		mu          sync.Mutex
		ifNoneMatch []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ifNoneMatch = append(ifNoneMatch, r.Header.Get("If-None-Match"))
		mu.Unlock()
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		http.ServeFile(w, r, filepath.Join("testdata", "software-installers", "fleet-osquery.msi"))
	}))
	t.Cleanup(srv.Close)
	msiURL := srv.URL + "/fleet-osquery.msi"

	type installerState struct {
		InstallerID          uint   `db:"installer_id"`
		TitleID              uint   `db:"title_id"`
		StorageID            string `db:"storage_id"`
		InstallerUpgradeCode string `db:"installer_upgrade_code"`
		TitleUpgradeCode     string `db:"title_upgrade_code"`
		UninstallScript      string `db:"uninstall_script"`
	}
	getState := func(teamID uint) installerState {
		var st installerState
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &st, `
				SELECT si.id AS installer_id, si.title_id, si.storage_id,
					si.upgrade_code AS installer_upgrade_code,
					COALESCE(st.upgrade_code, '') AS title_upgrade_code,
					sc.contents AS uninstall_script
				FROM software_installers si
				JOIN software_titles st ON st.id = si.title_id
				JOIN script_contents sc ON sc.id = si.uninstall_script_content_id
				WHERE si.global_or_team_id = ? AND si.is_active = 1`, teamID)
		})
		return st
	}
	batchDeleted := func(teamName string, payloads []*fleet.SoftwareInstallerPayload, dryRun bool) []fleet.DeletedSoftwarePackage {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: payloads},
			http.StatusAccepted, &resp, "team_name", teamName, "dry_run", strconv.FormatBool(dryRun))
		result := waitBatchSetSoftwareInstallers(t, &s.withServer, teamName, resp.RequestUUID)
		require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, result.Status, result.Message)
		return result.DeletedPackages
	}
	batch := func(teamName string, payload *fleet.SoftwareInstallerPayload, dryRun bool) {
		require.Empty(t, batchDeleted(teamName, []*fleet.SoftwareInstallerPayload{payload}, dryRun))
	}

	// Fresh download and parse; the uninstall script is omitted so Fleet generates it.
	batch(tm.Name, &fleet.SoftwareInstallerPayload{URL: msiURL}, false)
	fresh := getState(tm.ID)
	require.Equal(t, "{B681CB20-107E-428A-9B14-2D3C1AFED244}", fresh.InstallerUpgradeCode)
	require.Equal(t, fresh.InstallerUpgradeCode, fresh.TitleUpgradeCode)
	require.Contains(t, fresh.UninstallScript, fresh.InstallerUpgradeCode)

	// Reuse by hash from another fleet: fleet B gets its own installer with the same metadata.
	for _, dryRun := range []bool{true, false} {
		batch(tmB.Name, &fleet.SoftwareInstallerPayload{URL: msiURL, SHA256: fresh.StorageID}, dryRun)
	}
	copied := getState(tmB.ID)
	require.NotEqual(t, fresh.InstallerID, copied.InstallerID)
	copied.InstallerID = fresh.InstallerID
	require.Equal(t, fresh, copied)
	require.Equal(t, fresh, getState(tm.ID))

	// Reuse via conditional download (HTTP 304).
	batch(tm.Name, &fleet.SoftwareInstallerPayload{URL: msiURL}, false)
	require.Equal(t, fresh, getState(tm.ID))

	// Reuse by hash, without a download.
	for _, dryRun := range []bool{true, false} {
		batch(tm.Name, &fleet.SoftwareInstallerPayload{URL: msiURL, SHA256: fresh.StorageID}, dryRun)
		require.Equal(t, fresh, getState(tm.ID))
	}

	// Removing the MSI from the payload still reports it.
	deleted := batchDeleted(tm.Name, []*fleet.SoftwareInstallerPayload{}, true)
	require.Len(t, deleted, 1)
	require.Equal(t, fresh.TitleID, deleted[0].TitleID)

	mu.Lock()
	require.Equal(t, []string{"", etag}, ifNoneMatch)
	mu.Unlock()
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersKeepsTitleAttachedByName() {
	t := s.T()
	ctx := t.Context()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// A title from inventory with no bundle ID. The lookup prefers a title with the package's bundle ID, so drop any left by other tests.
	var titleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(ctx, `DELETE FROM software_titles WHERE name = 'DummyApp' OR bundle_identifier = 'com.example.dummy'`); err != nil {
			return err
		}
		res, err := q.ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES ('dummyapp', 'apps')`)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		titleID = uint(id) //nolint:gosec // dismiss G115
		return err
	})

	srv := httptest.NewServer(http.FileServer(http.Dir(filepath.Join("testdata", "software-installers"))))
	t.Cleanup(srv.Close)
	pkg := []*fleet.SoftwareInstallerPayload{{URL: srv.URL + "/dummy_installer.pkg"}}
	batchDeleted := func(payloads []*fleet.SoftwareInstallerPayload, dryRun bool) []fleet.DeletedSoftwarePackage {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: payloads},
			http.StatusAccepted, &resp, "team_name", tm.Name, "dry_run", strconv.FormatBool(dryRun))
		result := waitBatchSetSoftwareInstallers(t, &s.withServer, tm.Name, resp.RequestUUID)
		require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, result.Status, result.Message)
		return result.DeletedPackages
	}

	// The apply attaches the package to the inventory title by name.
	require.Empty(t, batchDeleted(pkg, false))
	var installerTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerTitleID, `SELECT title_id FROM software_installers WHERE global_or_team_id = ?`, tm.ID)
	})
	require.Equal(t, titleID, installerTitleID)

	// Batching it again reports nothing, dry run or not.
	for _, dryRun := range []bool{true, false} {
		require.Empty(t, batchDeleted(pkg, dryRun))
	}

	// Removing the package from the payload still reports it.
	deleted := batchDeleted([]*fleet.SoftwareInstallerPayload{}, true)
	require.Len(t, deleted, 1)
	require.Equal(t, titleID, deleted[0].TitleID)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersConditionalDownload() {
	t := s.T()

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	var (
		mu             sync.Mutex
		requestCount   int
		ifNoneMatchSet = []string{}
		etag           = `"ruby-deb-v1"`
		trailer        = "" // appended to the body to force a different content hash
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		ifNoneMatchSet = append(ifNoneMatchSet, r.Header.Get("If-None-Match"))
		currentETag := etag
		currentTrailer := trailer
		mu.Unlock()

		// Honor the conditional request: if the client sends our current
		// ETag, return 304 with no body.
		if r.Header.Get("If-None-Match") == currentETag {
			w.Header().Set("ETag", currentETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}

		file, err := os.Open(filepath.Join("testdata", "software-installers", "ruby.deb"))
		if !assert.NoError(t, err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		w.Header().Set("ETag", currentETag)
		_, err = io.Copy(w, file)
		assert.NoError(t, err)
		_, err = w.Write([]byte(currentTrailer))
		assert.NoError(t, err)
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// First run: full download, Fleet should capture the ETag and storage_id.
	softwareToInstall := []*fleet.SoftwareInstallerPayload{{URL: srv.URL}}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: softwareToInstall},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	firstHash := packages[0].HashSHA256
	require.NotEmpty(t, firstHash)

	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp,
		"available_for_install", "true", "team_id", fmt.Sprint(tm.ID))
	require.Len(t, titlesResp.SoftwareTitles, 1)
	titleID := titlesResp.SoftwareTitles[0].ID

	titleResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"team_id", fmt.Sprint(tm.ID))
	firstUploadedAt := titleResp.SoftwareTitle.SoftwarePackage.UploadedAt

	mu.Lock()
	require.Equal(t, 1, requestCount, "expected exactly one HTTP request on first run")
	require.Empty(t, ifNoneMatchSet[0], "first run must not send If-None-Match")
	mu.Unlock()

	// Second run: same URL, no changes. The server will return 304 and Fleet
	// should reuse the cached installer (uploaded_at must not change).
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: softwareToInstall},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.Equal(t, firstHash, packages[0].HashSHA256, "hash must match on cache hit")

	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, firstUploadedAt, titleResp.SoftwareTitle.SoftwarePackage.UploadedAt,
		"uploaded_at must not change on 304 cache hit")

	mu.Lock()
	require.Equal(t, 2, requestCount, "expected one additional HTTP request on second run")
	require.Equal(t, etag, ifNoneMatchSet[1], "second run must send If-None-Match with stored ETag")
	mu.Unlock()

	// Third run: server will again respond 304, but the user has supplied an
	// install_script that fails validation (exceeds SavedScriptMaxRuneLen).
	// The 304 fast-path must still run script validation and reject the batch.
	oversizeScript := strings.Repeat("a", fleet.SavedScriptMaxRuneLen+1)
	withBadScript := []*fleet.SoftwareInstallerPayload{{URL: srv.URL, InstallScript: oversizeScript}}
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: withBadScript},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	msg := waitBatchSetSoftwareInstallersFailed(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Contains(t, msg, "install script")

	// Fourth run: always_download bypasses conditional download, so the server
	// should receive a request with no If-None-Match header and return the full
	// body.
	withAlwaysDownload := []*fleet.SoftwareInstallerPayload{{URL: srv.URL, AlwaysDownload: true}}
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: withAlwaysDownload},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	_ = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)

	mu.Lock()
	require.GreaterOrEqual(t, requestCount, 4, "always_download must issue an HTTP request")
	require.Empty(t, ifNoneMatchSet[len(ifNoneMatchSet)-1], "always_download must not send If-None-Match")
	mu.Unlock()

	// Fifth run: flip always_download back off. The ETag captured during the
	// previous always_download run must still be usable, so this run should
	// immediately go back to sending If-None-Match and receiving 304 — no
	// "warm-up" download required.
	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"team_id", fmt.Sprint(tm.ID))
	uploadedAtBeforeConditional := titleResp.SoftwareTitle.SoftwarePackage.UploadedAt

	mu.Lock()
	countBeforeConditional := requestCount
	mu.Unlock()

	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: softwareToInstall},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)

	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uploadedAtBeforeConditional, titleResp.SoftwareTitle.SoftwarePackage.UploadedAt,
		"uploaded_at must not change when ETag from prior always_download run is reused")

	mu.Lock()
	require.Equal(t, countBeforeConditional+1, requestCount, "expected one HTTP request after re-enabling conditional download")
	require.Equal(t, `"ruby-deb-v1"`, ifNoneMatchSet[countBeforeConditional],
		"must send the ETag captured during the previous always_download run")
	mu.Unlock()

	// Sixth run: upstream content has changed — new ETag and modified body.
	// Fleet still sends the stored ETag (v1) as If-None-Match; the server
	// returns 200 with the new bytes and new ETag. Fleet must re-upload, bump
	// uploaded_at, and persist the new ETag.
	const newETag = `"ruby-deb-v2"`
	mu.Lock()
	etag = newETag
	trailer = " " // changes the body hash
	countBeforeContentChange := requestCount
	mu.Unlock()

	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"team_id", fmt.Sprint(tm.ID))
	uploadedAtBeforeContentChange := titleResp.SoftwareTitle.SoftwarePackage.UploadedAt
	hashBeforeContentChange := titleResp.SoftwareTitle.SoftwarePackage.StorageID

	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: softwareToInstall},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotEqual(t, hashBeforeContentChange, packages[0].HashSHA256, "hash must change when upstream content changes")

	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"team_id", fmt.Sprint(tm.ID))
	require.NotEqual(t, uploadedAtBeforeContentChange, titleResp.SoftwareTitle.SoftwarePackage.UploadedAt,
		"uploaded_at must advance when bytes are re-downloaded")

	mu.Lock()
	require.Equal(t, countBeforeContentChange+1, requestCount, "expected one HTTP request on sixth run")
	require.Equal(t, `"ruby-deb-v1"`, ifNoneMatchSet[countBeforeContentChange],
		"sixth run must send the previously stored ETag (v1) so the server can detect the mismatch")
	mu.Unlock()

	// Seventh run: no further content change. Fleet should now be sending the
	// new ETag (v2) it captured from run 6, and the server should reply 304.
	mu.Lock()
	countBeforeFinalRun := requestCount
	mu.Unlock()

	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: softwareToInstall},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)

	mu.Lock()
	require.Equal(t, countBeforeFinalRun+1, requestCount, "expected one HTTP request on seventh run")
	require.Equal(t, newETag, ifNoneMatchSet[countBeforeFinalRun],
		"seventh run must send the new ETag captured from the content-change run")
	mu.Unlock()
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersWithPoliciesAssociated() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	policy1Team1, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:  "team1Policy1",
			Query: "SELECT 1;",
		},
	)
	require.NoError(t, err)
	policy2Team2, err := s.ds.NewTeamPolicy(
		ctx, team2.ID, nil, fleet.PolicyPayload{
			Name:  "team2Policy2",
			Query: "SELECT 2;",
		},
	)
	require.NoError(t, err)

	// create an HTTP server to host software installers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var fileName string
		switch r.URL.Path {
		case "/ruby.deb", "/dummy_installer.pkg":
			fileName = strings.TrimPrefix(r.URL.Path, "/")
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		file, err := os.Open(filepath.Join("testdata", "software-installers", fileName))
		assert.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err = io.Copy(w, file)
		assert.NoError(t, err)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// team1 has ruby.deb
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{
			URL: srv.URL + "/ruby.deb",
		},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team1.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL+"/ruby.deb", packages[0].URL)

	// team2 has dummy_installer.pkg and ruby.deb.
	softwareToInstall = []*fleet.SoftwareInstallerPayload{
		{
			URL: srv.URL + "/dummy_installer.pkg",
		},
		{
			URL: srv.URL + "/ruby.deb",
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].URL < packages[j].URL
	})
	require.Len(t, packages, 2)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team2.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL+"/dummy_installer.pkg", packages[0].URL)
	require.NotNil(t, packages[1].TitleID)
	require.NotNil(t, packages[1].TeamID)
	require.Equal(t, team2.ID, *packages[1].TeamID)
	require.Equal(t, srv.URL+"/ruby.deb", packages[1].URL)

	// Associate ruby.deb to policy1Team1.
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyDebTitleID := resp.SoftwareTitles[0].ID
	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
	}, http.StatusOK, &mtplr)

	// Associate ruby.deb in team2 to policy2Team2.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy2Team2.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
	}, http.StatusOK, &mtplr)

	// Get rid of all installers in team1.
	softwareToInstall = []*fleet.SoftwareInstallerPayload{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Empty(t, packages)

	// policy1Team1 should not be associated to any installer.
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.SoftwareInstallerID)
	// team1 should be empty.
	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(team1.ID))
	require.Equal(t, 0, titlesResp.Count)

	// team2 should be untouched.
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(team2.ID))
	require.Equal(t, 2, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 2)
	require.NotNil(t, titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)
	require.Equal(t, srv.URL+"/dummy_installer.pkg", *titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)
	require.NotNil(t, titlesResp.SoftwareTitles[1].SoftwarePackage.PackageURL)
	require.Equal(t, srv.URL+"/ruby.deb", *titlesResp.SoftwareTitles[1].SoftwarePackage.PackageURL)

	// policy2Team2 should still be associated to ruby.deb of team2.
	policy2Team2, err = s.ds.Policy(ctx, policy2Team2.ID)
	require.NoError(t, err)
	require.NotNil(t, policy2Team2.SoftwareInstallerID)
}

func (s *integrationEnterpriseTestSuite) TestBatchSoftwareUploadWithSHAs() {
	t := s.T()

	ctx := context.Background()

	// create a team
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// create an HTTP server to host the software installer
	var hitDebURL, hitExeURL, hitPkgURL bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ruby.deb", "/updated/ruby.deb":
			hitDebURL = true
			file, err := os.Open(filepath.Join("testdata", "software-installers", "ruby.deb"))
			assert.NoError(t, err)
			defer file.Close()
			w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
			_, err = io.Copy(w, file)
			assert.NoError(t, err)
		case "/app.exe":
			hitExeURL = true
			file, err := os.Open(filepath.Join("..", "..", "pkg", "file", "testdata", "software-installers", "hello-world-installer.exe"))
			assert.NoError(t, err)
			defer file.Close()
			w.Header().Set("Content-Type", "application/vnd.microsoft.portable-executable")
			_, err = io.Copy(w, file)
			assert.NoError(t, err)
		case "/app.pkg":
			hitPkgURL = true
			file, err := os.Open(filepath.Join("testdata", "software-installers", "dummy_installer.pkg"))
			assert.NoError(t, err)
			defer file.Close()
			w.Header().Set("Content-Type", "application/x-newton-compatible-pkg")
			_, err = io.Copy(w, file)
			assert.NoError(t, err)

		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// do a request with a valid URL, but invalid hash, should fail
	rubyURL := srv.URL + "/ruby.deb"
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{URL: rubyURL, SHA256: "foobar"},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	errMsg := waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Equal(t, fmt.Sprintf("downloaded installer hash does not match provided hash for installer with url %s", rubyURL), errMsg)

	// payload without URL or hash should fail
	softwareToInstall[0].SHA256 = ""
	softwareToInstall[0].URL = ""
	resp := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusUnprocessableEntity, "team_name", team1.Name)
	require.Contains(t, extractServerErrorText(resp.Body), "Couldn't edit software. One or more software packages is missing url or hash_sha256 fields.")

	// Pass in valid URL
	softwareToInstall[0].URL = rubyURL
	softwareToInstall[0].SHA256 = ""
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team1.ID, *packages[0].TeamID)
	require.True(t, hitDebURL)
	hitDebURL = false

	var debHash string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &debHash, "SELECT storage_id FROM software_installers WHERE title_id = ?", packages[0].TitleID)
	})
	require.NotEmpty(t, debHash)

	// add the hash to the payload
	softwareToInstall[0].SHA256 = debHash

	// dry run shouldn't hit the download endpoint since we included the SHA
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name, "dry_run", "true")
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team1.ID, *packages[0].TeamID)
	require.False(t, hitDebURL)

	// since we provided the SHA and we'd already added the installer, we should not hit the
	// download endpoint
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team1.ID, *packages[0].TeamID)
	require.False(t, hitDebURL)

	// check that URL comes back on individual title fetch
	stResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *packages[0].TitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team1.ID))
	require.NotNil(t, stResp.SoftwareTitle.SoftwarePackage)
	require.Equal(t, "ruby", stResp.SoftwareTitle.Name)
	require.Equal(t, rubyURL, stResp.SoftwareTitle.SoftwarePackage.URL)

	// create a new team
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + "2",
		Description: "desc",
	})
	require.NoError(t, err)

	// create a new user for team 2; doesn't have access to team 1
	team2Admin := &fleet.User{
		Name:       "Team 2 Admin",
		Email:      uuid.NewString() + "@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team2,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, team2Admin.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), team2Admin)
	require.NoError(t, err)

	s.setTokenForTest(t, team2Admin.Email, test.GoodPassword)

	// make sure to not break other tests
	defer func() { s.token = s.getTestAdminToken() }()

	// Remove the URL; since the user doesn't have access to team 1 and only the hash is provided,
	// this should fail
	softwareToInstall[0].URL = ""
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, "package not found with hash")

	// user doesn't have access to team1: we should hit the download endpoint
	softwareToInstall[0].URL = rubyURL
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name, "dry_run", "true")
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Empty(t, packages)
	require.True(t, hitDebURL)
	hitDebURL = false

	// same for real run
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team2.ID, *packages[0].TeamID)
	require.True(t, hitDebURL)
	hitDebURL = false

	// Send payload with just the SHA; should succeed and not hit the download endpoint because we
	// downloaded it just above
	softwareToInstall[0].URL = ""
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Empty(t, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team2.ID, *packages[0].TeamID)
	require.False(t, hitDebURL)

	// Update the URL, should re-download
	updatedRubyURL := srv.URL + "/updated/ruby.deb"
	softwareToInstall[0].URL = updatedRubyURL
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, updatedRubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team2.ID, *packages[0].TeamID)
	require.True(t, hitDebURL)
	hitDebURL = false

	// add exe to payloads
	exeURL := srv.URL + "/app.exe"
	softwareToInstall = append(softwareToInstall, &fleet.SoftwareInstallerPayload{
		URL: exeURL,
	})

	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, "Install script is required for .exe packages.")

	softwareToInstall[1].InstallScript = "echo install"
	softwareToInstall[1].UninstallScript = "echo uninstall"
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 2)
	require.True(t, hitExeURL)
	hitExeURL = false

	var exeHash string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &exeHash, "SELECT storage_id FROM software_installers WHERE title_id = ?", packages[1].TitleID)
	})
	require.NotEmpty(t, exeHash)

	// remove the URL and scripts, and add the hash. we should get an error because scripts are required.
	softwareToInstall[1].InstallScript = ""
	softwareToInstall[1].UninstallScript = ""
	softwareToInstall[1].URL = ""
	softwareToInstall[1].SHA256 = exeHash
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, "Couldn't edit. Install script is required for .exe packages.")
	require.False(t, hitExeURL)

	// add the install script, but without uninstall script we should still fail
	softwareToInstall[1].InstallScript = "echo install"
	softwareToInstall[1].UninstallScript = ""
	softwareToInstall[1].URL = ""
	softwareToInstall[1].SHA256 = exeHash
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, "Couldn't edit. Uninstall script is required for .exe packages.")
	require.False(t, hitExeURL)

	// add both scripts to get a success
	softwareToInstall[1].InstallScript = "echo install 2"
	softwareToInstall[1].UninstallScript = "echo uninstall 2"

	// add the pkg installer with some custom scripts
	pkgURL := srv.URL + "/app.pkg"
	softwareToInstall = append(softwareToInstall, &fleet.SoftwareInstallerPayload{
		URL:             pkgURL,
		InstallScript:   "some install script",
		UninstallScript: "some uninstall script",
	})

	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 3)
	pkgTitleID := packages[2].TitleID
	require.NotNil(t, pkgTitleID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *pkgTitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team2.ID))
	require.NotNil(t, stResp.SoftwareTitle.SoftwarePackage)
	require.Equal(t, "DummyApp", stResp.SoftwareTitle.Name)
	require.Equal(t, pkgURL, stResp.SoftwareTitle.SoftwarePackage.URL)
	require.Equal(t, softwareToInstall[2].InstallScript, stResp.SoftwareTitle.SoftwarePackage.InstallScript)
	require.Equal(t, softwareToInstall[2].UninstallScript, stResp.SoftwareTitle.SoftwarePackage.UninstallScript)
	require.False(t, hitDebURL)
	require.False(t, hitExeURL)
	require.True(t, hitPkgURL)
	hitPkgURL = false

	// Attempt to add the exe to team 1 without scripts. Even though the admin has access to
	// both teams, it should fail because scripts are required for exes.
	// Check without either script first.
	s.token = s.getTestAdminToken()
	softwareToInstall[1].InstallScript = ""
	softwareToInstall[1].UninstallScript = ""
	softwareToInstall[1].URL = ""
	softwareToInstall[1].SHA256 = exeHash
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall[:2]}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, "Couldn't edit. Install script is required for .exe packages.")
	require.False(t, hitExeURL)

	// Now add just an install script, should still fail.
	softwareToInstall[1].InstallScript = "echo foo"
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall[:2]}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, "Couldn't edit. Uninstall script is required for .exe packages.")
	require.False(t, hitExeURL)

	// add the scripts back to not break subsequent calls
	softwareToInstall[1].InstallScript = "echo install 2"
	softwareToInstall[1].UninstallScript = "echo uninstall 2"

	expectedUninstallScript := `#!/bin/sh

# Fleet extracts and saves package IDs.
pkg_ids=(
  'com.example.dummy'
)

# For each package id, get all .app folders associated with the package and remove them.
for pkg_id in "${pkg_ids[@]}"
do
  # Get volume and location of the package.
  volume=$(pkgutil --pkg-info "$pkg_id" | grep -i "volume" | awk '{if (NF>1) print $NF}')
  location=$(pkgutil --pkg-info "$pkg_id" | grep -i "location" | awk '{if (NF>1) print $NF}')
  # Check if this package id corresponds to a valid/installed package
  if [[ ! -z "$volume" ]]; then
    # Remove individual directories that end with ".app" belonging to the package.
    # Only process directories that end with ".app" to prevent Fleet from removing top level directories.
    pkgutil --only-dirs --files "$pkg_id" | grep "\.app$" | sed -e 's@^@'"$volume""$location"'/@' | tr '\n' '\0' | xargs -n 1 -0 rm -rf
    # Remove receipts
    pkgutil --forget "$pkg_id"
  else
    echo "WARNING: volume is empty for package ID $pkg_id"
  fi
done
`

	// remove the custom scripts from the .pkg. We should get back the auto-generated ones.
	softwareToInstall[2].InstallScript = ""
	softwareToInstall[2].UninstallScript = ""
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 3)
	pkgTitleID = packages[2].TitleID
	require.NotNil(t, pkgTitleID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *pkgTitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team2.ID))
	require.NotNil(t, stResp.SoftwareTitle.SoftwarePackage)
	require.Equal(t, "DummyApp", stResp.SoftwareTitle.Name)
	require.Equal(t, pkgURL, stResp.SoftwareTitle.SoftwarePackage.URL)
	require.Equal(t, file.GetInstallScript("pkg"), stResp.SoftwareTitle.SoftwarePackage.InstallScript)
	require.Equal(t, expectedUninstallScript, stResp.SoftwareTitle.SoftwarePackage.UninstallScript)

	// Clean up all installers from the store. We don't care about how many installers are
	// removed in this test, so just check that the cleanup succeeded.
	_, err = s.softwareInstallStore.Cleanup(ctx, nil, time.Now())
	require.NoError(t, err)

	// At this point, the exe payload has no URL. Batch set should fail because the
	// installer bytes don't exist anymore and there is no URL provided.
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	errMsg = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Contains(t, errMsg, fmt.Sprintf("package not found with hash %s", exeHash))

	// Add the URL back and do the batch set. Should succeed and re-download all installers.
	softwareToInstall[1].URL = exeURL
	hitDebURL, hitExeURL, hitPkgURL = false, false, false
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 3)
	for _, v := range []bool{hitDebURL, hitExeURL, hitPkgURL} {
		require.True(t, v)
	}
}

func (s *integrationEnterpriseTestSuite) TestBatchSoftwareInstallerAndFMACategories() {
	t := s.T()

	ctx := context.Background()

	// create a team
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	token := "good_token"
	host := createOrbitEnrolledHost(t, "darwin", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &team1.ID, HostIDs: []uint{host.ID}}, http.StatusOK, &addHostsToTeamResponse{})

	// create an HTTP server to host the software installer
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fma.pkg":
			file, err := os.Open(filepath.Join("testdata", "software-installers", "dummy_installer.pkg"))
			assert.NoError(t, err)
			defer file.Close()
			w.Header().Set("Content-Type", "application/application/x-newton-compatible-pkg")
			_, err = io.Copy(w, file)
			assert.NoError(t, err)

		case "/non-fma.pkg":
			file, err := os.Open(filepath.Join("testdata", "software-installers", "EchoApp.pkg"))
			assert.NoError(t, err)
			defer file.Close()
			w.Header().Set("Content-Type", "application/application/x-newton-compatible-pkg")
			_, err = io.Copy(w, file)
			assert.NoError(t, err)

		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})

	installerServer := httptest.NewServer(handler)
	t.Cleanup(installerServer.Close)

	// Mock server to serve fleet maintained app manifest
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "6.0",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:      installerServer.URL + "/fma.pkg",
			SHA256:            "no_check", // See ma.noCheckHash
			DefaultCategories: []string{"Productivity"},
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
		}

		err := json.NewEncoder(w).Encode(manifest)
		assert.NoError(t, err)
	}))

	t.Cleanup(manifestServer.Close)
	t.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)

	maintained1, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "1Password",
		Slug:             "1password/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.1password.1password",
	})
	require.NoError(t, err)

	// a request with an invalid category silently drops the unknown name
	pkgURL := installerServer.URL + "/non-fma.pkg"
	softwareToInstall := []*fleet.SoftwareInstallerPayload{
		{URL: pkgURL, Categories: optjson.SetSlice([]string{"Not Found"}), SelfService: true},
		{Slug: &maintained1.Slug, SelfService: true},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 2)

	// an over-length category name is rejected up front
	softwareToInstall[0].Categories = optjson.SetSlice([]string{strings.Repeat("x", 256)})
	res := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusUnprocessableEntity, "team_name", team1.Name)
	require.Contains(t, extractServerErrorText(res.Body), "name must be at most 255 characters")

	testCases := []struct {
		desc                 string
		categories           optjson.Slice[string]
		fmaDefaultCategories []string
	}{
		{
			desc:       "duplicate categories provided",
			categories: optjson.SetSlice([]string{"🧰 Developer tools", "🌎 Browsers", "🌎 Browsers"}),
		},
		{
			desc:       "valid categories 1",
			categories: optjson.SetSlice([]string{"🧰 Developer tools", "🌎 Browsers"}),
		},
		{
			desc:       "valid categories 2",
			categories: optjson.SetSlice([]string{"👬 Communication", "🖥️ Productivity"}),
		},
		{
			desc:       "valid categories 3 - Security and Utilities",
			categories: optjson.SetSlice([]string{"🔐 Security", "🛠️ Utilities"}),
		},
		{
			desc:       "valid categories 4 - mixed with new categories",
			categories: optjson.SetSlice([]string{"🔐 Security", "🧰 Developer tools", "🛠️ Utilities"}),
		},
		{
			// omitted categories (unset) fall back to the FMA's manifest default
			desc:                 "omitted categories use FMA default",
			fmaDefaultCategories: []string{"🖥️ Productivity"},
		},
		{
			// an explicitly empty categories list sets zero categories (no manifest default)
			desc:       "explicit empty categories sets zero categories",
			categories: optjson.SetSlice([]string{}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			softwareToInstall[0].Categories = tc.categories
			softwareToInstall[1].Categories = tc.categories

			// remove duplicates if any
			wantCategories := server.RemoveDuplicatesFromSlice(tc.categories.Value)
			s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
			packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team1.Name, batchResponse.RequestUUID)
			require.Len(t, packages, 2)

			for _, p := range packages {
				require.NotNil(t, p.TitleID)
				require.NotNil(t, p.TeamID)
				require.Equal(t, team1.ID, *p.TeamID)

				// check that categories come back on individual title fetch
				stResp := getSoftwareTitleResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *p.TitleID), getSoftwareTitleRequest{}, http.StatusOK, &stResp, "team_id", fmt.Sprint(team1.ID))
				require.NotNil(t, stResp.SoftwareTitle.SoftwarePackage)
				if stResp.SoftwareTitle.SoftwarePackage.FleetMaintainedAppID != nil && !tc.categories.Set {
					// categories omitted on an FMA → fall back to the manifest default
					require.ElementsMatch(t, tc.fmaDefaultCategories, stResp.SoftwareTitle.SoftwarePackage.Categories)
					continue
				}
				require.ElementsMatch(t, wantCategories, stResp.SoftwareTitle.SoftwarePackage.Categories)
			}

			// check that the categories come back on the My device page
			res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
			getDeviceSw := getDeviceSoftwareResponse{}
			err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
			require.NoError(t, err)
			require.Len(t, getDeviceSw.Software, 2)
			for _, s := range getDeviceSw.Software {
				if s.Name == maintained1.Name && !tc.categories.Set {
					// categories omitted on an FMA → fall back to the manifest default
					require.ElementsMatch(t, tc.fmaDefaultCategories, s.SoftwarePackage.Categories)
					continue
				}

				require.ElementsMatch(t, wantCategories, s.SoftwarePackage.Categories)
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersLowercasesExtension() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team_" + t.Name()})
	require.NoError(t, err)

	states := map[string]*fmaTestState{
		"/zoom/windows.json": {
			version:        "1.0",
			installerBytes: []byte("xyz"),
			installerPath:  "/ZOOM-SETUP.MSI",
		},
	}
	startFMAServers(t, s.ds, states)

	var resp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
		http.StatusAccepted, &resp,
		"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
	)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

	// The column collation is case insensitive, so read back with a binary
	// collation to see the stored casing rather than a case-folded match.
	var stored struct {
		Filename  string `db:"filename"`
		Extension string `db:"extension"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &stored,
			`SELECT filename, extension COLLATE utf8mb4_bin AS extension FROM software_installers WHERE global_or_team_id = ?`,
			team.ID)
	})

	require.Equal(t, "ZOOM-SETUP.MSI", stored.Filename, "filename keeps the original casing")
	require.Equal(t, "msi", stored.Extension)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersUnsetsObsoletePatchPolicy() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team_" + t.Name()})
	require.NoError(t, err)

	// Set up two FMAs: zoom/windows and 1password/darwin.
	states := make(map[string]*fmaTestState, 2)
	states["/zoom/windows.json"] = &fmaTestState{
		version:        "1.0",
		installerBytes: []byte("xyz"),
		installerPath:  "/zoom.msi",
	}
	states["/1password/darwin.json"] = &fmaTestState{
		version:        "1.0",
		installerBytes: []byte("abc"),
		installerPath:  "/1password.pkg",
	}
	startFMAServers(t, s.ds, states)

	// Setup: batch set only zoom/windows.
	var resp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
		http.StatusAccepted, &resp,
		"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
	)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

	// Batch set both zoom/windows and 1password/darwin.
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
			{Slug: new("zoom/windows")},
			{Slug: new("1password/darwin")},
		}, TeamName: team.Name},
		http.StatusAccepted, &resp,
		"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
	)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

	// Create a patch policy for 1password/darwin.
	applyResp := fleet.ApplyPolicySpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/policies",
		fleet.ApplyPolicySpecsRequest{Specs: []*fleet.PolicySpec{{
			Name:                   "1password patch policy",
			Team:                   team.Name,
			Type:                   fleet.PolicyTypePatch,
			FleetMaintainedAppSlug: "1password/darwin",
		}}},
		http.StatusOK, &applyResp,
	)

	// Verify the patch policy exists.
	listPolResp := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
	require.Len(t, listPolResp.Policies, 1)
	require.Equal(t, fleet.PolicyTypePatch, listPolResp.Policies[0].Type)
	require.NotNil(t, listPolResp.Policies[0].PatchSoftware)

	// Batch set only zoom/windows — should not fail and should unset the obsolete patch policy's
	// patch_software_title_id without deleting the policy. Gitops then deletes it via the policy
	// delete API so the deletion produces a deleted_policy activity.
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
		http.StatusAccepted, &resp,
		"team_name", team.Name, "team_id", fmt.Sprint(team.ID),
	)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

	// Verify the patch policy still exists but no longer references a software title.
	listPolResp = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.ListTeamPoliciesRequest{}, http.StatusOK, &listPolResp, "page", "0")
	require.Len(t, listPolResp.Policies, 1)
	require.Equal(t, fleet.PolicyTypePatch, listPolResp.Policies[0].Type)
	require.Nil(t, listPolResp.Policies[0].PatchSoftware)
}

// Regression test for fleet#43659.
func (s *integrationEnterpriseTestSuite) TestBatchSetInstallersScriptByHash() {
	t := s.T()
	ctx := context.Background()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name(), Description: "desc"})
	require.NoError(t, err)

	scriptBytes, err := os.ReadFile(filepath.Join("testdata", "software-installers", "script.sh"))
	require.NoError(t, err)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		TeamID:      &tm.ID,
		Filename:    "script.sh",
		SelfService: true,
	}, http.StatusOK, "")

	var listResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp,
		"team_id", fmt.Sprintf("%d", tm.ID), "available_for_install", "true")

	var titleID uint
	for _, sw := range listResp.SoftwareTitles {
		if sw.SoftwarePackage != nil && sw.SoftwarePackage.Name == "script.sh" {
			titleID = sw.ID
			break
		}
	}
	require.NotZero(t, titleID)

	var storageID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &storageID,
			"SELECT storage_id FROM software_installers WHERE title_id = ? AND global_or_team_id = ?",
			titleID, tm.ID)
	})
	require.NotEmpty(t, storageID)

	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{SHA256: storageID}}},
		http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, batchResponse.RequestUUID)

	var installScript string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installScript, `
			SELECT sc.contents
			FROM software_installers si
			JOIN script_contents sc ON sc.id = si.install_script_content_id
			WHERE si.title_id = ? AND si.global_or_team_id = ?`,
			titleID, tm.ID)
	})
	require.Equal(t, string(scriptBytes), installScript)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersClearsFailedInstallCount() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "exit 1",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}, http.StatusOK, "")

	var installer struct {
		ID        uint   `db:"id"`
		StorageID string `db:"storage_id"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installer, `SELECT id, storage_id FROM software_installers WHERE global_or_team_id = ?`, team.ID)
	})
	require.NotZero(t, installer.ID)

	const hostID = 4242
	_, err = s.attemptCounter.RecordAttempt(ctx, hostID, installer.ID, fleet.PolicyAutomationInstallAttemptExpiry)
	require.NoError(t, err)

	// The batch matches the already uploaded package by hash, so the installer keeps its
	// ID and the count recorded against it is cleared instead of waiting for it to expire.
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
		{SHA256: installer.StorageID, InstallScript: "echo edited by batch"},
	}}, http.StatusAccepted, &batchResponse, "team_name", team.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchResponse.RequestUUID)

	attempts, err := s.attemptCounter.CountAttempts(ctx, hostID, installer.ID)
	require.NoError(t, err)
	require.Zero(t, attempts)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersFleetVariables() {
	t := s.T()
	ctx := context.Background()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// batch (gitops): script-only package with an unsupported variable is
	// rejected, including on dry runs; supported variables pass a dry run
	scriptOnly := []*fleet.SoftwareInstallerPayload{{
		URL:           "script://config.sh",
		InstallScript: "echo $FLEET_VAR_NONEXISTENT",
	}}
	for _, dryRun := range []string{"true", "false"} {
		resp := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: scriptOnly},
			http.StatusUnprocessableEntity, "team_name", tm.Name, "dry_run", dryRun)
		require.Contains(t, extractServerErrorText(resp.Body),
			"Fleet variable $FLEET_VAR_NONEXISTENT is not supported in scripts.", "dry_run=%s", dryRun)
	}
	scriptOnly[0].InstallScript = "echo $FLEET_VAR_HOST_UUID"
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: scriptOnly},
		http.StatusAccepted, "team_name", tm.Name, "dry_run", "true")
}

func (s *integrationEnterpriseTestSuite) TestSoftwareBatchDownloadProgressManyPackages() {
	t := s.T()
	teamName := "software-batch-download-progress"
	const packageCount = 100

	// One installer served under many names, instead of building an archive per package.
	installer, err := os.ReadFile(filepath.Join("testdata", "software-installers", "test.tar.gz"))
	require.NoError(t, err)
	payloads := make([]*fleet.SoftwareInstallerPayload, 0, packageCount)
	names := make([]string, 0, packageCount)

	// Serves an ETag and honors If-None-Match, so a second apply revalidates.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		etag := fmt.Sprintf("%q", name)
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(installer)
	}))
	t.Cleanup(srv.Close)

	for i := 1; i <= packageCount; i++ {
		name := fmt.Sprintf("tarball-package-%03d.tar.gz", i)
		names = append(names, name)
		payloads = append(payloads, &fleet.SoftwareInstallerPayload{
			URL:             srv.URL + "/" + name,
			InstallScript:   "echo installing",
			UninstallScript: "echo uninstalling",
		})
	}

	_, err = s.ds.NewTeam(t.Context(), &fleet.Team{Name: teamName})
	require.NoError(t, err)

	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: payloads},
		http.StatusAccepted, &batchResponse, "fleet_name", teamName)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, batchResponse.RequestUUID)
	require.Len(t, packages, packageCount)

	var batchResult batchSetSoftwareInstallersResultResponse
	s.DoJSON("GET", "/api/latest/fleet/software/batch/"+batchResponse.RequestUUID, nil, http.StatusOK,
		&batchResult, "fleet_name", teamName)

	// Every package reports on its own place in the payload, and none is left mid-download.
	require.Len(t, batchResult.DownloadProgress, packageCount)
	for i, progress := range batchResult.DownloadProgress {
		require.Equal(t, names[i], progress.Name, "package at index %d", i)
		require.Equal(t, fleet.SoftwarePackageDownloadFinished, progress.Status, "package at index %d", i)
	}

	// The same packages again: every conditional request gets a 304, so nothing transfers.
	batchResponse = batchSetSoftwareInstallersResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: payloads},
		http.StatusAccepted, &batchResponse, "fleet_name", teamName)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, batchResponse.RequestUUID)

	batchResult = batchSetSoftwareInstallersResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/batch/"+batchResponse.RequestUUID, nil, http.StatusOK,
		&batchResult, "fleet_name", teamName)

	require.Len(t, batchResult.DownloadProgress, packageCount)
	for i, progress := range batchResult.DownloadProgress {
		require.Equal(t, names[i], progress.Name, "package at index %d", i)
		require.Equal(t, fleet.SoftwarePackageDownloadSkipped, progress.Status, "package at index %d", i)
	}
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersFMARebuildSameVersion() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team_" + t.Name()})
	require.NoError(t, err)

	// Build A: version 1.0.
	states := map[string]*fmaTestState{
		"/zoom/windows.json": {
			version:        "1.0",
			installerBytes: []byte("zoom-build-1.0-a"),
			installerPath:  "/zoom-build-1.0-a.msi",
			installScript:  "install zoom-build-1.0-a.msi",
		},
	}
	startFMAServers(t, s.ds, states)

	var resp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
		http.StatusAccepted, &resp, "team_name", team.Name,
	)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

	var listResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "team_id", fmt.Sprintf("%d", team.ID), "available_for_install", "true")
	require.Len(t, listResp.SoftwareTitles, 1)
	titleID := listResp.SoftwareTitles[0].ID

	// Build A is cached coherently: version, filename, hash, and script all agree.
	metaA, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, titleID, true)
	require.NoError(t, err)
	require.Equal(t, "1.0", metaA.Version)
	require.Equal(t, "zoom-build-1.0-a.msi", metaA.Name)
	zoomState, ok := states["/zoom/windows.json"]
	require.True(t, ok)
	require.Equal(t, zoomState.sha256, metaA.StorageID)
	require.Equal(t, "install zoom-build-1.0-a.msi", metaA.InstallScript)

	// Build B: rebuilt under the SAME version 1.0 (the collision), with new bytes,
	// filename, and install script. Recompute the manifest hash for the new bytes.
	rebuild := zoomState
	rebuild.installerBytes = []byte("zoom-build-1.0-b")
	rebuild.installerPath = "/zoom-build-1.0-b.msi"
	rebuild.installScript = "install zoom-build-1.0-b.msi"
	rebuild.ComputeSHA(rebuild.installerBytes)

	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
		http.StatusAccepted, &resp, "team_name", team.Name,
	)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)

	// The rebuild must advance filename, hash, and script together — no skew.
	metaB, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, titleID, true)
	require.NoError(t, err)
	require.Equal(t, "1.0", metaB.Version)
	require.Equal(t, "zoom-build-1.0-b.msi", metaB.Name)
	require.Equal(t, rebuild.sha256, metaB.StorageID)
	require.Equal(t, "install zoom-build-1.0-b.msi", metaB.InstallScript)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersFMANoCheckHash() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team_" + t.Name()})
	require.NoError(t, err)

	state := &fmaTestState{
		version:            "1.0",
		installerBytes:     []byte("zoom-1.0"),
		installerPath:      "/api/desktop.latestRelease",
		installScript:      "install zoom-1.0.msi",
		noCheckSHA:         true,
		contentDisposition: `attachment; filename="zoom-1.0.msi"`,
	}
	downloads := startFMAServers(t, s.ds, map[string]*fmaTestState{"/zoom/windows.json": state})

	apply := func() {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{Slug: new("zoom/windows")}}, TeamName: team.Name},
			http.StatusAccepted, &resp, "team_name", team.Name,
		)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, resp.RequestUUID)
	}

	apply()
	require.Equal(t, 1, downloads("/api/desktop.latestRelease"))

	var listResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "team_id", fmt.Sprintf("%d", team.ID), "available_for_install", "true")
	require.Len(t, listResp.SoftwareTitles, 1)
	titleID := listResp.SoftwareTitles[0].ID

	// Re-applying the cached version must move no bytes, and must leave the
	// stored digest alone rather than writing the manifest's sentinel.
	apply()
	require.Equal(t, 1, downloads("/api/desktop.latestRelease"), "re-apply must not download")

	meta, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, titleID, true)
	require.NoError(t, err)
	require.Equal(t, "1.0", meta.Version)
	require.Equal(t, state.sha256, meta.StorageID)
	require.Equal(t, "install zoom-1.0.msi", meta.InstallScript)
	// The installer URL has no filename of its own, so a re-apply that derived one
	// from the URL instead of leaving the stored row alone would show up here.
	require.Equal(t, "zoom-1.0.msi", meta.Name)
	require.Equal(t, "msi", meta.Extension)

	// A newly published version is still downloaded.
	state.version = "2.0"
	state.installerBytes = []byte("zoom-2.0")
	state.installerPath = "/zoom-2.0.msi"
	state.installScript = "install zoom-2.0.msi"
	state.ComputeSHA(state.installerBytes)

	apply()
	require.Equal(t, 1, downloads("/zoom-2.0.msi"))

	meta, err = s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, titleID, true)
	require.NoError(t, err)
	require.Equal(t, "2.0", meta.Version)
	require.Equal(t, state.sha256, meta.StorageID)
	require.Equal(t, "install zoom-2.0.msi", meta.InstallScript)
}

func (s *integrationEnterpriseTestSuite) TestBatchSoftwareInstallerFMAManifestFetchedAfterResponse() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(filepath.Join("testdata", "software-installers", "dummy_installer.pkg"))
		if !assert.NoError(t, err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "application/application/x-newton-compatible-pkg")
		_, err = io.Copy(w, file)
		assert.NoError(t, err)
	}))
	t.Cleanup(installerServer.Close)

	// Serve each manifest only after the test releases it, and fail the fetch if no release comes within 5s
	manifestReleases := make(chan struct{}, 1)
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-manifestReleases:
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		manifest := ma.FMAManifestFile{
			Versions: []*ma.FMAManifestApp{{
				Version:           "6.0",
				Queries:           ma.FMAQueries{Exists: "SELECT 1 FROM osquery_info;"},
				InstallerURL:      installerServer.URL + "/fma.pkg",
				SHA256:            "no_check",
				DefaultCategories: []string{"Productivity"},
			}},
		}
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	}))
	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", manifestServer.URL, t)

	maintainedApp, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "1Password",
		Slug:             "1password/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.1password.1password",
	})
	require.NoError(t, err)

	testCases := []struct {
		name     string
		teamName string
	}{
		{
			name:     "batch for a fleet responds before the manifest is served and reports the manifest category",
			teamName: team.Name,
		},
		{
			name:     "batch for unassigned responds before the manifest is served and reports the manifest category",
			teamName: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// send the batch with the manifest still held back, the POST should respond
			var batchResponse batchSetSoftwareInstallersResponse
			s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{
				Software: []*fleet.SoftwareInstallerPayload{{Slug: &maintainedApp.Slug, SelfService: true}},
			}, http.StatusAccepted, &batchResponse, "team_name", tc.teamName)

			// release the manifest, the batch should complete with the manifest category
			manifestReleases <- struct{}{}
			batchResult := waitBatchSetSoftwareInstallers(t, &s.withServer, tc.teamName, batchResponse.RequestUUID)
			require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, batchResult.Status, batchResult.Message)
			require.Len(t, batchResult.Packages, 1)
			require.Equal(t, []string{"🖥️ Productivity"}, batchResult.Categories)
		})
	}
}
