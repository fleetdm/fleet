package service

// Software installer tests for the enterprise (premium) suite.
//
// Belongs here: uploading, downloading, editing, and deleting individual software packages and
// in-house apps (pkg, exe, msi, rpm, script packages, multiple packages per title), their
// validation and advanced options, automatic install policies created at upload, and
// user/self-service install and uninstall requests and their results and retries.
//
// Does not belong here: batch/GitOps software operations (integration_enterprise_software_batch_test.go),
// Fleet-maintained apps (integration_enterprise_software_fma_test.go), installs triggered by
// policy automations (integration_enterprise_policy_automations_test.go) or setup experience
// (integration_enterprise_setup_experience_test.go), and software listing
// (integration_enterprise_software_test.go).

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	strconv "strconv"
	"strings"
	"testing"
	"time"

	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestPackageUploadPreAuthAndTempFileCleanup() {
	t := s.T()
	tmpDir := t.TempDir()
	t.Setenv("TMPDIR", tmpDir)
	requireNoTempFiles := func() {
		files, err := filepath.Glob(filepath.Join(tmpDir, "multipart-*"))
		require.NoError(t, err)
		require.Empty(t, files)
	}

	ctx := t.Context()
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)
	users := []struct {
		name  string
		user  *fleet.User
		token string
	}{
		{name: "global observer", user: &fleet.User{GlobalRole: new(fleet.RoleObserver)}},
		{name: "team1 observer", user: &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleObserver}}}},
		{name: "team1 maintainer", user: &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleMaintainer}}}},
	}
	for i := range users {
		u := users[i].user
		u.Email = fmt.Sprintf("upload_preauth_%d@example.com", i)
		u.Name = u.Email
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(ctx, u)
		require.NoError(t, err)
		users[i].token = s.getTestToken(u.Email, test.GoodPassword)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("software", "a.pkg")
	require.NoError(t, err)
	_, err = fw.Write(bytes.Repeat([]byte{0}, 2<<20))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	contentType := mw.FormDataContentType()

	for _, r := range []struct{ method, path string }{
		{"POST", "/api/latest/fleet/software/package"},
		{"PATCH", "/api/latest/fleet/software/titles/1/package"},
		{"POST", "/api/latest/fleet/bootstrap"},
		{"POST", "/api/latest/fleet/mdm/bootstrap"},
		{"POST", "/api/latest/fleet/mdm/apple/bootstrap"},
	} {
		res := s.DoRawWithHeaders(r.method, r.path, body.Bytes(), http.StatusUnauthorized, map[string]string{"Content-Type": contentType})
		res.Body.Close()
		requireNoTempFiles()

		res = s.DoRawWithHeaders(r.method, r.path, body.Bytes(), http.StatusUnauthorized, map[string]string{
			"Content-Type": contentType, "Authorization": "Bearer invalid",
		})
		res.Body.Close()
		requireNoTempFiles()

		res = s.DoRawWithHeaders(r.method, r.path, body.Bytes(), http.StatusUnsupportedMediaType, map[string]string{
			"Content-Type": contentType, "Content-Encoding": "gzip", "Authorization": "Bearer " + s.token,
		})
		res.Body.Close()
		requireNoTempFiles()

		for _, user := range users[:2] {
			t.Run(r.method+" "+r.path+"/"+user.name, func(t *testing.T) {
				reader, writer := io.Pipe()
				mw := multipart.NewWriter(writer)
				requestCtx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()
				req, err := http.NewRequestWithContext(requestCtx, r.method, s.server.URL+r.path, reader)
				require.NoError(t, err)
				req.Header.Set("Content-Type", mw.FormDataContentType())
				req.Header.Set("Authorization", "Bearer "+user.token)
				releaseBody := make(chan struct{})
				writerDone := make(chan struct{})
				defer func() {
					close(releaseBody)
					_ = reader.Close()
					_ = writer.Close()
					<-writerDone
				}()
				go func() {
					defer close(writerDone)
					field := "software"
					if strings.HasSuffix(r.path, "/bootstrap") {
						field = "package"
					}
					if part, err := mw.CreateFormFile(field, "a.pkg"); err == nil {
						_, _ = part.Write(make([]byte, 1<<20))
					}
					// Keep the closing boundary and EOF back until a response arrives.
					<-releaseBody
				}()
				type result struct {
					status int
					err    error
				}
				response := make(chan result, 1)
				go func() {
					res, err := s.server.Client().Do(req)
					if err != nil {
						response <- result{err: err}
						return
					}
					res.Body.Close()
					response <- result{status: res.StatusCode}
				}()
				select {
				case res := <-response:
					require.NoError(t, res.err)
					require.Equal(t, http.StatusForbidden, res.status)
				case <-time.After(2 * time.Second):
					t.Fatal("upload didn't return 403 while the body was open")
				}
				requireNoTempFiles()
			})
		}
	}

	// An authenticated upload that parses but fails validation must not leave its file behind.
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/software/package", body.Bytes(), http.StatusBadRequest, map[string]string{
		"Content-Type": contentType, "Authorization": "Bearer " + s.token,
	})
	res.Body.Close()
	requireNoTempFiles()

	// A writer on another fleet still reaches the per-fleet check after parsing.
	var otherFleetBody bytes.Buffer
	mw = multipart.NewWriter(&otherFleetBody)
	fw, err = mw.CreateFormFile("software", "a.pkg")
	require.NoError(t, err)
	_, err = fw.Write(make([]byte, 2<<20))
	require.NoError(t, err)
	require.NoError(t, mw.WriteField("fleet_id", strconv.FormatUint(uint64(team2.ID), 10)))
	require.NoError(t, mw.Close())
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/software/package", otherFleetBody.Bytes(), http.StatusForbidden, map[string]string{
		"Content-Type": mw.FormDataContentType(), "Authorization": "Bearer " + users[2].token,
	})
	res.Body.Close()
	requireNoTempFiles()
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerUploadDownloadAndDelete() {
	t := s.T()

	openFile := func(name string) *os.File {
		f, err := os.Open(filepath.Join("testdata", "software-installers", name))
		require.NoError(t, err)
		return f
	}

	var expectBytes []byte
	var expectLen int
	f := openFile("ruby.deb")
	st, err := f.Stat()
	require.NoError(t, err)
	expectLen = int(st.Size())
	require.Equal(t, 11340, expectLen)
	expectBytes = make([]byte, expectLen)
	n, err := f.Read(expectBytes)
	require.NoError(t, err)
	require.Equal(t, n, expectLen)
	f.Close()

	checkDownloadResponse := func(t *testing.T, r *http.Response, expectedFilename string) {
		require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		require.Equal(t, fmt.Sprintf(`attachment;filename="%s"`, expectedFilename), r.Header.Get("Content-Disposition"))
		require.NotZero(t, r.ContentLength)
		require.Equal(t, expectLen, int(r.ContentLength))
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Len(t, b, expectLen)
		require.Equal(t, expectBytes, b)
	}

	t.Run("upload no team software installer", func(t *testing.T) {
		// status is reflected in list hosts responses and counts when filtering by software title and status
		// create a label to test also the counts per label with the software install status filter
		var labelResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name:  t.Name(),
			Query: "select 1"}, http.StatusOK, &labelResp)
		require.NotZero(t, labelResp.Label.ID)
		lblA := labelResp.Label

		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name:  "label_b" + t.Name(),
			Query: "select 1"}, http.StatusOK, &labelResp)
		require.NotZero(t, labelResp.Label.ID)
		lblB := labelResp.Label

		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name:  "label_c" + t.Name(),
			Query: "select 1"}, http.StatusOK, &labelResp)
		require.NotZero(t, labelResp.Label.ID)
		lblC := labelResp.Label

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:     "some install script",
			PreInstallQuery:   "some pre install query",
			PostInstallScript: "some post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:            "ruby",
			Version:          "1:2.5.1",
			Source:           "deb_packages",
			StorageID:        "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:         "linux",
			LabelsIncludeAny: []string{t.Name()},
		}

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
				payload := &fleet.UploadSoftwareInstallerPayload{
					InstallScript:     "some install script",
					PreInstallQuery:   "some pre install query",
					PostInstallScript: "some post install script",
					Filename:          "ruby.deb",
					// additional fields below are pre-populated so we can re-use the payload later for the test assertions
					Title:            "ruby",
					Version:          "1:2.5.1",
					Source:           "deb_packages",
					StorageID:        "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
					Platform:         "linux",
					LabelsIncludeAny: tc.incAny,
					LabelsIncludeAll: tc.incAll,
					LabelsExcludeAny: tc.exclAny,
				}

				s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any" or "labels_exclude_any" can be included.`)
			})
		}

		// upload with unsupported shebang in install script fails validation
		payloadBadShebang := &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "#!/usr/bin/perl\nprint 'hello'",
			Filename:      "ruby.deb",
		}
		s.uploadSoftwareInstaller(t, payloadBadShebang, http.StatusBadRequest, "Interpreter not supported")

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		_, titleID := checkSoftwareInstaller(t, s.ds, payload)

		// check activity
		activityData := fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": false, "software_title_id": %d, "labels_include_any": [{"id": %d, "name": %q}]}`,
			titleID, lblA.ID, lblA.Name)
		s.lastActivityMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

		// upload again fails: identical bytes on the same title are a hash duplicate
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "package is already added (same SHA-256 hash)")

		// update should succeed
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			SelfService:       new(true),
			InstallScript:     new("some install script"),
			PreInstallQuery:   new("some pre install query"),
			PostInstallScript: new("some post install script"),
			Filename:          "ruby.deb",
			TitleID:           titleID,
			TeamID:            nil,
		}, http.StatusOK, "")
		activityData = fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": true, "software_title_id": %d, "labels_include_any": [{"id": %d, "name": %q}], "software_display_name": "",
		"pinned_version": null}`,
			titleID, lblA.ID, lblA.Name)
		s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

		// update with unsupported shebang in install script fails validation
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			InstallScript: new("#!/usr/bin/perl\nprint 'hello'"),
			TitleID:       titleID,
			TeamID:        nil,
		}, http.StatusBadRequest, "Interpreter not supported")

		// patch the software installer to change the labels
		body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":            {"0"},
			"labels_exclude_any": {t.Name()},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)
		expectedPayload := *payload
		expectedPayload.LabelsIncludeAny = nil
		expectedPayload.LabelsExcludeAny = []string{lblA.Name}
		checkSoftwareInstaller(t, s.ds, &expectedPayload)

		// Create a host and assign the label to it
		host := createOrbitEnrolledHost(t, "linux", "label_host", s.ds)
		err = s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{lblA.ID: new(true)}, time.Now(), false)
		require.NoError(t, err)

		// Attempt to install. Should fail because label is "exclude any"
		resp := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(resp.Body), "Couldn't install. Host isn't member of the labels defined for this software title.")

		// patch the software installer again but this time change the pre install query and leave the labels as is
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":           {"0"},
			"pre_install_query": {"some other pre install query"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)
		expectedPayload.PreInstallQuery = "some other pre install query"
		expectedPayload.LabelsIncludeAny = nil                 // no change
		expectedPayload.LabelsExcludeAny = []string{lblA.Name} // no change
		checkSoftwareInstaller(t, s.ds, &expectedPayload)

		// update the label to be "include any". This should allow for the installation to happen.
		var b3 bytes.Buffer
		w3 := multipart.NewWriter(&b3)
		require.NoError(t, w3.WriteField("team_id", "0"))
		require.NoError(t, w3.WriteField("pre_install_query", "some other pre install query"))
		require.NoError(t, w3.WriteField("labels_include_any", lblA.Name))
		w3.Close()
		headers = map[string]string{
			"Content-Type":  w3.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", s.token),
		}
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), b3.Bytes(), http.StatusOK, headers)
		expectedPayload.PreInstallQuery = "some other pre install query"
		expectedPayload.LabelsIncludeAny = []string{lblA.Name}
		expectedPayload.LabelsExcludeAny = nil
		checkSoftwareInstaller(t, s.ds, &expectedPayload)

		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusAccepted)

		// update the installer succeeds
		body, headers = generateMultipartRequest(t, "software",
			"", []byte{}, s.token, map[string][]string{"self_service": {"true"}, "team_id": {"0"}})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

		activityData = fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": true, "labels_include_any": [{"id": %d, "name": %q}], "software_title_id": %d, "software_display_name": "",
		"pinned_version": null}`,
			lblA.ID, lblA.Name, titleID)
		s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

		// orbit-downloading fails with invalid orbit node key
		s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  123,
			OrbitNodeKey: uuid.NewString(),
		}, http.StatusUnauthorized)

		// download the installer
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusBadRequest)

		// delete the installer from nil team fails
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusBadRequest)

		// delete from team 0 succeeds
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", "0")
		activityData = fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": true, "labels_include_any": [{"id": %d, "name": %q}]}`,
			lblA.ID, lblA.Name)
		s.lastActivityMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(), activityData, 0)
	})

	t.Run("upload no team software installer with labels_include_all", func(t *testing.T) {
		// create a label to use for include_all scoping
		var labelResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name:  t.Name(),
			Query: "select 1"}, http.StatusOK, &labelResp)
		require.NotZero(t, labelResp.Label.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "some install script",
			PreInstallQuery:  "some pre install query",
			Filename:         "ruby.deb",
			Title:            "ruby",
			Version:          "1:2.5.1",
			Source:           "deb_packages",
			StorageID:        "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:         "linux",
			LabelsIncludeAll: []string{labelResp.Label.Name},
		}

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer metadata: LabelsIncludeAll should be persisted
		_, titleID := checkSoftwareInstaller(t, s.ds, payload)

		// check that the added-software activity carries labels_include_all
		activityData := fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": false, "software_title_id": %d, "labels_include_all": [{"id": %d, "name": %q}]}`,
			titleID, labelResp.Label.ID, t.Name())
		s.lastActivityMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

		// patch the installer to update an unrelated field; labels_include_all should be preserved
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			SelfService:     new(true),
			InstallScript:   new("some install script"),
			PreInstallQuery: new("some pre install query"),
			Filename:        "ruby.deb",
			TitleID:         titleID,
			TeamID:          nil,
		}, http.StatusOK, "")

		// the edited-software activity should still carry labels_include_all
		activityData = fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": true, "software_title_id": %d, "labels_include_all": [{"id": %d, "name": %q}], "software_display_name": "",
		"pinned_version": null}`,
			titleID, labelResp.Label.ID, t.Name())
		s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

		// create a host and assign the label — install should succeed since host has all required labels
		host := createOrbitEnrolledHost(t, "linux", "include_all_label_host", s.ds)
		err = s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{labelResp.Label.ID: new(true)}, time.Now(), false)
		require.NoError(t, err)
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusAccepted)

		// delete the installer; the deleted-software activity should carry labels_include_all
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", "0")
		activityData = fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": null,
		"team_id": null, "fleet_name": null, "fleet_id": null, "self_service": true, "labels_include_all": [{"id": %d, "name": %q}]}`,
			labelResp.Label.ID, t.Name())
		s.lastActivityMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(), activityData, 0)
	})

	t.Run("create team software installer", func(t *testing.T) {
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
			Name: t.Name(),
		}, http.StatusOK, &createTeamResp)
		require.NotZero(t, createTeamResp.Team.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:            &createTeamResp.Team.ID,
			InstallScript:     "another install script",
			PreInstallQuery:   "another pre install query",
			PostInstallScript: "another post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:       "ruby",
			Version:     "1:2.5.1",
			Source:      "deb_packages",
			StorageID:   "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:    "linux",
			SelfService: true,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		installerID, titleID := checkSoftwareInstaller(t, s.ds, payload)

		// check activity
		activityData := fmt.Sprintf(
			`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": "%s", "team_id": %d, "fleet_name": "%s", "fleet_id": %d, "self_service": true, "software_title_id": %d}`,
			createTeamResp.Team.Name,
			createTeamResp.Team.ID,
			createTeamResp.Team.Name,
			createTeamResp.Team.ID,
			titleID,
		)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

		// upload again fails: identical bytes on the same title are a hash duplicate
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "package is already added (same SHA-256 hash)")

		// download the installer
		r := s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusOK, "team_id", fmt.Sprintf("%d", *payload.TeamID))
		checkDownloadResponse(t, r, payload.Filename)

		// download the installer by getting token first
		tokenResp := getSoftwareInstallerTokenResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", titleID), nil, http.StatusOK,
			&tokenResp, "team_id", fmt.Sprintf("%d", *payload.TeamID))
		require.NotEmpty(t, tokenResp.Token)
		r = s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", titleID, tokenResp.Token), nil,
			http.StatusOK)
		checkDownloadResponse(t, r, payload.Filename)

		// downloading a second time using the same token should fail
		_ = s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", titleID, tokenResp.Token), nil,
			http.StatusForbidden)

		// alt != media should fail
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=bozo", titleID), nil,
			http.StatusUnprocessableEntity,
			&tokenResp, "team_id", fmt.Sprintf("%d", *payload.TeamID))

		// missing team_id should fail
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", titleID), nil,
			http.StatusBadRequest,
			&tokenResp)

		// create an orbit host that is not in the team
		hostNotInTeam := createOrbitEnrolledHost(t, "windows", "orbit-host-no-team", s.ds)
		// downloading installer doesn't work if the host doesn't have a pending install request
		s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostNotInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// create an orbit host, assign to team
		hostInTeam := createOrbitEnrolledHost(t, "linux", "orbit-host-team", s.ds)
		require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&createTeamResp.Team.ID, []uint{hostInTeam.ID})))

		// Create a software installation request
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)

		// requesting download with alt != media fails
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=FOOBAR", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusBadRequest)
		errMsg := extractServerErrorText(r.Body)
		require.Contains(t, errMsg, "only alt=media is supported")

		// valid download
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusOK)
		checkDownloadResponse(t, r, payload.Filename)

		// Get execution ID, normally comes from orbit config
		installUUID := getLatestSoftwareInstallExecID(t, s.ds, hostInTeam.ID)

		// Installation complete, host no longer has access to software
		s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID:                hostInTeam.ID,
				InstallUUID:           installUUID,
				InstallScriptExitCode: new(int(0)),
				InstallScriptOutput:   new("done"),
			},
		}, http.StatusNoContent)

		// while the installer exists, the install results carry its hash
		beforeDelete, err := s.ds.GetSoftwareInstallResults(context.Background(), installUUID)
		require.NoError(t, err)
		require.NotNil(t, beforeDelete.HashSHA256)
		wantHash := *beforeDelete.HashSHA256

		_ = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", fmt.Sprintf("%d", *payload.TeamID))

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(), fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": null, "team_name": "%s", "team_id": %d, "fleet_name": "%s", "fleet_id": %d, "self_service": true}`, createTeamResp.Team.Name, createTeamResp.Team.ID, createTeamResp.Team.Name, createTeamResp.Team.ID), 0)

		// the installed_software activity keeps its hash after the installer is deleted
		var hostActivities listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", hostInTeam.ID), nil, http.StatusOK, &hostActivities)
		require.Len(t, hostActivities.Activities, 1)
		require.Equal(t, fleet.ActivityTypeInstalledSoftware{}.ActivityName(), hostActivities.Activities[0].Type)
		var installedActivity fleet.ActivityTypeInstalledSoftware
		require.NoError(t, json.Unmarshal([]byte(*hostActivities.Activities[0].Details), &installedActivity))
		require.NotNil(t, installedActivity.HashSHA256)
		require.Equal(t, wantHash, *installedActivity.HashSHA256)

		// but the live install results lose it
		afterDelete, err := s.ds.GetSoftwareInstallResults(context.Background(), installUUID)
		require.NoError(t, err)
		require.Nil(t, afterDelete.HashSHA256)

		// download the installer, not found anymore
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusNotFound, "team_id", fmt.Sprintf("%d", *payload.TeamID))
	})

	t.Run("create team 0 software installer", func(t *testing.T) {
		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:            new(uint(0)),
			InstallScript:     "another install script",
			PreInstallQuery:   "another pre install query",
			PostInstallScript: "another post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:       "ruby",
			Version:     "1:2.5.1",
			Source:      "deb_packages",
			StorageID:   "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:    "linux",
			SelfService: true,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		installerID, titleID := checkSoftwareInstaller(t, s.ds, payload)

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(),
			fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": null, "team_id": 0, "fleet_name": null, "fleet_id": 0, "self_service": true, "software_title_id": %d}`, titleID), 0)

		// upload again fails: identical bytes on the same title are a hash duplicate
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "package is already added (same SHA-256 hash)")

		// download the installer
		r := s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusOK, "team_id", fmt.Sprintf("%d", 0))
		checkDownloadResponse(t, r, payload.Filename)

		// create an orbit host that is not in the team
		hostNotInTeam := createOrbitEnrolledHost(t, "windows", "orbit-host-no-team", s.ds)
		// downloading installer fails because there's no install request
		s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostNotInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// create an orbit host, assign to team
		hostInTeam := createOrbitEnrolledHost(t, "linux", "orbit-host-team", s.ds)

		// requesting download with alt != media fails
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=FOOBAR", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusBadRequest)
		errMsg := extractServerErrorText(r.Body)
		require.Contains(t, errMsg, "only alt=media is supported")

		// Create a software installation request
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)

		// valid download
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusOK)
		checkDownloadResponse(t, r, payload.Filename)

		// Get execution ID, normally comes from orbit config
		installUUID := getLatestSoftwareInstallExecID(t, s.ds, hostInTeam.ID)

		// Installation complete, host no longer has access to software
		s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID:                hostInTeam.ID,
				InstallUUID:           installUUID,
				InstallScriptExitCode: new(int(0)),
				InstallScriptOutput:   new("done"),
			},
		}, http.StatusNoContent)

		_ = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", fleet.OrbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		_, err := s.ds.CreateOrUpdateSoftwareTitleIcon(context.Background(), &fleet.UploadSoftwareTitleIconPayload{
			TitleID:   titleID,
			Filename:  "icon.png",
			TeamID:    0,
			StorageID: "icon_storage_id",
		})
		require.NoError(t, err)

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", "0")

		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &count, `SELECT COUNT(1) FROM software_title_icons WHERE team_id = ? and software_title_id = ?`, 0, titleID)
		})
		require.Equal(t, 0, count)

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(),
			fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "software_icon_url": "/api/latest/fleet/software/titles/%d/icon?fleet_id=0", "team_name": null, "team_id": null, "fleet_name": null, "fleet_id": null, "self_service": true}`, titleID), 0)

		// download the installer, not found anymore
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusNotFound, "team_id", fmt.Sprintf("%d", 0))
	})

	t.Run("uninstall migration for software installer", func(t *testing.T) {
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
			Name: t.Name(),
		}, http.StatusOK, &createTeamResp)
		require.NotZero(t, createTeamResp.Team.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:          &createTeamResp.Team.ID,
			InstallScript:   "another install script",
			UninstallScript: "exit 1",
			Filename:        "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:     "ruby",
			Version:   "1:2.5.1",
			Source:    "deb_packages",
			StorageID: "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:  "linux",
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		// Run the migration when nothing is to be done
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// check the software installer
		installerID, titleID := checkSoftwareInstaller(t, s.ds, payload)

		var origPackageIDs string
		var origExtension string
		// Update DB by clearing package id and tweaking extension
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if err := sqlx.GetContext(context.Background(), q, &origPackageIDs, `SELECT package_ids FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			require.NotEmpty(t, origPackageIDs)

			if err := sqlx.GetContext(context.Background(), q, &origExtension, `SELECT extension FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			require.NotEmpty(t, origExtension)

			if _, err = q.ExecContext(context.Background(), `UPDATE software_installers SET package_ids = '', extension = 'rb' WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			return nil
		})

		// Check title to make it works without package id
		respTitle := getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &respTitle, "team_id",
			fmt.Sprintf("%d", createTeamResp.Team.ID))
		require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
		assert.Equal(t, "another install script", respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
		assert.Equal(t, "exit 1", respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)

		// Run the migration
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// Check package ID and extension
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var packageIDs string
			if err := sqlx.GetContext(context.Background(), q, &packageIDs, `SELECT package_ids FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			assert.Equal(t, origPackageIDs, packageIDs)

			var extension string
			if err := sqlx.GetContext(context.Background(), q, &extension, `SELECT extension FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			assert.Equal(t, origExtension, extension)

			return nil
		})

		// Check uninstall script
		uninstallScript := file.GetUninstallScript("deb")
		uninstallScript = strings.ReplaceAll(uninstallScript, "$PACKAGE_ID", "'ruby'")
		respTitle = getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &respTitle, "team_id",
			fmt.Sprintf("%d", createTeamResp.Team.ID))
		require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
		assert.Equal(t, "another install script", respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
		assert.Equal(t, uninstallScript, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)

		// Running the migration again causes no issues.
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// Update DB by clearing package ids and swapping extension to one we skip
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err = q.ExecContext(context.Background(), `UPDATE software_installers SET package_ids = '', extension = 'tar.gz' WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			return nil
		})

		// Running the migration again causes no issues.
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// Package ID and extension should not have been modified
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var packageIDs string
			if err := sqlx.GetContext(context.Background(), q, &packageIDs, `SELECT package_ids FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			assert.Empty(t, packageIDs)

			var extension string
			if err := sqlx.GetContext(context.Background(), q, &extension, `SELECT extension FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			assert.Equal(t, "tar.gz", extension)

			return nil
		})

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
			"team_id", fmt.Sprintf("%d", *payload.TeamID))
	})

	t.Run("patch software installer categories", func(t *testing.T) {
		var labelResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name:  t.Name(),
			Query: "select 1"}, http.StatusOK, &labelResp)
		require.NotZero(t, labelResp.Label.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:     "some install script",
			PreInstallQuery:   "some pre install query",
			PostInstallScript: "some post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:            "ruby",
			Version:          "1:2.5.1",
			Source:           "deb_packages",
			StorageID:        "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:         "linux",
			LabelsIncludeAny: []string{t.Name()},
		}

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		_, titleID := checkSoftwareInstaller(t, s.ds, payload)

		// patch the software installer to change the categories and self-service
		body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":      {"0"},
			"categories":   {"Browsers", "Productivity"},
			"self_service": {"true"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

		expectedPayload := *payload
		expectedPayload.Categories = []string{"🌎 Browsers", "🖥️ Productivity"}
		expectedPayload.SelfService = true
		checkSoftwareInstaller(t, s.ds, payload)

		meta, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(context.Background(), nil, titleID, false)
		require.NoError(t, err)
		require.Equal(t, expectedPayload.Filename, meta.Name)
		require.Equal(t, expectedPayload.SelfService, meta.SelfService)
		require.ElementsMatch(t, expectedPayload.Categories, meta.Categories)

		// patch the software installer to change the categories and nothing else
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":    {"0"},
			"categories": {"Browsers"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

		expectedPayload.Categories = []string{"🌎 Browsers"}
		checkSoftwareInstaller(t, s.ds, payload)

		meta, err = s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(context.Background(), nil, titleID, false)
		require.NoError(t, err)
		require.Equal(t, expectedPayload.Filename, meta.Name)
		require.Equal(t, expectedPayload.SelfService, meta.SelfService)
		require.ElementsMatch(t, expectedPayload.Categories, meta.Categories)

		// patch the software installer to change the self-service and nothing else
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":      {"0"},
			"self_service": {"false"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

		expectedPayload.SelfService = false
		checkSoftwareInstaller(t, s.ds, payload)

		meta, err = s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(context.Background(), nil, titleID, false)
		require.NoError(t, err)
		require.Equal(t, expectedPayload.Filename, meta.Name)
		require.Equal(t, expectedPayload.SelfService, meta.SelfService)
		require.ElementsMatch(t, expectedPayload.Categories, meta.Categories)
	})
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerNewInstallRequestPlatformValidation() {
	t := s.T()

	hostsByPlatform := map[string]*fleet.Host{
		"linux": nil, "darwin": nil, "windows": nil,
	}

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	for platform := range hostsByPlatform {
		h, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + uuid.New().String()),
			NodeKey:         new(t.Name() + uuid.New().String()),
			Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
			Platform:        platform,
		})
		require.NoError(t, err)
		setOrbitEnrollment(t, h, s.ds)

		err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tm.ID, []uint{h.ID}))
		require.NoError(t, err)

		hostsByPlatform[platform] = h
	}

	softwareTitles := map[string]uint{
		"deb": 0, "msi": 0, "exe": 0, "pkg": 0,
	}

	for kind := range softwareTitles {
		// TODO(roberto): we need real binaries for exe, msi and pkg to
		// perform the API calls.
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			installScript := fmt.Sprintf(`echo '%s'`, kind)
			res, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`, installScript, installScript)
			if err != nil {
				return err
			}
			scriptContentID, _ := res.LastInsertId()

			uninstallScript := fmt.Sprintf(`echo uninstall '%s'`, kind)
			resUninstall, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`,
				uninstallScript, uninstallScript)
			if err != nil {
				return err
			}
			uninstallScriptContentID, _ := resUninstall.LastInsertId()

			res, err = q.ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES ('foo', ?)`, kind)
			if err != nil {
				return err
			}
			titleID, _ := res.LastInsertId()
			softwareTitles[kind] = uint(titleID) //nolint:gosec // dismiss G115

			platform := ""
			switch kind {
			case "deb":
				platform = "linux"
			case "msi", "exe":
				platform = "windows"
			case "pkg":
				platform = "darwin"
			}

			_, err = q.ExecContext(ctx, `
			INSERT INTO software_installers
				(title_id, filename, extension, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, team_id, global_or_team_id, pre_install_query, package_ids, is_active, patch_query)
			VALUES
				(?, ?, ?, ?, ?, ?, ?, unhex(?), ?, ?, ?, ?, TRUE, '')`,
				titleID, fmt.Sprintf("installer.%s", kind), kind, "v1.0.0", platform, scriptContentID, uninstallScriptContentID,
				hex.EncodeToString([]byte("test")), tm.ID, tm.ID, "foo", "")
			return err
		})
	}

	testCases := []struct {
		platform            string
		supportedInstallers []string
	}{
		{"windows", []string{"exe", "msi"}},
		{"darwin", []string{"pkg"}},
		{"linux", []string{"deb"}},
	}

	for _, tc := range testCases {
		for platform, host := range hostsByPlatform {
			for _, kind := range tc.supportedInstallers {
				wantStatus := http.StatusAccepted
				if tc.platform != platform {
					wantStatus = http.StatusBadRequest
				}

				var resp installSoftwareResponse
				s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, softwareTitles[kind]), nil,
					wantStatus, &resp)
			}
		}
	}
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerHostRequests() {
	t := s.T()

	// Enabling software inventory globally, which will be inherited by the team
	appConf, err := s.ds.AppConfig(context.Background())
	s.Require().NoError(err)
	appConf.Features.EnableSoftwareInventory = true
	appConf.ServerSettings.ScriptsDisabled = true // shouldn't stop installs/uninstalls
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	s.Require().NoError(err)
	time.Sleep(2 * time.Second) // Wait for the app config cache to clear

	t.Cleanup(func() {
		acr := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"scripts_disabled": false
		}
	}`), http.StatusOK, &acr)
	})

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name: t.Name(),
	}, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	teamID := &createTeamResp.Team.ID

	var resp installSoftwareResponse
	// non-existent host
	s.DoJSON("POST", "/api/latest/fleet/hosts/1/software/1/install", nil, http.StatusNotFound, &resp)
	s.DoJSON("POST", "/api/latest/fleet/hosts/1/software/1/uninstall", nil, http.StatusNotFound, &resp)

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
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(teamID, []uint{h.ID}))
	require.NoError(t, err)
	h.TeamID = teamID

	// request fails
	resp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/install", h.ID), nil, http.StatusUnprocessableEntity, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/uninstall", h.ID), nil, http.StatusUnprocessableEntity, &resp)

	// host installs fleetd
	orbitKey := setOrbitEnrollment(t, h, s.ds)
	h.OrbitNodeKey = &orbitKey

	// request fails because of non-existent title
	resp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/install", h.ID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/uninstall", h.ID), nil, http.StatusBadRequest, &resp)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "another install script",
		PreInstallQuery:   "another pre install query",
		PostInstallScript: "another post install script",
		UninstallScript:   "another uninstall script with $PACKAGE_ID",
		Filename:          "ruby.deb",
		Title:             "ruby",
		TeamID:            teamID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "deb_packages")

	// Get title with software installer
	respTitle := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &respTitle, "team_id",
		fmt.Sprintf("%d", *teamID))
	require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
	assert.Equal(t, "another install script", respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
	assert.Equal(t, `another uninstall script with 'ruby'`, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)

	// Upload another package for another platform
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		Filename: "dummy_installer.pkg",
		Title:    "DummyApp",
		TeamID:   teamID,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	pkgTitleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", pkgTitleID), nil, http.StatusOK, &respTitle, "team_id",
		fmt.Sprintf("%d", *teamID))
	require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
	assert.NotEmpty(t, respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
	assert.NotEmpty(t, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)
	assert.NotContains(t, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript, "$PACKAGE_ID")
	assert.Contains(t, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript, "com.example.dummy")

	// install/uninstall request fails for the wrong platform
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, pkgTitleID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, pkgTitleID), nil, http.StatusBadRequest, &resp)

	// delete software installer which we will not use
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", pkgTitleID), nil, http.StatusNoContent,
		"team_id", fmt.Sprintf("%d", *teamID))

	// install software request succeeds
	resp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titleID), nil, http.StatusAccepted, &resp)

	// Get the results, should be pending
	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	require.NotNil(t, getHostSoftwareResp.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSoftwareResp.Software[0].Status)
	assert.Nil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	installUUID := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID

	gsirr := getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &gsirr)
	require.NoError(t, gsirr.Err)
	require.NotNil(t, gsirr.Results)
	results := gsirr.Results
	require.Equal(t, installUUID, results.InstallUUID)
	require.Equal(t, fleet.SoftwareInstallPending, results.Status)

	// Can't install/uninstall if software install is pending
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titleID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusBadRequest, &resp)

	// create 3 more hosts, will have statuses installed, failed and one with two
	// install requests - one failed and the latest install pending
	h2 := createOrbitEnrolledHost(t, "linuxmint", "host2", s.ds)
	h3 := createOrbitEnrolledHost(t, "debian", "host3", s.ds)
	h4 := createOrbitEnrolledHost(t, "tuxedo", "host4", s.ds)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(teamID, []uint{h2.ID, h3.ID, h4.ID}))
	require.NoError(t, err)

	// cancel pending refetches so we can see when refetches are queued
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), h2.ID, false))
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), h3.ID, false))
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), h4.ID, false))

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h2.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID2 := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *h2.OrbitNodeKey, installUUID2)), http.StatusNoContent)

	software := []fleet.Software{
		{Name: payload.Title, Version: payload.Version, Source: "deb_packages"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), h2.ID, software)
	require.NoError(t, err)

	// Note: no need to use the "include_available_for_install" query param
	// since we simulated ingesting the software above
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)

	// Verify refetch requested is set after successful install
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h2.ID), nil, http.StatusOK, &hostResp)
	require.True(t, hostResp.Host.RefetchRequested, "RefetchRequested should be true after successful software install")

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h3.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h3.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID3 := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *h3.OrbitNodeKey, installUUID3)), http.StatusNoContent)

	// Verify refetch requested is NOT set after failed install
	var hostRespFailed getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h3.ID), nil, http.StatusOK, &hostRespFailed)
	require.False(t, hostRespFailed.Host.RefetchRequested, "RefetchRequested should be false after failed software install")

	// Exhaust automatic retries for h3 so it reaches terminal "failed" state.
	// Server-side retries queue up to MaxSoftwareInstallAttempts attempts.
	for attempt := 2; attempt <= fleet.MaxSoftwareInstallAttempts; attempt++ {
		getHostSoftwareResp = getHostSoftwareResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h3.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
		require.Len(t, getHostSoftwareResp.Software, 1)
		require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage)
		require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
		retryUUID := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
		require.NotEqual(t, installUUID3, retryUUID, "retry should have a new install UUID (attempt %d)", attempt)
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
				"orbit_node_key": %q,
				"install_uuid": %q,
				"pre_install_condition_output": "ok",
				"install_script_exit_code": 1,
				"install_script_output": "retry %d failed"
			}`, *h3.OrbitNodeKey, retryUUID, attempt)), http.StatusNoContent)
		installUUID3 = retryUUID
	}

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h4.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h4.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID4a := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": ""
		}`, *h4.OrbitNodeKey, installUUID4a)), http.StatusNoContent)

	// Verify refetch requested is NOT set after failed pre-install condition
	var hostRespPreInstallFailed getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h4.ID), nil, http.StatusOK, &hostRespPreInstallFailed)
	require.False(t, hostRespPreInstallFailed.Host.RefetchRequested, "RefetchRequested should be false after failed pre-install condition")

	// Exhaust automatic retries for h4 so it reaches terminal "failed" state.
	for attempt := 2; attempt <= fleet.MaxSoftwareInstallAttempts; attempt++ {
		getHostSoftwareResp = getHostSoftwareResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h4.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
		require.Len(t, getHostSoftwareResp.Software, 1)
		require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage)
		require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
		retryUUID := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
		require.NotEqual(t, installUUID4a, retryUUID, "retry should have a new install UUID (attempt %d)", attempt)
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
				"orbit_node_key": %q,
				"install_uuid": %q,
				"pre_install_condition_output": ""
			}`, *h4.OrbitNodeKey, retryUUID)), http.StatusNoContent)
		installUUID4a = retryUUID
	}

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h4.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h4.ID), nil, http.StatusOK, &getHostSoftwareResp, "include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID4b := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	_ = installUUID4b

	// status is reflected in software title response
	titleResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id",
		fmt.Sprint(*teamID))
	// TODO: confirm expected behavior of the title response host counts (unspecified)
	require.Zero(t, titleResp.SoftwareTitle.HostsCount)
	require.Nil(t, titleResp.SoftwareTitle.CountsUpdatedAt)
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
	require.Equal(t, "ruby.deb", titleResp.SoftwareTitle.SoftwarePackage.Name)
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage.Status)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{
		Installed:      1,
		PendingInstall: 2,
		FailedInstall:  1,
	}, *titleResp.SoftwareTitle.SoftwarePackage.Status)

	// status is reflected in list hosts responses and counts when filtering by software title and status
	// create a label to test also the counts per label with the software install status filter
	var labelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name:  "test",
		Hosts: []string{h.Hostname, h2.Hostname, h3.Hostname, h4.Hostname}}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)

	cases := []struct {
		status  string
		count   int
		hostIDs []uint
	}{
		{"pending", 2, []uint{h.ID, h4.ID}},
		{"failed", 1, []uint{h3.ID}},
		{"installed", 1, []uint{h2.ID}},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			var listResp listHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", c.status, "team_id",
				fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID))
			require.Len(t, listResp.Hosts, c.count)
			gotIDs := make([]uint, 0, c.count)
			for _, h := range listResp.Hosts {
				gotIDs = append(gotIDs, h.ID)
			}
			require.ElementsMatch(t, c.hostIDs, gotIDs)

			var countResp countHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", c.status, "team_id",
				fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID))
			require.Equal(t, c.count, countResp.Count)

			// count with label filter
			countResp = countHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", c.status, "team_id",
				fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID), "label_id", fmt.Sprint(labelResp.Label.ID))
			require.Equal(t, c.count, countResp.Count)

			listResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelResp.Label.ID), nil, http.StatusOK, &listResp,
				"software_status", c.status, "team_id", fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID))
			require.Len(t, listResp.Hosts, c.count)
			gotIDs = make([]uint, 0, c.count)
			for _, h := range listResp.Hosts {
				gotIDs = append(gotIDs, h.ID)
			}
			require.ElementsMatch(t, c.hostIDs, gotIDs)
		})
	}

	// filter validations
	r := s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "uninstalled")
	require.Contains(t, extractServerErrorText(r.Body), "Invalid software_status")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed")
	require.Contains(t, extractServerErrorText(r.Body), "Missing software_title_id")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "software_title_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Missing team_id")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "team_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Missing software_title_id")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "team_id", "1", "software_title_id", "1", "software_version_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Invalid parameters. The combination of software_version_id and software_title_id is not allowed.")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "team_id", "1", "software_title_id", "1", "software_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Invalid parameters. The combination of software_id and software_title_id is not allowed.")

	// Test: software_title_id DNE and software_status is valid (should return 0 hosts, 0 count)
	nonExistentTitleID := uint(999999) // unlikely to exist

	validStatuses := []string{"installed", "pending", "failed"}
	for _, status := range validStatuses {
		t.Run("nonexistent_title_id_"+status, func(t *testing.T) {
			// List hosts
			var listResp listHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp,
				"software_status", status,
				"team_id", fmt.Sprint(*teamID),
				"software_title_id", fmt.Sprint(nonExistentTitleID),
			)
			require.Empty(t, listResp.Hosts)

			// Count hosts
			var countResp countHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp,
				"software_status", status,
				"team_id", fmt.Sprint(*teamID),
				"software_title_id", fmt.Sprint(nonExistentTitleID),
			)
			require.Equal(t, 0, countResp.Count)
		})
	}

	// Return installed app with software detail query
	distributedReq := submitDistributedQueryResultsRequestShim{
		NodeKey: *h2.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_linux": json.RawMessage(fmt.Sprintf(
				`[{"name": "%s", "version": "1.0", "type": "Package (deb)",
					"source": "deb_packages", "last_opened_at": "",
					"installed_path": "/bin/ruby"}]`, payload.Title)),
		},
		Statuses: map[string]any{
			hostDistributedQueryPrefix + "software_linux": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp,
		"include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].Status)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.NotEmpty(t, getHostSoftwareResp.Software[0].InstalledVersions, "Installed versions should exist")

	// Remove the installed app of h2 by not returning it
	distributedReq = submitDistributedQueryResultsRequestShim{
		NodeKey: *h2.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_linux": json.RawMessage(`[]`),
		},
		Statuses: map[string]any{
			hostDistributedQueryPrefix + "software_linux": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp,
		"include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.Nil(t, getHostSoftwareResp.Software[0].Status)
	assert.Nil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Empty(t, getHostSoftwareResp.Software[0].InstalledVersions, "Installed versions should now not exist")

	// Mark original h install successful
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *h.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// simulate a lock/unlock on h; this creates the host_mdm_actions table, which reproduces #25144
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", h.ID), nil, http.StatusOK)
	status, err := s.ds.GetHostLockWipeStatus(context.Background(), h)
	require.NoError(t, err)
	var orbitScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *h.OrbitNodeKey, status.LockScript.ExecutionID)),
		http.StatusOK, &orbitScriptResp)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", h.ID), nil, http.StatusOK)
	status, err = s.ds.GetHostLockWipeStatus(context.Background(), h)
	require.NoError(t, err)
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *h.OrbitNodeKey, status.UnlockScript.ExecutionID)),
		http.StatusOK, &orbitScriptResp)

	token := "secret_token"
	createDeviceTokenForHost(t, s.ds, h.ID, token)

	// Remove pending refresh so we can see if uninstall sets it
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), h.ID, false))

	// Do uninstall on h
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp,
		"include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Equal(t, fleet.SoftwareUninstallPending, *getHostSoftwareResp.Software[0].Status)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	uninstallExecutionID := getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall.ExecutionID

	// Uninstall should show up as a pending activity
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", h.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), listUpcomingAct.Activities[0].Type)
	details := make(map[string]any, 5)
	require.NoError(t, json.Unmarshal(*listUpcomingAct.Activities[0].Details, &details))
	assert.EqualValues(t, fleet.SoftwareUninstallPending, details["status"])

	// should be visible from My device
	s.DoRawNoAuth("GET", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%s/results", token, uninstallExecutionID), nil, http.StatusOK)

	// Check that status is reflected in software title response
	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id",
		fmt.Sprint(*teamID))
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
	assert.Equal(t, "ruby.deb", titleResp.SoftwareTitle.SoftwarePackage.Name)
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage.Status)
	assert.Equal(t, fleet.SoftwareInstallerStatusSummary{
		PendingInstall:   1,
		FailedInstall:    1,
		PendingUninstall: 1,
	}, *titleResp.SoftwareTitle.SoftwarePackage.Status)

	// Another install/uninstall cannot be sent once an uninstall is pending
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titleID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusBadRequest, &resp)

	// expect uninstall script (uninstall software works via a script) to be pending
	var orbitResp fleet.OrbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Len(t, orbitResp.Notifications.PendingScriptExecutionIDs, 1)
	require.Equal(t, uninstallExecutionID, orbitResp.Notifications.PendingScriptExecutionIDs[0])

	// Refetch should not be requested yet as the uninstall is still pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.False(t, hostResp.Host.RefetchRequested, "RefetchRequested should not be true yet")

	// Host sends successful uninstall result
	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *h.OrbitNodeKey,
			uninstallExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// Verify refetch requested is set after successful uninstall
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.True(t, hostResp.Host.RefetchRequested, "RefetchRequested should be true after successful software uninstall")

	// Check activity feed
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", h.ID), nil, http.StatusOK, &activitiesResp, "order_key", "id",
		"order_direction", "desc")
	require.NotEmpty(t, activitiesResp.Activities)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), activitiesResp.Activities[0].Type)
	details = make(map[string]any, 5)
	require.NoError(t, json.Unmarshal(*activitiesResp.Activities[0].Details, &details))
	assert.Equal(t, "uninstalled", details["status"])

	// should be visible from My device
	s.DoRawNoAuth("GET", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%s/results", token, uninstallExecutionID), nil, http.StatusOK)

	// Software should be available for install again
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp,
		"include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	assert.Nil(t, getHostSoftwareResp.Software[0].Status)

	// Remove pending refresh so we can see if failed uninstall sets it
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), h.ID, false))

	// Uninstall again, but this time with a failed result
	beforeUninstall := time.Now()
	// Since host_script_results does not use fine-grained timestamps yet, we adjust
	beforeUninstall = beforeUninstall.Add(-time.Second)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp,
		"include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Equal(t, fleet.SoftwareUninstallPending, *getHostSoftwareResp.Software[0].Status)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	uninstallExecutionID = getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall.ExecutionID
	// Host sends failed uninstall result
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 1, "output": "not ok"}`, *h.OrbitNodeKey,
			uninstallExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// Verify refetch requested is NOT set after failed uninstall
	var hostRespFailedUninstall getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostRespFailedUninstall)
	require.False(t, hostRespFailedUninstall.Host.RefetchRequested, "RefetchRequested should be false after failed software uninstall")

	// Check activity feed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", h.ID), nil, http.StatusOK, &activitiesResp, "order_key", "id",
		"order_direction", "desc")
	require.NotEmpty(t, activitiesResp.Activities)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), activitiesResp.Activities[0].Type)
	details = make(map[string]any, 5)
	require.NoError(t, json.Unmarshal(*activitiesResp.Activities[0].Details, &details))
	assert.Equal(t, "failed", details["status"])

	// Access software install/uninstall result after host is deleted
	err = s.ds.DeleteHost(context.Background(), h.ID)
	require.NoError(t, err)

	instResult, err := s.ds.GetSoftwareInstallResults(context.Background(), installUUID)
	require.NoError(t, err)
	require.NotNil(t, instResult.HostDeletedAt)

	gsirr = getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &gsirr)
	require.NoError(t, gsirr.Err)
	require.NotNil(t, gsirr.Results)
	results = gsirr.Results
	require.Equal(t, installUUID, results.InstallUUID)
	require.Equal(t, fleet.SoftwareInstalled, results.Status)

	var scriptResultResp fleet.GetScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+uninstallExecutionID, nil, http.StatusOK, &scriptResultResp)
	assert.Equal(t, h.ID, scriptResultResp.HostID)
	assert.NotEmpty(t, scriptResultResp.ScriptContents)
	require.NotNil(t, scriptResultResp.ExitCode)
	assert.EqualValues(t, 1, *scriptResultResp.ExitCode)
	assert.Equal(t, "not ok", scriptResultResp.Output)
	assert.Less(t, beforeUninstall, scriptResultResp.CreatedAt)

	appConf.ServerSettings.ScriptsDisabled = false // set back to normal
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	s.Require().NoError(err)
}

func (s *integrationEnterpriseTestSuite) TestSelfServiceSoftwareInstallUninstall() {
	t := s.T()

	host1 := createOrbitEnrolledHost(t, "ubuntu", "", s.ds)
	token := "secret_token"
	createDeviceTokenForHost(t, s.ds, host1.ID, token)

	host2 := createOrbitEnrolledHost(t, "ubuntu", "2", s.ds)
	token2 := "secret_token2"
	createDeviceTokenForHost(t, s.ds, host2.ID, token2)

	// Create a label and assign it to the host
	var labelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name:  t.Name(),
		Query: "select 1"}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)

	err := s.ds.RecordLabelQueryExecutions(context.Background(), host1, map[uint]*bool{labelResp.Label.ID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	payloadNoSS := &fleet.UploadSoftwareInstallerPayload{
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "install",
		PostInstallScript: "echo hi",
		Filename:          "ruby.deb",
		Title:             "ruby",
		SelfService:       false,
	}
	s.uploadSoftwareInstaller(t, payloadNoSS, http.StatusOK, "")
	titleIDNoSS := getSoftwareTitleID(t, s.ds, payloadNoSS.Title, "deb_packages")

	payloadSS := &fleet.UploadSoftwareInstallerPayload{
		PreInstallQuery:   "SELECT 2",
		InstallScript:     "install again",
		PostInstallScript: "echo bye",
		Filename:          "emacs.deb",
		Title:             "emacs",
		SelfService:       true,
		LabelsIncludeAny:  []string{labelResp.Label.Name},
	}
	s.uploadSoftwareInstaller(t, payloadSS, http.StatusOK, "")
	titleIDSS := getSoftwareTitleID(t, s.ds, payloadSS.Title, "deb_packages")

	// cannot self-install if software installer does not allow it
	res := s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token, titleIDNoSS), nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Software title is not available through self-service")

	// Add an installer with an exclude any label. Installation attempt should fail.
	payloadLabelSS := &fleet.UploadSoftwareInstallerPayload{
		PreInstallQuery:   "SELECT 42",
		InstallScript:     "install again",
		PostInstallScript: "echo bye",
		Filename:          "vim.deb",
		Title:             "vim",
		SelfService:       true,
		LabelsExcludeAny:  []string{labelResp.Label.Name},
	}
	s.uploadSoftwareInstaller(t, payloadLabelSS, http.StatusOK, "")
	titleIDLabelSS := getSoftwareTitleID(t, s.ds, payloadLabelSS.Title, "deb_packages")

	resp := s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token, titleIDLabelSS), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(resp.Body), "Couldn't install. Host isn't member of the labels defined for this software title.")

	// request self-install of software that allows it (is self-service + label scoped)
	s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token, titleIDSS), nil, http.StatusAccepted)

	// it shows up as "self-installed" in the upcoming activities of the host
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	require.Nil(t, listUpcomingAct.Activities[0].ActorID)
	require.False(t, listUpcomingAct.Activities[0].FleetInitiated)

	var details fleet.ActivityTypeInstalledSoftware
	err = json.Unmarshal([]byte(*listUpcomingAct.Activities[0].Details), &details)
	require.NoError(t, err)
	require.Equal(t, host1.ID, details.HostID)
	require.Equal(t, details.SoftwareTitle, payloadSS.Title)
	require.True(t, details.SelfService)
	require.EqualValues(t, fleet.SoftwareInstallPending, details.Status)
	installID := details.InstallUUID

	// record the installation results
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *host1.OrbitNodeKey, installID)),
		http.StatusNoContent)

	// nothing in upcoming activities anymore
	listUpcomingAct = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Empty(t, listUpcomingAct.Activities)

	// installation shows up in past activities
	var listPastAct listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host1.ID), nil, http.StatusOK, &listPastAct)
	require.Len(t, listPastAct.Activities, 1)
	require.Nil(t, listPastAct.Activities[0].ActorID)
	require.False(t, listPastAct.Activities[0].FleetInitiated)

	err = json.Unmarshal([]byte(*listPastAct.Activities[0].Details), &details)
	require.NoError(t, err)
	require.Equal(t, host1.ID, details.HostID)
	require.Equal(t, details.SoftwareTitle, payloadSS.Title)
	require.True(t, details.SelfService)
	require.EqualValues(t, fleet.SoftwareInstalled, details.Status)

	// Do uninstall on host
	s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%d", token, titleIDSS), nil, http.StatusAccepted)
	var getHostSoftwareResp getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSoftwareResp,
		"include_available_for_install", "true")
	require.Len(t, getHostSoftwareResp.Software, 2)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Equal(t, fleet.SoftwareUninstallPending, *getHostSoftwareResp.Software[0].Status)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	uninstallExecutionID := getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall.ExecutionID

	// Uninstall should show up as a pending activity
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), listUpcomingAct.Activities[0].Type)
	uninstallDetails := make(map[string]any, 5)
	require.NoError(t, json.Unmarshal(*listUpcomingAct.Activities[0].Details, &uninstallDetails))
	assert.EqualValues(t, fleet.SoftwareUninstallPending, uninstallDetails["status"])
	assert.EqualValues(t, true, uninstallDetails["self_service"])
	assert.Nil(t, listUpcomingAct.Activities[0].ActorID)
	assert.False(t, listUpcomingAct.Activities[0].FleetInitiated)

	// Check uninstall results via device endpoint before execution
	res = s.DoRawNoAuth("GET", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%s/results", token, uninstallExecutionID), nil, http.StatusOK)
	uninstallResult := fleet.GetScriptResultResponse{}
	err = json.NewDecoder(res.Body).Decode(&uninstallResult)
	require.NoError(t, err)
	require.Equal(t, host1.DisplayName(), uninstallResult.HostName)
	require.Equal(t, uninstallExecutionID, uninstallResult.ExecutionID)
	require.Nil(t, uninstallResult.ExitCode)
	require.Empty(t, uninstallResult.Output)

	// Host sends successful uninstall result
	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host1.OrbitNodeKey,
			uninstallExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// Check activity feed
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host1.ID), nil, http.StatusOK, &activitiesResp, "order_key", "id",
		"order_direction", "desc")
	require.NotEmpty(t, activitiesResp.Activities)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), activitiesResp.Activities[0].Type)
	uninstallDetails = make(map[string]any, 5)
	require.NoError(t, json.Unmarshal(*activitiesResp.Activities[0].Details, &uninstallDetails))
	assert.Equal(t, "uninstalled", uninstallDetails["status"])
	assert.EqualValues(t, true, uninstallDetails["self_service"])

	// Check uninstall results via device endpoint after execution
	res = s.DoRawNoAuth("GET", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%s/results", token, uninstallExecutionID), nil, http.StatusOK)
	uninstallResult = fleet.GetScriptResultResponse{}
	err = json.NewDecoder(res.Body).Decode(&uninstallResult)
	require.NoError(t, err)
	require.Equal(t, host1.DisplayName(), uninstallResult.HostName)
	require.Equal(t, uninstallExecutionID, uninstallResult.ExecutionID)
	require.Zero(t, *uninstallResult.ExitCode)
	require.Equal(t, "ok", uninstallResult.Output)

	// make sure uninstall endpoint errors properly
	s.DoRawNoAuth("GET", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%s/results", token2, uninstallExecutionID), nil, http.StatusNotFound)
	s.DoRawNoAuth("GET", fmt.Sprintf("/api/v1/fleet/device/%s/software/uninstall/%s/results", token, uninstallExecutionID+`f`), nil, http.StatusNotFound)
}

func (s *integrationEnterpriseTestSuite) TestHostSoftwareInstallResult() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)

	// Create software installers and corresponding host install requests.
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.deb",
		Title:             "ruby",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "deb_packages")
	payload2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script 2",
		PreInstallQuery:   "pre install query 2",
		PostInstallScript: "post install script 2",
		Filename:          "vim.deb",
		Title:             "vim",
	}
	s.uploadSoftwareInstaller(t, payload2, http.StatusOK, "")
	titleID2 := getSoftwareTitleID(t, s.ds, payload2.Title, "deb_packages")
	payload3 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script 3",
		PreInstallQuery:   "pre install query 3",
		PostInstallScript: "post install script 3",
		Filename:          "emacs.deb",
		Title:             "emacs",
	}
	s.uploadSoftwareInstaller(t, payload3, http.StatusOK, "")
	titleID3 := getSoftwareTitleID(t, s.ds, payload3.Title, "deb_packages")

	latestInstallUUID := func() string {
		var id string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT execution_id FROM upcoming_activities ORDER BY id DESC LIMIT 1`)
		})
		return id
	}

	// create some install requests for the host
	beforeInstall := time.Now()
	installUUIDs := make([]string, 3)
	titleIDs := []uint{titleID, titleID2, titleID3}
	for i := range installUUIDs {
		resp := installSoftwareResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/software/%d/install", host.ID, titleIDs[i]), nil, http.StatusAccepted, &resp)
		installUUIDs[i] = latestInstallUUID()
	}

	type result struct {
		HostID                  uint
		InstallUUID             string
		Status                  fleet.SoftwareInstallerStatus
		Output                  *string
		PostInstallScriptOutput *string
		PreInstallQueryOutput   *string
	}
	checkResults := func(want result) {
		var resp getSoftwareInstallResultsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/install/%s/results", want.InstallUUID), nil, http.StatusOK, &resp)

		assert.Equal(t, want.HostID, resp.Results.HostID)
		assert.Equal(t, want.InstallUUID, resp.Results.InstallUUID)
		assert.Equal(t, want.Status, resp.Results.Status)
		assert.Equal(t, want.PreInstallQueryOutput, resp.Results.PreInstallQueryOutput)
		assert.Equal(t, want.Output, resp.Results.Output)
		assert.Equal(t, want.PostInstallScriptOutput, resp.Results.PostInstallScriptOutput)
		assert.Less(t, beforeInstall, resp.Results.CreatedAt)
		assert.Greater(t, time.Now(), resp.Results.CreatedAt)
	}

	installHash := func(installUUID string) *string {
		res, err := s.ds.GetSoftwareInstallResults(ctx, installUUID)
		require.NoError(t, err)
		return res.HashSHA256
	}

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host.OrbitNodeKey, installUUIDs[0])),
		http.StatusNoContent)

	checkResults(result{
		HostID:                host.ID,
		InstallUUID:           installUUIDs[0],
		Status:                fleet.SoftwareInstallFailed,
		PreInstallQueryOutput: new(fleet.SoftwareInstallerQuerySuccessCopy),
		Output:                new(fmt.Sprintf(fleet.SoftwareInstallerInstallFailCopy, "failed")),
	})
	wantAct := fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload.Title,
		SoftwarePackage: payload.Filename,
		InstallUUID:     installUUIDs[0],
		Status:          string(fleet.SoftwareInstallFailed),
		Source:          new("deb_packages"),
		HashSHA256:      installHash(installUUIDs[0]),
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": ""
		}`, *host.OrbitNodeKey, installUUIDs[1])),
		http.StatusNoContent)

	checkResults(result{
		HostID:                host.ID,
		InstallUUID:           installUUIDs[1],
		Status:                fleet.SoftwareInstallFailed,
		PreInstallQueryOutput: new(fleet.SoftwareInstallerQueryFailCopy),
	})
	wantAct = fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload2.Title,
		SoftwarePackage: payload2.Filename,
		InstallUUID:     installUUIDs[1],
		Status:          string(fleet.SoftwareInstallFailed),
		Source:          new("deb_packages"),
		HashSHA256:      installHash(installUUIDs[1]),
	}
	s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "success",
			"post_install_script_exit_code": 0,
			"post_install_script_output": "ok"
		}`, *host.OrbitNodeKey, installUUIDs[2])),
		http.StatusNoContent)

	checkResults(result{
		HostID:                  host.ID,
		InstallUUID:             installUUIDs[2],
		Status:                  fleet.SoftwareInstalled,
		PreInstallQueryOutput:   new(fleet.SoftwareInstallerQuerySuccessCopy),
		Output:                  new(fmt.Sprintf(fleet.SoftwareInstallerInstallSuccessCopy, "success")),
		PostInstallScriptOutput: new(fmt.Sprintf(fleet.SoftwareInstallerPostInstallSuccessCopy, "ok")),
	})
	wantAct = fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload3.Title,
		SoftwarePackage: payload3.Filename,
		InstallUUID:     installUUIDs[2],
		Status:          string(fleet.SoftwareInstalled),
		Source:          new("deb_packages"),
		HashSHA256:      installHash(installUUIDs[2]),
	}
	lastActID := s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	// non-existing installation uuid
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": "uuid-no-such",
			"pre_install_condition_output": ""
		}`, *host.OrbitNodeKey)),
		http.StatusNotFound)
	// no new activity created
	s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), lastActID)

	// "re-install", but this time there's an installer download failure
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"install_script_exit_code": %d,
			"install_script_output": "Installer download failed"
		}`, *host.OrbitNodeKey, installUUIDs[2], fleet.ExitCodeInstallerDownloadFailed)),
		http.StatusNoContent)
	checkResults(result{
		HostID:      host.ID,
		InstallUUID: installUUIDs[2],
		Status:      fleet.SoftwareInstallFailed,
		Output:      new(fleet.SoftwareInstallerDownloadFailedCopy),
	})
	wantAct = fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload3.Title,
		SoftwarePackage: payload3.Filename,
		InstallUUID:     installUUIDs[2],
		Status:          string(fleet.SoftwareInstallFailed),
		Source:          new("deb_packages"),
		HashSHA256:      installHash(installUUIDs[2]),
	}
	s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	var hostActivityResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &hostActivityResp)
	require.Len(t, hostActivityResp.Activities, 4)
	require.NotNil(t, hostActivityResp.Activities[0].Details)
	var actDetails fleet.ActivityTypeInstalledSoftware
	require.NoError(t, json.Unmarshal(*hostActivityResp.Activities[0].Details, &actDetails))
	require.Equal(t, wantAct, actDetails)
}

