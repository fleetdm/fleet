package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	software_mock "github.com/fleetdm/fleet/v4/server/mock/software"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
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
		Logger:               log.NewLogfmtLogger(os.Stdout),
		EnableCachedDS:       true,
		SoftwareInstallStore: softwareInstallStore,
		FleetConfig:          &fleetConfig,
	}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		installConfig.Logger = kitlog.NewNopLogger()
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
// We are
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
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string) (string, error) {
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
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, payload.TeamID, payload.Filename)
	})
	require.NotZero(t, id)

	meta, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), id)
	require.NoError(t, err)
	titleID := *meta.TitleID

	// create an orbit host, assign to team
	hostInTeam := createOrbitEnrolledHost(t, "linux", "orbit-host-team", s.ds)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &createTeamResp.Team.ID, []uint{hostInTeam.ID}))

	// Create a software installation request
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, titleID), installSoftwareRequest{},
		http.StatusAccepted)

	// Get the InstallerUUID
	var installUUID string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &installUUID,
			"SELECT execution_id FROM host_software_installs WHERE host_id = ?", hostInTeam.ID)
	})

	// Fetch installer details
	var orbitSoftwareResp orbitGetSoftwareInstallResponse
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", orbitGetSoftwareInstallRequest{
		InstallUUID:  installUUID,
		OrbitNodeKey: *hostInTeam.OrbitNodeKey,
	}, http.StatusOK, &orbitSoftwareResp)
	assert.Equal(t, meta.InstallerID, orbitSoftwareResp.InstallerID)
	require.NotNil(t, orbitSoftwareResp.SoftwareInstallerURL)
	assert.Equal(t, "https://example.com/signed", orbitSoftwareResp.SoftwareInstallerURL.URL)
	require.Equal(t, filename, orbitSoftwareResp.SoftwareInstallerURL.Filename)

	// Error in signing -- we simply don't return the URL
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string) (string, error) {
		return "", errors.New("error signing")
	}
	orbitSoftwareResp = orbitGetSoftwareInstallResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", orbitGetSoftwareInstallRequest{
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
	s.softwareInstallStore.SignFunc = func(ctx context.Context, fileID string) (string, error) {
		return s3Store.Sign(ctx, fileID)
	}
	s.DoJSON("POST", "/api/fleet/orbit/software_install/details", orbitGetSoftwareInstallRequest{
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
