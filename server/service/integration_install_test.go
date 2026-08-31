package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	software_mock "github.com/fleetdm/fleet/v4/server/mock/software"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsInstall(t *testing.T) {
	testingSuite := new(integrationInstallTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationInstallTestSuite struct {
	withServer
	suite.Suite
	softwareInstallStore *software_mock.SoftwareInstallerStore
}

func (s *integrationInstallTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationInstallTestSuite")

	// Create a mock S3 software install store
	softwareInstallStore := &software_mock.SoftwareInstallerStore{}
	s.softwareInstallStore = softwareInstallStore

	fleetConfig := config.TestConfig()
	signer, _ := rsa.GenerateKey(rand.Reader, 2048)
	fleetConfig.S3.SoftwareInstallersCloudFrontSigner = signer
	installConfig := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Logger:               slog.New(slog.NewTextHandler(os.Stdout, nil)),
		EnableCachedDS:       true,
		SoftwareInstallStore: softwareInstallStore,
		FleetConfig:          &fleetConfig,
	}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		installConfig.Logger = slog.New(slog.DiscardHandler)
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &installConfig)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedTokens = make(map[string]string)
}

func (s *integrationInstallTestSuite) TearDownTest() {
	s.withServer.commonTearDownTest(s.T())
}

// TestSoftwareInstallerSignedURL tests that the software installer signed URL is returned.
// We test using both mock and real fleet.SoftwareInstallerStore.Sign functions.
func (s *integrationInstallTestSuite) TestSoftwareInstallerSignedURL() {
	t := s.T()

	openFile := func(name string) *os.File {
		f, err := os.Open(filepath.Join("testdata", "software-installers", name))
		require.NoError(t, err)
		return f
	}

	filename := "ruby.deb"
	var expectBytes []byte
	var expectLen int
	f := openFile(filename)
	st, err := f.Stat()
	require.NoError(t, err)
	expectLen = int(st.Size())
	require.Equal(t, expectLen, 11340)
	expectBytes = make([]byte, expectLen)
	n, err := f.Read(expectBytes)
	require.NoError(t, err)
	require.Equal(t, n, expectLen)
	f.Close()

	// Set up mocks
	var myInstallerID string
	s.softwareInstallStore.ExistsFunc = func(ctx context.Context, installerID string) (bool, error) {
		return installerID == myInstallerID, nil
	}
	s.softwareInstallStore.PutFunc = func(ctx context.Context, installerID string, content io.ReadSeeker) error {
		myInstallerID = installerID
		return nil
	}
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
		return "https://example.com/signed", nil
	}

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
		Filename:          filename,
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
	var id uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, payload.TeamID, payload.Filename)
	})
	require.NotZero(t, id)

	meta, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), id)
	require.NoError(t, err)
	titleID := *meta.TitleID

	// create an orbit host, assign to team
	hostInTeam := createOrbitEnrolledHost(t, "linux", "orbit-host-team", s.ds)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&createTeamResp.Team.ID, []uint{hostInTeam.ID})))

	// Create a software installation request
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, titleID), installSoftwareRequest{},
		http.StatusAccepted)

	// Get the InstallerUUID
	installUUID := getLatestSoftwareInstallExecID(t, s.ds, hostInTeam.ID)

	// Fetch installer details
	var orbitSoftwareResp fleet.OrbitGetSoftwareInstallResponse
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  installUUID,
		OrbitNodeKey: *hostInTeam.OrbitNodeKey,
	}, http.StatusOK, &orbitSoftwareResp)
	assert.Equal(t, meta.InstallerID, orbitSoftwareResp.InstallerID)
	require.NotNil(t, orbitSoftwareResp.SoftwareInstallerURL)
	assert.Equal(t, "https://example.com/signed", orbitSoftwareResp.SoftwareInstallerURL.URL)
	require.Equal(t, filename, orbitSoftwareResp.SoftwareInstallerURL.Filename)

	// Error in signing -- we simply don't return the URL
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
		return "", errors.New("error signing")
	}
	orbitSoftwareResp = fleet.OrbitGetSoftwareInstallResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  installUUID,
		OrbitNodeKey: *hostInTeam.OrbitNodeKey,
	}, http.StatusOK, &orbitSoftwareResp)
	assert.Equal(t, meta.InstallerID, orbitSoftwareResp.InstallerID)
	assert.Nil(t, orbitSoftwareResp.SoftwareInstallerURL)

	// Now test with the real sign function
	signer, _ := rsa.GenerateKey(rand.Reader, 2048)

	s3Config := config.S3Config{
		SoftwareInstallersCloudFrontURL:                   "https://example.cloudfront.net",
		SoftwareInstallersCloudFrontURLSigningPublicKeyID: "ABC123XYZ",
		SoftwareInstallersCloudFrontSigner:                signer,
	}
	s3Store, err := s3.NewTestSoftwareInstallerStore(s3Config)
	require.NoError(t, err)
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
		return s3Store.Sign(ctx, fileID, fleet.SoftwareInstallerSignedURLExpiry)
	}
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  installUUID,
		OrbitNodeKey: *hostInTeam.OrbitNodeKey,
	}, http.StatusOK, &orbitSoftwareResp)
	assert.Equal(t, meta.InstallerID, orbitSoftwareResp.InstallerID)
	require.NotNil(t, orbitSoftwareResp.SoftwareInstallerURL)
	assert.True(t,
		strings.HasPrefix(orbitSoftwareResp.SoftwareInstallerURL.URL,
			s3Config.SoftwareInstallersCloudFrontURL+"/software-installers/"+payload.StorageID+"?Expires="),
		orbitSoftwareResp.SoftwareInstallerURL.URL)
	assert.Contains(t, orbitSoftwareResp.SoftwareInstallerURL.URL, "&Signature=")
	assert.Contains(t, orbitSoftwareResp.SoftwareInstallerURL.URL,
		"&Key-Pair-Id="+s3Config.SoftwareInstallersCloudFrontURLSigningPublicKeyID)
	require.Equal(t, filename, orbitSoftwareResp.SoftwareInstallerURL.Filename)

}