func (s *integrationEnterpriseTestSuite) TestScriptOnlyPackageAdvancedOptions() {
	t := s.T()

	// Pre-install query, post-install, and uninstall scripts were previously
	// stripped for script-only packages; they should now persist. The install
	// script stays file-driven and isn't editable via PATCH.
	teamID := uint(0)
	payload := &fleet.UploadSoftwareInstallerPayload{
		Filename:          "script.sh",
		TeamID:            &teamID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1 FROM osquery_info;",
		PostInstallScript: "#!/bin/sh\necho post-install\n",
		UninstallScript:   "#!/bin/sh\necho uninstall\n",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	titleID := getSoftwareTitleID(t, s.ds, "script", "sh_packages")

	getPackage := func() *fleet.SoftwareInstaller {
		var resp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{}, http.StatusOK, &resp, "team_id", "0")
		require.NotNil(t, resp.SoftwareTitle)
		require.NotNil(t, resp.SoftwareTitle.SoftwarePackage)
		return resp.SoftwareTitle.SoftwarePackage
	}

	pkg := getPackage()
	// The install script is the uploaded .sh file's contents.
	require.Contains(t, pkg.InstallScript, `echo "script"`)
	require.Equal(t, "SELECT 1 FROM osquery_info;", pkg.PreInstallQuery)
	require.Contains(t, pkg.PostInstallScript, "echo post-install")
	require.Contains(t, pkg.UninstallScript, "echo uninstall")

	// PATCH new advanced-option values; they should persist. A PATCH to
	// install_script is ignored for script-only packages (file-driven).
	body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
		"team_id":             {"0"},
		"pre_install_query":   {"SELECT 2 FROM osquery_info;"},
		"post_install_script": {"#!/bin/sh\necho post-install-2\n"},
		"uninstall_script":    {"#!/bin/sh\necho uninstall-2\n"},
		"install_script":      {"#!/bin/sh\necho ignored\n"},
	})
	s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

	pkg = getPackage()
	require.Equal(t, "SELECT 2 FROM osquery_info;", pkg.PreInstallQuery)
	require.Contains(t, pkg.PostInstallScript, "echo post-install-2")
	require.Contains(t, pkg.UninstallScript, "echo uninstall-2")
	// Install script stays the uploaded file's contents; the PATCH is ignored.
	require.Contains(t, pkg.InstallScript, `echo "script"`)
	require.NotContains(t, pkg.InstallScript, "echo ignored")
}

