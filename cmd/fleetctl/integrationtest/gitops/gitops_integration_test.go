package gitops

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/fleetctltest"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/integrationtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	appleMdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/svctest"
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
	users, server := svctest.RunServerForTestsWithDS(s.T(), s.DS, &serverConfig)
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL) // fleetctl always uses this env var in tests
	s.Server = server
	s.Users = users
	s.fleetCfg = fleetCfg

	appConf, err = s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	// Disable gitops exceptions so that existing tests can freely use labels, secrets, etc. in their YAML.
	appConf.GitOpsConfig.Exceptions = fleet.GitOpsExceptions{}
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
	_ = fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})

	// Real run
	_ = fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
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
	_ = fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})
	secrets, err := s.DS.GetSecretVariables(ctx, []string{secretName1})
	require.NoError(t, err)
	require.Empty(t, secrets)

	// Real run
	_ = fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
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

func (s *integrationGitopsTestSuite) TestFleetGitopsDDMFleetVarsRequiresPremium() {
	t := s.T()
	fleetctlConfig := s.createFleetctlConfig()

	// Create a DDM declaration with a Fleet variable
	declDir := t.TempDir()
	declFile := path.Join(declDir, "decl-fleetvar.json")
	err := os.WriteFile(declFile, []byte(`{
		"Type": "com.apple.configuration.management.test",
		"Identifier": "com.example.fleetvar-test",
		"Payload": {"Value": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}
	}`), 0o644)
	require.NoError(t, err)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(fmt.Sprintf(`
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: %s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
queries:
`, declFile))
	require.NoError(t, err)

	t.Setenv("FLEET_URL", s.Server.URL)

	// Applying a DDM declaration with Fleet variables should fail without a premium license
	_, err = fleetctltest.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})
	require.ErrorContains(t, err, "missing or invalid license")
}

// customHostVitalsGlobalYAML writes a minimal global GitOps file with the
// given `custom_host_vitals:` body (e.g. "- name: Foo\n- name: Bar", or ""
// to omit the key entirely, which is the declarative clear-all case).
func (s *integrationGitopsTestSuite) customHostVitalsGlobalYAML(customHostVitalsBody string) string {
	t := s.T()
	f, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	customHostVitalsKey := ""
	if customHostVitalsBody != "" {
		customHostVitalsKey = "custom_host_vitals:\n" + customHostVitalsBody
	}
	_, err = f.WriteString(fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
%s
`, customHostVitalsKey))
	require.NoError(t, err)
	return f.Name()
}

func (s *integrationGitopsTestSuite) TestFleetGitopsCustomHostVitals() {
	t := s.T()
	ctx := t.Context()
	fleetctlConfig := s.createFleetctlConfig()
	t.Setenv("FLEET_URL", s.Server.URL)

	listVitals := func() []fleet.CustomHostVital {
		vitals, _, _, err := s.DS.ListCustomHostVitals(ctx, fleet.ListOptions{})
		require.NoError(t, err)
		return vitals
	}
	names := func(vitals []fleet.CustomHostVital) []string {
		out := make([]string, 0, len(vitals))
		for _, v := range vitals {
			out = append(out, v.Name)
		}
		return out
	}

	// Ensure a clean slate: this is global state shared across the suite's tests.
	defer func() {
		globalFile := s.customHostVitalsGlobalYAML("")
		fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
		require.Empty(t, listVitals())
	}()

	// Dry run validates without persisting, and logs what would've been created.
	globalFileV1 := s.customHostVitalsGlobalYAML("  - name: Asset tag\n  - name: Department\n")
	out := fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileV1, "--dry-run"})
	assert.Contains(t, out, "[+] would've created 2 custom host vitals")
	assert.Contains(t, out, "gitops dry run succeeded")
	assert.Empty(t, listVitals())

	// Real run creates both, and logs what it created.
	out = fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileV1})
	assert.Contains(t, out, "[+] creating 2 custom host vitals")
	assert.Contains(t, out, "gitops succeeded")
	require.ElementsMatch(t, []string{"Asset tag", "Department"}, names(listVitals()))

	byName := make(map[string]uint)
	for _, v := range listVitals() {
		byName[v.Name] = v.ID
	}
	assetTagID := byName["Asset tag"]

	// Re-applying with "Department" dropped and "Role" added: Department is
	// deleted, Role is created, Asset tag is retained with the same ID.
	globalFileV2 := s.customHostVitalsGlobalYAML("  - name: Asset tag\n  - name: Role\n")
	out = fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileV2})
	assert.Contains(t, out, "[-] deleting custom host vital 'Department'")
	assert.Contains(t, out, "[+] creating 1 custom host vital")
	require.ElementsMatch(t, []string{"Asset tag", "Role"}, names(listVitals()))
	for _, v := range listVitals() {
		if v.Name == "Asset tag" {
			require.Equal(t, assetTagID, v.ID)
		}
	}

	// A dry run still sends the request to the server (with DryRun set), so
	// server-side validation -- like the collation-aware duplicate-name check,
	// which is not enforced by the local diff -- is still exercised without
	// persisting.
	globalFileDupe := s.customHostVitalsGlobalYAML("  - name: Asset tag\n  - name: asset tag\n")
	_, err := fleetctltest.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileDupe, "--dry-run"})
	require.ErrorContains(t, err, "duplicate custom host vital names")
	require.ElementsMatch(t, []string{"Asset tag", "Role"}, names(listVitals()))

	// A script referencing $FLEET_HOST_VITAL_<id> for "Asset tag" blocks its
	// removal: applying a config that drops it from custom_host_vitals errors
	// the whole run, and no vitals are removed.
	script, err := s.DS.NewScript(ctx, &fleet.Script{
		Name:           "collect-asset-tag.sh",
		ScriptContents: fmt.Sprintf("echo $%s%d", fleet.CustomHostVitalPrefix, assetTagID),
	})
	require.NoError(t, err)

	globalFileV3 := s.customHostVitalsGlobalYAML("  - name: Role\n")
	_, err = fleetctltest.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileV3})
	require.ErrorContains(t, err, "Couldn't delete")
	require.ErrorContains(t, err, "Asset tag")
	require.ElementsMatch(t, []string{"Asset tag", "Role"}, names(listVitals()))

	require.NoError(t, s.DS.DeleteScript(ctx, script.ID))

	// With the reference gone, the same config now succeeds.
	fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileV3})
	require.ElementsMatch(t, []string{"Role"}, names(listVitals()))

	// An absent `custom_host_vitals:` key is a declarative clear-all.
	globalFileAbsent := s.customHostVitalsGlobalYAML("")
	fleetctltest.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileAbsent})
	require.Empty(t, listVitals())
}