func getLatestSoftwareInstallExecID(t *testing.T, ds *mysql.Datastore, hostID uint) string {
	var installUUID string
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &installUUID,
			"SELECT execution_id FROM host_software_installs WHERE host_id = ? ORDER BY id desc", hostID)
	})
	return installUUID
}

// TestShScriptInstallOnDarwin tests that .sh script packages (stored as platform='linux')
// can be installed on darwin (macOS) hosts through the full HTTP API flow.
func (s *integrationInstallTestSuite) TestShScriptInstallOnDarwin() {
	t := s.T()

	filename := "test-script.sh"

	// Create a .sh script file in-memory
	tfr, err := fleet.NewTempFileReader(strings.NewReader("#!/bin/bash\necho 'hello world'\n"), t.TempDir)
	require.NoError(t, err)
	defer tfr.Close()

	// Set up mocks
	var myInstallerID string
	s.softwareInstallStore.ExistsFunc = func(ctx context.Context, installerID string) (bool, error) {
		return installerID == myInstallerID, nil
	}
	s.softwareInstallStore.PutFunc = func(ctx context.Context, installerID string, content io.ReadSeeker) error {
		myInstallerID = installerID
		return nil
	}
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
		return "https://example.com/signed-sh", nil
	}

	// Create a team
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name: t.Name(),
	}, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)

	// Upload .sh script package
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		TeamID:        &createTeamResp.Team.ID,
		Filename:      filename,
		InstallerFile: tfr,
	}, http.StatusOK, "")

	// Get the title ID from the database
	var id uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, createTeamResp.Team.ID, filename)
	})
	require.NotZero(t, id)

	meta, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "linux", meta.Platform, ".sh file should be stored with platform=linux")
	titleID := *meta.TitleID

	// Create a darwin (macOS) orbit host and assign to team
	darwinHost := createOrbitEnrolledHost(t, "darwin", "darwin-sh-host", s.ds)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&createTeamResp.Team.ID, []uint{darwinHost.ID})))

	// Install .sh on darwin should succeed
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", darwinHost.ID, titleID), installSoftwareRequest{},
		http.StatusAccepted)

	// Get the install UUID
	installUUID := getLatestSoftwareInstallExecID(t, s.ds, darwinHost.ID)

	// Fetch installer details via orbit endpoint
	var orbitSoftwareResp fleet.OrbitGetSoftwareInstallResponse
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  installUUID,
		OrbitNodeKey: *darwinHost.OrbitNodeKey,
	}, http.StatusOK, &orbitSoftwareResp)
	assert.Equal(t, meta.InstallerID, orbitSoftwareResp.InstallerID)
	require.NotNil(t, orbitSoftwareResp.SoftwareInstallerURL)
	assert.Equal(t, "https://example.com/signed-sh", orbitSoftwareResp.SoftwareInstallerURL.URL)
	require.Equal(t, filename, orbitSoftwareResp.SoftwareInstallerURL.Filename)
}