// 1. software title uploaded doesn't match existing title
// 2. host reports software with the same bundle identifier
// 3. reconciler runs, doesn't create a new title
func (s *integrationEnterpriseTestSuite) TestPKGNewSoftwareTitleFlow() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		{Name: "DummyApp", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.dummy"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 4)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	// still one because the counts didn't update yet
	require.Len(t, resp.SoftwareTitles, 1)

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)

	// host reports another version of dummy, but this one has a
	// different name, but same identifier
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		{Name: "DummyApp", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.dummy"},
		{Name: "AppDummy", Version: "2.0.0", Source: "apps", BundleIdentifier: "com.example.dummy"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 5)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// titles are the same
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)
}

func (s *integrationEnterpriseTestSuite) TestPKGNoVersion() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some installer script",
		Filename:      "no_version.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	// title shows up with blank version
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.Equal(t, "NoVersion", resp.SoftwareTitles[0].Name)
	require.Empty(t, resp.SoftwareTitles[0].SoftwarePackage.Version)
}

func (s *integrationEnterpriseTestSuite) TestPKGNoBundleIdentifier() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some installer script",
		Filename:      "no_bundle_identifier.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Unable to extract necessary metadata.")
}

func (s *integrationEnterpriseTestSuite) TestEXEPackageUploads() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some installer script",
		Filename:      "hello-world-installer.exe",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Uninstall script is required for .exe packages.")

	payload = &fleet.UploadSoftwareInstallerPayload{
		Filename: "hello-world-installer.exe",
		TeamID:   &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Install script is required for .exe packages.")

	payload = &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some installer script",
		UninstallScript:  "some uninstall script",
		Filename:         "hello-world-installer.exe",
		TeamID:           &team.ID,
		AutomaticInstall: true,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Couldn't add. Fleet can't create a policy to detect existing installations for .exe packages. Please add the software, add a custom policy, and enable the install software policy automation.")

	payload = &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "some installer script",
		UninstallScript: "some uninstall script",
		Filename:        "hello-world-installer.exe",
		TeamID:          &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	var titleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &titleID, `SELECT title_id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, &team.ID, payload.Filename)
	})
	require.NotZero(t, titleID)

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript: new(""),
		TitleID:       titleID,
		TeamID:        &team.ID,
	}, http.StatusBadRequest, "Couldn't edit. Install script is required for .exe packages.")

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		Filename:        "hello-world-installer.exe",
		UninstallScript: new(""),
		TitleID:         titleID,
		TeamID:          &team.ID,
	}, http.StatusBadRequest, "Couldn't edit. Uninstall script is required for .exe packages.")

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		Filename:          "hello-world-installer.exe",
		PostInstallScript: new("foo bar baz"),
		TitleID:           titleID,
		TeamID:            &team.ID,
	}, http.StatusOK, "")
}

func (s *integrationEnterpriseTestSuite) TestScriptPackageUploads() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	// Create temporary script files for testing
	shContent := "#!/bin/bash\necho 'Installing...'\nexit 0\n"
	ps1Content := "Write-Host 'Installing...'\nExit 0\n"

	shFile, err := fleet.NewTempFileReader(strings.NewReader(shContent), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer shFile.Close()

	ps1File, err := fleet.NewTempFileReader(strings.NewReader(ps1Content), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer ps1File.Close()

	// Test .sh file with automatic_install (should be rejected)
	payload := &fleet.UploadSoftwareInstallerPayload{
		Filename:         "install-app.sh",
		TeamID:           &team.ID,
		AutomaticInstall: true,
		InstallerFile:    shFile,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Couldn't add. Fleet can't create a policy to detect existing installations for .sh packages.")

	// Rewind the file reader for reuse
	err = shFile.Rewind()
	require.NoError(t, err)

	// Test .ps1 file with automatic_install (should be rejected)
	payload = &fleet.UploadSoftwareInstallerPayload{
		Filename:         "install-app.ps1",
		TeamID:           &team.ID,
		AutomaticInstall: true,
		InstallerFile:    ps1File,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Couldn't add. Fleet can't create a policy to detect existing installations for .ps1 packages.")

	// Rewind the file reader for reuse
	err = shFile.Rewind()
	require.NoError(t, err)

	// .sh script package with advanced options, which should persist.
	payload = &fleet.UploadSoftwareInstallerPayload{
		Filename:          "install-app.sh",
		TeamID:            &team.ID,
		UninstallScript:   "echo 'uninstall'",
		PostInstallScript: "echo 'post'",
		PreInstallQuery:   "SELECT 1;",
		InstallerFile:     shFile,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	var titleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &titleID, `SELECT title_id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, &team.ID, payload.Filename)
	})
	require.NotZero(t, titleID)

	// Verify advanced options persisted (pre_install_query has its trailing ; trimmed).
	var scriptContents struct {
		UninstallScript   string `db:"uninstall_script"`
		PostInstallScript string `db:"post_install_script"`
		PreInstallQuery   string `db:"pre_install_query"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &scriptContents, `
			SELECT
				COALESCE(uninst.contents, '') AS uninstall_script,
				COALESCE(postinst.contents, '') AS post_install_script,
				pre_install_query
			FROM software_installers si
			LEFT JOIN script_contents uninst ON uninst.id = si.uninstall_script_content_id
			LEFT JOIN script_contents postinst ON postinst.id = si.post_install_script_content_id
			WHERE si.title_id = ? AND si.global_or_team_id = ?`, titleID, team.ID)
	})
	require.Equal(t, "echo 'uninstall'", scriptContents.UninstallScript)
	require.Equal(t, "echo 'post'", scriptContents.PostInstallScript)
	require.Equal(t, "SELECT 1;", scriptContents.PreInstallQuery)

	// Editing a script package updates its advanced options.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		Filename:          "install-app.sh",
		UninstallScript:   new("echo 'updated uninstall'"),
		PostInstallScript: new("echo 'updated post'"),
		PreInstallQuery:   new("SELECT 2"),
		TitleID:           titleID,
		TeamID:            &team.ID,
	}, http.StatusOK, "")

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &scriptContents, `
			SELECT
				COALESCE(uninst.contents, '') AS uninstall_script,
				COALESCE(postinst.contents, '') AS post_install_script,
				pre_install_query
			FROM software_installers si
			LEFT JOIN script_contents uninst ON uninst.id = si.uninstall_script_content_id
			LEFT JOIN script_contents postinst ON postinst.id = si.post_install_script_content_id
			WHERE si.title_id = ? AND si.global_or_team_id = ?`, titleID, team.ID)
	})
	require.Equal(t, "echo 'updated uninstall'", scriptContents.UninstallScript)
	require.Equal(t, "echo 'updated post'", scriptContents.PostInstallScript)
	require.Equal(t, "SELECT 2", scriptContents.PreInstallQuery)

	// Replacing the file updates the install script (the file is the install
	// script), keeping install_script_content_id consistent with storage_id.
	newSHContent := "#!/bin/bash\necho 'Installing v2...'\nexit 0\n"
	newSHFile, err := fleet.NewTempFileReader(strings.NewReader(newSHContent), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer newSHFile.Close()

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		Filename:      "install-app.sh",
		InstallerFile: newSHFile,
		TitleID:       titleID,
		TeamID:        &team.ID,
	}, http.StatusOK, "")

	var afterReplace struct {
		InstallScript string `db:"install_script"`
		StorageID     string `db:"storage_id"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &afterReplace, `
			SELECT COALESCE(inst.contents, '') AS install_script, si.storage_id
			FROM software_installers si
			LEFT JOIN script_contents inst ON inst.id = si.install_script_content_id
			WHERE si.title_id = ? AND si.global_or_team_id = ?`, titleID, team.ID)
	})
	require.Equal(t, newSHContent, afterReplace.InstallScript, "replacing the file should update install_script")
	expectedHash := sha256.Sum256([]byte(newSHContent))
	require.Equal(t, hex.EncodeToString(expectedHash[:]), afterReplace.StorageID, "storage_id should match the new file's contents")

	// GitOps adds a path-based script package by sending a "script://filename" placeholder
	// url plus the script hash. The placeholder must not be persisted.
	installScript := "echo 'hello world'"
	scriptSum := sha256.Sum256([]byte(installScript))
	scriptPkg := []*fleet.SoftwareInstallerPayload{
		{URL: "script://hello world.sh", SHA256: hex.EncodeToString(scriptSum[:]), InstallScript: installScript},
	}

	// First apply: no matching installer yet, so this exercises the download path.
	var batchResp batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: scriptPkg}, http.StatusAccepted, &batchResp, "team_name", team.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchResp.RequestUUID)
	require.Len(t, packages, 1)

	var storedURL string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &storedURL, `SELECT url FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, &team.ID, "hello world.sh")
	})
	require.Empty(t, storedURL, "script:// placeholder url must not be persisted")

	// Seed a stale placeholder url like an older Fleet would, then re-apply. The hash and
	// url now match an existing installer, so this hits the cache-hit path, which must
	// also drop the placeholder.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET url = ? WHERE global_or_team_id = ? AND filename = ?`, "script://hello world.sh", &team.ID, "hello world.sh")
		return err
	})
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: scriptPkg}, http.StatusAccepted, &batchResp, "team_name", team.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, team.Name, batchResp.RequestUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &storedURL, `SELECT url FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, &team.ID, "hello world.sh")
	})
	require.Empty(t, storedURL, "cache-hit re-apply must drop the placeholder url too")

	pyContent := "#!/usr/bin/env python3\nprint('Installing...')\n"
	pyFile, err := fleet.NewTempFileReader(strings.NewReader(pyContent), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer pyFile.Close()

	payload = &fleet.UploadSoftwareInstallerPayload{
		Filename:         "install-app.py",
		TeamID:           &team.ID,
		AutomaticInstall: true,
		InstallerFile:    pyFile,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Couldn't add. Fleet can't create a policy to detect existing installations for .py packages.")

	err = pyFile.Rewind()
	require.NoError(t, err)

	badPyContent := "print('no shebang')\n"
	badPyFile, err := fleet.NewTempFileReader(strings.NewReader(badPyContent), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer badPyFile.Close()
	payload = &fleet.UploadSoftwareInstallerPayload{
		Filename:      "no-shebang.py",
		TeamID:        &team.ID,
		InstallerFile: badPyFile,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Script validation failed")

	// install_script is derived from the file, so any install_script param is ignored.
	payload = &fleet.UploadSoftwareInstallerPayload{
		Filename:          "install-app.py",
		Title:             "install-app.py",
		TeamID:            &team.ID,
		InstallScript:     "this should be ignored",
		UninstallScript:   "echo 'uninstall py'",
		PostInstallScript: "echo 'post py'",
		PreInstallQuery:   "SELECT 1;",
		InstallerFile:     pyFile,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	var pyStored struct {
		Source            string `db:"source"`
		InstallScript     string `db:"install_script"`
		UninstallScript   string `db:"uninstall_script"`
		PostInstallScript string `db:"post_install_script"`
		PreInstallQuery   string `db:"pre_install_query"`
		Platform          string `db:"platform"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &pyStored, `
			SELECT
				st.source,
				COALESCE(inst.contents, '') AS install_script,
				COALESCE(uninst.contents, '') AS uninstall_script,
				COALESCE(postinst.contents, '') AS post_install_script,
				si.pre_install_query,
				si.platform
			FROM software_installers si
			JOIN software_titles st ON st.id = si.title_id
			LEFT JOIN script_contents inst ON inst.id = si.install_script_content_id
			LEFT JOIN script_contents uninst ON uninst.id = si.uninstall_script_content_id
			LEFT JOIN script_contents postinst ON postinst.id = si.post_install_script_content_id
			WHERE si.global_or_team_id = ? AND si.filename = ?`, team.ID, payload.Filename)
	})
	require.Equal(t, "py_packages", pyStored.Source, "py package should be stored with py_packages source")
	require.Equal(t, pyContent, pyStored.InstallScript, "install_script should be the .py file contents, not the ignored param")
	require.Equal(t, "echo 'uninstall py'", pyStored.UninstallScript)
	require.Equal(t, "echo 'post py'", pyStored.PostInstallScript)
	require.Equal(t, "SELECT 1;", pyStored.PreInstallQuery)
	require.Equal(t, "linux", pyStored.Platform, ".py packages are stored with the linux platform")

	// Fresh team so filename collisions from earlier assertions don't leak in.
	crossTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "cross"})
	require.NoError(t, err)

	crossScript := "echo 'cross-platform hello'"
	crossHash := sha256.Sum256([]byte(crossScript))
	darwinOnly := []string{"darwin"}
	crossPkg := []*fleet.SoftwareInstallerPayload{
		{
			URL:                      "script://cross-hello.sh",
			SHA256:                   hex.EncodeToString(crossHash[:]),
			InstallScript:            crossScript,
			SetupExperiencePlatforms: &darwinOnly,
		},
	}

	// [darwin] on a .sh (native=linux): cross-table row for darwin, install_during_setup stays off.
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkg}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)

	var crossRows []struct {
		SoftwareInstallerID uint   `db:"software_installer_id"`
		Platform            string `db:"platform"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, crossTeam.ID)
	})
	require.Len(t, crossRows, 1, "expected exactly one cross-platform selection")
	require.Equal(t, "darwin", crossRows[0].Platform)
	require.NotZero(t, crossRows[0].SoftwareInstallerID)

	var installDuringSetup bool
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installDuringSetup,
			`SELECT install_during_setup FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, crossTeam.ID, "cross-hello.sh")
	})
	require.False(t, installDuringSetup, "linux not selected → install_during_setup should stay false")

	// Omitting the field on re-apply leaves the cross-table alone — the batch
	// only reconciles when at least one payload opts in, so UI-set selections
	// aren't clobbered by callers that don't know about the field.
	crossPkgNoField := []*fleet.SoftwareInstallerPayload{
		{URL: "script://cross-hello.sh", SHA256: hex.EncodeToString(crossHash[:]), InstallScript: crossScript},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkgNoField}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, crossTeam.ID)
	})
	require.Len(t, crossRows, 1, "omitting the field is a no-op; prior selection survives")

	// Explicit empty list opts into reconcile with nothing selected → clears.
	emptyPlatforms := []string{}
	crossPkgEmpty := []*fleet.SoftwareInstallerPayload{
		{URL: "script://cross-hello.sh", SHA256: hex.EncodeToString(crossHash[:]), InstallScript: crossScript, SetupExperiencePlatforms: &emptyPlatforms},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkgEmpty}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, crossTeam.ID)
	})
	require.Empty(t, crossRows, "explicit empty list clears the cross-table row")

	// Re-apply is idempotent.
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkg}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, crossTeam.ID)
	})
	require.Len(t, crossRows, 1, "cross-table row should be restored on re-apply")

	// Both platforms in the list: install_during_setup flips on from the list
	// alone (native "linux" is present), and the darwin cross-row is
	// preserved. When SetupExperiencePlatforms is set it's authoritative for
	// both the native flag and the cross-table.
	bothPlatforms := []string{"darwin", "linux"}
	crossPkgBoth := []*fleet.SoftwareInstallerPayload{
		{
			URL:                      "script://cross-hello.sh",
			SHA256:                   hex.EncodeToString(crossHash[:]),
			InstallScript:            crossScript,
			SetupExperiencePlatforms: &bothPlatforms,
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkgBoth}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installDuringSetup,
			`SELECT install_during_setup FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, crossTeam.ID, "cross-hello.sh")
	})
	require.True(t, installDuringSetup, "native in list should set install_during_setup=1 without a separate setup_experience bool")
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, crossTeam.ID)
	})
	require.Len(t, crossRows, 1, "darwin cross-row still present when [macos, linux] is set")

	// Native-only in the list is equivalent to setup_experience: true:
	// install_during_setup=1, cross-table cleared for this installer.
	linuxOnly := []string{"linux"}
	crossPkgNative := []*fleet.SoftwareInstallerPayload{
		{URL: "script://cross-hello.sh", SHA256: hex.EncodeToString(crossHash[:]), InstallScript: crossScript, SetupExperiencePlatforms: &linuxOnly},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkgNative}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installDuringSetup,
			`SELECT install_during_setup FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, crossTeam.ID, "cross-hello.sh")
	})
	require.True(t, installDuringSetup, "native-only list keeps install_during_setup=1")
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, crossTeam.ID)
	})
	require.Empty(t, crossRows, "native-only list clears the cross-table for this installer")

	// Extension/platform mismatch: windows on a .sh.
	var failure string
	windows := []string{"windows"}
	crossPkgBad := []*fleet.SoftwareInstallerPayload{
		{URL: "script://cross-hello.sh", SHA256: hex.EncodeToString(crossHash[:]), InstallScript: crossScript, SetupExperiencePlatforms: &windows},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: crossPkgBad}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	failure = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)
	require.Contains(t, failure, `platform "windows" is not a valid "setup_experience_platform" value for a .sh package`)

	// Multi-installer batch with mixed opt-in. Installer A opts into
	// [darwin]; installer B leaves SetupExperiencePlatforms nil. B's existing
	// darwin cross-row (seeded via a prior explicit apply) must survive the
	// batch instead of being wiped by A's opt-in.
	scriptB := "echo 'sibling'"
	scriptBHash := sha256.Sum256([]byte(scriptB))
	// First, give both A and B a darwin cross-row so we have prior state to
	// preserve.
	seedA := []string{"darwin"}
	seedB := []string{"darwin"}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{
		{URL: "script://cross-hello.sh", SHA256: hex.EncodeToString(crossHash[:]), InstallScript: crossScript, SetupExperiencePlatforms: &seedA},
		{URL: "script://sibling.sh", SHA256: hex.EncodeToString(scriptBHash[:]), InstallScript: scriptB, SetupExperiencePlatforms: &seedB},
	}}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ? ORDER BY software_installer_id`, crossTeam.ID)
	})
	require.Len(t, crossRows, 2, "seed step: both installers should have darwin rows")

	// Now re-apply with A opting into [] (clear) and B leaving the field
	// nil. B's row must survive.
	empty := []string{}
	mixed := []*fleet.SoftwareInstallerPayload{
		{URL: "script://cross-hello.sh", SHA256: hex.EncodeToString(crossHash[:]), InstallScript: crossScript, SetupExperiencePlatforms: &empty},
		{URL: "script://sibling.sh", SHA256: hex.EncodeToString(scriptBHash[:]), InstallScript: scriptB},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: mixed}, http.StatusAccepted, &batchResp, "team_name", crossTeam.Name)
	waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, crossTeam.Name, batchResp.RequestUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &crossRows,
			`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ? ORDER BY software_installer_id`, crossTeam.ID)
	})
	require.Len(t, crossRows, 1, "A's row cleared by explicit []; B's row untouched by nil")

	var siblingID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &siblingID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, crossTeam.ID, "sibling.sh")
	})
	require.Equal(t, siblingID, crossRows[0].SoftwareInstallerID, "surviving row should be sibling.sh (B)")

	// .py packages are cross-platform on the same terms as .sh.
	pyTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "pycross"})
	require.NoError(t, err)

	pyCrossScript := "#!/usr/bin/env python3\nprint('cross-platform hello')\n"
	pyCrossHash := sha256.Sum256([]byte(pyCrossScript))
	pyPkg := func(platforms *[]string) []*fleet.SoftwareInstallerPayload {
		return []*fleet.SoftwareInstallerPayload{{
			URL:                      "script://cross-hello.py",
			SHA256:                   hex.EncodeToString(pyCrossHash[:]),
			InstallScript:            pyCrossScript,
			SetupExperiencePlatforms: platforms,
		}}
	}
	applyPy := func(platforms *[]string) {
		s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: pyPkg(platforms)},
			http.StatusAccepted, &batchResp, "team_name", pyTeam.Name)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, pyTeam.Name, batchResp.RequestUUID)
	}
	pyState := func() ([]struct {
		SoftwareInstallerID uint   `db:"software_installer_id"`
		Platform            string `db:"platform"`
	}, bool,
	) {
		var rows []struct {
			SoftwareInstallerID uint   `db:"software_installer_id"`
			Platform            string `db:"platform"`
		}
		var duringSetup bool
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if err := sqlx.SelectContext(ctx, q, &rows,
				`SELECT software_installer_id, platform FROM setup_experience_software_installers WHERE global_or_team_id = ?`, pyTeam.ID); err != nil {
				return err
			}
			return sqlx.GetContext(ctx, q, &duringSetup,
				`SELECT install_during_setup FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, pyTeam.ID, "cross-hello.py")
		})
		return rows, duringSetup
	}

	applyPy(&darwinOnly)
	pyRows, pyDuringSetup := pyState()
	require.Len(t, pyRows, 1, "expected one cross-platform selection for the .py package")
	require.Equal(t, "darwin", pyRows[0].Platform)
	require.False(t, pyDuringSetup, "linux not selected → install_during_setup should stay false")

	var pyStoredShape struct {
		Platform  string `db:"platform"`
		Extension string `db:"extension"`
		Source    string `db:"source"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &pyStoredShape,
			`SELECT si.platform, si.extension, st.source FROM software_installers si
			 JOIN software_titles st ON st.id = si.title_id
			 WHERE si.global_or_team_id = ? AND si.filename = ?`, pyTeam.ID, "cross-hello.py")
	})
	require.Equal(t, "linux", pyStoredShape.Platform)
	require.Equal(t, "py", pyStoredShape.Extension)
	require.Equal(t, "py_packages", pyStoredShape.Source)

	// Omitting the field is a no-op; only an explicit empty value clears.
	// See reconcileGitOpsSetupExperienceCrossInstallers.
	applyPy(nil)
	pyRows, _ = pyState()
	require.Len(t, pyRows, 1, "omitting the field leaves the prior selection alone")

	applyPy(&emptyPlatforms)
	pyRows, pyDuringSetup = pyState()
	require.Empty(t, pyRows, "explicit empty list clears the cross-table row")
	require.False(t, pyDuringSetup)

	applyPy(&bothPlatforms)
	pyRows, pyDuringSetup = pyState()
	require.Len(t, pyRows, 1)
	require.True(t, pyDuringSetup, "native in list selects the native platform")

	applyPy(&linuxOnly)
	pyRows, pyDuringSetup = pyState()
	require.Empty(t, pyRows, "native-only list clears the cross-table")
	require.True(t, pyDuringSetup)

	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: pyPkg(&windows)},
		http.StatusAccepted, &batchResp, "team_name", pyTeam.Name)
	failure = waitBatchSetSoftwareInstallersFailed(t, &s.withServer, pyTeam.Name, batchResp.RequestUUID)
	require.Contains(t, failure, `platform "windows" is not a valid "setup_experience_platform" value for a .py package`)
}

func (s *integrationEnterpriseTestSuite) TestSoftwareMultiplePackagesPerTitle() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	// Two script packages with the same filename resolve to the same title
	// ("deploy"), and different contents give them distinct content hashes, so
	// they coexist as two packages under one title.
	contentA := "#!/bin/bash\necho 'A'\n"
	contentB := "#!/bin/bash\necho 'B'\n"
	contentC := "#!/bin/bash\necho 'C'\n"
	hashOf := func(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }

	upload := func(content string, selfService bool, expectedStatus int, expectedErr string) {
		fr, err := fleet.NewTempFileReader(strings.NewReader(content), func() string { return t.TempDir() })
		require.NoError(t, err)
		defer fr.Close()
		s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
			Filename:      "deploy.sh",
			TeamID:        &team.ID,
			SelfService:   selfService,
			InstallerFile: fr,
		}, expectedStatus, expectedErr)
	}

	// rawSoftwareMultipart adds or edits a script package and returns the raw
	// response body, so we can assert exactly which package the endpoint echoes.
	rawSoftwareMultipart := func(method, path, content string, extra map[string]string) []byte {
		fr, err := fleet.NewTempFileReader(strings.NewReader(content), func() string { return t.TempDir() })
		require.NoError(t, err)
		defer fr.Close()
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("software", "deploy.sh")
		require.NoError(t, err)
		_, err = io.Copy(fw, fr)
		require.NoError(t, err)
		require.NoError(t, w.WriteField("team_id", fmt.Sprintf("%d", team.ID)))
		require.NoError(t, w.WriteField("fleet_id", fmt.Sprintf("%d", team.ID)))
		for k, v := range extra {
			require.NoError(t, w.WriteField(k, v))
		}
		require.NoError(t, w.Close())
		headers := map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", s.token),
		}
		r := s.DoRawWithHeaders(method, path, body.Bytes(), http.StatusOK, headers)
		defer r.Body.Close()
		respBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		return respBody
	}

	// Add a first package (A, not self-service), then a second (B, self-service).
	// The POST response must echo the package that was just added (not the title's
	// first-added one).
	upload(contentA, false, http.StatusOK, "")
	postBody := rawSoftwareMultipart("POST", "/api/latest/fleet/software/package", contentB, map[string]string{"self_service": "true"})
	var addResp uploadSoftwareInstallerResponse
	require.NoError(t, json.Unmarshal(postBody, &addResp))
	require.NotNil(t, addResp.SoftwarePackage)
	require.Equal(t, hashOf(contentB), addResp.SoftwarePackage.StorageID, "POST echoes the just-added package, not first-added")

	// Adding a second package emits the same added_software activity.
	s.lastActivityMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), ``, 0)

	// Re-uploading identical bytes is rejected (per-title hash dedupe).
	upload(contentA, false, http.StatusConflict, "already added (same SHA-256 hash)")

	var titleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleID,
			`SELECT DISTINCT title_id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, team.ID, "deploy.sh")
	})
	require.NotZero(t, titleID)

	getTitle := func() *fleet.SoftwareTitle {
		var resp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), getSoftwareTitleRequest{},
			http.StatusOK, &resp, "team_id", fmt.Sprintf("%d", team.ID))
		return resp.SoftwareTitle
	}

	// --- Detail endpoint: full packages[] shape, software_package == first-added ---
	title := getTitle()
	require.Len(t, title.Packages, 2)
	require.NotNil(t, title.SoftwarePackage)
	require.Equal(t, title.Packages[0].InstallerID, title.SoftwarePackage.InstallerID)
	require.Equal(t, hashOf(contentA), title.Packages[0].StorageID)
	require.Equal(t, hashOf(contentB), title.Packages[1].StorageID)
	require.Equal(t, hashOf(contentA), title.SoftwarePackage.StorageID)
	// per-package fields are independent
	require.False(t, title.Packages[0].SelfService)
	require.True(t, title.Packages[1].SelfService)
	// scripts are hydrated (for a script package the install script is the file)
	require.Equal(t, contentA, title.Packages[0].InstallScript)
	require.Equal(t, contentB, title.Packages[1].InstallScript)

	installerA := title.Packages[0].InstallerID
	installerB := title.Packages[1].InstallerID

	// --- List endpoint: trimmed packages[] ---
	var listResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listResp,
		"query", "deploy", "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, listResp.SoftwareTitles, 1)
	lt := listResp.SoftwareTitles[0]
	require.Len(t, lt.Packages, 2)
	require.NotNil(t, lt.SoftwarePackage)
	require.Equal(t, "deploy.sh", lt.SoftwarePackage.Name)
	require.Equal(t, "deploy.sh", lt.Packages[0].Name)
	require.NotNil(t, lt.Packages[0].SelfService)
	require.False(t, *lt.Packages[0].SelfService)
	require.NotNil(t, lt.Packages[1].SelfService)
	require.True(t, *lt.Packages[1].SelfService)

	// The list packages[] must omit the host-only last_install/last_uninstall fields.
	listRes := s.Do("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK,
		"query", "deploy", "team_id", fmt.Sprintf("%d", team.ID))
	rawList, err := io.ReadAll(listRes.Body)
	require.NoError(t, err)
	listRes.Body.Close()
	var rawListParsed struct {
		SoftwareTitles []struct {
			Packages []map[string]json.RawMessage `json:"packages"`
		} `json:"software_titles"`
	}
	require.NoError(t, json.Unmarshal(rawList, &rawListParsed))
	require.Len(t, rawListParsed.SoftwareTitles, 1)
	require.Len(t, rawListParsed.SoftwareTitles[0].Packages, 2)
	for _, p := range rawListParsed.SoftwareTitles[0].Packages {
		_, hasLastInstall := p["last_install"]
		_, hasLastUninstall := p["last_uninstall"]
		require.False(t, hasLastInstall, "list packages[] must omit last_install")
		require.False(t, hasLastUninstall, "list packages[] must omit last_uninstall")
	}

	// --- Download token: installer_id pins the token to a specific package (#49239) ---
	// Bug: the token endpoint ignored installer_id and always dispatched
	// first-added, so the UI download button on any row of a multi-package title
	// fetched A regardless of which row the user clicked.
	readAllAndClose := func(t *testing.T, r *http.Response) []byte {
		t.Helper()
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		return b
	}
	tokenDownload := func(t *testing.T, args ...string) []byte {
		t.Helper()
		var tok getSoftwareInstallerTokenResponse
		s.DoJSON("POST",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", titleID),
			nil, http.StatusOK, &tok, args...)
		require.NotEmpty(t, tok.Token)
		r := s.DoRawNoAuth("GET",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", titleID, tok.Token),
			nil, http.StatusOK)
		return readAllAndClose(t, r)
	}
	teamArg := []string{"team_id", fmt.Sprintf("%d", team.ID)}
	// Omitting installer_id falls back to first-added (A), preserving single-package back-compat.
	require.Equal(t, contentA, string(tokenDownload(t, teamArg...)),
		"no installer_id defaults to first-added (A)")
	// Explicit installer_id=A resolves to A.
	require.Equal(t, contentA,
		string(tokenDownload(t, append(teamArg, "installer_id", fmt.Sprintf("%d", installerA))...)),
		"installer_id=A returns A's content")
	// Explicit installer_id=B resolves to B (the fix).
	require.Equal(t, contentB,
		string(tokenDownload(t, append(teamArg, "installer_id", fmt.Sprintf("%d", installerB))...)),
		"installer_id=B returns B's content, not first-added")

	// Direct (non-token) download endpoint honors installer_id too.
	rDirect := s.Do("GET",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID),
		nil, http.StatusOK,
		"team_id", fmt.Sprintf("%d", team.ID),
		"installer_id", fmt.Sprintf("%d", installerB))
	require.Equal(t, contentB, string(readAllAndClose(t, rDirect)),
		"direct download with installer_id=B returns B")

	// --- Edit without installer_id on a multi-package title -> 400 ---
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     titleID,
		TeamID:      &team.ID,
		SelfService: new(true),
	}, http.StatusBadRequest, "installer_id is required")

	// --- Edit B's file to A's content (sibling hash collision) -> 409, both unchanged ---
	collideFile, err := fleet.NewTempFileReader(strings.NewReader(contentA), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer collideFile.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		InstallerID:   installerB,
		TeamID:        &team.ID,
		Filename:      "deploy.sh",
		InstallerFile: collideFile,
	}, http.StatusConflict, "already added (same SHA-256 hash)")

	title = getTitle()
	require.Len(t, title.Packages, 2)
	require.Equal(t, hashOf(contentA), title.Packages[0].StorageID)
	require.Equal(t, hashOf(contentB), title.Packages[1].StorageID)

	// --- Edit B's file to a new hash -> ok; A untouched; PATCH echoes the edited package ---
	patchBody := rawSoftwareMultipart("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID),
		contentC, map[string]string{"installer_id": fmt.Sprintf("%d", installerB)})
	var patchResp getSoftwareInstallerResponse
	require.NoError(t, json.Unmarshal(patchBody, &patchResp))
	require.NotNil(t, patchResp.SoftwareInstaller)
	require.Equal(t, installerB, patchResp.SoftwareInstaller.InstallerID, "PATCH echoes the edited package, not first-added")
	require.Equal(t, hashOf(contentC), patchResp.SoftwareInstaller.StorageID)
	title = getTitle()
	require.Equal(t, hashOf(contentA), title.Packages[0].StorageID)
	require.Equal(t, hashOf(contentC), title.Packages[1].StorageID)

	// --- Re-save B's current file -> no-op (200) ---
	sameFile, err := fleet.NewTempFileReader(strings.NewReader(contentC), func() string { return t.TempDir() })
	require.NoError(t, err)
	defer sameFile.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		InstallerID:   installerB,
		TeamID:        &team.ID,
		Filename:      "deploy.sh",
		InstallerFile: sameFile,
	}, http.StatusOK, "")
	title = getTitle()
	require.Len(t, title.Packages, 2)
	require.Equal(t, hashOf(contentC), title.Packages[1].StorageID)

	// --- Per-package categories: a targeted PATCH echo must match GET, not the title-merged union ---
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: titleID, InstallerID: installerA, TeamID: &team.ID, Categories: []string{"Browsers"},
	}, http.StatusOK, "")
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: titleID, InstallerID: installerB, TeamID: &team.ID, Categories: []string{"Productivity"},
	}, http.StatusOK, "")
	title = getTitle()
	catsA := title.Packages[0].Categories
	catsB := title.Packages[1].Categories
	require.NotEmpty(t, catsA)
	require.NotEmpty(t, catsB)
	require.NotEqual(t, catsA, catsB, "packages carry distinct categories")

	// A noop PATCH targeting the first-added installer must echo that installer's own
	// categories, not the union across sibling packages.
	var noopBody bytes.Buffer
	noopW := multipart.NewWriter(&noopBody)
	require.NoError(t, noopW.WriteField("team_id", fmt.Sprintf("%d", team.ID)))
	require.NoError(t, noopW.WriteField("installer_id", fmt.Sprintf("%d", installerA)))
	require.NoError(t, noopW.Close())
	noopResp := s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID),
		noopBody.Bytes(), http.StatusOK, map[string]string{
			"Content-Type":  noopW.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", s.token),
		})
	noopRaw, err := io.ReadAll(noopResp.Body)
	require.NoError(t, err)
	noopResp.Body.Close()
	var noopParsed getSoftwareInstallerResponse
	require.NoError(t, json.Unmarshal(noopRaw, &noopParsed))
	require.NotNil(t, noopParsed.SoftwareInstaller)
	require.Equal(t, installerA, noopParsed.SoftwareInstaller.InstallerID)
	require.ElementsMatch(t, catsA, noopParsed.SoftwareInstaller.Categories,
		"noop PATCH echo must return the targeted installer's own categories, not the title-merged set")

	// A title-level display name is shared across sibling packages; deleting one must not wipe it.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, ?)`,
			team.ID, titleID, "My Deploy Tool")
		return err
	})
	displayName := func() string {
		var name string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &name,
				`SELECT COALESCE(MAX(display_name), '') FROM software_title_display_names WHERE team_id = ? AND software_title_id = ?`,
				team.ID, titleID)
		})
		return name
	}

	// --- Delete one package (A) -> 204, B remains and becomes first-added ---
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil,
		http.StatusNoContent, "team_id", fmt.Sprintf("%d", team.ID), "installer_id", fmt.Sprintf("%d", installerA))
	title = getTitle()
	require.Len(t, title.Packages, 1)
	require.Equal(t, installerB, title.Packages[0].InstallerID)
	require.Equal(t, installerB, title.SoftwarePackage.InstallerID)
	require.Equal(t, "My Deploy Tool", displayName(), "deleting one of several packages must not wipe the shared display name")

	// --- Delete all remaining (no installer_id) -> 204, no packages left ---
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil,
		http.StatusNoContent, "team_id", fmt.Sprintf("%d", team.ID))
	require.Empty(t, displayName(), "deleting the last package removes the title display name")
	var remaining int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &remaining,
			`SELECT COUNT(*) FROM software_installers WHERE global_or_team_id = ? AND title_id = ?`, team.ID, titleID)
	})
	require.Zero(t, remaining, "title-level delete removes all packages")
}

