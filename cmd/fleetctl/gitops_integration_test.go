package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

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
	testingSuite.suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationGitopsTestSuite struct {
	suite.Suite
	withServer
	fleetCfg config.FleetConfig
}

func (s *integrationGitopsTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationGitopsTestSuite")

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	testCert, testKey, err := appleMdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")
	fleetCfg.Osquery.EnrollCooldown = 0

	mdmStorage, err := s.ds.NewMDMAppleMDMStorage()
	require.NoError(s.T(), err)
	depStorage, err := s.ds.NewMDMAppleDEPStorage()
	require.NoError(s.T(), err)
	scepStorage, err := s.ds.NewSCEPDepot()
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
	err = s.ds.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetSCEPChallenge, Value: []byte("scepchallenge")},
	}, nil)
	require.NoError(s.T(), err)
	users, server := service.RunServerForTestsWithDS(s.T(), s.ds, &serverConfig)
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL) // fleetctl always uses this env var in tests
	s.server = server
	s.users = users
	s.fleetCfg = fleetCfg

	appConf, err = s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

func (s *integrationGitopsTestSuite) TearDownSuite() {
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
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
	t.Setenv("FLEET_URL", s.server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	globalFile := path.Join(repoDir, "default.yml")

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})

	// Real run
	_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})

}

func (s *integrationGitopsTestSuite) createFleetctlConfig() *os.File {
	t := s.T()
	// Create a temporary fleetctl config file
	fleetctlConfig, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	// GitOps user is a premium feature, so we simply use an admin user.
	token := s.getTestToken("admin1@example.com", test.GoodPassword)
	configStr := fmt.Sprintf(
		`
contexts:
  default:
    address: %s
    tls-skip-verify: true
    token: %s
`, s.server.URL, token,
	)
	_, err = fleetctlConfig.WriteString(configStr)
	require.NoError(t, err)
	return fleetctlConfig
}

func (s *integrationGitopsTestSuite) TestFleetGitopsWithFleetSecrets() {
	t := s.T()
	const secretName = "NAME"
	ctx := context.Background()
	fleetctlConfig := s.createFleetctlConfig()

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_SECRET_"+secretName, "secret_value")
	globalFile := path.Join("testdata", "gitops", "global_integration.yml")

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})
	secrets, err := s.ds.GetSecretVariables(ctx, []string{secretName})
	require.NoError(t, err)
	require.Empty(t, secrets)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
	secrets, err = s.ds.GetSecretVariables(ctx, []string{secretName})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, secretName, secrets[0].Name)
	assert.Equal(t, "secret_value", secrets[0].Value)
	scriptID, err := s.ds.GetScriptIDByName(ctx, "fleet-secret.sh", nil)
	require.NoError(t, err)
	expected, err := os.ReadFile("testdata/gitops/lib/fleet-secret.sh")
	require.NoError(t, err)
	script, err := s.ds.GetScriptContents(ctx, scriptID)
	require.NoError(t, err)
	assert.Equal(t, expected, script)

}