func (s *integrationInstallTestSuite) TestGetInHouseAppManifestSignedURL() {
	// Test that the signed URL is used if cloudfrontsigner is configured
	t := s.T()
	teamID := ptr.Uint(0)

	signURL := `https://example.cloudfront.net/software-installers/storage_id?Expires=1766462733&Signature=some_signature&Key-Pair-Id=ABC123XYZ`

	// Set up mocks
	var myInstallerID string
	s.softwareInstallStore.ExistsFunc = func(ctx context.Context, installerID string) (bool, error) {
		return installerID == myInstallerID, nil
	}
	s.softwareInstallStore.PutFunc = func(ctx context.Context, installerID string, content io.ReadSeeker) error {
		myInstallerID = installerID
		return nil
	}
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
		return signURL, nil
	}

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa"}, http.StatusOK, "")

	var titleResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{
		SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{Platform: "ios"},
	}, http.StatusOK, &titleResp, "team_id", "0")
	require.Len(t, titleResp.SoftwareTitles, 1)
	require.Equal(t, "ipa_test", titleResp.SoftwareTitles[0].Name)
	titleID := titleResp.SoftwareTitles[0].ID

	readManifest := func(res *http.Response) []byte {
		buf, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		res.Body.Close()
		return buf
	}

	// Mint directly; the activation path is exercised in end-to-end tests.
	token := uuid.NewString()
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return s.ds.CreateInHouseAppInstallToken(context.Background(), q, token, titleID, *teamID, 1)
	})
	res := s.DoRawNoAuth("GET",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/in_house_app/manifest/%s", titleID, token),
		nil, http.StatusOK)

	manifest := readManifest(res)
	require.NotNil(t, manifest)
	escapedURL := `https://example.cloudfront.net/software-installers/storage_id?Expires=1766462733&amp;Signature=some_signature&amp;Key-Pair-Id=ABC123XYZ`
	require.Contains(t, string(manifest), escapedURL)
}