// 1. host reports software
// 2. reconciler runs, creates title
// 3. installer is uploaded, creates a new software title
func (s *integrationEnterpriseTestSuite) TestPKGSoftwareAlreadyReported() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		// note: the source is not "apps". Homebrew packages do not report bundle_identifier
		// so it is possible to add a pkg installer with the same name
		{Name: "DummyApp", Version: "1.0.0", Source: "homebrew"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 4)

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}

	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 4)
	require.ElementsMatch(
		t,
		[]any{[]string{"foo", "homebrew"}, []string{"bar", "apps"}, []string{"DummyApp", "homebrew"}, []string{"DummyApp", "apps"}},
		[]any{
			[]string{resp.SoftwareTitles[0].Name, resp.SoftwareTitles[0].Source},
			[]string{resp.SoftwareTitles[1].Name, resp.SoftwareTitles[1].Source},
			[]string{resp.SoftwareTitles[2].Name, resp.SoftwareTitles[2].Source},
			[]string{resp.SoftwareTitles[3].Name, resp.SoftwareTitles[3].Source},
		},
	)
}

// 1. host reports software
// 2. installer is uploaded, creates new software title
// 3. reconciler runs, no changes
func (s *integrationEnterpriseTestSuite) TestPKGSoftwareReconciliation() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		// note: the source is not "apps". Homebrew packages do not report bundle_identifier
		// so it is possible to add a pkg installer with the same name
		{Name: "DummyApp", Version: "1.0.0", Source: "homebrew"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 4)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	// only one title (the uploaded software) because the cron didn't run yet
	require.Len(t, resp.SoftwareTitles, 1)
	require.ElementsMatch(
		t,
		[]string{"DummyApp"},
		[]string{resp.SoftwareTitles[0].Name},
	)
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 4)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp", "DummyApp"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
			resp.SoftwareTitles[3].Name,
		},
	)
}

