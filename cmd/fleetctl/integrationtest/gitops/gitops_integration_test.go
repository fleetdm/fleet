package gitops

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
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

// for https://github.com/fleetdm/fleet/issues/28107,
// scenario 1: profile with 2 labels, one is deleted and removed from profile in the same apply
func (s *integrationGitopsTestSuite) TestGitopsDeleteLabelAndRemoveFromProfileScenario1() {
	t := s.T()
	ctx := t.Context()
	fleetctlConfig := s.createFleetctlConfig()

	gitopsDir := t.TempDir()

	// create the labels files
	labelA := writeGitopsFile(t, gitopsDir, "labelA-*.yml", `
- name: A
  query: SELECT 1;
  label_membership_type: dynamic
`)

	labelB := writeGitopsFile(t, gitopsDir, "labelB-*.yml", `
- name: B
  query: SELECT 2;
  label_membership_type: dynamic
`)

	// create the profile file
	profile := writeGitopsFile(t, gitopsDir, "profile-*.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadDescription</key><string>prof1</string>
  <key>PayloadDisplayName</key><string>prof1</string>
  <key>PayloadIdentifier</key><string>com.fleet.1</string>
  <key>PayloadOrganization</key><string>Fleet</string>
  <key>PayloadRemovalDisallowed</key><false/>
  <key>PayloadScope</key><string>System</string>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadUUID</key><string>D399FCFD-C68A-4939-BFA1-CD2814778D25</string>
  <key>PayloadVersion</key><integer>1</integer>
</dict>
</plist>
`)

	// create the gitops file
	globalFile := writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
        labels_include_any:
        - A
        - B
labels:
  - path: ./%s
  - path: ./%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
`, filepath.Base(profile), filepath.Base(labelA), filepath.Base(labelB)))

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})

	// at this point the profile is valid and has "include any" with labels A and B
	profs, _, err := s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, "prof1", profs[0].Name)
	require.Len(t, profs[0].LabelsIncludeAny, 2)
	// labels are sorted by name so this is deterministic
	require.Equal(t, "A", profs[0].LabelsIncludeAny[0].LabelName)
	require.False(t, profs[0].LabelsIncludeAny[0].Broken)
	require.Equal(t, "B", profs[0].LabelsIncludeAny[1].LabelName)
	require.False(t, profs[0].LabelsIncludeAny[1].Broken)

	// update the gitops config to remove label A from the profile and delete it
	// from Fleet at the same time (so it shouldn't be "broken" in the proflie as
	// it is removed from it).
	globalFile = writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
        labels_include_any:
        - B
labels:
  - path: ./%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
`, filepath.Base(profile), filepath.Base(labelB)))

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})

	profs, _, err = s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, "prof1", profs[0].Name)
	// TODO: the following line should fail, it should show as 2 labels and 1
	// broken, but it doesn't... maybe it's a race?
	require.Len(t, profs[0].LabelsIncludeAny, 1)
	require.Equal(t, "B", profs[0].LabelsIncludeAny[0].LabelName)
	require.False(t, profs[0].LabelsIncludeAny[0].Broken)
}

// for https://github.com/fleetdm/fleet/issues/28107,
// scenario 2: profile with 2 labels, one is deleted and both are removed from profile in the same apply
func (s *integrationGitopsTestSuite) TestGitopsDeleteLabelAndRemoveFromProfileScenario2() {
	t := s.T()
	ctx := t.Context()
	fleetctlConfig := s.createFleetctlConfig()

	gitopsDir := t.TempDir()

	// create the labels files
	labelA := writeGitopsFile(t, gitopsDir, "labelA-*.yml", `
- name: A
  query: SELECT 1;
  label_membership_type: dynamic
`)

	labelB := writeGitopsFile(t, gitopsDir, "labelB-*.yml", `
- name: B
  query: SELECT 2;
  label_membership_type: dynamic
`)

	// create the profile file
	profile := writeGitopsFile(t, gitopsDir, "profile-*.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadDescription</key><string>prof1</string>
  <key>PayloadDisplayName</key><string>prof1</string>
  <key>PayloadIdentifier</key><string>com.fleet.1</string>
  <key>PayloadOrganization</key><string>Fleet</string>
  <key>PayloadRemovalDisallowed</key><false/>
  <key>PayloadScope</key><string>System</string>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadUUID</key><string>D399FCFD-C68A-4939-BFA1-CD2814778D25</string>
  <key>PayloadVersion</key><integer>1</integer>
</dict>
</plist>
`)

	// create the gitops file
	globalFile := writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
        labels_include_any:
        - A
        - B
labels:
  - path: ./%s
  - path: ./%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
`, filepath.Base(profile), filepath.Base(labelA), filepath.Base(labelB)))

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})

	// at this point the profile is valid and has "include any" with labels A and B
	profs, _, err := s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, "prof1", profs[0].Name)
	require.Len(t, profs[0].LabelsIncludeAny, 2)
	// labels are sorted by name so this is deterministic
	require.Equal(t, "A", profs[0].LabelsIncludeAny[0].LabelName)
	require.False(t, profs[0].LabelsIncludeAny[0].Broken)
	require.Equal(t, "B", profs[0].LabelsIncludeAny[1].LabelName)
	require.False(t, profs[0].LabelsIncludeAny[1].Broken)

	// update the gitops config to remove label A from the profile and delete it
	// from Fleet at the same time (so it shouldn't be "broken" in the proflie as
	// it is removed from it).
	globalFile = writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
labels:
  - path: ./%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
`, filepath.Base(profile), filepath.Base(labelB)))

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})

	profs, _, err = s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, "prof1", profs[0].Name)
	// TODO: the following line should fail, it should show as 2 labels and 1
	// broken, but it doesn't... maybe it's a race?
	require.Len(t, profs[0].LabelsIncludeAny, 0)
}
