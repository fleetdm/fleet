package gitops

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/integrationtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	appleMdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsGitops(t *testing.T) {
	testingSuite := new(integrationGitopsTestSuite)
	testingSuite.WithServer.Suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationGitopsTestSuite struct {
	suite.Suite
	integrationtest.WithServer
	fleetCfg config.FleetConfig
}

func (s *integrationGitopsTestSuite) SetupSuite() {
	s.WithDS.SetupSuite("integrationGitopsTestSuite")

	appConf, err := s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	appConf.MDM.WindowsEnabledAndConfigured = true
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	testCert, testKey, err := appleMdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, "../../../../server/service/testdata")
	fleetCfg.Osquery.EnrollCooldown = 0

	mdmStorage, err := s.DS.NewMDMAppleMDMStorage()
	require.NoError(s.T(), err)
	depStorage, err := s.DS.NewMDMAppleDEPStorage()
	require.NoError(s.T(), err)
	scepStorage, err := s.DS.NewSCEPDepot()
	require.NoError(s.T(), err)
	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)

	serverConfig := service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierFree,
		},
		FleetConfig: &fleetCfg,
		MDMStorage:  mdmStorage,
		DEPStorage:  depStorage,
		SCEPStorage: scepStorage,
		Pool:        redisPool,
		APNSTopic:   "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
	}
	err = s.DS.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetSCEPChallenge, Value: []byte("scepchallenge")},
	}, nil)
	require.NoError(s.T(), err)
	users, server := service.RunServerForTestsWithDS(s.T(), s.DS, &serverConfig)
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL) // fleetctl always uses this env var in tests
	s.Server = server
	s.Users = users
	s.fleetCfg = fleetCfg

	appConf, err = s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

func (s *integrationGitopsTestSuite) TearDownSuite() {
	appConf, err := s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

// TestFleetGitops runs `fleetctl gitops` command on configs in https://github.com/fleetdm/fleet-gitops repo.
// Changes to that repo may cause this test to fail.
func (s *integrationGitopsTestSuite) TestFleetGitops() {
	t := s.T()
	const fleetGitopsRepo = "https://github.com/fleetdm/fleet-gitops"

	fleetctlConfig := s.createFleetctlConfig()

	// Clone git repo
	repoDir := t.TempDir()
	_, err := git.PlainClone(
		repoDir, false, &git.CloneOptions{
			ReferenceName: "main",
			SingleBranch:  true,
			Depth:         1,
			URL:           fleetGitopsRepo,
			Progress:      os.Stdout,
		},
	)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	globalFile := path.Join(repoDir, "default.yml")

	// Dry run
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})

	// Real run
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})

}

func (s *integrationGitopsTestSuite) createFleetctlConfig() *os.File {
	t := s.T()
	// Create a temporary fleetctl config file
	fleetctlConfig, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	// GitOps user is a premium feature, so we simply use an admin user.
	token := s.GetTestToken("admin1@example.com", test.GoodPassword)
	configStr := fmt.Sprintf(
		`
contexts:
  default:
    address: %s
    tls-skip-verify: true
    token: %s
`, s.Server.URL, token,
	)
	_, err = fleetctlConfig.WriteString(configStr)
	require.NoError(t, err)
	return fleetctlConfig
}

func (s *integrationGitopsTestSuite) TestFleetGitopsWithFleetSecrets() {
	t := s.T()
	const (
		secretName1 = "NAME"
		secretName2 = "length"
	)
	ctx := context.Background()
	fleetctlConfig := s.createFleetctlConfig()

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_SECRET_"+secretName1, "secret_value")
	t.Setenv("FLEET_SECRET_"+secretName2, "2")
	globalFile := path.Join("..", "..", "fleetctl", "testdata", "gitops", "global_integration.yml")

	// Dry run
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})
	secrets, err := s.DS.GetSecretVariables(ctx, []string{secretName1})
	require.NoError(t, err)
	require.Empty(t, secrets)

	// Real run
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
	// Check secrets
	secrets, err = s.DS.GetSecretVariables(ctx, []string{secretName1, secretName2})
	require.NoError(t, err)
	require.Len(t, secrets, 2)
	for _, secret := range secrets {
		switch secret.Name {
		case secretName1:
			assert.Equal(t, "secret_value", secret.Value)
		case secretName2:
			assert.Equal(t, "2", secret.Value)
		default:
			t.Fatalf("unexpected secret %s", secret.Name)
		}
	}

	// Check script(s)
	scriptID, err := s.DS.GetScriptIDByName(ctx, "fleet-secret.sh", nil)
	require.NoError(t, err)
	expected, err := os.ReadFile("../../fleetctl/testdata/gitops/lib/fleet-secret.sh")
	require.NoError(t, err)
	script, err := s.DS.GetScriptContents(ctx, scriptID)
	require.NoError(t, err)
	assert.Equal(t, expected, script)

	// Check Apple profiles
	profiles, err := s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Contains(t, string(profiles[0].Mobileconfig), "$FLEET_SECRET_"+secretName1)
	// Check Windows profiles
	allProfiles, _, err := s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, allProfiles, 2)
	var windowsProfileUUID string
	for _, profile := range allProfiles {
		if profile.Platform == "windows" {
			windowsProfileUUID = profile.ProfileUUID
		}
	}
	require.NotEmpty(t, windowsProfileUUID)
	winProfile, err := s.DS.GetMDMWindowsConfigProfile(ctx, windowsProfileUUID)
	require.NoError(t, err)
	assert.Contains(t, string(winProfile.SyncML), "${FLEET_SECRET_"+secretName2+"}")
}