func (s *integrationEnterpriseTestSuite) TestNonPolicySoftwareInstallRetries() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{},
		http.StatusOK, &resp, "query", "DummyApp", "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	softwareTitleID := resp.SoftwareTitles[0].ID

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team.ID, "dummy_installer.pkg")
	})
	require.NotZero(t, installerID)

	getPendingInstall := func() *fleet.HostSoftwareInstallerResult {
		var results []*fleet.HostSoftwareInstallerResult
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &results, `
				SELECT
					id,
					execution_id,
					host_id,
					software_installer_id,
					self_service,
					install_script_exit_code,
					attempt_number
				FROM host_software_installs
				WHERE host_id = ? AND install_script_exit_code IS NULL AND policy_id IS NULL
				ORDER BY id ASC
			`, host.ID)
		})
		if len(results) == 0 {
			return nil
		}
		return results[0]
	}
	submitInstallResult := func(installUUID string, exitCode int) {
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
				"orbit_node_key": %q,
				"install_uuid": %q,
				"install_script_exit_code": %d,
				"install_script_output": "install output"
			}`, *host.OrbitNodeKey, installUUID, exitCode),
		), http.StatusNoContent)
	}
	getInstallResults := func() []*fleet.HostSoftwareInstallerResult {
		var results []*fleet.HostSoftwareInstallerResult
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &results, `
				SELECT id, execution_id, host_id, software_installer_id, self_service,
					install_script_exit_code, attempt_number
				FROM host_software_installs
				WHERE host_id = ? AND software_installer_id = ? AND policy_id IS NULL
				ORDER BY id ASC
			`, host.ID, installerID)
		})
		return results
	}
	countActivities := func() int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*)
				FROM activity_past
				WHERE activity_type = 'installed_software'
					AND JSON_EXTRACT(details, '$.host_id') = ?
					AND JSON_EXTRACT(details, '$.status') = 'failed_install'
			`, host.ID)
		})
		return count
	}

	// Trigger install from host details (admin-initiated, non-policy)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, softwareTitleID),
		nil, http.StatusAccepted)

	// Wait for install to be queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "install should be queued")
	}, 5*time.Second, 100*time.Millisecond)

	// Fail the first attempt
	pendingInstall := getPendingInstall()
	require.NotNil(t, pendingInstall)
	installUUID1 := pendingInstall.InstallUUID
	submitInstallResult(installUUID1, 1)

	// Verify attempt 1 and retry queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		results := getInstallResults()
		assert.GreaterOrEqual(t, len(results), 2, "should have completed attempt and pending retry")
		if len(results) >= 2 {
			assert.NotNil(t, results[0].AttemptNumber)
			assert.Equal(t, 1, *results[0].AttemptNumber)
			assert.NotNil(t, results[0].InstallScriptExitCode)
			assert.Nil(t, results[1].InstallScriptExitCode)
		}
	}, 5*time.Second, 100*time.Millisecond)

	// Activity created for each failure
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 1, countActivities(), "activity should be created for attempt 1")
	}, 5*time.Second, 100*time.Millisecond)

	// Wait for pending retry
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall)
		if pendingInstall != nil {
			assert.NotEqual(t, installUUID1, pendingInstall.InstallUUID)
		}
	}, 5*time.Second, 100*time.Millisecond)

	// Fail attempt 2
	pendingInstall = getPendingInstall()
	require.NotNil(t, pendingInstall)
	installUUID2 := pendingInstall.InstallUUID
	submitInstallResult(installUUID2, 1)

	// Verify attempt 2 and retry queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		results := getInstallResults()
		assert.GreaterOrEqual(t, len(results), 3)
		if len(results) >= 3 {
			assert.NotNil(t, results[1].AttemptNumber)
			assert.Equal(t, 2, *results[1].AttemptNumber)
			assert.Nil(t, results[2].InstallScriptExitCode)
		}
	}, 5*time.Second, 100*time.Millisecond)
	require.Equal(t, 2, countActivities())

	// Wait for second retry
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall)
	}, 5*time.Second, 100*time.Millisecond)

	// Fail attempt 3 (final)
	pendingInstall = getPendingInstall()
	require.NotNil(t, pendingInstall)
	installUUID3 := pendingInstall.InstallUUID
	submitInstallResult(installUUID3, 1)

	// Verify attempt 3 is final — no more retries
	results := getInstallResults()
	require.Len(t, results, 3, "should have exactly 3 completed attempts")
	require.NotNil(t, results[2].AttemptNumber)
	require.Equal(t, 3, *results[2].AttemptNumber)
	require.NotNil(t, results[2].InstallScriptExitCode)

	// All 3 attempts create activities
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 3, countActivities(), "activity should be created for each attempt")
	}, 5*time.Second, 100*time.Millisecond)

	// No more retries
	time.Sleep(2 * time.Second)
	pendingInstall = getPendingInstall()
	require.Nil(t, pendingInstall)

	// Verify exactly 3 attempts
	results = getInstallResults()
	require.Len(t, results, 3)
	require.Equal(t, 1, *results[0].AttemptNumber)
	require.Equal(t, 2, *results[1].AttemptNumber)
	require.Equal(t, 3, *results[2].AttemptNumber)

	// Manual re-install resets retry count
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, softwareTitleID),
		nil, http.StatusAccepted)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall)
	}, 5*time.Second, 100*time.Millisecond)

	// Fail the new first attempt
	pendingInstall = getPendingInstall()
	require.NotNil(t, pendingInstall)
	submitInstallResult(pendingInstall.InstallUUID, 1)

	// Verify retry is queued (fresh sequence starts at attempt 1)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		results := getInstallResults()
		// Find the last completed result
		for _, result := range slices.Backward(results) {
			if result.InstallScriptExitCode != nil {
				assert.NotNil(t, result.AttemptNumber)
				assert.Equal(t, 1, *result.AttemptNumber, "fresh sequence should start at attempt 1")
				break
			}
		}
		// Should have a pending retry
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "retry should be queued for fresh sequence")
	}, 5*time.Second, 100*time.Millisecond)

	// Succeed the retry
	pendingInstall = getPendingInstall()
	require.NotNil(t, pendingInstall)
	submitInstallResult(pendingInstall.InstallUUID, 0) // exit code 0 = success

	// No more retries after success
	time.Sleep(2 * time.Second)
	pendingInstall = getPendingInstall()
	require.Nil(t, pendingInstall)
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallersWithoutBundleIdentifier() {
	t := s.T()
	ctx := context.Background()

	// Create a host without a team
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "DummyApp", Version: "0.0.2", Source: "apps"},
	}
	// we must ingest the title with an empty bundle identifier for this
	// test to be valid
	require.Empty(t, software[0].BundleIdentifier)
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Version:       "0.0.2",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
}

