package gitops

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/integrationtest"
	"github.com/fleetdm/fleet/v4/pkg/spec"
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
		secretName2 = "LENGTH"
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

// TestProfileValidationWithFleetSecretInDataTag directly tests that profiles with FLEET_SECRET_
// variables in <data> tags fail validation
func (s *integrationGitopsTestSuite) TestProfileValidationWithFleetSecretInDataTag() {
	t := s.T()

	// Create a profile with $FLEET_SECRET_DUO_CERTIFICATE in a <data> tag
	profileContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>PayloadContent</key>
        <array>
            <dict>
                <key>PayloadType</key>
                <string>com.apple.security.root</string>
                <key>PayloadVersion</key>
                <integer>1</integer>
                <key>PayloadIdentifier</key>
                <string>com.example.test.cert</string>
                <key>PayloadUUID</key>
                <string>11111111-2222-3333-4444-555555555555</string>
                <key>PayloadDisplayName</key>
                <string>Test Root Certificate</string>
                <key>PayloadContent</key>
                <data>$FLEET_SECRET_DUO_CERTIFICATE</data>
            </dict>
        </array>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
        <key>PayloadIdentifier</key>
        <string>com.example.test.profile</string>
        <key>PayloadUUID</key>
        <string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
        <key>PayloadDisplayName</key>
        <string>Test MDM Profile with Base64</string>
    </dict>
</plist>`

	// Try to validate the profile - this should fail with "illegal base64 data"
	// This is what happens in getProfilesContents at client.go:360
	_, err := fleet.NewMDMAppleConfigProfile([]byte(profileContent), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal base64 data")
}

// TestProfileWithSecretInName tests that profiles with FLEET_SECRET_ variables in PayloadDisplayName are rejected
func (s *integrationGitopsTestSuite) TestProfileWithSecretInName() {
	t := s.T()

	// Create a profile with $FLEET_SECRET_ in PayloadDisplayName - this should be rejected
	profileContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
        <key>PayloadIdentifier</key>
        <string>com.example.test.profile</string>
        <key>PayloadUUID</key>
        <string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
        <key>PayloadDisplayName</key>
        <string>Profile with $FLEET_SECRET_PASSWORD in name</string>
    </dict>
</plist>`

	// This profile should parse fine without expansion
	mc, err := fleet.NewMDMAppleConfigProfile([]byte(profileContent), nil)
	require.NoError(t, err)

	// But the name contains a FLEET_SECRET variable which should be rejected
	require.Contains(t, mc.Name, "$FLEET_SECRET_PASSWORD")

	// Verify that our validation would catch this BEFORE expansion
	containsSecrets := len(fleet.ContainsPrefixVars(mc.Name, fleet.ServerSecretPrefix)) > 0
	require.True(t, containsSecrets, "Name should be detected as containing secrets")

	// Test the security vulnerability fix: secrets in name with expansion
	t.Setenv("FLEET_SECRET_PASSWORD", "mysecretpassword")

	// Expand the profile content to simulate what the vulnerable code would do
	expandedContent, err := spec.ExpandEnvBytesIncludingSecrets([]byte(profileContent))
	require.NoError(t, err)

	// Parse the expanded content
	mcExpanded, err := fleet.NewMDMAppleConfigProfile(expandedContent, nil)
	require.NoError(t, err)

	// CRITICAL: After expansion, the secret is no longer detectable in the name
	expandedContainsSecrets := len(fleet.ContainsPrefixVars(mcExpanded.Name, fleet.ServerSecretPrefix)) > 0
	require.False(t, expandedContainsSecrets, "Expanded name should NOT contain FLEET_SECRET_ pattern (this is the vulnerability)")

	// The expanded name now contains the actual secret value
	require.Contains(t, mcExpanded.Name, "mysecretpassword")
	require.NotContains(t, mcExpanded.Name, "$FLEET_SECRET_PASSWORD")

	// This demonstrates why we must check the ORIGINAL content before expansion
}