func (s *integrationInstallTestSuite) TestSoftwareInstallerFleetVariables() {
	t := s.T()
	ctx := context.Background()

	s.softwareInstallStore.ExistsFunc = func(ctx context.Context, installerID string) (bool, error) {
		return true, nil
	}
	s.softwareInstallStore.PutFunc = func(ctx context.Context, installerID string, content io.ReadSeeker) error {
		return nil
	}
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
		return "https://example.com/signed", nil
	}

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name()}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID

	const unsupportedVarErrMsg = "Fleet variable $FLEET_VAR_NONEXISTENT is not supported in scripts."

	// upload validation: unsupported and CA variables are rejected, naming the script
	uploadCases := []struct {
		payload *fleet.UploadSoftwareInstallerPayload
		errMsg  string
	}{
		{&fleet.UploadSoftwareInstallerPayload{TeamID: &teamID, Filename: "ruby.deb", InstallScript: "echo $FLEET_VAR_NONEXISTENT"}, unsupportedVarErrMsg},
		{&fleet.UploadSoftwareInstallerPayload{TeamID: &teamID, Filename: "ruby.deb", PostInstallScript: "echo ${FLEET_VAR_NONEXISTENT}"}, unsupportedVarErrMsg},
		{&fleet.UploadSoftwareInstallerPayload{TeamID: &teamID, Filename: "ruby.deb", UninstallScript: "echo $FLEET_VAR_NDES_SCEP_CHALLENGE"}, "Fleet variable $FLEET_VAR_NDES_SCEP_CHALLENGE is not supported in scripts."},
	}
	for _, c := range uploadCases {
		s.uploadSoftwareInstaller(t, c.payload, http.StatusUnprocessableEntity, c.errMsg)
	}

	// supported variables in all three scripts are accepted and stored unexpanded
	payload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:            &teamID,
		Filename:          "ruby.deb",
		InstallScript:     "install $FLEET_VAR_HOST_HARDWARE_SERIAL",
		PostInstallScript: "post ${FLEET_VAR_HOST_UUID}",
		UninstallScript:   "uninstall $FLEET_VAR_HOST_PLATFORM",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, teamID, payload.Filename)
	})
	meta, err := s.ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)
	titleID := *meta.TitleID

	host := createOrbitEnrolledHost(t, "ubuntu", "installer-vars", s.ds)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&teamID, []uint{host.ID})))

	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{},
		http.StatusAccepted)
	installUUID := getLatestSoftwareInstallExecID(t, s.ds, host.ID)

	// stored contents are unexpanded; the orbit details fetch resolves them for the host
	stored, err := s.ds.GetSoftwareInstallDetails(ctx, installUUID)
	require.NoError(t, err)
	require.Equal(t, "install $FLEET_VAR_HOST_HARDWARE_SERIAL", stored.InstallScript)

	var detailsResp fleet.OrbitGetSoftwareInstallResponse
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  installUUID,
		OrbitNodeKey: *host.OrbitNodeKey,
	}, http.StatusOK, &detailsResp)
	require.Equal(t, "install "+host.HardwareSerial, detailsResp.InstallScript)
	require.Equal(t, "post "+host.UUID, detailsResp.PostInstallScript)
	require.Equal(t, "uninstall ubuntu", detailsResp.UninstallScript)

	// the host completes the install so the queue is free for the failure case
	s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
		OrbitNodeKey: *host.OrbitNodeKey,
		HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
			HostID:                host.ID,
			InstallUUID:           installUUID,
			InstallScriptExitCode: new(0),
			InstallScriptOutput:   new("ok"),
		},
	}, http.StatusNoContent)

	// update validation: unsupported variable is rejected
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		TeamID:        &teamID,
		InstallScript: new("echo $FLEET_VAR_NONEXISTENT"),
	}, http.StatusUnprocessableEntity, unsupportedVarErrMsg)

	// update to an IdP variable the host can't resolve
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:       titleID,
		TeamID:        &teamID,
		InstallScript: new("install $FLEET_VAR_HOST_END_USER_IDP_USERNAME"),
	}, http.StatusOK, "")

	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), installSoftwareRequest{},
		http.StatusAccepted)
	failUUID := getLatestSoftwareInstallExecID(t, s.ds, host.ID)

	// the details fetch records the failure server-side and returns not found
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  failUUID,
		OrbitNodeKey: *host.OrbitNodeKey,
	}, http.StatusNotFound, &detailsResp)

	results, err := s.ds.GetSoftwareInstallResults(ctx, failUUID)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallFailed, results.Status)
	require.NotNil(t, results.Output)
	require.Contains(t, *results.Output, "There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_USERNAME.")

	// the user-facing results endpoint renders the reason, not a generic error
	var installResultsResp getSoftwareInstallResultsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", failUUID), nil, http.StatusOK, &installResultsResp)
	require.NotNil(t, installResultsResp.Results.Output)
	require.Contains(t, *installResultsResp.Results.Output, "Fleet couldn't resolve variables in this software's scripts.")
	require.Contains(t, *installResultsResp.Results.Output, "There is no IdP username for this host.")

	// a repeated fetch of the failed install stays not-found and does not
	// record a second result
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID:  failUUID,
		OrbitNodeKey: *host.OrbitNodeKey,
	}, http.StatusNotFound, &detailsResp)
	resultsAgain, err := s.ds.GetSoftwareInstallResults(ctx, failUUID)
	require.NoError(t, err)
	require.Equal(t, results.UpdatedAt, resultsAgain.UpdatedAt)

	// the failed execution left the host's upcoming queue; a retry of the
	// install may be queued under a new execution id
	var upcomingResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &upcomingResp)
	for _, act := range upcomingResp.Activities {
		require.NotContains(t, string(*act.Details), failUUID)
	}

}