func (s *integrationEnterpriseTestSuite) TestSoftwareUploadRPM() {
	t := s.T()

	// Fedora and RHEL have hosts.platform = 'rhel'.
	host := createOrbitEnrolledHost(t, "rhel", "", s.ds)

	// Upload an RPM package.
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.rpm",
		Title:             "ruby",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "rpm_packages")

	// Send a request to the host to install the RPM package.
	var installSoftwareResp installSoftwareResponse
	beforeInstallRequest := time.Now()
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusAccepted, &installSoftwareResp)
	installUUID := getLatestSoftwareInstallExecID(t, s.ds, host.ID)

	// Simulate host installing the RPM package.
	beforeInstallResult := time.Now()
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host.OrbitNodeKey, installUUID)),
		http.StatusNoContent,
	)

	var resp getSoftwareInstallResultsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &resp)
	assert.Equal(t, host.ID, resp.Results.HostID)
	assert.Equal(t, installUUID, resp.Results.InstallUUID)
	assert.Equal(t, fleet.SoftwareInstallFailed, resp.Results.Status)
	assert.NotNil(t, resp.Results.PreInstallQueryOutput)
	assert.Equal(t, fleet.SoftwareInstallerQuerySuccessCopy, *resp.Results.PreInstallQueryOutput)
	assert.NotNil(t, resp.Results.Output)
	assert.Equal(t, fmt.Sprintf(fleet.SoftwareInstallerInstallFailCopy, "failed"), *resp.Results.Output)
	assert.Empty(t, resp.Results.PostInstallScriptOutput)
	assert.Less(t, beforeInstallRequest, resp.Results.CreatedAt)
	assert.Greater(t, time.Now(), resp.Results.CreatedAt)
	// created_at comes from the installed software activity, which is created after the result
	assert.Less(t, beforeInstallResult, resp.Results.CreatedAt)

	wantAct := fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload.Title,
		SoftwarePackage: payload.Filename,
		InstallUUID:     installUUID,
		Status:          string(fleet.SoftwareInstallFailed),
		Source:          new("rpm_packages"),
		HashSHA256:      resp.Results.HashSHA256,
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)
}

func (s *integrationEnterpriseTestSuite) TestAutomaticPolicies() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	// Upload dummy_installer.pkg to team1 without automatic policy.
	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some pkg install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team1.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	// Check no policies were created.
	ts := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Empty(t, ts.Policies)
	require.Empty(t, ts.InheritedPolicies)

	// Delete and try again with automatic policy turned on.
	pkgTitleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", pkgTitleID), nil, http.StatusNoContent,
		"team_id", fmt.Sprintf("%d", team1.ID))

	// Upload dummy_installer.pkg to team1 with automatic policy.
	pkgPayload = &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some pkg install script 2",
		Filename:         "dummy_installer.pkg",
		TeamID:           &team1.ID,
		AutomaticInstall: true,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	pkgTitleID = getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	respTitle := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d?team_id=%d", pkgTitleID, team1.ID), listSoftwareTitlesRequest{}, http.StatusOK, &respTitle)
	require.NotNil(t, respTitle.SoftwareTitle)
	require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
	require.Len(t, respTitle.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies, 1)
	require.Equal(t, "[Install software] DummyApp (pkg)", respTitle.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies[0].Name)

	// check that we created an activity for the policy creation
	wantAct := fleet.ActivityTypeCreatedPolicy{
		ID:   respTitle.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies[0].ID,
		Name: respTitle.SoftwareTitle.SoftwarePackage.AutomaticInstallPolicies[0].Name,
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	// Check a policy was created on team1.
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	require.Empty(t, ts.InheritedPolicies)
	require.Equal(t, "[Install software] DummyApp (pkg)", ts.Policies[0].Name)

	// Upload dummy_installer.pkg to team2 with automatic policy.
	pkgPayload = &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some pkg install script 3",
		Filename:         "dummy_installer.pkg",
		TeamID:           &team2.ID,
		AutomaticInstall: true,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	// Check a policy was created on team2.
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team2.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	require.Empty(t, ts.InheritedPolicies)
	require.Equal(t, "[Install software] DummyApp (pkg)", ts.Policies[0].Name)

	// Upload ruby.deb to team1 with automatic policy.
	payloadRubyDEB := &fleet.UploadSoftwareInstallerPayload{
		Filename:         "ruby.deb",
		TeamID:           &team1.ID,
		AutomaticInstall: true,
	}
	s.uploadSoftwareInstaller(t, payloadRubyDEB, http.StatusOK, "")

	payloadRubyRPM := &fleet.UploadSoftwareInstallerPayload{
		Filename:         "ruby.rpm",
		TeamID:           &team1.ID,
		AutomaticInstall: true,
	}
	s.uploadSoftwareInstaller(t, payloadRubyRPM, http.StatusOK, "")

	// Upload fleet-osquery.msi to team1 with automatic policy.
	fleetOsqueryPayload := &fleet.UploadSoftwareInstallerPayload{
		Filename:         "fleet-osquery.msi",
		TeamID:           &team1.ID,
		AutomaticInstall: true,
	}
	s.uploadSoftwareInstaller(t, fleetOsqueryPayload, http.StatusOK, "")

	// Check policies were created on team1.
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 4)
	require.Empty(t, ts.InheritedPolicies)
	require.Equal(t, "[Install software] ruby (deb)", ts.Policies[1].Name)
	require.Equal(t, "[Install software] ruby (rpm)", ts.Policies[2].Name)
	require.Equal(t, "[Install software] Fleet osquery (msi)", ts.Policies[3].Name)
}

// test for #26668
func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerOrbitDownloadFailure() {
	t := s.T()

	// upload an installer
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.deb",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

	// create an orbit host
	host := createOrbitEnrolledHost(t, "linux", "orbit-host", s.ds)
	// create a software installation request, is immediately activated
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)

	// should be listed in upcoming activities
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	swInstallExecID := listUpcomingAct.Activities[0].UUID

	// add a script execution request, not activated yet
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo 'hello'"}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	// software install exec ID is returned in orbit notifications, but not script exec
	var orbitResp fleet.OrbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitResp)
	require.Equal(t, []string{swInstallExecID}, orbitResp.Notifications.PendingSoftwareInstallerIDs)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// simulate a failed orbit download of the installer - it sends an empty
	// result (i.e. no result)
	s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
		OrbitNodeKey: *host.OrbitNodeKey,
		HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
			InstallUUID: swInstallExecID,
		},
	}, http.StatusNoContent)

	// software install exec ID is still returned in orbit notifications, but not script exec
	orbitResp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitResp)
	require.Equal(t, []string{swInstallExecID}, orbitResp.Notifications.PendingSoftwareInstallerIDs)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// installer is still listed as first upcoming activity
	listUpcomingAct = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 2)
	require.Equal(t, swInstallExecID, listUpcomingAct.Activities[0].UUID)
	scriptExecID := listUpcomingAct.Activities[1].UUID

	// record an actual result for the installer
	s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
		OrbitNodeKey: *host.OrbitNodeKey,
		HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
			InstallUUID:           swInstallExecID,
			InstallScriptExitCode: new(int(0)),
			InstallScriptOutput:   new("hello"),
		},
	}, http.StatusNoContent)

	// now the script exec ID is returned in notifications
	orbitResp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitResp)
	require.Equal(t, []string{scriptExecID}, orbitResp.Notifications.PendingScriptExecutionIDs)
	require.Empty(t, orbitResp.Notifications.PendingSoftwareInstallerIDs)

	// only script is now returned in upcoming
	listUpcomingAct = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	require.Equal(t, scriptExecID, listUpcomingAct.Activities[0].UUID)

	// past activity is created for the software install
	installRes, err := s.ds.GetSoftwareInstallResults(context.Background(), swInstallExecID)
	require.NoError(t, err)
	wantAct := fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   "ruby",
		SoftwarePackage: payload.Filename,
		InstallUUID:     swInstallExecID,
		Status:          string(fleet.SoftwareInstalled),
		Source:          new("deb_packages"),
		HashSHA256:      installRes.HashSHA256,
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)
}

// TestScriptPackageUploadValidation tests that script packages (.sh and .ps1)
// persist post_install_script, uninstall_script, and pre_install_query when
// uploaded via the API. The install_script field stays the uploaded file's
// contents (file-driven) and any provided install_script value is ignored.
func (s *integrationEnterpriseTestSuite) TestScriptPackageUploadValidation() {
	t := s.T()

	tmpDir := t.TempDir()

	shScriptPath := filepath.Join(tmpDir, "test-script.sh")
	shScriptContent := []byte("#!/bin/bash\necho 'Installing...'\n")
	err := os.WriteFile(shScriptPath, shScriptContent, 0o644)
	require.NoError(t, err)

	ps1ScriptPath := filepath.Join(tmpDir, "test-script.ps1")
	ps1ScriptContent := []byte("Write-Host 'Installing...'\n")
	err = os.WriteFile(ps1ScriptPath, ps1ScriptContent, 0o644)
	require.NoError(t, err)

	pyScriptPath := filepath.Join(tmpDir, "test-script.py")
	pyScriptContent := []byte("#!/usr/bin/env python3\nprint('Installing...')\n")
	err = os.WriteFile(pyScriptPath, pyScriptContent, 0o644)
	require.NoError(t, err)

	t.Run("sh script package preserves advanced options", func(t *testing.T) {
		installerFile, err := fleet.NewKeepFileReader(shScriptPath)
		require.NoError(t, err)
		defer installerFile.Close()

		// install_script is ignored (the file is the install script); the rest persist.
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:     "echo 'install_script is ignored'",
			PostInstallScript: "echo 'post-install'",
			UninstallScript:   "echo 'uninstall'",
			PreInstallQuery:   "SELECT 1",
			Filename:          "test-script.sh",
			InstallerFile:     installerFile,
		}

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		var listResp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "team_id", "0", "available_for_install", "true")

		var found bool
		var titleID uint
		for _, sw := range listResp.SoftwareTitles {
			if sw.SoftwarePackage != nil && sw.SoftwarePackage.Name == "test-script.sh" {
				found = true
				titleID = sw.ID
				break
			}
		}
		require.True(t, found, "Script package should be created")

		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")

		require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
		installer := titleResp.SoftwareTitle.SoftwarePackage

		require.Equal(t, string(shScriptContent), installer.InstallScript, ".sh script package should have install_script from file contents")
		require.NotEqual(t, "echo 'install_script is ignored'", installer.InstallScript, "user-provided install_script should be overwritten")
		require.Equal(t, "echo 'post-install'", installer.PostInstallScript, ".sh script package should persist post_install_script")
		require.Equal(t, "echo 'uninstall'", installer.UninstallScript, ".sh script package should persist uninstall_script")
		require.Equal(t, "SELECT 1", installer.PreInstallQuery, ".sh script package should persist pre_install_query")

		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, 204, "team_id", "0")
	})

	t.Run("ps1 script package preserves advanced options", func(t *testing.T) {
		installerFile, err := fleet.NewKeepFileReader(ps1ScriptPath)
		require.NoError(t, err)
		defer installerFile.Close()

		// install_script is ignored (the file is the install script); the rest persist.
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:     "Write-Host 'install_script is ignored'",
			PostInstallScript: "Write-Host 'post-install'",
			UninstallScript:   "Write-Host 'uninstall'",
			PreInstallQuery:   "SELECT 1",
			Filename:          "test-script.ps1",
			InstallerFile:     installerFile,
		}

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		var listResp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "team_id", "0", "available_for_install", "true")

		var found bool
		var titleID uint
		for _, sw := range listResp.SoftwareTitles {
			if sw.SoftwarePackage != nil && sw.SoftwarePackage.Name == "test-script.ps1" {
				found = true
				titleID = sw.ID
				break
			}
		}
		require.True(t, found, "Script package should be created")

		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")

		require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
		installer := titleResp.SoftwareTitle.SoftwarePackage

		require.Equal(t, string(ps1ScriptContent), installer.InstallScript, ".ps1 script package should have install_script from file contents")
		require.NotEqual(t, "Write-Host 'install_script is ignored'", installer.InstallScript, "user-provided install_script should be overwritten")
		require.Equal(t, "Write-Host 'post-install'", installer.PostInstallScript, ".ps1 script package should persist post_install_script")
		require.Equal(t, "Write-Host 'uninstall'", installer.UninstallScript, ".ps1 script package should persist uninstall_script")
		require.Equal(t, "SELECT 1", installer.PreInstallQuery, ".ps1 script package should persist pre_install_query")

		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, 204, "team_id", "0")
	})

	t.Run("py script package preserves advanced options", func(t *testing.T) {
		installerFile, err := fleet.NewKeepFileReader(pyScriptPath)
		require.NoError(t, err)
		defer installerFile.Close()

		// install_script is ignored (the file is the install script); the rest persist.
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:     "print('install_script is ignored')",
			PostInstallScript: "echo 'post-install'",
			UninstallScript:   "echo 'uninstall'",
			PreInstallQuery:   "SELECT 1",
			Filename:          "test-script.py",
			InstallerFile:     installerFile,
		}

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		var listResp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "team_id", "0", "available_for_install", "true")

		var found bool
		var titleID uint
		for _, sw := range listResp.SoftwareTitles {
			if sw.SoftwarePackage != nil && sw.SoftwarePackage.Name == "test-script.py" {
				found = true
				titleID = sw.ID
				break
			}
		}
		require.True(t, found, "Script package should be created")

		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id", "0")

		require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
		installer := titleResp.SoftwareTitle.SoftwarePackage

		require.Equal(t, "py_packages", titleResp.SoftwareTitle.Source, ".py script package should have py_packages source")
		require.Equal(t, string(pyScriptContent), installer.InstallScript, ".py script package should have install_script from file contents")
		require.NotEqual(t, "print('install_script is ignored')", installer.InstallScript, "user-provided install_script should be overwritten")
		require.Equal(t, "echo 'post-install'", installer.PostInstallScript, ".py script package should persist post_install_script")
		require.Equal(t, "echo 'uninstall'", installer.UninstallScript, ".py script package should persist uninstall_script")
		require.Equal(t, "SELECT 1", installer.PreInstallQuery, ".py script package should persist pre_install_query")

		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, 204, "team_id", "0")
	})
}

func (s *integrationEnterpriseTestSuite) TestInHouseAppCRUD() {
	t := s.T()
	ctx := context.Background()

	t.Run("upload update delete in-house app", func(t *testing.T) {
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
			Name: t.Name(),
		}, http.StatusOK, &createTeamResp)
		require.NotZero(t, createTeamResp.Team.ID)

		var labelResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name:  "iha_label",
			Query: "select 1"}, http.StatusOK, &labelResp)
		require.NotZero(t, labelResp.Label.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:      &createTeamResp.Team.ID,
			Filename:    "ipa_test2.ipa",
			Version:     "1.0.0",
			StorageID:   uuid.New().String(),
			SelfService: true,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		var titleID uint
		var installerID uint
		var titleName string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			require.NoError(t, sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM in_house_apps WHERE filename = ?`, payload.Filename))
			require.NoError(t, sqlx.GetContext(ctx, q, &installerID, `SELECT title_id FROM in_house_apps WHERE filename = ?`, payload.Filename))
			require.NoError(t, sqlx.GetContext(ctx, q, &titleName, `SELECT name FROM software_titles WHERE id = ?`, titleID))
			return nil
		})
		require.Equal(t, "ipa_test", titleName) // Title name should be different than filename

		// check activity
		activityData := fmt.Sprintf(
			`{"software_title": "ipa_test", "software_package": "ipa_test2.ipa", "team_name": "%s", "team_id": %d, "fleet_name": "%s", "fleet_id": %d, "self_service": true, "software_title_id": %d}`,
			createTeamResp.Team.Name,
			createTeamResp.Team.ID,
			createTeamResp.Team.Name,
			createTeamResp.Team.ID,
			titleID+1, // iOS title is created first and returned, so the latest title is the iPadOS one
		)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

		// upload again fails
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "already exists")

		// patch the in house app to change labels, categories
		body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":            {fmt.Sprintf("%d", createTeamResp.Team.ID)},
			"labels_exclude_any": {"iha_label"},
			"categories":         {"Productivity", "Browsers"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)
		expectedPayload := *payload
		expectedPayload.LabelsExcludeAny = []string{labelResp.Label.Name}
		expectedPayload.Categories = []string{"🖥️ Productivity", "🌎 Browsers"}

		meta, err := s.ds.GetInHouseAppMetadataByTeamAndTitleID(context.Background(), &createTeamResp.Team.ID, installerID)
		require.NoError(t, err)
		require.Equal(t, expectedPayload.Filename, meta.Name)
		require.Equal(t, expectedPayload.LabelsExcludeAny[0], meta.LabelsExcludeAny[0].LabelName)
		require.Equal(t, expectedPayload.SelfService, meta.SelfService)
		require.ElementsMatch(t, expectedPayload.Categories, meta.Categories)

		// patch only the categories (test fix for https://github.com/fleetdm/fleet/issues/36069)
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":    {fmt.Sprintf("%d", createTeamResp.Team.ID)},
			"categories": {"Browsers"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)
		expectedPayload.Categories = []string{"🌎 Browsers"}

		meta, err = s.ds.GetInHouseAppMetadataByTeamAndTitleID(context.Background(), &createTeamResp.Team.ID, installerID)
		require.NoError(t, err)
		require.Equal(t, expectedPayload.Filename, meta.Name)
		require.Equal(t, expectedPayload.LabelsExcludeAny[0], meta.LabelsExcludeAny[0].LabelName)
		require.Equal(t, expectedPayload.SelfService, meta.SelfService)
		require.ElementsMatch(t, expectedPayload.Categories, meta.Categories)

		// patch only the self-service
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":      {fmt.Sprintf("%d", createTeamResp.Team.ID)},
			"self_service": {"false"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)
		expectedPayload.SelfService = false

		meta, err = s.ds.GetInHouseAppMetadataByTeamAndTitleID(context.Background(), &createTeamResp.Team.ID, installerID)
		require.NoError(t, err)
		require.Equal(t, expectedPayload.Filename, meta.Name)
		require.Equal(t, expectedPayload.LabelsExcludeAny[0], meta.LabelsExcludeAny[0].LabelName)
		require.Equal(t, expectedPayload.SelfService, meta.SelfService)
		require.ElementsMatch(t, expectedPayload.Categories, meta.Categories)

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", fmt.Sprintf("%d", *payload.TeamID))

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(),
			fmt.Sprintf(`{"labels_exclude_any":  [{"id": %d, "name": "%s"}], "software_title": "ipa_test", "software_package": "ipa_test2.ipa", "software_icon_url": null, "team_name": "%s", "team_id": %d, "fleet_name": "%s", "fleet_id": %d, "self_service": false}`,
				labelResp.Label.ID, labelResp.Label.Name, createTeamResp.Team.Name, createTeamResp.Team.ID, createTeamResp.Team.Name, createTeamResp.Team.ID), 0)

		// download the installer, not found anymore
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusNotFound, "team_id", fmt.Sprintf("%d", *payload.TeamID))
	})

	t.Run("managed app configuration", func(t *testing.T) {
		var teamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name()}, http.StatusOK, &teamResp)

		const validPlist = `<dict><key>K</key><string>v</string></dict>`
		const validPlist2 = `<dict><key>K2</key><string>v2</string></dict>`

		// Upload .ipa with configuration.
		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:        &teamResp.Team.ID,
			Filename:      "ipa_test2.ipa",
			Version:       "1.0.0",
			StorageID:     uuid.New().String(),
			Configuration: []byte(validPlist),
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		var titleID, installerID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			require.NoError(t, sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM in_house_apps WHERE filename = ? AND platform = 'ios' AND team_id = ?`, payload.Filename, teamResp.Team.ID))
			require.NoError(t, sqlx.GetContext(ctx, q, &installerID, `SELECT id FROM in_house_apps WHERE filename = ? AND platform = 'ios' AND team_id = ?`, payload.Filename, teamResp.Team.ID))
			return nil
		})

		// Stored configuration matches what was sent.
		storedCfg, err := s.ds.GetInHouseAppConfiguration(ctx, installerID)
		require.NoError(t, err)
		require.Equal(t, []byte(validPlist), storedCfg)

		// Update with a new configuration.
		body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":       {fmt.Sprintf("%d", teamResp.Team.ID)},
			"configuration": {validPlist2},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

		storedCfg, err = s.ds.GetInHouseAppConfiguration(ctx, installerID)
		require.NoError(t, err)
		require.Equal(t, []byte(validPlist2), storedCfg)

		// GET title returns the iOS configuration as a JSON string of plist; unmarshal to recover.
		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID),
			&getSoftwareTitleRequest{ID: titleID, TeamID: &teamResp.Team.ID},
			http.StatusOK, &titleResp, "fleet_id", fmt.Sprint(teamResp.Team.ID))
		require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
		var gotPlist string
		require.NoError(t, json.Unmarshal(titleResp.SoftwareTitle.SoftwarePackage.Configuration, &gotPlist))
		require.Equal(t, validPlist2, gotPlist)

		// Update with empty configuration → cleared.
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":       {fmt.Sprintf("%d", teamResp.Team.ID)},
			"configuration": {""},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusOK, headers)

		_, err = s.ds.GetInHouseAppConfiguration(ctx, installerID)
		require.True(t, fleet.IsNotFound(err), "expected configuration cleared, got %v", err)

		// Update with invalid plist → 422.
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":       {fmt.Sprintf("%d", teamResp.Team.ID)},
			"configuration": {"not actually a plist"},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusUnprocessableEntity, headers)

		// Update with disallowed Fleet variable → 422.
		body, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
			"team_id":       {fmt.Sprintf("%d", teamResp.Team.ID)},
			"configuration": {`<dict><key>K</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`},
		})
		s.DoRawWithHeaders("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID), body.Bytes(), http.StatusUnprocessableEntity, headers)

		// Upload with invalid plist → 422.
		invalidPayload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:        &teamResp.Team.ID,
			Filename:      "ipa_test2.ipa",
			Version:       "1.0.0",
			StorageID:     uuid.New().String(),
			Configuration: []byte("not actually a plist"),
		}
		s.uploadSoftwareInstaller(t, invalidPayload, http.StatusUnprocessableEntity, "invalid plist")
	})
}

// TestSoftwareInstallerUploadPayloadTooLarge verifies that uploading (or updating) a
// software installer that exceeds the configured max size returns HTTP 400 with
// a user-friendly error message, not a generic HTTP 413.
//
// In production, serve.go wraps the request body with http.MaxBytesReader. The
// DecodeRequest methods in software_installers.go catch the resulting
// MaxBytesError and return a fleet.BadRequestError. The fix being tested here
// ensures that the BadRequestError does NOT wrap the original MaxBytesError in
// InternalErr, which would cause MakeDecoder to convert it into a generic 413.
func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerUploadPayloadTooLarge() {
	t := s.T()

	// Wrap the test server's handler with the same MaxBytesReader middleware
	// that serve.go applies in production, but with a tiny limit so we can
	// trigger it with a small test file.
	const maxSize int64 = 512 // 512 B
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/fleet/software/package")) ||
			(r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/package") && strings.Contains(r.URL.Path, "/fleet/software/titles/")) {
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			r = r.WithContext(installersize.NewContext(r.Context(), maxSize))
		}
		s.server.Config.Handler.ServeHTTP(w, r)
	})
	wrappedServer := httptest.NewServer(wrappedHandler)
	defer wrappedServer.Close()

	// doMultipartUpload builds a multipart request with a file exceeding the
	// limit and sends it to the wrapped server.
	doMultipartUpload := func(t *testing.T, method, path string) *http.Response {
		t.Helper()

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("software", "test_installer.pkg")
		require.NoError(t, err)
		// Write content that, together with multipart overhead, exceeds maxSize.
		_, err = fw.Write(bytes.Repeat([]byte("x"), int(maxSize)+1))
		require.NoError(t, err)
		require.NoError(t, w.WriteField("install_script", "echo install"))
		w.Close()

		req, err := http.NewRequest(method, wrappedServer.URL+path, &b)
		require.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		return resp
	}

	t.Run("upload", func(t *testing.T) {
		resp := doMultipartUpload(t, "POST", "/api/latest/fleet/software/package")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		errMsg := extractServerErrorText(resp.Body)
		require.Contains(t, errMsg, "The maximum file size is")
	})

	t.Run("update", func(t *testing.T) {
		// First upload a small installer via the normal (unwrapped) server so
		// we have a valid title to update.
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "echo install",
			Filename:      "ruby.deb",
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// Find the title ID of the uploaded installer.
		var listResp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "available_for_install", "true")
		titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

		resp := doMultipartUpload(t, "PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package", titleID))
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		errMsg := extractServerErrorText(resp.Body)
		require.Contains(t, errMsg, "The maximum file size is")
	})
}

// TestSoftwareMultiplePackagesInstallPrecedence verifies install-time first-added precedence when a
// title holds multiple label-scoped packages: a host matching more than one package installs the
// first-added one, while a host matching only a later package installs that one (correct scoping).
func (s *integrationEnterpriseTestSuite) TestSoftwareMultiplePackagesInstallPrecedence() {
	t := s.T()
	ctx := context.Background()

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name()}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID
	user := s.users["admin1@example.com"]

	labelA, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "labelA_" + t.Name()})
	require.NoError(t, err)
	labelB, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "labelB_" + t.Name()})
	require.NoError(t, err)

	// Two packages under one title (same bundle id, different content hash), each scoped to one label.
	newPkg := func(storage, filename, version string, label *fleet.Label) uint {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("hello-"+storage), t.TempDir)
		require.NoError(t, err)
		id, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			UninstallScript:  "uninstall",
			InstallerFile:    tfr,
			StorageID:        storage,
			Filename:         filename,
			Title:            "MultiPkgApp",
			Version:          version,
			Source:           "deb_packages",
			Extension:        "deb",
			BundleIdentifier: "com.example.multipkgapp",
			UserID:           user.ID,
			TeamID:           &teamID,
			Platform:         "linux",
			ValidatedLabels: &fleet.LabelIdentsWithScope{
				LabelScope: fleet.LabelScopeIncludeAny,
				ByName:     map[string]fleet.LabelIdent{label.Name: {LabelName: label.Name, LabelID: label.ID}},
			},
		})
		require.NoError(t, err)
		return id
	}
	installerA := newPkg("storage-a", "pkgA.deb", "1.0", labelA)
	installerB := newPkg("storage-b", "pkgB.deb", "2.0", labelB)
	require.Less(t, installerA, installerB)

	titleID := getSoftwareTitleID(t, s.ds, "MultiPkgApp", "deb_packages")

	newLinuxHost := func(suffix string, labelIDs []uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(t.Name() + suffix + uuid.New().String()),
			NodeKey:         new(t.Name() + suffix + uuid.New().String()),
			Hostname:        fmt.Sprintf("%s-%s.local", t.Name(), suffix),
			Platform:        "ubuntu",
		})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&teamID, []uint{h.ID})))
		if len(labelIDs) > 0 {
			require.NoError(t, s.ds.AddLabelsToHost(ctx, h.ID, labelIDs))
		}
		h.LabelUpdatedAt = time.Now()
		require.NoError(t, s.ds.UpdateHost(ctx, h))
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	// queuedInstallerID returns the software_installer_id queued for the host's pending install.
	queuedInstallerID := func(hostID uint) uint {
		var ids []uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &ids, `
				SELECT siua.software_installer_id
				FROM software_install_upcoming_activities siua
				JOIN upcoming_activities ua ON ua.id = siua.upcoming_activity_id
				WHERE ua.host_id = ?`, hostID)
		})
		require.Len(t, ids, 1)
		return ids[0]
	}

	// Host in BOTH labels matches pkgA and pkgB → first-added (pkgA) installs.
	hostBoth := newLinuxHost("both", []uint{labelA.ID, labelB.ID})
	var resp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostBoth.ID, titleID), nil, http.StatusAccepted, &resp)
	require.Equal(t, installerA, queuedInstallerID(hostBoth.ID))

	var installAUUID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installAUUID, `
			SELECT execution_id FROM host_software_installs
			WHERE host_id = ? AND software_installer_id = ?
			ORDER BY id DESC LIMIT 1`, hostBoth.ID, installerA)
	})
	_, err = s.ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                hostBoth.ID,
		InstallUUID:           installAUUID,
		InstallScriptExitCode: new(int(0)),
	}, nil)
	require.NoError(t, err)

	installBUUID, err := s.ds.InsertSoftwareInstallRequest(ctx, hostBoth.ID, installerB, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	_, err = s.ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                hostBoth.ID,
		InstallUUID:           installBUUID,
		InstallScriptExitCode: new(int(0)),
	}, nil)
	require.NoError(t, err)

	var hostSoftwareResp getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", hostBoth.ID), nil, http.StatusOK, &hostSoftwareResp,
		"include_available_for_install", "true")
	var title *fleet.HostSoftwareWithInstaller
	for _, software := range hostSoftwareResp.Software {
		if software.ID == titleID {
			title = software
			break
		}
	}
	require.NotNil(t, title)
	require.NotNil(t, title.SoftwarePackage)
	require.Equal(t, "pkgA.deb", title.SoftwarePackage.Name)
	require.Equal(t, "1.0", title.SoftwarePackage.Version)
	require.Equal(t, new(fleet.SoftwareInstalled), title.Status)
	require.NotNil(t, title.SoftwarePackage.LastInstall)
	require.Equal(t, installAUUID, title.SoftwarePackage.LastInstall.InstallUUID)

	// Host in only labelB matches only pkgB → pkgB installs (never the first-added pkgA).
	hostSecond := newLinuxHost("second", []uint{labelB.ID})
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostSecond.ID, titleID), nil, http.StatusAccepted, &resp)
	require.Equal(t, installerB, queuedInstallerID(hostSecond.ID))

	// Host in neither label is in scope for no package → install is rejected.
	hostNeither := newLinuxHost("neither", nil)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostNeither.ID, titleID), nil, http.StatusBadRequest, &resp)
}

func (s *integrationEnterpriseTestSuite) TestScriptPackageFleetVariables() {
	t := s.T()
	ctx := context.Background()

	tmpDir := t.TempDir()
	writeScript := func(name, contents string) string {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
		return path
	}

	const unsupportedVarErrMsg = "Fleet variable $FLEET_VAR_NONEXISTENT is not supported in scripts."

	// a script package's install script is the uploaded file itself, so an
	// unsupported variable in the file is rejected at upload
	badFile, err := fleet.NewKeepFileReader(writeScript("vars-pkg-bad.sh", "#!/bin/sh\necho $FLEET_VAR_NONEXISTENT\n"))
	require.NoError(t, err)
	defer badFile.Close()
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename:      "vars-pkg.sh",
		InstallerFile: badFile,
	}, http.StatusUnprocessableEntity, unsupportedVarErrMsg)

	// supported variables in the file are accepted
	goodFile, err := fleet.NewKeepFileReader(writeScript("vars-pkg-good.sh", "#!/bin/sh\necho $FLEET_VAR_HOST_UUID\n"))
	require.NoError(t, err)
	defer goodFile.Close()
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename:      "vars-pkg.sh",
		InstallerFile: goodFile,
	}, http.StatusOK, "")

	var listResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listResp, "team_id", "0", "available_for_install", "true")
	var titleID uint
	for _, sw := range listResp.SoftwareTitles {
		if sw.SoftwarePackage != nil && sw.SoftwarePackage.Name == "vars-pkg.sh" {
			titleID = sw.ID
		}
	}
	require.NotZero(t, titleID)

	// replacing the package file is rejected too: the new file's contents
	// become the install script
	badUpdateFile, err := fleet.NewKeepFileReader(writeScript("vars-pkg-update-bad.sh", "#!/bin/sh\necho hi\necho $FLEET_VAR_NONEXISTENT\n"))
	require.NoError(t, err)
	defer badUpdateFile.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		Filename:      "vars-pkg.sh",
		InstallerFile: badUpdateFile,
	}, http.StatusUnprocessableEntity, unsupportedVarErrMsg)

	// a replacement file with a supported variable is accepted and stored unexpanded
	const updatedContents = "#!/bin/sh\necho $FLEET_VAR_HOST_HARDWARE_SERIAL\n"
	goodUpdateFile, err := fleet.NewKeepFileReader(writeScript("vars-pkg-update-good.sh", updatedContents))
	require.NoError(t, err)
	defer goodUpdateFile.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		Filename:      "vars-pkg.sh",
		InstallerFile: goodUpdateFile,
	}, http.StatusOK, "")

	meta, err := s.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, titleID, true)
	require.NoError(t, err)
	require.Equal(t, updatedContents, meta.InstallScript)
}

func (s *integrationEnterpriseTestSuite) TestStagedUploadUnavailable() {
	t := s.T()
	res := s.Do("POST", "/api/latest/fleet/staged_upload",
		createStagedUploadRequest{Target: fleet.StagedUploadTargetSoftwarePackage, Size: 1}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "Direct upload isn't available")

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{StagedUploadID: uuid.NewString(), Filename: "ruby.deb"},
		http.StatusBadRequest, "Direct upload isn't available")
}
