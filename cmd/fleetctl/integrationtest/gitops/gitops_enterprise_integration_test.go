package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"text/template"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/integrationtest"
	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	appleMdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	mock_pkg "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const fleetGitopsRepo = "https://github.com/fleetdm/fleet-gitops"

func TestIntegrationsEnterpriseGitops(t *testing.T) {
	testingSuite := new(enterpriseIntegrationGitopsTestSuite)
	testingSuite.WithServer.Suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type enterpriseIntegrationGitopsTestSuite struct {
	suite.Suite
	integrationtest.WithServer
	fleetCfg               config.FleetConfig
	softwareTitleIconStore fleet.SoftwareTitleIconStore
	iconDir                string
	activityMock           *mock_pkg.MockActivityService
}

func (s *enterpriseIntegrationGitopsTestSuite) SetupSuite() {
	s.WithDS.SetupSuite("enterpriseIntegrationGitopsTestSuite")

	appConf, err := s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	testCert, testKey, err := appleMdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, "../../../../server/service/testdata")
	fleetCfg.Osquery.EnrollCooldown = 0

	err = s.DS.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetAPNSCert, Value: testCertPEM},
		{Name: fleet.MDMAssetAPNSKey, Value: testKeyPEM},
		{Name: fleet.MDMAssetCACert, Value: testCertPEM},
		{Name: fleet.MDMAssetCAKey, Value: testKeyPEM},
	}, nil)
	require.NoError(s.T(), err)

	mdmStorage, err := s.DS.NewMDMAppleMDMStorage()
	require.NoError(s.T(), err)
	depStorage, err := s.DS.NewMDMAppleDEPStorage()
	require.NoError(s.T(), err)
	scepStorage, err := s.DS.NewSCEPDepot()
	require.NoError(s.T(), err)
	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)

	// Create a software title icon store
	iconDir := s.T().TempDir()
	softwareTitleIconStore, err := filesystem.NewSoftwareTitleIconStore(iconDir)
	require.NoError(s.T(), err)
	s.softwareTitleIconStore = softwareTitleIconStore
	s.iconDir = iconDir

	serverConfig := service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		FleetConfig:            &fleetCfg,
		MDMStorage:             mdmStorage,
		DEPStorage:             depStorage,
		SCEPStorage:            scepStorage,
		Pool:                   redisPool,
		APNSTopic:              "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
		SCEPConfigService:      scep.NewSCEPConfigService(slog.New(slog.NewTextHandler(os.Stdout, nil)), nil),
		DigiCertService:        digicert.NewService(),
		SoftwareTitleIconStore: softwareTitleIconStore,
	}
	err = s.DS.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetSCEPChallenge, Value: []byte("scepchallenge")},
	}, nil)
	require.NoError(s.T(), err)

	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		serverConfig.Logger = slog.New(slog.DiscardHandler)
	}
	users, server := service.RunServerForTestsWithDS(s.T(), s.DS, &serverConfig)
	s.activityMock = serverConfig.ActivityMock
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL) // fleetctl always uses this env var in tests
	s.Server = server
	s.Users = users
	s.fleetCfg = fleetCfg

	appConf, err = s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	// Disable gitops exceptions so that existing tests can freely use labels, secrets, etc. in their YAML.
	// Tests that specifically test exception enforcement should re-enable them.
	appConf.GitOpsConfig.Exceptions = fleet.GitOpsExceptions{}
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

func (s *enterpriseIntegrationGitopsTestSuite) TearDownSuite() {
	appConf, err := s.DS.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

func (s *enterpriseIntegrationGitopsTestSuite) TearDownTest() {
	t := s.T()
	ctx := context.Background()

	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		// Delete certificate templates before CAs and teams to avoid FK constraints.
		if _, err := q.ExecContext(ctx, `DELETE FROM certificate_templates`); err != nil {
			return err
		}
		_, err := q.ExecContext(ctx, `DELETE FROM certificate_authorities`)
		return err
	})

	teams, err := s.DS.ListTeams(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
	require.NoError(t, err)
	for _, tm := range teams {
		err := s.DS.DeleteTeam(ctx, tm.ID)
		require.NoError(t, err)
	}

	// Delete policies in "Unassigned" (the others are deleted in ts.DS.DeleteTeam above).
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM policies WHERE team_id = 0;`)
		return err
	})
	// Clean software installers in "Unassigned" (the others are deleted in ts.DS.DeleteTeam above).
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM software_installers WHERE global_or_team_id = 0;`)
		return err
	})

	vppTokens, err := s.DS.ListVPPTokens(ctx)
	require.NoError(t, err)
	for _, tok := range vppTokens {
		err := s.DS.DeleteVPPToken(ctx, tok.ID)
		require.NoError(t, err)
	}

	mysql.ExecAdhocSQL(t, s.DS, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_apps;")
		return err
	})
	mysql.ExecAdhocSQL(t, s.DS, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM in_house_apps;")
		return err
	})

	lbls, err := s.DS.ListLabels(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{}, false)
	require.NoError(t, err)
	for _, lbl := range lbls {
		if lbl.LabelType != fleet.LabelTypeBuiltIn {
			err := s.DS.DeleteLabel(ctx, lbl.Name, fleet.TeamFilter{User: test.UserAdmin})
			require.NoError(t, err)
		}
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) assertDryRunOutput(t *testing.T, output string) {
	s.assertDryRunOutputWithDeprecation(t, output, false)
}

func (s *enterpriseIntegrationGitopsTestSuite) assertDryRunOutputWithDeprecation(t *testing.T, output string, expectDeprecation bool) {
	var sawDeprecation bool
	allowedVerbs := []string{
		"moved",
		"deleted",
		"updated",
		"applied",
		"added",
		"created",
		"set",
	}
	pattern := fmt.Sprintf("\\[([+\\-!])] would've (%s)", strings.Join(allowedVerbs, "|"))
	reg := regexp.MustCompile(pattern)
	for line := range strings.SplitSeq(output, "\n") {
		if expectDeprecation && line != "" && strings.Contains(line, "is deprecated") {
			sawDeprecation = true
			continue
		}
		if line != "" && !strings.Contains(line, "succeeded") {
			assert.Regexp(t, reg, line, "on dry run")
		}
	}
	if expectDeprecation {
		assert.True(t, sawDeprecation, "expected to see deprecation warning in dry run output")
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) assertRealRunOutput(t *testing.T, output string) {
	s.assertRealRunOutputWithDeprecation(t, output, false)
}

func (s *enterpriseIntegrationGitopsTestSuite) assertRealRunOutputWithDeprecation(t *testing.T, output string, allowDeprecation bool) {
	allowedVerbs := []string{
		"moving",
		"deleted",
		"updated",
		"applied",
		"added",
		"created",
		"set",
		"applying", // this is used when doing groups operations before the operation starts, e.g. "Applying 10 policies"
		"deleting", // ditto
	}
	pattern := fmt.Sprintf("\\[([+\\-!])] (%s)", strings.Join(allowedVerbs, "|"))
	reg := regexp.MustCompile(pattern)
	for line := range strings.SplitSeq(output, "\n") {
		if allowDeprecation && line != "" && strings.Contains(line, "is deprecated") {
			continue
		}
		if line != "" && !strings.Contains(line, "succeeded") {
			assert.Regexp(t, reg, line, "on real run")
		}
	}
}

// TestFleetGitops runs `fleetctl gitops` command on configs in https://github.com/fleetdm/fleet-gitops repo.
// Changes to that repo may cause this test to fail.
func (s *enterpriseIntegrationGitopsTestSuite) TestFleetGitops() {
	os.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)
	defer os.Unsetenv("FLEET_ENABLE_LOG_TOPICS")

	t := s.T()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

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
	t.Setenv("FLEET_WORKSTATIONS_ENROLL_SECRET", "workstations_enroll_secret")
	t.Setenv("FLEET_PERSONAL_MOBILE_DEVICES_ENROLL_SECRET", "personal_mobile_devices_enroll_secret")
	t.Setenv("FLEET_DEDICATED_DEVICES_ENROLL_SECRET", "dedicated_devices_enroll_secret")
	t.Setenv("FLEET_EMPLOYEE_ISSUED_MOBILE_DEVICES_ENROLL_SECRET", "employee_issued_mobile_devices_enroll_secret")
	t.Setenv("FLEET_IT_SERVERS_ENROLL_SECRET", "it_servers_enroll_secret")
	globalFile := path.Join(repoDir, "default.yml")
	teamsDir := path.Join(repoDir, "teams")
	teamFiles, err := os.ReadDir(teamsDir)
	require.NoError(t, err)
	teamFileNames := make([]string, 0, len(teamFiles))
	for _, file := range teamFiles {
		if filepath.Ext(file.Name()) == ".yml" {
			teamFileNames = append(teamFileNames, path.Join(teamsDir, file.Name()))
		}
	}

	// Create a team to be deleted.
	deletedTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	const deletedTeamName = "team_to_be_deleted"

	_, err = deletedTeamFile.WriteString(
		fmt.Sprintf(
			`
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"deleted_team_secret"}]
`, deletedTeamName,
		),
	)
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, s.DS)

	// Apply the team to be deleted
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", deletedTeamFile.Name()}), true)

	// Dry run
	// NOTE: The fleet-gitops repo may still use deprecated keys (queries, team_settings),
	// so we allow deprecation warnings in this test.
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"}), true)
	for _, fileName := range teamFileNames {
		// When running no-teams, global config must also be provided ...
		if strings.Contains(fileName, "unassigned.yml") {
			s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "-f", globalFile, "--dry-run"}), true)
		} else {
			s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "--dry-run"}), true)
		}
	}

	// Dry run with all the files
	args := []string{"gitops", "--config", fleetctlConfig.Name(), "--dry-run", "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)

	// Real run with all the files, but don't delete other teams
	args = []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)

	// Check that all the teams exist
	teamsJSON := fleetctl.RunAppForTest(t, []string{"get", "teams", "--config", fleetctlConfig.Name(), "--json"})
	assert.Equal(t, 3, strings.Count(teamsJSON, "fleet_id"))

	// Real run with all the files, and delete other teams
	args = []string{"gitops", "--config", fleetctlConfig.Name(), "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)

	// Check that only the right teams exist
	teamsJSON = fleetctl.RunAppForTest(t, []string{"get", "teams", "--config", fleetctlConfig.Name(), "--json"})
	assert.Equal(t, 2, strings.Count(teamsJSON, "fleet_id"))
	assert.NotContains(t, teamsJSON, deletedTeamName)

	// Real run with one file at a time
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile}), true)
	for _, fileName := range teamFileNames {
		// When running no-teams, global config must also be provided ...
		if strings.Contains(fileName, "unassigned.yml") {
			s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "-f", globalFile}), true)
		} else {
			s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName}), true)
		}
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) createFleetctlConfig(t *testing.T, user fleet.User) *os.File {
	fleetctlConfig, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	token := s.GetTestToken(user.Email, test.GoodPassword)
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

func (s *enterpriseIntegrationGitopsTestSuite) createGitOpsUser(t *testing.T) fleet.User {
	user := fleet.User{
		Name:       "GitOps User " + uuid.NewString(),
		Email:      uuid.NewString() + "@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.DS.NewUser(context.Background(), &user)
	require.NoError(t, err)
	return user
}

// TestDeleteMacOSSetup tests the deletion of macOS setup assets by `fleetctl gitops` command.
func (s *enterpriseIntegrationGitopsTestSuite) TestDeleteMacOSSetup() {
	t := s.T()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(`
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(
		fmt.Sprintf(
			`
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`, teamName,
		),
	)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	// Add bootstrap packages
	require.NoError(t, s.DS.InsertMDMAppleBootstrapPackage(context.Background(), &fleet.MDMAppleBootstrapPackage{
		Name:   "bootstrap.pkg",
		TeamID: 0,
		Bytes:  []byte("bootstrap package"),
		Token:  uuid.NewString(),
		Sha256: []byte("sha256"),
	}, nil))
	team, err := s.DS.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = s.DS.DeleteTeam(context.Background(), team.ID)
	})
	require.NoError(t, s.DS.InsertMDMAppleBootstrapPackage(context.Background(), &fleet.MDMAppleBootstrapPackage{
		Name:   "bootstrap.pkg",
		TeamID: team.ID,
		Bytes:  []byte("bootstrap package"),
		Token:  uuid.NewString(),
		Sha256: []byte("sha256"),
	}, nil))
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		stmt := "SELECT COUNT(*) FROM mdm_apple_bootstrap_packages WHERE team_id IN (?, ?)"
		var result int
		require.NoError(t, sqlx.GetContext(context.Background(), q, &result, stmt, 0, team.ID))
		assert.Equal(t, 2, result)
		return nil
	})

	// Add enrollment profiles
	_, err = s.DS.SetOrUpdateMDMAppleSetupAssistant(context.Background(), &fleet.MDMAppleSetupAssistant{
		TeamID:  nil,
		Name:    "enrollment_profile.json",
		Profile: []byte(`{"foo":"bar"}`),
	})
	require.NoError(t, err)
	_, err = s.DS.SetOrUpdateMDMAppleSetupAssistant(context.Background(), &fleet.MDMAppleSetupAssistant{
		TeamID:  &team.ID,
		Name:    "enrollment_profile.json",
		Profile: []byte(`{"foo":"bar"}`),
	})
	require.NoError(t, err)
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		stmt := "SELECT COUNT(*) FROM mdm_apple_setup_assistants WHERE global_or_team_id IN (?, ?)"
		var result int
		require.NoError(t, sqlx.GetContext(context.Background(), q, &result, stmt, 0, team.ID))
		assert.Equal(t, 2, result)
		return nil
	})

	// Re-apply configs and expect the macOS setup assets to be cleared
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		stmt := "SELECT COUNT(*) FROM mdm_apple_bootstrap_packages WHERE team_id IN (?, ?)"
		var result int
		require.NoError(t, sqlx.GetContext(context.Background(), q, &result, stmt, 0, team.ID))
		assert.Equal(t, 0, result)
		return nil
	})
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		stmt := "SELECT COUNT(*) FROM mdm_apple_setup_assistants WHERE global_or_team_id IN (?, ?)"
		var result int
		require.NoError(t, sqlx.GetContext(context.Background(), q, &result, stmt, 0, team.ID))
		assert.Equal(t, 0, result)
		return nil
	})
}

// TestCAIntegrations enables DigiCert and Custom SCEP CAs via GitOps.
// At the same time, GitOps uploads Apple profiles that use the newly configured CAs.
func (s *enterpriseIntegrationGitopsTestSuite) TestCAIntegrations() {
	t := s.T()
	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	var (
		gotProfileMu sync.Mutex
		gotProfile   bool
	)
	digiCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			matches := regexp.MustCompile(`^/mpki/api/v2/profile/([a-zA-Z0-9_-]+)$`).FindStringSubmatch(r.URL.Path)
			if len(matches) != 2 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			profileID := matches[1]

			resp := map[string]string{
				"id":     profileID,
				"name":   "DigiCert",
				"status": "Active",
			}
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
			gotProfileMu.Lock()
			gotProfile = profileID == "digicert_profile_id"
			defer gotProfileMu.Unlock()
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(digiCertServer.Close)

	scepServer := scep_server.StartTestSCEPServer(t)

	// Get the path to the directory of this test file
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get runtime caller info")
	dirPath := filepath.Dir(currentFile)
	// Resolve ../../fleetctl relative to the source file directory
	dirPath = filepath.Join(dirPath, "../../fleetctl")
	// Clean and convert to absolute path
	dirPath, err := filepath.Abs(filepath.Clean(dirPath))
	require.NoError(t, err)

	apiToken := "digicert_api_token" // nolint:gosec // G101: Potential hardcoded credentials
	profileID := "digicert_profile_id"
	certCN := "digicert_cn"
	certSeatID := "digicert_seat_id"
	_, err = s.DS.NewCertificateAuthority(t.Context(), &fleet.CertificateAuthority{
		Type:                          string(fleet.CATypeDigiCert),
		Name:                          ptr.String("DigiCert"),
		URL:                           &digiCertServer.URL,
		APIToken:                      &apiToken,
		ProfileID:                     &profileID,
		CertificateCommonName:         &certCN,
		CertificateUserPrincipalNames: &[]string{"digicert_upn"},
		CertificateSeatID:             &certSeatID,
	})
	require.NoError(t, err)
	challenge := "challenge"
	_, err = s.DS.NewCertificateAuthority(t.Context(), &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("CustomScepProxy"),
		URL:       &scepServer.URL,
		Challenge: &challenge,
	})
	require.NoError(t, err)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(fmt.Sprintf(`
agent_options:
controls:
  apple_settings:
    configuration_profiles:
      - path: %s/testdata/gitops/lib/scep-and-digicert.mobileconfig
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
  certificate_authorities:
    digicert:
      - name: DigiCert
        url: %s
        api_token: digicert_api_token
        profile_id: digicert_profile_id
        certificate_common_name: digicert_cn
        certificate_user_principal_names: ["digicert_upn"]
        certificate_seat_id: digicert_seat_id
    custom_scep_proxy:
      - name: CustomScepProxy
        url: %s
        challenge: challenge
policies:
reports:
`,
		dirPath,
		digiCertServer.URL,
		scepServer.URL+"/scep",
	))
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	groupedCAs, err := s.DS.GetGroupedCertificateAuthorities(t.Context(), false)
	require.NoError(t, err)

	// check digicert
	require.Len(t, groupedCAs.DigiCert, 1)
	digicertCA := groupedCAs.DigiCert[0]
	require.Equal(t, "DigiCert", digicertCA.Name)
	require.Equal(t, digiCertServer.URL, digicertCA.URL)
	require.Equal(t, fleet.MaskedPassword, digicertCA.APIToken)
	require.Equal(t, "digicert_profile_id", digicertCA.ProfileID)
	require.Equal(t, "digicert_cn", digicertCA.CertificateCommonName)
	require.Equal(t, []string{"digicert_upn"}, digicertCA.CertificateUserPrincipalNames)
	require.Equal(t, "digicert_seat_id", digicertCA.CertificateSeatID)
	gotProfileMu.Lock()
	require.False(t, gotProfile) // external digicert service was NOT called because stored config was not modified
	gotProfileMu.Unlock()

	// check custom SCEP proxy
	require.Len(t, groupedCAs.CustomScepProxy, 1)
	customSCEPProxyCA := groupedCAs.CustomScepProxy[0]
	require.Equal(t, "CustomScepProxy", customSCEPProxyCA.Name)
	require.Equal(t, scepServer.URL+"/scep", customSCEPProxyCA.URL)
	require.Equal(t, fleet.MaskedPassword, customSCEPProxyCA.Challenge)

	profiles, _, err := s.DS.ListMDMConfigProfiles(context.Background(), nil, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, profiles, 1)

	// now modify the stored config and confirm that external digicert service is called
	_, err = globalFile.WriteString(fmt.Sprintf(`
agent_options:
controls:
  apple_settings:
    configuration_profiles:
      - path: %s/testdata/gitops/lib/scep-and-digicert.mobileconfig
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
  certificate_authorities:
    digicert:
      - name: DigiCert
        url: %s
        api_token: digicert_api_token
        profile_id: digicert_profile_id
        certificate_common_name: digicert_cn
        certificate_user_principal_names: [%q]
        certificate_seat_id: digicert_seat_id
    custom_scep_proxy:
      - name: CustomScepProxy
        url: %s
        challenge: challenge
policies:
reports:
`,
		dirPath,
		digiCertServer.URL,
		"digicert_upn_2", // minor modification to stored config so gitops run is not a no-op and triggers call to external digicert service
		scepServer.URL+"/scep",
	))
	require.NoError(t, err)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	groupedCAs, err = s.DS.GetGroupedCertificateAuthorities(t.Context(), false)
	require.NoError(t, err)

	// check digicert
	require.Len(t, groupedCAs.DigiCert, 1)
	digicertCA = groupedCAs.DigiCert[0]
	require.Equal(t, "DigiCert", digicertCA.Name)
	require.Equal(t, digiCertServer.URL, digicertCA.URL)
	require.Equal(t, fleet.MaskedPassword, digicertCA.APIToken)
	require.Equal(t, "digicert_profile_id", digicertCA.ProfileID)
	require.Equal(t, "digicert_cn", digicertCA.CertificateCommonName)
	require.Equal(t, []string{"digicert_upn_2"}, digicertCA.CertificateUserPrincipalNames)
	require.Equal(t, "digicert_seat_id", digicertCA.CertificateSeatID)
	gotProfileMu.Lock()
	require.True(t, gotProfile) // external digicert service was called because stored config was modified
	gotProfileMu.Unlock()

	// Now test that we can clear the configs
	_, err = globalFile.WriteString(`
agent_options:
controls:
  apple_settings:
    configuration_profiles:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`)
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	groupedCAs, err = s.DS.GetGroupedCertificateAuthorities(t.Context(), true)
	require.NoError(t, err)
	assert.Empty(t, groupedCAs.DigiCert)
	assert.Empty(t, groupedCAs.CustomScepProxy)
}

// TestDeleteCAWithCertificateTemplates tests the GitOps ordering when deleting a certificate
// authority that is referenced by certificate templates on a team.
func (s *enterpriseIntegrationGitopsTestSuite) TestDeleteCAWithCertificateTemplates() {
	t := s.T()
	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	scepServer := scep_server.StartTestSCEPServer(t)

	t.Setenv("FLEET_URL", s.Server.URL)

	// Step 1: Create a CA and a team with a certificate template referencing it via GitOps.
	globalFileWithCA := s.writeConfigFile(t, fmt.Sprintf(`
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
  certificate_authorities:
    custom_scep_proxy:
      - name: TestSCEP
        url: %s
        challenge: challenge
policies:
reports:
`, scepServer.URL+"/scep"))

	teamFileWithCert := s.writeConfigFile(t, `
name: CA Test Team
controls:
  android_settings:
    certificates:
      - name: TestCert
        certificate_authority_name: TestSCEP
        subject_name: "CN=test,O=Fleet"
settings:
  secrets:
    - secret: ca_test_team_secret
agent_options:
policies:
reports:
software:
`)

	// Apply both files to create the CA, team, and certificate template.
	fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFileWithCA, "-f", teamFileWithCert,
	})

	// Verify the CA was created via GitOps.
	groupedCAs, err := s.DS.GetGroupedCertificateAuthorities(t.Context(), false)
	require.NoError(t, err)
	require.Len(t, groupedCAs.CustomScepProxy, 1)
	require.Equal(t, "TestSCEP", groupedCAs.CustomScepProxy[0].Name)

	// Verify the team was created and the certificate template exists.
	teams, err := s.DS.ListTeams(t.Context(), fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
	require.NoError(t, err)
	var teamID uint
	for _, tm := range teams {
		if tm.Name == "CA Test Team" {
			teamID = tm.ID
			break
		}
	}
	require.NotZero(t, teamID, "team 'CA Test Team' should exist")

	certTemplates, _, err := s.DS.GetCertificateTemplatesByTeamID(t.Context(), teamID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certTemplates, 1)
	require.Equal(t, "TestCert", certTemplates[0].Name)

	// Step 2: Run gitops removing the CA but WITHOUT the team file.
	// This should fail because the team's certificate templates still reference the CA.
	globalFileWithoutCA := s.writeConfigFile(t, `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`)

	_, err = fleetctl.RunAppNoChecks([]string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFileWithoutCA,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), fleet.DeleteCAReferencedByTemplatesErrMsg)

	// Verify CA and certificate template still exist.
	groupedCAs, err = s.DS.GetGroupedCertificateAuthorities(t.Context(), false)
	require.NoError(t, err)
	require.Len(t, groupedCAs.CustomScepProxy, 1)

	certTemplates, _, err = s.DS.GetCertificateTemplatesByTeamID(t.Context(), teamID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certTemplates, 1)

	// Step 3: Run gitops removing BOTH the CA and the certificate template.
	// This should succeed because the postOp ordering ensures certificate templates
	// are deleted before the CA.
	teamFileWithoutCert := s.writeConfigFile(t, `
name: CA Test Team
controls:
settings:
  secrets:
    - secret: ca_test_team_secret
agent_options:
policies:
reports:
software:
`)

	fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFileWithoutCA, "-f", teamFileWithoutCert,
	})

	// Verify CA and certificate template are deleted.
	groupedCAs, err = s.DS.GetGroupedCertificateAuthorities(t.Context(), false)
	require.NoError(t, err)
	assert.Empty(t, groupedCAs.CustomScepProxy)

	certTemplates, _, err = s.DS.GetCertificateTemplatesByTeamID(t.Context(), teamID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, certTemplates)
}

func (s *enterpriseIntegrationGitopsTestSuite) writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// TestUnsetConfigurationProfileLabels tests the removal of labels associated with a
// configuration profile via gitops.
func (s *enterpriseIntegrationGitopsTestSuite) TestUnsetConfigurationProfileLabels() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	profileFile, err := os.CreateTemp(t.TempDir(), "*.mobileconfig")
	require.NoError(t, err)
	_, err = profileFile.WriteString(test.GenerateMDMAppleProfile("test", "test", uuid.NewString()))
	require.NoError(t, err)
	err = profileFile.Close()
	require.NoError(t, err)

	const (
		globalTemplate = `
agent_options:
labels:
  - name: Label1
    query: select 1
controls:
  apple_settings:
    configuration_profiles:
      - path: %s
%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`
		withLabelsIncludeAny = `
        labels_include_any:
          - Label1
`
		emptyLabelsIncludeAny = `
        labels_include_any:
`
		teamTemplate = `
controls:
  apple_settings:
    configuration_profiles:
      - path: %s
%s
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
		withLabelsIncludeAll = `
        labels_include_all:
          - Label1
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(fmt.Sprintf(globalTemplate, profileFile.Name(), withLabelsIncludeAny))
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamTemplate, profileFile.Name(), withLabelsIncludeAll, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	// get the team ID
	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// the custom setting is scoped by the label for no team
	profs, _, err := s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Len(t, profs[0].LabelsIncludeAny, 1)
	require.Equal(t, "Label1", profs[0].LabelsIncludeAny[0].LabelName)

	// the custom setting is scoped by the label for team
	profs, _, err = s.DS.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Len(t, profs[0].LabelsIncludeAll, 1)
	require.Equal(t, "Label1", profs[0].LabelsIncludeAll[0].LabelName)

	// remove the label conditions
	err = os.WriteFile(globalFile.Name(), []byte(fmt.Sprintf(globalTemplate, profileFile.Name(), emptyLabelsIncludeAny)), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), []byte(fmt.Sprintf(teamTemplate, profileFile.Name(), "", teamName)), 0o644)
	require.NoError(t, err)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	// the custom setting is not scoped by label anymore
	profs, _, err = s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Len(t, profs[0].LabelsIncludeAny, 0)

	profs, _, err = s.DS.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Len(t, profs[0].LabelsIncludeAll, 0)
}

// TestUnsetSoftwareInstallerLabels tests the removal of labels associated with a
// software installer via gitops.
func (s *enterpriseIntegrationGitopsTestSuite) TestUnsetSoftwareInstallerLabels() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		globalTemplate = `
agent_options:
labels:
  - name: Label1
    query: select 1
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		noTeamTemplate = `name: Unassigned
controls:
policies:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
%s
`
		withLabelsIncludeAny = `
      labels_include_any:
        - Label1
`
		emptyLabelsIncludeAny = `
      labels_include_any:
`
		withLabelsIncludeAll = `
      labels_include_all:
        - Label1
`
		teamTemplate = `
controls:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
%s
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
		withLabelsExcludeAny = `
      labels_exclude_any:
        - Label1
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, withLabelsIncludeAny))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamTemplate, withLabelsExcludeAny, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}))

	// get the team ID
	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// the installer is scoped by the label for no team
	titles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.NotNil(t, titles[0].SoftwarePackage)
	noTeamTitleID := titles[0].ID
	meta, err := s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, noTeamTitleID, false)
	require.NoError(t, err)
	require.Len(t, meta.LabelsIncludeAny, 1)
	require.Equal(t, "Label1", meta.LabelsIncludeAny[0].LabelName)

	// the installer is scoped by the label for team
	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team.ID}, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.NotNil(t, titles[0].SoftwarePackage)
	teamTitleID := titles[0].ID
	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.Len(t, meta.LabelsExcludeAny, 1)
	require.Equal(t, "Label1", meta.LabelsExcludeAny[0].LabelName)

	// switch both to labels_include_all
	err = os.WriteFile(noTeamFilePath, fmt.Appendf(nil, noTeamTemplate, withLabelsIncludeAll), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, withLabelsIncludeAll, teamName), 0o644)
	require.NoError(t, err)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}))

	// the installer is now scoped by labels_include_all for no team
	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, noTeamTitleID, false)
	require.NoError(t, err)
	require.Empty(t, meta.LabelsIncludeAny)
	require.Empty(t, meta.LabelsExcludeAny)
	require.Len(t, meta.LabelsIncludeAll, 1)
	require.Equal(t, "Label1", meta.LabelsIncludeAll[0].LabelName)

	// the installer is now scoped by labels_include_all for team
	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.Empty(t, meta.LabelsIncludeAny)
	require.Empty(t, meta.LabelsExcludeAny)
	require.Len(t, meta.LabelsIncludeAll, 1)
	require.Equal(t, "Label1", meta.LabelsIncludeAll[0].LabelName)

	// remove the label conditions
	err = os.WriteFile(noTeamFilePath, []byte(fmt.Sprintf(noTeamTemplate, emptyLabelsIncludeAny)), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), []byte(fmt.Sprintf(teamTemplate, "", teamName)), 0o644)
	require.NoError(t, err)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}))

	// the installer is not scoped by label anymore
	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, noTeamTitleID, false)
	require.NoError(t, err)
	require.NotNil(t, meta.TitleID)
	require.Equal(t, noTeamTitleID, *meta.TitleID)
	require.Len(t, meta.LabelsExcludeAny, 0)
	require.Len(t, meta.LabelsIncludeAny, 0)
	require.Len(t, meta.LabelsIncludeAll, 0)

	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.NotNil(t, meta.TitleID)
	require.Equal(t, teamTitleID, *meta.TitleID)
	require.Len(t, meta.LabelsExcludeAny, 0)
	require.Len(t, meta.LabelsIncludeAny, 0)
	require.Len(t, meta.LabelsIncludeAll, 0)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestDeletingNoTeamYAML() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// global file setup
	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	// setup script
	const testScriptTemplate = `echo "Hello, world!"`

	scriptFile, err := os.CreateTemp(t.TempDir(), "*.sh")
	require.NoError(t, err)
	_, err = scriptFile.WriteString(testScriptTemplate)
	require.NoError(t, err)
	err = scriptFile.Close()
	require.NoError(t, err)

	// no team file setup
	const (
		noTeamTemplate = `name: Unassigned
policies:
controls:
  setup_experience:
    macos_script: %s
software:
`
	)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, scriptFile.Name()))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath}))

	// Check script existence
	_, err = s.DS.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)

	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})

	// Check script does not exist
	_, err = s.DS.GetSetupExperienceScript(ctx, nil)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
}

// Helper function to get No Team webhook settings from the database
func getNoTeamWebhookSettings(ctx context.Context, t *testing.T, ds *mysql.Datastore) fleet.FailingPoliciesWebhookSettings {
	cfg, err := ds.DefaultTeamConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	return cfg.WebhookSettings.FailingPoliciesWebhook
}

// Helper function to verify No Team webhook settings match expected values
func verifyNoTeamWebhookSettings(ctx context.Context, t *testing.T, ds *mysql.Datastore, expected fleet.FailingPoliciesWebhookSettings) {
	actual := getNoTeamWebhookSettings(ctx, t, ds)

	require.Equal(t, expected.Enable, actual.Enable)
	if expected.Enable {
		require.Equal(t, expected.DestinationURL, actual.DestinationURL)
		require.Equal(t, expected.HostBatchSize, actual.HostBatchSize)
		require.ElementsMatch(t, expected.PolicyIDs, actual.PolicyIDs)
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) TestNoTeamWebhookSettings() {
	t := s.T()
	ctx := t.Context()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	var webhookSettings fleet.FailingPoliciesWebhookSettings

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Create a global config file
	const globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: global_secret
policies:
reports:
`

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	// Create a no-team.yml file with webhook settings
	const noTeamTemplateWithWebhook = `
name: Unassigned
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://example.com/no-team-webhook
      host_batch_size: 50
      policy_ids:
        - 1
        - 2
        - 3
`

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(noTeamTemplateWithWebhook)
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	// Test dry-run first
	output := fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"})
	s.assertDryRunOutputWithDeprecation(t, output, true)

	// Check that webhook settings are mentioned in the output
	require.Contains(t, output, "would've applied webhook settings for unassigned hosts")

	// Apply the configuration (non-dry-run)
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})
	s.assertRealRunOutputWithDeprecation(t, output, true)

	// Verify the output mentions webhook settings were applied
	require.Contains(t, output, "applying webhook settings for unassigned hosts")
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook settings were actually applied by checking the database
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable:         true,
		DestinationURL: "https://example.com/no-team-webhook",
		HostBatchSize:  50,
		PolicyIDs:      []uint{1, 2, 3},
	})

	// Test updating webhook settings
	const noTeamTemplateUpdatedWebhook = `
name: Unassigned
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: false
      destination_url: https://updated.example.com/webhook
      host_batch_size: 100
      policy_ids:
        - 4
        - 5
`

	noTeamFileUpdated, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFileUpdated.WriteString(noTeamTemplateUpdatedWebhook)
	require.NoError(t, err)
	err = noTeamFileUpdated.Close()
	require.NoError(t, err)
	noTeamFilePathUpdated := filepath.Join(filepath.Dir(noTeamFileUpdated.Name()), "unassigned.yml")
	err = os.Rename(noTeamFileUpdated.Name(), noTeamFilePathUpdated)
	require.NoError(t, err)

	// Apply the updated configuration
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathUpdated})

	// Verify the output still mentions webhook settings were applied
	require.Contains(t, output, "applying webhook settings for unassigned hosts")
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook settings were updated
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable:         false,
		DestinationURL: "https://updated.example.com/webhook",
		HostBatchSize:  100,
		PolicyIDs:      []uint{4, 5},
	})

	// Test removing webhook settings entirely
	const noTeamTemplateNoWebhook = `
name: Unassigned
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
`

	noTeamFileNoWebhook, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFileNoWebhook.WriteString(noTeamTemplateNoWebhook)
	require.NoError(t, err)
	err = noTeamFileNoWebhook.Close()
	require.NoError(t, err)
	noTeamFilePathNoWebhook := filepath.Join(filepath.Dir(noTeamFileNoWebhook.Name()), "unassigned.yml")
	err = os.Rename(noTeamFileNoWebhook.Name(), noTeamFilePathNoWebhook)
	require.NoError(t, err)

	// Apply configuration without webhook settings
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathNoWebhook})

	// Verify webhook settings are mentioned as being applied (they're applied as nil to clear)
	require.Contains(t, output, "applying webhook settings for unassigned hosts")
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook settings were cleared
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable: false,
	})

	// Test case: team_settings exists but webhook_settings is nil
	// First, set webhook settings again
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook was set
	webhookSettings = getNoTeamWebhookSettings(ctx, t, s.DS)
	require.True(t, webhookSettings.Enable)

	// Now apply config with team_settings but no webhook_settings
	const noTeamTemplateTeamSettingsNoWebhook = `
name: Unassigned
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
settings:
`
	noTeamFileTeamNoWebhook, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFileTeamNoWebhook.WriteString(noTeamTemplateTeamSettingsNoWebhook)
	require.NoError(t, err)
	err = noTeamFileTeamNoWebhook.Close()
	require.NoError(t, err)
	noTeamFilePathTeamNoWebhook := filepath.Join(filepath.Dir(noTeamFileTeamNoWebhook.Name()), "unassigned.yml")
	err = os.Rename(noTeamFileTeamNoWebhook.Name(), noTeamFilePathTeamNoWebhook)
	require.NoError(t, err)

	// Apply configuration with team_settings but no webhook_settings
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathTeamNoWebhook})

	// Verify webhook settings are cleared
	require.Contains(t, output, "applying webhook settings for unassigned hosts")
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook settings are disabled
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable: false,
	})

	// Test case: webhook_settings exists but failing_policies_webhook is nil
	// First, set webhook settings again
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook was set
	webhookSettings = getNoTeamWebhookSettings(ctx, t, s.DS)
	require.True(t, webhookSettings.Enable)

	// Now apply config with webhook_settings but no failing_policies_webhook
	const noTeamTemplateWebhookNoFailing = `
name: Unassigned
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
settings:
  webhook_settings:
`
	noTeamFileWebhookNoFailing, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFileWebhookNoFailing.WriteString(noTeamTemplateWebhookNoFailing)
	require.NoError(t, err)
	err = noTeamFileWebhookNoFailing.Close()
	require.NoError(t, err)
	noTeamFilePathWebhookNoFailing := filepath.Join(filepath.Dir(noTeamFileWebhookNoFailing.Name()), "unassigned.yml")
	err = os.Rename(noTeamFileWebhookNoFailing.Name(), noTeamFilePathWebhookNoFailing)
	require.NoError(t, err)

	// Apply configuration with webhook_settings but no failing_policies_webhook
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathWebhookNoFailing})

	// Verify webhook settings are cleared
	require.Contains(t, output, "applying webhook settings for unassigned hosts")
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook settings are disabled
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable: false,
	})
}

func (s *enterpriseIntegrationGitopsTestSuite) TestRemoveCustomSettingsFromDefaultYAML() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// setup custom settings profile
	profileFile, err := os.CreateTemp(t.TempDir(), "*.mobileconfig")
	require.NoError(t, err)
	_, err = profileFile.WriteString(test.GenerateMDMAppleProfile("test", "test", uuid.NewString()))
	require.NoError(t, err)
	err = profileFile.Close()
	require.NoError(t, err)

	// global file setup with custom settings
	const (
		globalTemplateWithCustomSettings = `
agent_options:
controls:
  apple_settings:
    configuration_profiles:
      - path: %s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(fmt.Sprintf(globalTemplateWithCustomSettings, profileFile.Name()))
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	profiles, err := s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(profiles))

	// global file setup without custom settings
	const (
		globalTemplateWithoutCustomSettings = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`
	)

	globalFile, err = os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplateWithoutCustomSettings)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	// Check profile does not exist
	profiles, err = s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(profiles))
}

func (s *enterpriseIntegrationGitopsTestSuite) TestMacOSSetup() {
	t := s.T()
	ctx := context.Background()

	originalAppConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := s.DS.SaveAppConfig(ctx, originalAppConfig)
		require.NoError(t, err)
	})

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/signed.pkg")
	}))
	defer bootstrapServer.Close()

	const (
		globalConfig = `
agent_options:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		globalConfigOnly = `
agent_options:
controls:
  setup_experience:
    macos_bootstrap_package: %s
    macos_manual_agent_install: %t
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		noTeamConfig = `name: Unassigned
controls:
  setup_experience:
    macos_bootstrap_package: %s
    macos_manual_agent_install: true
policies:
software:
`

		teamConfig = `
controls:
  setup_experience:
    macos_bootstrap_package: %s
    macos_manual_agent_install: %t
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalConfig)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamConfig, bootstrapServer.URL))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamConfig, bootstrapServer.URL, true, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)
	teamFileClear, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFileClear.WriteString(fmt.Sprintf(teamConfig, bootstrapServer.URL, false, teamName))
	require.NoError(t, err)
	err = teamFileClear.Close()
	require.NoError(t, err)

	globalFileOnlySet, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileOnlySet.WriteString(fmt.Sprintf(globalConfigOnly, bootstrapServer.URL, true))
	require.NoError(t, err)
	err = globalFileOnlySet.Close()
	require.NoError(t, err)
	globalFileOnlyClear, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileOnlyClear.WriteString(fmt.Sprintf(globalConfigOnly, bootstrapServer.URL, false))
	require.NoError(t, err)
	err = globalFileOnlyClear.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}))

	appConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.True(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.True(t, team.Config.MDM.MacOSSetup.ManualAgentInstall.Value)

	// Apply global configs without no-team
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlyClear.Name(), "-f", teamFileClear.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlyClear.Name(), "-f", teamFileClear.Name()}))
	appConfig, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.False(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)
	team, err = s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.False(t, team.Config.MDM.MacOSSetup.ManualAgentInstall.Value)

	// Apply global configs only
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlySet.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlySet.Name()}))
	appConfig, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.True(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestFleetGitOpsDeletesNonManagedLabels() {
	t := s.T()
	ctx := context.Background()

	// Set the required environment variables
	t.Setenv("FLEET_SERVER_URL", s.Server.URL)
	t.Setenv("ORG_NAME", "Around the block")

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	var someUser fleet.User
	for _, u := range s.Users {
		someUser = u
		break
	}

	// 'nonManagedLabel' is associated with a software installer and is
	// not managed in the ops file so it should be deleted.
	nonManagedLabel, err := s.DS.NewLabel(ctx, &fleet.Label{
		Name:  t.Name(),
		Query: "bye bye label",
	})
	require.NoError(t, err)

	installer, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)

	_, _, err = s.DS.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install zoo",
		InstallerFile: installer,
		StorageID:     uuid.NewString(),
		Filename:      "zoo.pkg",
		Title:         "zoo",
		Source:        "apps",
		Version:       "0.0.1",
		UserID:        someUser.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{
			LabelScope: fleet.LabelScopeIncludeAny,
			ByName: map[string]fleet.LabelIdent{nonManagedLabel.Name: {
				LabelID:   nonManagedLabel.ID,
				LabelName: nonManagedLabel.Name,
			}},
		},
	})
	require.NoError(t, err)

	opsFile := path.Join("..", "..", "fleetctl", "testdata", "gitops", "global_config_no_paths.yml")
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", opsFile})

	// Check label was removed successfully
	result, err := s.DS.LabelIDsByName(ctx, []string{nonManagedLabel.Name}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Empty(t, result)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestMacOSSetupScriptWithFleetSecret() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const secretName = "MY_SECRET"
	const secretValue = "my-secret-value"

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_SECRET_"+secretName, secretValue)

	// Create a script file that uses the fleet secret
	scriptFile, err := os.CreateTemp(t.TempDir(), "*.sh")
	require.NoError(t, err)
	_, err = scriptFile.WriteString(`echo "Using secret: $FLEET_SECRET_` + secretName)
	require.NoError(t, err)
	err = scriptFile.Close()
	require.NoError(t, err)

	// Create a no-team file with the script
	const noTeamTemplate = `name: Unassigned
policies:
controls:
  setup_experience:
    macos_script: %s
software:
`
	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, scriptFile.Name()))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	// Create a global file
	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(`
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`)
	require.NoError(t, err)

	// Apply the configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath}))

	// Verify the script was saved
	_, err = s.DS.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)

	// Verify the secret was saved
	secretVariables, err := s.DS.GetSecretVariables(ctx, []string{secretName})
	require.NoError(t, err)
	require.Equal(t, secretVariables[0].Name, secretName)
	require.Equal(t, secretVariables[0].Value, secretValue)
}

// TestEnvSubstitutionInProfiles tests that only FLEET_SECRET_ prefixed env vars are saved as secrets
func (s *enterpriseIntegrationGitopsTestSuite) TestEnvSubstitutionInProfiles() {
	t := s.T()
	ctx := t.Context()
	tempDir := t.TempDir()

	// Create a test configuration profile with both valid and invalid secret references
	profileContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadDisplayName</key>
            <string>Test Profile</string>
            <key>PayloadIdentifier</key>
            <string>com.fleet.test.env</string>
            <key>PayloadType</key>
            <string>Configuration</string>
            <key>PayloadUUID</key>
            <string>12345678-1234-1234-1234-123456789012</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>TestSecretValue</key>
            <string>$FLEET_SECRET_TEST_SECRET</string>
            <key>TestInvalidSecret</key>
            <string>$FLEET_DUO_CERTIFICATE_SECRET</string>
            <key>TestPlainValue</key>
            <string>$HOME</string>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Test Profile</string>
    <key>PayloadIdentifier</key>
    <string>com.fleet.test.env</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>12345678-1234-1234-1234-123456789012</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`

	// Write the profile to a file
	profilePath := filepath.Join(tempDir, "test-profile.mobileconfig")
	err := os.WriteFile(profilePath, []byte(profileContent), 0o644) //nolint:gosec // test code
	require.NoError(t, err)

	// Create a GitOps config file that references the profile
	// Note: Environment variables in the YAML config itself get expanded,
	// but not in the referenced profile files
	gitopsConfig := fmt.Sprintf(`
org_settings:
  server_settings:
    server_url: %s
  secrets:
    - secret: test_secret
agent_options:
  config:
    decorators:
      load:
        - SELECT uuid AS host_uuid FROM system_info;
controls:
  apple_settings:
    configuration_profiles:
      - path: %s
reports: []
policies: []
`, s.Server.URL, profilePath)

	configPath := filepath.Join(tempDir, "gitops.yml")
	err = os.WriteFile(configPath, []byte(gitopsConfig), 0o644) //nolint:gosec // test code
	require.NoError(t, err)

	// Create a GitOps user
	gitOpsUser := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, gitOpsUser)

	// Set the environment variable for the valid secret
	t.Setenv("FLEET_SECRET_TEST_SECRET", "super_secret_value_123")
	t.Setenv("FLEET_DUO_CERTIFICATE_SECRET", "should_not_be_saved")
	t.Setenv("HOME", "also_not_saved")

	// Run GitOps dry-run - should fail without the required secret
	// First, unset the environment variable to trigger the error
	_ = os.Unsetenv("FLEET_SECRET_TEST_SECRET")
	_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", configPath, "--dry-run"})
	require.ErrorContains(t, err, "FLEET_SECRET_TEST_SECRET")

	// Set the env var again and run for real
	t.Setenv("FLEET_SECRET_TEST_SECRET", "super_secret_value_123")
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", configPath})

	// Verify that the secret was saved to the server
	secrets, err := s.DS.GetSecretVariables(ctx, []string{"TEST_SECRET"})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "TEST_SECRET", secrets[0].Name)
	assert.Equal(t, "super_secret_value_123", secrets[0].Value)

	// Verify that non-FLEET_SECRET_ variables were NOT saved
	notSaved, err := s.DS.GetSecretVariables(ctx, []string{"DUO_CERTIFICATE_SECRET", "HOME"})
	require.NoError(t, err)
	assert.Empty(t, notSaved, "Non-FLEET_SECRET_ variables should not be saved")

	// Verify that the profile content has the expected substitutions:
	// - $FLEET_SECRET_* variables should remain as-is (substituted at delivery time)
	// - Other env vars should be expanded during GitOps
	profiles, err := s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)

	foundProfile := false
	for _, profile := range profiles {
		t.Logf("Found profile: %s", profile.Name)
		if strings.Contains(profile.Name, "test-profile") || strings.Contains(profile.Identifier, "com.fleet.test.env") {
			foundProfile = true
			// $FLEET_SECRET_* variables should NOT be expanded (they're expanded at delivery time)
			assert.Contains(t, string(profile.Mobileconfig), "$FLEET_SECRET_TEST_SECRET")
			// Non-FLEET_SECRET_/FLEET_VAR_ variables SHOULD be expanded during GitOps
			assert.Contains(t, string(profile.Mobileconfig), "should_not_be_saved") // Value of $FLEET_DUO_CERTIFICATE_SECRET
			assert.Contains(t, string(profile.Mobileconfig), "also_not_saved")      // Value of $HOME
			// The original variable names should NOT be present
			assert.NotContains(t, string(profile.Mobileconfig), "$FLEET_DUO_CERTIFICATE_SECRET")
			assert.NotContains(t, string(profile.Mobileconfig), "$HOME")
			break
		}
	}
	assert.True(t, foundProfile, "Profile should be uploaded to the server")
}

// TestFleetSecretInDataTag tests that FLEET_SECRET_ variables in <data> tags of Apple profiles
// are handled properly.
func (s *enterpriseIntegrationGitopsTestSuite) TestFleetSecretInDataTag() {
	t := s.T()
	tempDir := t.TempDir()
	ctx := context.Background()

	// Sample certificate in base64 format (this is a dummy test certificate)
	testCertBase64 := `MIIDaTCCAlGgAwIBAgIUNQLezMUpmZK18DcLKt/XTRcLlK8wDQYJKoZIhvcNAQELBQAwRDEfMB0GA1UEAwwWRHVtbXkgVGVzdCBDZXJ0aWZpY2F0ZTEUMBIGA1UECgwLRXhhbXBsZSBPcmcxCzAJBgNVBAYTAlVTMB4XDTI1MDgxOTE3NDMwN1oXDTI2MDgxOTE3NDMwN1owRDEfMB0GA1UEAwwWRHVtbXkgVGVzdCBDZXJ0aWZpY2F0ZTEUMBIGA1UECgwLRXhhbXBsZSBPcmcxCzAJBgNVBAYTAlVTMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA07Np/w5WpFVLlMKX3dZSxwo+c2uwP2glTN0HA5c/6UOQRR9c91yoGGJsD4pfqhtIMSTFw7po3n/PjhGDe/WH+utK+ZIcD0nGD6SvmOggyoohHs81eIOjJAEJjxzhk7eLTVpUI2EnPe/24ei/dgkK59As9qQyH/y+CoR8JIYbNCJH5YLC2Pa44V84QWa2I5DHKUKrUXo9WsrRp1N1JjyaG/6hxLBJZ69e0QTrxxScboreRqVUR6oIEJRTchB+rDG5dxXzCQE6/F8N3qR76t23wd3CLmrcXoEc1P2P331Qzi0KXNXjdJFf0plmfRkT/IWgfM81Vfon1QwENwRSBNmPfQIDAQABo1MwUTAdBgNVHQ4EFgQU9q7SDfQRbJ31snRt2sZzx5sdEpYwHwYDVR0jBBgwFoAU9q7SDfQRbJ31snRt2sZzx5sdEpYwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAYwH42JP45SnZejSF74OcYt8fp08jCWHOiFC3QEo3cROXVn6AWjEbzuQpOxRWF9EizNA83c4E6I+kQVztiuv9bUKGuLFeYb9lUZe8HOvH+j22MtGvZrDPygsJc8TavdkxAsu6OiNQZrYiCFzixkKS9b5p/1B93GBh62OFnV1nUBS8PzAZhOAyJ8UcEhr+GNzZG99/wOkcB0uwxmIb8x8sB3KnQ0qef/qnmgeWxlJlDc/SZ2/4PgtaluZ+noDfNPzaQn4eJNnBz0OTqZ9yuKALeE1WHk8U13zSdc1GNVLhXOrEHegPK5bBmA/lpIQ6HrkwUX7MJ3vK0AD3LjaTzXltDQ==`

	// Create a test team first
	team, err := s.DS.NewTeam(ctx, &fleet.Team{
		Name: "Test Team for Secret in Data Tag",
	})
	require.NoError(t, err)

	// Create a profile with $FLEET_SECRET_DUO_CERTIFICATE in a <data> tag
	// This mimics the real-world scenario where the certificate should be base64 encoded
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

	// Write the profile to a file
	profilePath := filepath.Join(tempDir, "rootcert-secret.mobileconfig")
	err = os.WriteFile(profilePath, []byte(profileContent), 0o644) //nolint:gosec
	require.NoError(t, err)

	// Create a team GitOps config file that references the profile
	teamConfig := fmt.Sprintf(`
name: %s
settings:
  secrets:
    - secret: test_secret
agent_options:
  config:
    decorators:
      load:
        - SELECT uuid AS host_uuid FROM system_info;
controls:
  apple_settings:
    configuration_profiles:
      - path: %s
reports:
policies:
software:
`, team.Name, profilePath)

	configPath := filepath.Join(tempDir, "team-gitops.yml")
	err = os.WriteFile(configPath, []byte(teamConfig), 0o644) //nolint:gosec
	require.NoError(t, err)

	// Create a GitOps user
	gitOpsUser := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, gitOpsUser)

	// Set the environment variable with the base64-encoded certificate
	t.Setenv("FLEET_SECRET_DUO_CERTIFICATE", testCertBase64)

	// The fix expands FLEET_SECRET_ variables for validation only, allowing the profile to be parsed
	_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", configPath, "--dry-run"})
	require.NoError(t, err, "GitOps dry-run should succeed with the fix")

	// Also test without dry-run to confirm it works
	_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", configPath})
	require.NoError(t, err, "GitOps should succeed with the fix")

	// Verify that the profile stored on the server still has the unexpanded variable
	profiles, err := s.DS.ListMDMAppleConfigProfiles(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	// The stored profile should still contain the unexpanded variable, not the actual secret
	require.Contains(t, string(profiles[0].Mobileconfig), "$FLEET_SECRET_DUO_CERTIFICATE",
		"Profile should still contain unexpanded FLEET_SECRET variable")
}

func (s *enterpriseIntegrationGitopsTestSuite) TestAddManualLabels() {
	t := s.T()
	ctx := context.Background()

	user := fleet.User{
		Name:       "Admin User",
		Email:      uuid.NewString() + "@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.DS.NewUser(context.Background(), &user)
	require.NoError(t, err)

	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Add some hosts
	host1, err := s.DS.NewHost(context.Background(), &fleet.Host{
		UUID:           "uuid-1",
		Hostname:       "host1",
		Platform:       "linux",
		HardwareSerial: "serial1",
	})
	require.NoError(t, err)
	host2, err := s.DS.NewHost(context.Background(), &fleet.Host{
		UUID:           "uuid-2",
		Hostname:       "host2",
		Platform:       "linux",
		HardwareSerial: "serial2",
	})
	require.NoError(t, err)
	host3, err := s.DS.NewHost(context.Background(), &fleet.Host{
		UUID:           "uuid-3",
		Hostname:       "host3",
		Platform:       "linux",
		HardwareSerial: "serial3",
	})
	require.NoError(t, err)
	host4, err := s.DS.NewHost(context.Background(), &fleet.Host{
		UUID:           "uuid-4",
		Hostname:       "host4",
		Platform:       "linux",
		HardwareSerial: "serial4",
	})
	require.NoError(t, err)
	// Add a host whose UUID starts with the ID of host4 (probably ID 4,
	// but get it from the record just in case.)
	// host4 should _not_ be added to the label (see issue #34236).
	host5, err := s.DS.NewHost(context.Background(), &fleet.Host{
		UUID:           fmt.Sprintf("%duuid-5", host4.ID),
		Hostname:       "dummy",
		Platform:       "linux",
		HardwareSerial: "dummy",
	})
	require.NoError(t, err)

	// Create a global file
	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(fmt.Sprintf(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
  - name: my-label
    label_membership_type: manual
    hosts:
    - %s
    - %s
    - %d
    - %s
    - dummy
`, host1.Hostname, host2.HardwareSerial, host3.ID, host5.UUID))
	require.NoError(t, err)

	// Apply the configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	// Verify the label was created and has the correct hosts
	labels, err := s.DS.LabelsByName(ctx, []string{"my-label"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, labels, 1)
	label := labels["my-label"]
	// Get the hosts for the label
	labelHosts, err := s.DS.ListHostsInLabel(ctx, fleet.TeamFilter{User: &user}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, labelHosts, 4)
	// Get the IDs of the hosts
	var labelHostIDs []uint
	for _, h := range labelHosts {
		labelHostIDs = append(labelHostIDs, h.ID)
	}
	// Verify the correct hosts were added to the label
	require.ElementsMatch(t, labelHostIDs, []uint{host1.ID, host2.ID, host3.ID, host5.ID})
}

func (s *enterpriseIntegrationGitopsTestSuite) TestManualLabelOmitHostsPreservesMembership() {
	t := s.T()
	ctx := context.Background()

	user := fleet.User{
		Name:       "Admin User",
		Email:      uuid.NewString() + "@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.DS.NewUser(context.Background(), &user)
	require.NoError(t, err)

	fleetctlConfig := s.createFleetctlConfig(t, user)

	host1, err := s.DS.NewHost(ctx, &fleet.Host{
		UUID:           "omit-hosts-uuid-1",
		Hostname:       "omit-hosts-host1",
		Platform:       "linux",
		HardwareSerial: "omit-hosts-serial1",
	})
	require.NoError(t, err)
	host2, err := s.DS.NewHost(ctx, &fleet.Host{
		UUID:           "omit-hosts-uuid-2",
		Hostname:       "omit-hosts-host2",
		Platform:       "linux",
		HardwareSerial: "omit-hosts-serial2",
	})
	require.NoError(t, err)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	// Step 1: Apply a manual label with hosts.
	_, err = globalFile.WriteString(fmt.Sprintf(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
  - name: preserve-membership-label
    description: A manual label
    label_membership_type: manual
    hosts:
    - %s
    - %s
`, host1.Hostname, host2.Hostname))
	require.NoError(t, err)

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	labels, err := s.DS.LabelsByName(ctx, []string{"preserve-membership-label"}, fleet.TeamFilter{})
	require.NoError(t, err)
	label := labels["preserve-membership-label"]
	require.NotNil(t, label)
	labelHosts, err := s.DS.ListHostsInLabel(ctx, fleet.TeamFilter{User: &user}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, labelHosts, 2)

	// Step 2: Re-apply the same label without the hosts key.
	require.NoError(t, os.WriteFile(globalFile.Name(), []byte(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
  - name: preserve-membership-label
    description: A manual label
    label_membership_type: manual
`), 0o644))

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	// Verify the hosts are still attached.
	labelHosts, err = s.DS.ListHostsInLabel(ctx, fleet.TeamFilter{User: &user}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, labelHosts, 2, "omitting hosts key should preserve existing membership")
	var hostIDs []uint
	for _, h := range labelHosts {
		hostIDs = append(hostIDs, h.ID)
	}
	require.ElementsMatch(t, hostIDs, []uint{host1.ID, host2.ID})

	// Step 3: Re-apply with empty hosts to clear membership.
	require.NoError(t, os.WriteFile(globalFile.Name(), []byte(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
  - name: preserve-membership-label
    description: A manual label
    label_membership_type: manual
    hosts: []
`), 0o644))

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}))

	// Verify hosts were cleared.
	labelHosts, err = s.DS.ListHostsInLabel(ctx, fleet.TeamFilter{User: &user}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, labelHosts, 0, "empty hosts list should clear membership")
}

func (s *enterpriseIntegrationGitopsTestSuite) TestIPASoftwareInstallers() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	lbl, err := s.DS.NewLabel(ctx, &fleet.Label{Name: "Label1", Query: "SELECT 1"})
	require.NoError(t, err)
	require.NotZero(t, lbl.ID)

	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
labels:
  - name: Label1
    label_membership_type: dynamic
    query: SELECT 1
`

		noTeamTemplate = `name: Unassigned
controls:
policies:
software:
  packages:
%s
`
		teamTemplate = `
controls:
software:
  packages:
%s
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	// create an .ipa software for the no-team config
	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, `
      - url: ${SOFTWARE_INSTALLER_URL}/ipa_test.ipa
        self_service: true
`))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath}))

	// the ipa installer was created for no team
	titles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)

	require.Len(t, titles, 2)
	var sources, platforms []string
	for _, title := range titles {
		require.Equal(t, "ipa_test", title.Name)
		require.NotNil(t, title.BundleIdentifier)
		require.Equal(t, "com.ipa-test.ipa-test", *title.BundleIdentifier)
		sources = append(sources, title.Source)

		require.NotNil(t, title.SoftwarePackage)
		platforms = append(platforms, title.SoftwarePackage.Platform)
		require.Equal(t, "ipa_test.ipa", title.SoftwarePackage.Name)

		meta, err := s.DS.GetInHouseAppMetadataByTeamAndTitleID(ctx, nil, title.ID)
		require.NoError(t, err)
		require.True(t, meta.SelfService)
		require.Empty(t, meta.LabelsExcludeAny)
		require.Empty(t, meta.LabelsIncludeAny)
		require.Empty(t, meta.LabelsIncludeAll)
	}
	require.ElementsMatch(t, []string{"ios_apps", "ipados_apps"}, sources)
	require.ElementsMatch(t, []string{"ios", "ipados"}, platforms)

	// create a dummy install script, should be ignored for ipa apps
	scriptFile, err := os.CreateTemp(t.TempDir(), "*.sh")
	require.NoError(t, err)
	_, err = scriptFile.WriteString(`echo "dummy install script"`)
	require.NoError(t, err)
	err = scriptFile.Close()
	require.NoError(t, err)

	// create an .ipa software for the team config
	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamTemplate, `
      - url: ${SOFTWARE_INSTALLER_URL}/ipa_test.ipa
        self_service: false
        install_script:
          path: `+scriptFile.Name()+`
        labels_include_any:
          - Label1
`, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	// get the team ID
	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// the ipa installer was created for the team
	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)

	require.Len(t, titles, 2)
	sources, platforms = []string{}, []string{}
	for _, title := range titles {
		require.Equal(t, "ipa_test", title.Name)
		require.NotNil(t, title.BundleIdentifier)
		require.Equal(t, "com.ipa-test.ipa-test", *title.BundleIdentifier)
		sources = append(sources, title.Source)

		require.NotNil(t, title.SoftwarePackage)
		platforms = append(platforms, title.SoftwarePackage.Platform)
		require.Equal(t, "ipa_test.ipa", title.SoftwarePackage.Name)

		meta, err := s.DS.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, title.ID)
		require.NoError(t, err)
		require.False(t, meta.SelfService)
		require.Empty(t, meta.LabelsExcludeAny)
		require.Empty(t, meta.LabelsIncludeAll)
		require.Len(t, meta.LabelsIncludeAny, 1)
		require.Equal(t, lbl.ID, meta.LabelsIncludeAny[0].LabelID)
		require.Empty(t, meta.InstallScript) // install script should be ignored for ipa apps
	}
	require.ElementsMatch(t, []string{"ios_apps", "ipados_apps"}, sources)
	require.ElementsMatch(t, []string{"ios", "ipados"}, platforms)

	// update the team config to clear the label condition
	err = os.WriteFile(teamFile.Name(), []byte(fmt.Sprintf(teamTemplate, `
      - url: ${SOFTWARE_INSTALLER_URL}/ipa_test.ipa
        labels_include_any:
`, teamName)), 0o644)
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	// the ipa installer was created for the team
	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)

	require.Len(t, titles, 2)
	sources, platforms = []string{}, []string{}
	for _, title := range titles {
		require.Equal(t, "ipa_test", title.Name)
		require.NotNil(t, title.BundleIdentifier)
		require.Equal(t, "com.ipa-test.ipa-test", *title.BundleIdentifier)
		sources = append(sources, title.Source)

		require.NotNil(t, title.SoftwarePackage)
		platforms = append(platforms, title.SoftwarePackage.Platform)
		require.Equal(t, "ipa_test.ipa", title.SoftwarePackage.Name)

		meta, err := s.DS.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, title.ID)
		require.NoError(t, err)
		require.False(t, meta.SelfService)
		require.Empty(t, meta.LabelsExcludeAny)
		require.Empty(t, meta.LabelsIncludeAny)
		require.Empty(t, meta.LabelsIncludeAll)
	}
	require.ElementsMatch(t, []string{"ios_apps", "ipados_apps"}, sources)
	require.ElementsMatch(t, []string{"ios", "ipados"}, platforms)

	// update the team config to clear all installers
	err = os.WriteFile(teamFile.Name(), []byte(fmt.Sprintf(teamTemplate, "", teamName)), 0o644)
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}))

	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 0)
}

// TestGitOpsSoftwareDisplayName tests that display names for software packages and VPP apps
// are properly applied via GitOps.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareDisplayName() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		noTeamTemplate = `name: Unassigned
controls:
policies:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
      display_name: Custom Ruby Name
`

		teamTemplate = `
controls:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
      display_name: Team Custom Ruby
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(noTeamTemplate)
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamTemplate, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}))

	// get the team ID
	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// Verify display name for no team
	noTeamTitles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, noTeamTitles, 1)
	require.NotNil(t, noTeamTitles[0].SoftwarePackage)
	noTeamTitleID := noTeamTitles[0].ID

	// Verify the display name is stored in the database for no team
	var noTeamDisplayName string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &noTeamDisplayName,
			"SELECT display_name FROM software_title_display_names WHERE team_id = ? AND software_title_id = ?",
			0, noTeamTitleID)
	})
	require.Equal(t, "Custom Ruby Name", noTeamDisplayName)

	// Verify display name for team
	teamTitles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team.ID}, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, teamTitles, 1)
	require.NotNil(t, teamTitles[0].SoftwarePackage)
	teamTitleID := teamTitles[0].ID

	// Verify the display name is stored in the database for team
	var teamDisplayName string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &teamDisplayName,
			"SELECT display_name FROM software_title_display_names WHERE team_id = ? AND software_title_id = ?",
			team.ID, teamTitleID)
	})
	require.Equal(t, "Team Custom Ruby", teamDisplayName)
}

// TestGitOpsSoftwareIcons tests that custom icons for software packages
// and fleet maintained apps are properly applied via GitOps.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareIcons() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		noTeamTemplate = `name: Unassigned
controls:
policies:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
      icon:
        path: %s/testdata/gitops/lib/icon.png
  fleet_maintained_apps:
    - slug: foo/darwin
      icon:
        path: %s/testdata/gitops/lib/icon.png
`

		teamTemplate = `
controls:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
      icon:
        path: %s/testdata/gitops/lib/icon.png
  fleet_maintained_apps:
    - slug: foo/darwin
      icon:
        path: %s/testdata/gitops/lib/icon.png
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	// Get the absolute path to the directory of this test file
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get runtime caller info")
	dirPath := filepath.Dir(currentFile)
	dirPath = filepath.Join(dirPath, "../../fleetctl")
	dirPath, err := filepath.Abs(filepath.Clean(dirPath))
	require.NoError(t, err)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(noTeamFile, noTeamTemplate, dirPath, dirPath)
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(teamFile, teamTemplate, dirPath, dirPath, teamName)
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	// Mock server to serve fleet maintained app installer
	installerBytes := []byte("foo")
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(installerBytes)
	}))
	defer installerServer.Close()

	// Mock server to serve fleet maintained app manifest
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "6.0",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       installerServer.URL + "/foo.pkg",
			InstallScriptRef:   "foobaz",
			UninstallScriptRef: "foobaz",
			SHA256:             "no_check", // See ma.noCheckHash
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs: map[string]string{
				"foobaz": "Hello World!",
			},
		}

		err := json.NewEncoder(w).Encode(manifest)
		require.NoError(t, err)
	}))

	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL, t)

	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
			VALUES ('foo', 'foo/darwin', 'darwin', 'com.example.foo')`)
		return err
	})

	// Apply configs
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}))

	// Get the team ID
	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// Verify titles were added for no team
	noTeamTitles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, noTeamTitles, 2)
	require.NotNil(t, noTeamTitles[0].SoftwarePackage)
	require.NotNil(t, noTeamTitles[1].SoftwarePackage)
	noTeamTitleIDs := []uint{noTeamTitles[0].ID, noTeamTitles[1].ID}

	// Verify the custom icon is stored in the database for no team
	var noTeamIconFilenames []string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		stmt, args, err := sqlx.In("SELECT filename FROM software_title_icons WHERE team_id = ? AND software_title_id IN (?)", 0, noTeamTitleIDs)
		if err != nil {
			return err
		}
		return sqlx.SelectContext(ctx, q, &noTeamIconFilenames, stmt, args...)
	})
	require.Len(t, noTeamIconFilenames, 2)
	require.Equal(t, "icon.png", noTeamIconFilenames[0])
	require.Equal(t, "icon.png", noTeamIconFilenames[1])

	// Verify titles were added for team
	teamTitles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team.ID}, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, teamTitles, 2)
	require.NotNil(t, teamTitles[0].SoftwarePackage)
	require.NotNil(t, teamTitles[1].SoftwarePackage)
	teamTitleIDs := []uint{teamTitles[0].ID, teamTitles[1].ID}

	// Verify the custom icon is stored in the database for team
	var teamIconFilenames []string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		stmt, args, err := sqlx.In("SELECT filename FROM software_title_icons WHERE team_id = ? AND software_title_id IN (?)", team.ID, teamTitleIDs)
		if err != nil {
			return err
		}
		return sqlx.SelectContext(ctx, q, &teamIconFilenames, stmt, args...)
	})
	require.Len(t, teamIconFilenames, 2)
	require.Equal(t, "icon.png", teamIconFilenames[0])
	require.Equal(t, "icon.png", teamIconFilenames[1])
}

// TestGitOpsIconRecoversAfterClearingStaleHash documents the recovery path
// when an icon row's storage_id refers to bytes that are no longer in the
// store. The planner short-circuits any subsequent gitops run because the
// row's hash still matches the local YAML; clearing the row's storage_id
// forces the next run to re-upload.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsIconRecoversAfterClearingStaleHash() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		teamTemplate = `
controls:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
      icon:
        path: %s/testdata/gitops/lib/icon.png
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	// Resolve the absolute path to the testdata icon used by the YAML.
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get runtime caller info")
	dirPath := filepath.Dir(currentFile)
	dirPath = filepath.Join(dirPath, "../../fleetctl")
	dirPath, err := filepath.Abs(filepath.Clean(dirPath))
	require.NoError(t, err)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	require.NoError(t, globalFile.Close())

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(teamFile, teamTemplate, dirPath, teamName)
	require.NoError(t, err)
	require.NoError(t, teamFile.Close())

	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	firstRunOut := fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", teamFile.Name(),
	})
	s.assertRealRunOutput(t, firstRunOut)
	require.Contains(t, firstRunOut, "set icons on 1 software title")

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	titles, _, _, err := s.DS.ListSoftwareTitles(ctx,
		fleet.SoftwareTitleListOptions{TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 1)

	var storageID string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &storageID,
			"SELECT storage_id FROM software_title_icons WHERE team_id = ? AND software_title_id = ?",
			team.ID, titles[0].ID)
	})
	require.NotEmpty(t, storageID)

	iconPath := filepath.Join(s.iconDir, "software-title-icons", storageID)
	info, err := os.Stat(iconPath)
	require.NoError(t, err)
	originalSize := info.Size()
	require.Positive(t, originalSize)

	// Truncate the bytes on disk while leaving the icon row intact.
	require.NoError(t, os.Truncate(iconPath, 0))

	// A second gitops run with the unchanged YAML silently no-ops because
	// the planner short-circuits before any API call. This pins down that
	// known limitation; the recovery below is what restores the icon.
	secondRunOut := fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", teamFile.Name(),
	})
	s.assertRealRunOutput(t, secondRunOut)
	require.Contains(t, secondRunOut, "set icons on 0 software titles")
	info, err = os.Stat(iconPath)
	require.NoError(t, err)
	require.Equal(t, int64(0), info.Size())

	// Recovery: clearing storage_id forces a hash mismatch and re-upload
	// on the next run.
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			"UPDATE software_title_icons SET storage_id = '' WHERE team_id = ? AND software_title_id = ?",
			team.ID, titles[0].ID)
		return err
	})

	thirdRunOut := fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", teamFile.Name(),
	})
	s.assertRealRunOutput(t, thirdRunOut)
	require.Contains(t, thirdRunOut, "set icons on 1 software title")

	info, err = os.Stat(iconPath)
	require.NoError(t, err)
	require.Equal(t, originalSize, info.Size())
}

// TestGitOpsIconUpdateRecoversWhenBytesAreMissing exercises the gitops
// client's fallback from a metadata-only icon update to a full upload
// when the server reports the bytes for a known hash are missing. Two
// titles in the same team share an icon. After the first run we mutate
// one title's icon row to point at a hash with no bytes and truncate
// the real bytes; the next gitops run forces that title through the
// update path with a hash whose bytes are missing, and the client must
// recover by re-uploading.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsIconUpdateRecoversWhenBytesAreMissing() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		teamTemplate = `
controls:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
      icon:
        path: %s/testdata/gitops/lib/icon.png
    - url: ${SOFTWARE_INSTALLER_URL}/vim.deb
      icon:
        path: %s/testdata/gitops/lib/icon.png
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get runtime caller info")
	dirPath := filepath.Dir(currentFile)
	dirPath = filepath.Join(dirPath, "../../fleetctl")
	dirPath, err := filepath.Abs(filepath.Clean(dirPath))
	require.NoError(t, err)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	require.NoError(t, globalFile.Close())

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(teamFile, teamTemplate, dirPath, dirPath, teamName)
	require.NoError(t, err)
	require.NoError(t, teamFile.Close())

	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	firstRunOut := fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", teamFile.Name(),
	})
	s.assertRealRunOutput(t, firstRunOut)
	require.Contains(t, firstRunOut, "set icons on 2 software titles")

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	titles, _, _, err := s.DS.ListSoftwareTitles(ctx,
		fleet.SoftwareTitleListOptions{TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 2)

	var storageIDs []string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &storageIDs,
			"SELECT DISTINCT storage_id FROM software_title_icons WHERE team_id = ?", team.ID)
	})
	require.Len(t, storageIDs, 1)
	storageID := storageIDs[0]

	iconPath := filepath.Join(s.iconDir, "software-title-icons", storageID)
	info, err := os.Stat(iconPath)
	require.NoError(t, err)
	originalSize := info.Size()
	require.Positive(t, originalSize)

	// Force divergence: redirect one title's icon row to a fake hash
	// (no bytes will exist at that storage id) and truncate the real
	// bytes so the integrity check on the genuine hash also fails. On
	// the next gitops run, the title with the fake storage_id will be
	// routed through IconsToUpdate (its server hash differs from the
	// local hash, but the local hash is in UploadedHashes from the
	// other title's row), and the server will refuse the metadata-only
	// update because the bytes for the local hash are gone.
	const fakeHash = "deadbeef0000000000000000000000000000000000000000000000000000beef"
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			"UPDATE software_title_icons SET storage_id = ? WHERE team_id = ? AND software_title_id = ?",
			fakeHash, team.ID, titles[0].ID)
		return err
	})
	require.NoError(t, os.Truncate(iconPath, 0))

	secondRunOut := fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", teamFile.Name(),
	})
	s.assertRealRunOutput(t, secondRunOut)

	info, err = os.Stat(iconPath)
	require.NoError(t, err)
	require.Equal(t, originalSize, info.Size())

	var afterStorageIDs []string
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &afterStorageIDs,
			"SELECT DISTINCT storage_id FROM software_title_icons WHERE team_id = ?", team.ID)
	})
	require.Len(t, afterStorageIDs, 1)
	require.Equal(t, storageID, afterStorageIDs[0])
}

// TestGitOpsTeamLabels tests operations around team labels
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsTeamLabels() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetCfg := s.createFleetctlConfig(t, user)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	// -----------------------------------------------------------------
	// First, let's validate that we can add labels to the global scope
	// -----------------------------------------------------------------
	require.NoError(t, os.WriteFile(globalFile.Name(), []byte(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
  - name: global-label-one
    label_membership_type: dynamic
    query: SELECT 1
  - name: global-label-two
    label_membership_type: dynamic
    query: SELECT 1
`), 0o644))

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name()}))

	expected := make(map[string]uint)
	expected["global-label-one"] = 0
	expected["global-label-two"] = 0

	got := labelTeamIDResult(t, s, ctx)

	require.True(t, maps.Equal(expected, got))

	// ---------------------------------------------------------------
	// Now, let's validate that we can add and remove labels in a team
	// ---------------------------------------------------------------
	// TeamOne already exists
	teamOneName := uuid.NewString()
	teamOne, err := s.DS.NewTeam(context.Background(), &fleet.Team{Name: teamOneName})
	require.NoError(t, err)

	teamOneFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(teamOneFile.Name(), fmt.Appendf(nil,
		`
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
labels:
  - name: team-one-label-one
    label_membership_type: dynamic
    query: SELECT 2
  - name: team-one-label-two
    label_membership_type: dynamic
    query: SELECT 3
`, teamOneName), 0o644))

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", teamOneFile.Name(), "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", teamOneFile.Name()}))

	got = labelTeamIDResult(t, s, ctx)

	expected = make(map[string]uint)
	expected["global-label-one"] = 0
	expected["global-label-two"] = 0
	expected["team-one-label-one"] = teamOne.ID
	expected["team-one-label-two"] = teamOne.ID

	require.True(t, maps.Equal(expected, got))

	// Try removing one label from teamOne
	require.NoError(t, os.WriteFile(teamOneFile.Name(), fmt.Appendf(nil,
		`
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
labels:
  - name: team-one-label-one
    label_membership_type: dynamic
    query: SELECT 2
`, teamOneName), 0o644))

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name()}))

	expected = make(map[string]uint)
	expected["global-label-one"] = 0
	expected["global-label-two"] = 0
	expected["team-one-label-one"] = teamOne.ID

	got = labelTeamIDResult(t, s, ctx)

	require.True(t, maps.Equal(expected, got))

	// ------------------------------------------------
	// Finally, let's validate that we can move labels around
	// ------------------------------------------------
	require.NoError(t, os.WriteFile(globalFile.Name(), []byte(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
  - name: global-label-one
    label_membership_type: dynamic
    query: SELECT 1

`), 0o644))

	require.NoError(t, os.WriteFile(teamOneFile.Name(), fmt.Appendf(nil,

		`
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
labels:
  - name: team-one-label-two
    label_membership_type: dynamic
    query: SELECT 3
  - name: global-label-two
    label_membership_type: dynamic
    query: SELECT 1
`, teamOneName), 0o644))

	teamTwoName := uuid.NewString()
	teamTwoFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(teamTwoFile.Name(), fmt.Appendf(nil, `
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret2"}]
labels:
  - name: team-one-label-one
    label_membership_type: dynamic
    query: SELECT 2
`, teamTwoName), 0o644))

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name(), "-f", teamTwoFile.Name(), "--dry-run"}))

	// TODO: Seems like we require two passes to achieve equilibrium?
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name(), "-f", teamTwoFile.Name()}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name(), "-f", teamTwoFile.Name()}))

	teamTwo, err := s.DS.TeamByName(ctx, teamTwoName)
	require.NoError(t, err)

	got = labelTeamIDResult(t, s, ctx)

	expected = make(map[string]uint)
	expected["global-label-one"] = 0
	expected["team-one-label-two"] = teamOne.ID
	expected["global-label-two"] = teamOne.ID
	expected["team-one-label-one"] = teamTwo.ID

	require.True(t, maps.Equal(expected, got))
}

// Tests a gitops setup where every team runs from an independent repo. Multiple repos are simulated by
// copying over the example repository multiple times.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsTeamLabelsMultipleRepos() {
	t := s.T()
	ctx := context.Background()

	var users []fleet.User
	var cfgPaths []*os.File
	var reposDir []string

	for range 2 {
		user := s.createGitOpsUser(t)
		users = append(users, user)

		cfg := s.createFleetctlConfig(t, user)
		cfgPaths = append(cfgPaths, cfg)

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
		reposDir = append(reposDir, repoDir)
	}

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_ENROLL_SECRET", "workstations_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET", "workstations_canary_enroll_secret")

	type tmplParams struct {
		Name    string
		Queries string
		Labels  string
	}
	teamCfgTmpl, err := template.New("t1").Parse(`
controls:
software:
reports:{{ .Queries }}
policies:
labels:{{ .Labels }}
agent_options:
name:{{ .Name }}
settings:
  secrets: [{"secret":"{{ .Name}}_secret"}]
`)
	require.NoError(t, err)

	// --------------------------------------------------
	// First, lets simulate adding a new team per repo
	// --------------------------------------------------
	for i, repo := range reposDir {
		globalFile := path.Join(repo, "default.yml")

		newTeamCfgFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)

		require.NoError(t, teamCfgTmpl.Execute(newTeamCfgFile, tmplParams{
			Name:    fmt.Sprintf(" team-%d", i),
			Queries: fmt.Sprintf("\n  - name: query-%d\n    query: SELECT 1", i),
			Labels:  fmt.Sprintf("\n  - name: label-%d\n    label_membership_type: dynamic\n    query: SELECT 1", i),
		}))

		args := []string{"gitops", "--config", cfgPaths[i].Name(), "-f", globalFile, "-f", newTeamCfgFile.Name()}
		s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)
	}

	for i, user := range users {
		team, err := s.DS.TeamByName(ctx, fmt.Sprintf("team-%d", i))
		require.NoError(t, err)
		require.NotNil(t, team)

		queries, _, _, _, err := s.DS.ListQueries(ctx, fleet.ListQueryOptions{TeamID: &team.ID})
		require.NoError(t, err)
		require.Len(t, queries, 1)
		require.Equal(t, fmt.Sprintf("query-%d", i), queries[0].Name)
		require.Equal(t, "SELECT 1", queries[0].Query)
		require.NotNil(t, queries[0].TeamID)
		require.Equal(t, *queries[0].TeamID, team.ID)
		require.NotNil(t, queries[0].AuthorID)
		require.Equal(t, *queries[0].AuthorID, user.ID)

		label, err := s.DS.LabelByName(ctx, fmt.Sprintf("label-%d", i), fleet.TeamFilter{User: &fleet.User{ID: user.ID}})
		require.NoError(t, err)
		require.NotNil(t, label)
		require.NotNil(t, label.TeamID)
		require.Equal(t, *label.TeamID, team.ID)
		require.NotNil(t, label.AuthorID)
		require.Equal(t, *label.AuthorID, user.ID)
	}

	// -----------------------------------------------------------------
	// Then, lets simulate a mutation by dropping the labels on team one
	// -----------------------------------------------------------------
	for i, repo := range reposDir {
		globalFile := path.Join(repo, "default.yml")

		newTeamCfgFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)

		params := tmplParams{
			Name:    fmt.Sprintf(" team-%d", i),
			Queries: fmt.Sprintf("\n  - name: query-%d\n    query: SELECT 1", i),
		}
		if i != 0 {
			params.Labels = fmt.Sprintf("\n  - name: label-%d\n    label_membership_type: dynamic\n    query: SELECT 1", i)
		}

		require.NoError(t, teamCfgTmpl.Execute(newTeamCfgFile, params))

		args := []string{"gitops", "--config", cfgPaths[i].Name(), "-f", globalFile, "-f", newTeamCfgFile.Name()}
		s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)
	}

	for i, user := range users {
		team, err := s.DS.TeamByName(ctx, fmt.Sprintf("team-%d", i))
		require.NoError(t, err)
		require.NotNil(t, team)

		queries, _, _, _, err := s.DS.ListQueries(ctx, fleet.ListQueryOptions{TeamID: &team.ID})
		require.NoError(t, err)
		require.Len(t, queries, 1)
		require.Equal(t, fmt.Sprintf("query-%d", i), queries[0].Name)
		require.Equal(t, "SELECT 1", queries[0].Query)
		require.NotNil(t, queries[0].TeamID)
		require.Equal(t, *queries[0].TeamID, team.ID)
		require.NotNil(t, queries[0].AuthorID)
		require.Equal(t, *queries[0].AuthorID, user.ID)

		label, err := s.DS.LabelByName(ctx, fmt.Sprintf("label-%d", i), fleet.TeamFilter{User: &fleet.User{ID: user.ID}})
		if i == 0 {
			require.Error(t, err)
			require.Nil(t, label)
		} else {
			require.NoError(t, err)
			require.NotNil(t, label)
			require.NotNil(t, label.TeamID)
			require.Equal(t, *label.TeamID, team.ID)
			require.NotNil(t, label.AuthorID)
			require.Equal(t, *label.AuthorID, user.ID)
		}
	}
}

func labelTeamIDResult(t *testing.T, s *enterpriseIntegrationGitopsTestSuite, ctx context.Context) map[string]uint {
	type labelResult struct {
		Name   string `db:"name"`
		TeamID *uint  `db:"team_id"`
	}
	var result []labelResult
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		require.NoError(t, sqlx.SelectContext(ctx, q, &result, "SELECT name, team_id FROM labels WHERE label_type = 0"))
		return nil
	})
	got := make(map[string]uint)
	for _, r := range result {
		var teamID uint
		if r.TeamID != nil {
			teamID = *r.TeamID
		}
		got[r.Name] = teamID
	}
	return got
}

// captureLabelActivities replaces the suite's activity mock with a recorder.
// It returns a function that returns and resets the captured label activities.
// Cleanup restores the previous NewActivityFunc.
func (s *enterpriseIntegrationGitopsTestSuite) captureLabelActivities(t *testing.T) func() []activity_api.ActivityDetails {
	t.Helper()
	require.NotNil(t, s.activityMock, "activity mock should be wired up via TestServerOpts.ActivityMock")
	prev := s.activityMock.NewActivityFunc
	var (
		mu       sync.Mutex
		captured []activity_api.ActivityDetails
	)
	s.activityMock.NewActivityFunc = func(ctx context.Context, user *activity_api.User, a activity_api.ActivityDetails) error {
		switch a.(type) {
		case fleet.ActivityTypeCreatedLabel, fleet.ActivityTypeEditedLabel, fleet.ActivityTypeDeletedLabel:
			mu.Lock()
			captured = append(captured, a)
			mu.Unlock()
		}
		if prev != nil {
			return prev(ctx, user, a)
		}
		return nil
	}
	t.Cleanup(func() { s.activityMock.NewActivityFunc = prev })

	return func() []activity_api.ActivityDetails {
		mu.Lock()
		defer mu.Unlock()
		out := captured
		captured = nil
		return out
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsLabelActivities() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetCfg := s.createFleetctlConfig(t, user)

	teamName := uuid.NewString()
	_, err := s.DS.NewTeam(ctx, &fleet.Team{Name: teamName})
	require.NoError(t, err)

	// Hosts to assign to the manual label in later phases.
	for _, h := range []string{"label-act-host-1", "label-act-host-2"} {
		_, err := s.DS.NewHost(ctx, &fleet.Host{
			UUID:           h,
			Hostname:       h,
			Platform:       "linux",
			HardwareSerial: h,
		})
		require.NoError(t, err)
	}

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	writeGlobal := func(body string) {
		require.NoError(t, os.WriteFile(globalFile.Name(), []byte(`
agent_options:
controls:
org_settings:
  secrets:
  - secret: test_secret
policies:
reports:
labels:
`+body), 0o644))
	}
	writeTeam := func(body string) {
		require.NoError(t, os.WriteFile(teamFile.Name(), fmt.Appendf(nil, `
controls:
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
labels:
%s`, teamName, body), 0o644))
	}
	apply := func() {
		s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
			"gitops", "--config", fleetCfg.Name(),
			"-f", globalFile.Name(), "-f", teamFile.Name(),
		}))
	}

	flush := s.captureLabelActivities(t)

	// Phase 1: initial apply creates one global and one team label.
	writeGlobal(`  - name: lbl-global
    description: original-global
    label_membership_type: dynamic
    query: SELECT 1
`)
	writeTeam(`  - name: lbl-team
    description: original-team
    label_membership_type: dynamic
    query: SELECT 2
`)
	apply()
	got := flush()
	require.Len(t, got, 2, "expected created_label for global + team")

	byName := map[string]fleet.ActivityTypeCreatedLabel{}
	for _, a := range got {
		c, ok := a.(fleet.ActivityTypeCreatedLabel)
		require.True(t, ok, "expected created_label, got %T", a)
		byName[c.Name] = c
	}
	require.Contains(t, byName, "lbl-global")
	require.Contains(t, byName, "lbl-team")
	require.Nil(t, byName["lbl-global"].FleetID, "global label should have nil fleet_id")
	require.NotNil(t, byName["lbl-team"].FleetID, "team label should have a fleet_id")
	require.NotNil(t, byName["lbl-team"].FleetName)
	require.Equal(t, teamName, *byName["lbl-team"].FleetName)

	// Phase 2: re-apply identical specs — no activity should fire.
	apply()
	require.Empty(t, flush(), "no-op apply should produce no activity")

	// Phase 3: edit the global label's description and the team label's query.
	writeGlobal(`  - name: lbl-global
    description: edited-global
    label_membership_type: dynamic
    query: SELECT 1
`)
	writeTeam(`  - name: lbl-team
    description: original-team
    label_membership_type: dynamic
    query: SELECT 99
`)
	apply()
	got = flush()
	require.Len(t, got, 2, "expected edited_label for both edits")
	editedNames := map[string]struct{}{}
	for _, a := range got {
		e, ok := a.(fleet.ActivityTypeEditedLabel)
		require.True(t, ok, "expected edited_label, got %T", a)
		editedNames[e.Name] = struct{}{}
	}
	require.Contains(t, editedNames, "lbl-global")
	require.Contains(t, editedNames, "lbl-team")

	// Phase 4: change lbl-global from dynamic to manual (membership_type swap)
	// and assign one host. Should emit a single edited_label for lbl-global.
	writeGlobal(`  - name: lbl-global
    description: edited-global
    label_membership_type: manual
    hosts:
      - label-act-host-1
`)
	writeTeam(`  - name: lbl-team
    description: original-team
    label_membership_type: dynamic
    query: SELECT 99
`)
	apply()
	got = flush()
	require.Len(t, got, 1, "expected edited_label for membership_type change")
	e, ok := got[0].(fleet.ActivityTypeEditedLabel)
	require.True(t, ok, "expected edited_label, got %T", got[0])
	require.Equal(t, "lbl-global", e.Name)

	// Phase 5: extend the manual label's host list. Should emit a single
	// edited_label even though all other fields stayed the same.
	writeGlobal(`  - name: lbl-global
    description: edited-global
    label_membership_type: manual
    hosts:
      - label-act-host-1
      - label-act-host-2
`)
	apply()
	got = flush()
	require.Len(t, got, 1, "expected edited_label for host list change")
	e, ok = got[0].(fleet.ActivityTypeEditedLabel)
	require.True(t, ok, "expected edited_label, got %T", got[0])
	require.Equal(t, "lbl-global", e.Name)

	// Phase 5b: re-apply same host list — must be a no-op.
	apply()
	require.Empty(t, flush(), "no-op host list should produce no activity")

	// Phase 6: add a host_vitals label alongside the existing labels.
	writeGlobal(`  - name: lbl-global
    description: edited-global
    label_membership_type: manual
    hosts:
      - label-act-host-1
      - label-act-host-2
  - name: lbl-host-vitals
    description: vitals-label
    label_membership_type: host_vitals
    criteria:
      vital: end_user_idp_group
      value: original-group
`)
	apply()
	got = flush()
	require.Len(t, got, 1, "expected created_label for host_vitals label")
	c, ok := got[0].(fleet.ActivityTypeCreatedLabel)
	require.True(t, ok, "expected created_label, got %T", got[0])
	require.Equal(t, "lbl-host-vitals", c.Name)

	// Phase 7: update the host_vitals criteria — should emit edited_label.
	writeGlobal(`  - name: lbl-global
    description: edited-global
    label_membership_type: manual
    hosts:
      - label-act-host-1
      - label-act-host-2
  - name: lbl-host-vitals
    description: vitals-label
    label_membership_type: host_vitals
    criteria:
      vital: end_user_idp_group
      value: updated-group
`)
	apply()
	got = flush()
	require.Len(t, got, 1, "expected edited_label for criteria change")
	e, ok = got[0].(fleet.ActivityTypeEditedLabel)
	require.True(t, ok, "expected edited_label, got %T", got[0])
	require.Equal(t, "lbl-host-vitals", e.Name)

	// Phase 7b: re-apply the same criteria — must be a no-op.
	apply()
	require.Empty(t, flush(), "no-op criteria re-apply should produce no activity")

	// Phase 8: remove all labels from the spec — gitops issues delete calls
	// per name, which should each emit a deleted_label activity.
	writeGlobal("")
	writeTeam("")
	apply()
	got = flush()
	require.Len(t, got, 3, "expected deleted_label for all three labels")
	deletedNames := map[string]struct{}{}
	for _, a := range got {
		d, ok := a.(fleet.ActivityTypeDeletedLabel)
		require.True(t, ok, "expected deleted_label, got %T", a)
		deletedNames[d.Name] = struct{}{}
	}
	require.Contains(t, deletedNames, "lbl-global")
	require.Contains(t, deletedNames, "lbl-team")
	require.Contains(t, deletedNames, "lbl-host-vitals")
}

// captureDeletedPolicyActivities replaces the suite's activity mock with a
// recorder for deleted_policy activities. It returns a function that returns
// and resets the captured activities. Cleanup restores the previous
// NewActivityFunc.
func (s *enterpriseIntegrationGitopsTestSuite) captureDeletedPolicyActivities(t *testing.T) func() []fleet.ActivityTypeDeletedPolicy {
	t.Helper()
	require.NotNil(t, s.activityMock, "activity mock should be wired up via TestServerOpts.ActivityMock")
	prev := s.activityMock.NewActivityFunc
	var (
		mu       sync.Mutex
		captured []fleet.ActivityTypeDeletedPolicy
	)
	s.activityMock.NewActivityFunc = func(ctx context.Context, user *activity_api.User, a activity_api.ActivityDetails) error {
		if d, ok := a.(fleet.ActivityTypeDeletedPolicy); ok {
			mu.Lock()
			captured = append(captured, d)
			mu.Unlock()
		}
		if prev != nil {
			return prev(ctx, user, a)
		}
		return nil
	}
	t.Cleanup(func() { s.activityMock.NewActivityFunc = prev })

	return func() []fleet.ActivityTypeDeletedPolicy {
		mu.Lock()
		defer mu.Unlock()
		out := captured
		captured = nil
		return out
	}
}

// setupDarwinFMA inserts a darwin FMA record and starts a per-FMA installer
// server. Returns the slug and the installer server URL. The caller is
// responsible for wiring a manifest server (FLEET_DEV_MAINTAINED_APPS_BASE_URL)
// that serves a manifest for /<slug>.json — calling this helper twice within
// the same test requires a single combined manifest server because
// dev_mode.SetOverride only supports one base URL at a time.
func (s *enterpriseIntegrationGitopsTestSuite) setupDarwinFMA(t *testing.T) (slug, installerURL string) {
	t.Helper()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	slug = fmt.Sprintf("foo%s/darwin", suffix)
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(),
			`INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
			 VALUES (?, ?, 'darwin', ?)`, "foo"+suffix, slug, "com.example.foo"+suffix)
		return err
	})

	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("foo"))
	}))
	t.Cleanup(installerServer.Close)
	return slug, installerServer.URL
}

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsRemovedFMAEmitsPolicyDeletedActivities() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	t.Setenv("FLEET_URL", s.Server.URL)

	sharedSlug, sharedInstaller := s.setupDarwinFMA(t)
	patchAndInstallSlug, patchAndInstallInstaller := s.setupDarwinFMA(t)
	teamName := uuid.NewString()

	manifestFor := func(installerURL string) ma.FMAManifestFile {
		return ma.FMAManifestFile{
			Versions: []*ma.FMAManifestApp{{
				Version:            "1.0",
				Queries:            ma.FMAQueries{Exists: "SELECT 1 FROM osquery_info;"},
				InstallerURL:       installerURL + "/foo.pkg",
				InstallScriptRef:   "fooscript",
				UninstallScriptRef: "fooscript",
				SHA256:             "no_check", // See ma.noCheckHash
			}},
			Refs: map[string]string{"fooscript": "echo hello"},
		}
	}
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var installerURL string
		switch r.URL.Path {
		case "/" + sharedSlug + ".json":
			installerURL = sharedInstaller
		case "/" + patchAndInstallSlug + ".json":
			installerURL = patchAndInstallInstaller
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(manifestFor(installerURL))
	}))
	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL, t)

	const globalConfig = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`
	teamWithFMAAndPolicies := fmt.Sprintf(`
controls:
software:
  fleet_maintained_apps:
    - slug: %s
    - slug: %s
policies:
  - name: install-policy
    query: SELECT 1
    install_software:
      fleet_maintained_app_slug: %s
  - name: patch-policy
    type: patch
    fleet_maintained_app_slug: %s
  - name: patch-and-install-policy
    type: patch
    fleet_maintained_app_slug: %s
    install_software:
      fleet_maintained_app_slug: %s
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
reports:
`, sharedSlug, patchAndInstallSlug, sharedSlug, sharedSlug, patchAndInstallSlug, patchAndInstallSlug, teamName)
	teamEmpty := fmt.Sprintf(`
controls:
software:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
reports:
`, teamName)

	globalFile := filepath.Join(t.TempDir(), "global.yml")
	require.NoError(t, os.WriteFile(globalFile, []byte(globalConfig), 0o644))
	teamFile := filepath.Join(t.TempDir(), "team.yml")

	require.NoError(t, os.WriteFile(teamFile, []byte(teamWithFMAAndPolicies), 0o644))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", teamFile,
	}))

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	pols, err := s.DS.ListMergedTeamPolicies(ctx, team.ID, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, pols, 3)
	policyIDsByName := map[string]uint{}
	for _, p := range pols {
		policyIDsByName[p.Name] = p.ID
	}
	require.Contains(t, policyIDsByName, "install-policy")
	require.Contains(t, policyIDsByName, "patch-policy")
	require.Contains(t, policyIDsByName, "patch-and-install-policy")

	flush := s.captureDeletedPolicyActivities(t)
	require.NoError(t, os.WriteFile(teamFile, []byte(teamEmpty), 0o644))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", teamFile,
	}))

	pols, err = s.DS.ListMergedTeamPolicies(ctx, team.ID, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Empty(t, pols, "all policies should be removed after FMA installer is removed")

	deletedIDs := map[uint]bool{}
	for _, d := range flush() {
		deletedIDs[d.ID] = true
	}
	for _, name := range []string{"install-policy", "patch-policy", "patch-and-install-policy"} {
		require.True(t, deletedIDs[policyIDsByName[name]],
			"expected deleted_policy activity for %q (id=%d), got activities for IDs %v",
			name, policyIDsByName[name], deletedIDs)
	}
}

// TestGitOpsVPPAppAutoUpdate tests that auto-update settings for VPP apps (iOS/iPadOS)
// are properly applied via GitOps.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsVPPAppAutoUpdate() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Create a global VPP token (location is "Jungle")
	test.CreateInsertGlobalVPPToken(t, s.DS)

	// Generate team name upfront since we need it in the global template
	teamName := uuid.NewString()

	// The global template includes VPP token assignment to the team
	// The location "Jungle" comes from test.CreateInsertGlobalVPPToken
	globalTemplate := fmt.Sprintf(`
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
  mdm:
    volume_purchasing_program:
      - location: Jungle
        teams:
          - %s
policies:
reports:
`, teamName)

	teamTemplate := `
controls:
software:
  app_store_apps:
    - app_store_id: "2"
      platform: ios
      self_service: false
      auto_update_enabled: true
      auto_update_window_start: "02:00"
      auto_update_window_end: "06:00"
    - app_store_id: "2"
      platform: ipados
      self_service: false
      auto_update_enabled: true
      auto_update_window_start: "03:00"
      auto_update_window_end: "07:00"
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamTemplate, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	t.Setenv("FLEET_URL", s.Server.URL)

	testing_utils.StartAndServeVPPServer(t)

	dryRunOutput := fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	require.Contains(t, dryRunOutput, "gitops dry run succeeded")

	realRunOutput := fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()})
	require.Contains(t, realRunOutput, "gitops succeeded")

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// Verify VPP apps were added
	titles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 2) // One for iOS, one for iPadOS

	// Verify auto-update schedules were created in the database
	type autoUpdateSchedule struct {
		TitleID   uint   `db:"title_id"`
		TeamID    uint   `db:"team_id"`
		Enabled   bool   `db:"enabled"`
		StartTime string `db:"start_time"`
		EndTime   string `db:"end_time"`
	}
	var schedules []autoUpdateSchedule
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &schedules,
			`SELECT title_id, team_id, enabled, start_time, end_time
			FROM software_update_schedules
			WHERE team_id = ?
			ORDER BY title_id`, team.ID)
	})

	require.Len(t, schedules, 2)

	for _, schedule := range schedules {
		require.Equal(t, team.ID, schedule.TeamID)
		require.True(t, schedule.Enabled)

		var foundTitle *fleet.SoftwareTitleListResult
		for i := range titles {
			if titles[i].ID == schedule.TitleID {
				foundTitle = &titles[i]
				break
			}
		}
		require.NotNil(t, foundTitle, "should find title for schedule")

		// Verify the correct start/end times based on source
		switch foundTitle.Source {
		case "ios_apps":
			require.Equal(t, "02:00", schedule.StartTime)
			require.Equal(t, "06:00", schedule.EndTime)
		case "ipados_apps":
			require.Equal(t, "03:00", schedule.StartTime)
			require.Equal(t, "07:00", schedule.EndTime)
		default:
			t.Fatalf("unexpected source: %s", foundTitle.Source)
		}
	}

	// Now apply a config without auto-update fields for the iPadOS app and verify they're cleared
	teamTemplateNoAutoUpdate := `
controls:
software:
  app_store_apps:
    - app_store_id: "2"
      platform: ios
      self_service: false
      auto_update_enabled: true
      auto_update_window_start: "02:00"
      auto_update_window_end: "06:00"
    - app_store_id: "2"
      platform: ipados
      self_service: false
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`

	teamFileNoAutoUpdate, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFileNoAutoUpdate.WriteString(fmt.Sprintf(teamTemplateNoAutoUpdate, teamName))
	require.NoError(t, err)
	err = teamFileNoAutoUpdate.Close()
	require.NoError(t, err)

	// Apply the updated config
	realRunOutput = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFileNoAutoUpdate.Name()})
	require.Contains(t, realRunOutput, "gitops succeeded")

	// Verify auto-update schedules: iOS should still have settings, iPadOS should be disabled
	var updatedSchedules []autoUpdateSchedule
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &updatedSchedules,
			`SELECT title_id, team_id, enabled, start_time, end_time
			FROM software_update_schedules
			WHERE team_id = ?
			ORDER BY title_id`, team.ID)
	})

	require.Len(t, updatedSchedules, 2)

	for _, schedule := range updatedSchedules {
		var foundTitle *fleet.SoftwareTitleListResult
		for i := range titles {
			if titles[i].ID == schedule.TitleID {
				foundTitle = &titles[i]
				break
			}
		}
		require.NotNil(t, foundTitle, "should find title for schedule")

		switch foundTitle.Source {
		case "ios_apps":
			// iOS app should still have auto-update enabled
			require.True(t, schedule.Enabled)
			require.Equal(t, "02:00", schedule.StartTime)
			require.Equal(t, "06:00", schedule.EndTime)
		case "ipados_apps":
			// iPadOS app should now have auto-update disabled (fields removed from config)
			// but the previous start/end times should still be preserved in the database
			require.False(t, schedule.Enabled)
			require.Equal(t, "03:00", schedule.StartTime)
			require.Equal(t, "07:00", schedule.EndTime)
		default:
			t.Fatalf("unexpected source: %s", foundTitle.Source)
		}
	}
}

// TestFleetDesktopSettingsBrowserAlternativeHost tests that user can mutate the fleet_desktop.alternative_browser_host
// setting via GitOps.
func (s *enterpriseIntegrationGitopsTestSuite) TestFleetDesktopSettingsBrowserAlternativeHost() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetCfg := s.createFleetctlConfig(t, user)

	type tmplParams struct {
		AlternativeBrowserHost string
	}
	globalCfgTpl, err := template.New("t1").Parse(`
agent_options:
controls:
reports:
policies:
org_settings:
  secrets:
    - secret: test_secret
  fleet_desktop:
    {{ .AlternativeBrowserHost }}
`)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_ENROLL_SECRET", "workstations_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET", "workstations_canary_enroll_secret")

	testCases := []struct {
		Name                   string
		AlternativeBrowserHost string
		Expected               string
		ShouldError            bool
	}{
		{
			Name:                   "custom",
			AlternativeBrowserHost: `alternative_browser_host: "example1.com"`,
			Expected:               "example1.com",
		},
		{
			Name:                   "empty value",
			AlternativeBrowserHost: `alternative_browser_host: ""`,
			Expected:               "",
		},
		{
			Name:                   "invalid value",
			AlternativeBrowserHost: `alternative_browser_host: "http://example2.com"`,
			ShouldError:            true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			globalCfgFile, err := os.CreateTemp(t.TempDir(), "*.yml")
			require.NoError(t, err)

			require.NoError(t, globalCfgTpl.Execute(globalCfgFile, tmplParams{
				AlternativeBrowserHost: testCase.AlternativeBrowserHost,
			}))

			if testCase.ShouldError {
				fleetctl.RunAppCheckErr(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalCfgFile.Name()}, "applying fleet config: PATCH /api/latest/fleet/config received status 422 Validation Failed: must be a valid hostname or IP address")
			} else {
				s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalCfgFile.Name(), "--dry-run"}))
				s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalCfgFile.Name()}))
			}

			storedCfg, err := s.DS.AppConfig(ctx)
			require.NoError(t, err)
			require.NotNil(t, storedCfg)
			require.Equal(t, testCase.Expected, storedCfg.FleetDesktop.AlternativeBrowserHost)
		})
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) TestSpecialCaseTeamsVPPApps() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Create a global VPP token (location is "Jungle")
	test.CreateInsertGlobalVPPToken(t, s.DS)

	// Generate team name upfront since we need it in the global template
	teamName := uuid.NewString()

	// The global template includes VPP token assignment to the team
	// The location "Jungle" comes from test.CreateInsertGlobalVPPToken
	globalTemplate := `agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
  - secret: foobar
  mdm:
    volume_purchasing_program:
      - location: Jungle
        teams:
          - %s
policies:
reports:
`

	teamTemplate := `
controls:
software:
  app_store_apps:
    - app_store_id: "2"
      platform: ios
      self_service: false
      auto_update_enabled: true
      auto_update_window_start: "02:00"
      auto_update_window_end: "06:00"
    - app_store_id: "2"
      platform: ipados
      self_service: false
      auto_update_enabled: true
      auto_update_window_start: "03:00"
      auto_update_window_end: "07:00"
reports:
policies:
agent_options:
name: %s
settings:
  %s
`

	testCases := []struct {
		specialCase  string
		teamName     string
		teamSettings string
	}{
		{
			specialCase:  "All teams",
			teamName:     teamName,
			teamSettings: `secrets: [{"secret":"enroll_secret"}]`,
		},
		{
			specialCase: "No team",
			teamName:    "Unassigned",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.specialCase, func(t *testing.T) {
			globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
			require.NoError(t, err)
			globalYAML := fmt.Sprintf(globalTemplate, tc.specialCase)
			_, err = globalFile.WriteString(globalYAML)
			require.NoError(t, err)
			err = globalFile.Close()
			require.NoError(t, err)
			teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
			require.NoError(t, err)
			_, err = fmt.Fprintf(teamFile, teamTemplate, tc.teamName, tc.teamSettings)
			require.NoError(t, err)
			err = teamFile.Close()
			require.NoError(t, err)

			teamFileName := teamFile.Name()

			if tc.specialCase == "No team" {
				noTeamFilePath := filepath.Join(filepath.Dir(teamFile.Name()), "unassigned.yml")
				err = os.Rename(teamFile.Name(), noTeamFilePath)
				require.NoError(t, err)

				teamFileName = noTeamFilePath
			}

			t.Setenv("FLEET_URL", s.Server.URL)

			testing_utils.StartAndServeVPPServer(t)

			dryRunOutput := fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFileName, "--dry-run"})
			require.Contains(t, dryRunOutput, "gitops dry run succeeded")

			realRunOutput := fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFileName})
			require.Contains(t, realRunOutput, "gitops succeeded")

			var teamID uint
			if tc.specialCase != "No team" {
				team, err := s.DS.TeamByName(ctx, teamName)
				require.NoError(t, err)
				teamID = team.ID
			}

			// Verify VPP apps were added
			titles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &teamID},
				fleet.TeamFilter{User: test.UserAdmin})
			require.NoError(t, err)
			require.Len(t, titles, 2) // One for iOS, one for iPadOS
		})
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) TestDisallowSoftwareSetupExperience() {
	t := s.T()
	ctx := context.Background()

	originalAppConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.DS.SaveAppConfig(ctx, originalAppConfig)
		require.NoError(t, err)
	})

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	test.CreateInsertGlobalVPPToken(t, s.DS)
	teamName := uuid.NewString()

	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/signed.pkg")
	}))
	defer bootstrapServer.Close()

	// The global template includes VPP token assignment to the team
	// The location "Jungle" comes from test.CreateInsertGlobalVPPToken
	globalTemplate := `agent_options:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
  - secret: foobar
  mdm:
    volume_purchasing_program:
      - location: Jungle
        teams:
          - %s
policies:
controls:
queries:
`

	testVPP := `
controls:
  setup_experience:
    macos_bootstrap_package: %s
    macos_manual_agent_install: true
software:
  app_store_apps:
    - app_store_id: "2"
      platform: darwin
      setup_experience: true
    - app_store_id: "2"
      platform: ios
      setup_experience: true
    - app_store_id: "2"
      platform: ipados
      setup_experience: true
queries:
policies:
agent_options:
name: %s
team_settings:
  %s
`

	//nolint:gosec // test code
	testPackages := `
controls:
  setup_experience:
    macos_bootstrap_package: %s
    macos_manual_agent_install: true
software:
  app_store_apps:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/dummy_installer.pkg
      setup_experience: true
queries:
policies:
agent_options:
name: %s
team_settings:
  %s
`

	testCases := []struct {
		VPPTeam      string
		testName     string
		teamName     string
		teamTemplate string
		teamSettings string
		errContains  *string
	}{
		{
			testName:     "All VPP with setup experience",
			VPPTeam:      "All teams",
			teamName:     teamName,
			teamTemplate: testVPP,
			teamSettings: `secrets: [{"secret":"enroll_secret"}]`,
			errContains:  ptr.String("Couldn't edit software."),
		},
		{
			testName:     "Packages fail",
			VPPTeam:      "All teams",
			teamName:     teamName,
			teamTemplate: testPackages,
			teamSettings: `secrets: [{"secret":"enroll_secret"}]`,
			errContains:  ptr.String("Couldn't edit software."),
		},
		{
			testName:     "No team VPP",
			VPPTeam:      "No team",
			teamName:     "Unassigned",
			teamTemplate: testVPP,
			errContains:  ptr.String("Couldn't edit software."),
		},
		{
			testName:     "No team Installers",
			VPPTeam:      "No team",
			teamName:     "Unassigned",
			teamTemplate: testPackages,
			errContains:  ptr.String("Couldn't edit software."),
		},
		// left out more possible combinations of setup experience being set for different platforms
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
			require.NoError(t, err)
			globalYAML := fmt.Sprintf(globalTemplate, tc.VPPTeam)
			_, err = globalFile.WriteString(globalYAML)
			require.NoError(t, err)
			err = globalFile.Close()
			require.NoError(t, err)
			teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
			require.NoError(t, err)
			_, err = fmt.Fprintf(teamFile, tc.teamTemplate, bootstrapServer.URL, tc.teamName, tc.teamSettings)
			require.NoError(t, err)
			err = teamFile.Close()
			require.NoError(t, err)

			teamFileName := teamFile.Name()

			if tc.VPPTeam == "No team" {
				noTeamFilePath := filepath.Join(filepath.Dir(teamFile.Name()), "unassigned.yml")
				err = os.Rename(teamFile.Name(), noTeamFilePath)
				require.NoError(t, err)
				teamFileName = noTeamFilePath
			}

			t.Setenv("FLEET_URL", s.Server.URL)
			testing_utils.StartSoftwareInstallerServer(t)
			testing_utils.StartAndServeVPPServer(t)

			// Don't attempt dry runs because they would not actually create the team, so the config would not be found
			_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFileName})

			if tc.errContains != nil {
				require.ErrorContains(t, err, *tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func (s *enterpriseIntegrationGitopsTestSuite) TestConfigurationProfileEscaping() {
	t := s.T()
	ctx := context.Background()
	tempDir := t.TempDir()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Get the testdata mobileconfig profile
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	profilePath := filepath.Join(filepath.Dir(currentFile), "testdata", "apple-profile.mobileconfig")
	require.FileExists(t, profilePath)

	const secretPasswordValue = "custom&password<tag>"
	t.Setenv("FLEET_SECRET_PASSWORD", secretPasswordValue)
	t.Setenv("API_KEY", "my-api-key&value")
	t.Setenv("FLEET_URL", s.Server.URL)

	gitopsConfig := fmt.Sprintf(`
org_settings:
  server_settings:
    server_url: %s
  org_info:
    org_name: Fleet
  secrets:
    - secret: test_secret
agent_options:
controls:
  apple_settings:
    configuration_profiles:
      - path: %s
policies:
reports:
`, s.Server.URL, profilePath)

	configPath := filepath.Join(tempDir, "gitops.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(gitopsConfig), 0o644)) //nolint:gosec

	// Run gitops
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", configPath})

	// Verify the stored profile in the DB, based on my testing these vars are okay as they are not host-specific and we can avoid enrolling a host and checking the host specific payload.
	profiles, err := s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)

	var storedProfile *fleet.MDMAppleConfigProfile
	for _, p := range profiles {
		if strings.Contains(p.Identifier, "fleet.9913C522") {
			storedProfile = p
			break
		}
	}
	require.NotNil(t, storedProfile, "should find the uploaded profile")
	profileBody := string(storedProfile.Mobileconfig)

	// $FLEET_SECRET_PASSWORD should NOT be expanded, as it will be expanded server-side
	assert.Contains(t, profileBody, "$FLEET_SECRET_PASSWORD",
		"stored profile should still contain the FLEET_SECRET_ placeholder")
	// $API_KEY should be expanded and its value should have been escaped during gitops
	assert.Contains(t, profileBody, "my-api-key&amp;value",
		"stored profile should have the expanded $API_KEY value")
	assert.NotContains(t, profileBody, "$API_KEY",
		"stored profile should not contain the $API_KEY variable reference")

	// Verify the secret was saved to the server unescaped, to avoid double encoding secrets.
	secrets, err := s.DS.GetSecretVariables(ctx, []string{"PASSWORD"})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, secretPasswordValue, secrets[0].Value,
		"secret should be stored as the raw value (not XML-escaped)")
}

// TestJSONConfigurationProfileEscaping covers issue #38013 — JSON profiles
// (Apple DDM declarations) must have variable values JSON-escaped at expansion
// time, while the underlying secret is still stored on the server unescaped.
func (s *enterpriseIntegrationGitopsTestSuite) TestJSONConfigurationProfileEscaping() {
	t := s.T()
	ctx := context.Background()
	tempDir := t.TempDir()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		declIdentifier     = "com.fleetdm.json.escape.test"
		secretPasswordName = "JSON_ESCAPE_PASSWORD"

		// Values contain characters that break naive JSON string interpolation
		// (double quote, backslash, and XML-significant chars for completeness).
		secretPasswordValue = `custom"password\tag&<>` //nolint:gosec // G101: test fixture, not a credential
		apiKeyValue         = `my"api&key\v`           //nolint:gosec // G101: test fixture, not a credential
	)

	t.Setenv("FLEET_SECRET_"+secretPasswordName, secretPasswordValue)
	t.Setenv("API_KEY", apiKeyValue)
	t.Setenv("FLEET_URL", s.Server.URL)

	declPath := filepath.Join(tempDir, "decl.json")
	declBody := fmt.Sprintf(`{
		"Type": "com.apple.configuration.management.test",
		"Identifier": %q,
		"Payload": {
			"Password": "$FLEET_SECRET_%s",
			"ApiKey": "$API_KEY"
		}
	}`, declIdentifier, secretPasswordName)
	require.NoError(t, os.WriteFile(declPath, []byte(declBody), 0o644)) //nolint:gosec

	gitopsConfig := fmt.Sprintf(`
org_settings:
  server_settings:
    server_url: %s
  org_info:
    org_name: Fleet
  secrets:
    - secret: json_escape_test_secret
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: %s
policies:
reports:
`, s.Server.URL, declPath)

	configPath := filepath.Join(tempDir, "gitops.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(gitopsConfig), 0o644)) //nolint:gosec

	// Before the fix, this run would fail with
	// "Declaration profiles should include valid JSON."
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", configPath})

	// Retrieve the stored declaration by listing profiles and matching on identifier.
	profs, _, err := s.DS.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	var declUUID string
	for _, p := range profs {
		if p.Platform == "darwin" && p.Identifier == declIdentifier {
			declUUID = p.ProfileUUID
			break
		}
	}
	require.NotEmpty(t, declUUID, "uploaded declaration should be listed")

	decl, err := s.DS.GetMDMAppleDeclaration(ctx, declUUID)
	require.NoError(t, err)
	stored := string(decl.RawJSON)

	// Stored bytes must be valid JSON — this is the regression guard for #38013.
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(decl.RawJSON, &parsed),
		"stored declaration must be valid JSON")

	// $FLEET_SECRET_* placeholder must be stored literally; it is expanded
	// server-side at delivery time so secrets are never double-encoded.
	assert.Contains(t, stored, "$FLEET_SECRET_"+secretPasswordName,
		"stored declaration should still contain the FLEET_SECRET_ placeholder")

	// $API_KEY must be JSON-escaped: `"` → `\"`, `\` → `\\`.
	payload, ok := parsed["Payload"].(map[string]any)
	require.True(t, ok, "Payload should be an object")
	assert.Equal(t, apiKeyValue, payload["ApiKey"],
		"ApiKey should round-trip to the raw env var value after JSON parse")
	assert.NotContains(t, stored, "$API_KEY",
		"stored declaration should not contain the unexpanded $API_KEY reference")

	// The custom secret must be stored raw so server-side expansion doesn't double-encode it.
	dbSecrets, err := s.DS.GetSecretVariables(ctx, []string{secretPasswordName})
	require.NoError(t, err)
	require.Len(t, dbSecrets, 1)
	assert.Equal(t, secretPasswordValue, dbSecrets[0].Value,
		"secret should be stored as the raw value (not JSON-escaped)")
}

// TestGitOpsSoftwareWithEnvVarInstalledByPolicy tests that a software package
// with an environment variable in the URL can be referenced by a policy to be
// installed automatically.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareWithEnvVarInstalledByPolicy() {
	t := s.T()
	ctx := t.Context()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	const (
		globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		noTeamTemplate = `name: Unassigned
controls:
policies:
  - description: Test policy.
    install_software:
      package_path: ./lib/ruby.yml
    name: Install ruby
    platform: linux
    query: SELECT 1 FROM file WHERE path = "/usr/local/bin/ruby";
    resolution: Install ruby.
software:
  packages:
    - path: ./lib/ruby.yml
`

		packageTemplate = `- url: ${CUSTOM_SOFTWARE_INSTALLER_URL}/ruby.deb`

		fleetTemplate = `
controls:
software:
  packages:
    - path: ./lib/ruby.yml
reports:
policies:
  - description: Test policy.
    install_software:
      package_path: ./lib/ruby.yml
    name: Install team ruby
    platform: linux
    query: SELECT 1 FROM file WHERE path = "/usr/local/bin/ruby";
    resolution: Install ruby.
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	tempDir := t.TempDir()

	globalFile := filepath.Join(tempDir, "global.yml")
	err := os.WriteFile(globalFile, []byte(globalTemplate), 0o644) //nolint:gosec
	require.NoError(t, err)

	noTeamFile := filepath.Join(tempDir, "unassigned.yml")
	err = os.WriteFile(noTeamFile, []byte(noTeamTemplate), 0o644)
	require.NoError(t, err)

	fleetName := uuid.NewString()
	fleetFile := filepath.Join(tempDir, "fleet.yml")
	err = os.WriteFile(fleetFile, fmt.Appendf(nil, fleetTemplate, fleetName), 0o644)
	require.NoError(t, err)

	pkgFile := filepath.Join(tempDir, "lib", "ruby.yml")
	err = os.MkdirAll(filepath.Dir(pkgFile), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(pkgFile, []byte(packageTemplate), 0o644)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	// Apply configs, installer URL env var is not defined yet
	_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", noTeamFile, "-f", fleetFile, "--dry-run"})
	require.ErrorContains(t, err, `environment variable "CUSTOM_SOFTWARE_INSTALLER_URL" not set`)
	_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", noTeamFile, "-f", fleetFile})
	require.ErrorContains(t, err, `environment variable "CUSTOM_SOFTWARE_INSTALLER_URL" not set`)

	// define the URL env var and apply again, should succeed
	t.Setenv("CUSTOM_SOFTWARE_INSTALLER_URL", os.Getenv("SOFTWARE_INSTALLER_URL"))
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", noTeamFile, "-f", fleetFile, "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", noTeamFile, "-f", fleetFile}))

	// no-team has a ruby custom installer
	titles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: ptr.Uint(0)}, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, "ruby", titles[0].Name)
	require.NotNil(t, titles[0].SoftwarePackage)
	installer, err := s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, titles[0].ID, false)
	require.NoError(t, err)

	tmPols, err := s.DS.ListMergedTeamPolicies(ctx, 0, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, tmPols, 1)
	require.Equal(t, "Install ruby", tmPols[0].Name)
	require.NotNil(t, tmPols[0].SoftwareInstallerID)
	require.Equal(t, installer.InstallerID, *tmPols[0].SoftwareInstallerID)

	// Get the team ID
	tm, err := s.DS.TeamByName(ctx, fleetName)
	require.NoError(t, err)

	// team has a ruby custom installer
	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &tm.ID}, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, "ruby", titles[0].Name)
	require.NotNil(t, titles[0].SoftwarePackage)
	installer, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &tm.ID, titles[0].ID, false)
	require.NoError(t, err)

	tmPols, err = s.DS.ListMergedTeamPolicies(ctx, tm.ID, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, tmPols, 1)
	require.Equal(t, "Install team ruby", tmPols[0].Name)
	require.NotNil(t, tmPols[0].SoftwareInstallerID)
	require.Equal(t, installer.InstallerID, *tmPols[0].SoftwareInstallerID)
}

// TestOmittedTopLevelKeysGlobal verifies that omitting top-level keys from a global
// gitops file clears the corresponding settings (e.g. policies, agent_options).
func (s *enterpriseIntegrationGitopsTestSuite) TestOmittedTopLevelKeysGlobal() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	t.Setenv("FLEET_URL", s.Server.URL)

	// Step 1: Apply a full global config with policies, agent_options, controls, and reports.
	const fullGlobalConfig = `
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: boofar
agent_options:
  config:
    options:
      pack_delimiter: /
controls:
  enable_disk_encryption: true
policies:
  - name: Test Global Policy
    query: SELECT 1;
reports:
  - name: Test Global Report
    query: SELECT 1;
    automations_enabled: false
labels:
  - name: Test Global Label
    label_membership_type: dynamic
    query: SELECT 1
`
	fullFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fullFile.WriteString(fullGlobalConfig)
	require.NoError(t, err)
	require.NoError(t, fullFile.Close())

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fullFile.Name()}))

	// Verify policy, agent_options, controls, and reports were applied.
	policies, err := s.DS.ListGlobalPolicies(ctx, fleet.ListOptions{}, "")
	require.NoError(t, err)
	require.Len(t, policies, 1)
	require.Equal(t, "Test Global Policy", policies[0].Name)

	appCfg, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, appCfg.AgentOptions)
	require.Contains(t, string(*appCfg.AgentOptions), "pack_delimiter")
	require.True(t, appCfg.MDM.EnableDiskEncryption.Value)

	queries, _, _, _, err := s.DS.ListQueries(ctx, fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, 1)
	require.Equal(t, "Test Global Report", queries[0].Name)

	globalSecrets, err := s.DS.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, globalSecrets, 1)
	require.Equal(t, "boofar", globalSecrets[0].Secret)

	labels, err := s.DS.LabelsByName(ctx, []string{"Test Global Label"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, labels, 1)

	// Step 2: Apply a minimal global config that omits policies, agent_options, reports, labels.
	const minimalGlobalConfig = `
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
`
	minimalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = minimalFile.WriteString(minimalGlobalConfig)
	require.NoError(t, err)
	require.NoError(t, minimalFile.Close())

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", minimalFile.Name()}))

	// Verify policies were cleared.
	policies, err = s.DS.ListGlobalPolicies(ctx, fleet.ListOptions{}, "")
	require.NoError(t, err)
	require.Len(t, policies, 0)

	appCfg, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)

	// Verify agent_options were cleared (set to null).
	require.Nil(t, appCfg.AgentOptions)

	// Verify controls were cleared (disk encryption reverts to false).
	require.False(t, appCfg.MDM.EnableDiskEncryption.Value)

	// Verify reports were cleared.
	queries, _, _, _, err = s.DS.ListQueries(ctx, fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, 0)

	// Verify secrets are cleared.
	globalSecrets, err = s.DS.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, globalSecrets, 0)

	// Verify labels are cleared.
	labels, err = s.DS.LabelsByName(ctx, []string{"Test Global Label"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, labels, 0)
}

// TestOmittedTopLevelKeysFleet verifies that omitting top-level keys from a fleet
// gitops file clears the corresponding settings (e.g. policies, agent_options, settings).
func (s *enterpriseIntegrationGitopsTestSuite) TestOmittedTopLevelKeysFleet() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	fleetName := "Test Omitted Keys " + uuid.NewString()

	// Step 1: Apply a full fleet config with policies, agent_options, controls, features, reports, and software.
	fullFleetConfig := fmt.Sprintf(`
name: %s
settings:
  secrets:
    - secret: foobar
  features:
    enable_host_users: false
agent_options:
  config:
    options:
      pack_delimiter: /
controls:
  enable_disk_encryption: true
policies:
  - name: Test Fleet Policy
    query: SELECT 1;
reports:
  - name: Test Fleet Report
    query: SELECT 1;
    automations_enabled: false
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
labels:
  - name: Test Fleet Label
    label_membership_type: dynamic
    query: SELECT 1

`, fleetName)

	fullFleetFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fullFleetFile.WriteString(fullFleetConfig)
	require.NoError(t, err)
	require.NoError(t, fullFleetFile.Close())

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(), "-f", fullFleetFile.Name(),
	}))

	// Verify policy, agent_options, controls, features, and reports were applied.
	fl, err := s.DS.TeamByName(ctx, fleetName)
	require.NoError(t, err)

	flPols, err := s.DS.ListMergedTeamPolicies(ctx, fl.ID, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, flPols, 1)
	require.Equal(t, "Test Fleet Policy", flPols[0].Name)

	require.NotNil(t, fl.Config.AgentOptions)
	require.Contains(t, string(*fl.Config.AgentOptions), "pack_delimiter")
	require.True(t, fl.Config.MDM.EnableDiskEncryption)
	require.False(t, fl.Config.Features.EnableHostUsers)

	flQueries, _, _, _, err := s.DS.ListQueries(ctx, fleet.ListQueryOptions{TeamID: &fl.ID})
	require.NoError(t, err)
	require.Len(t, flQueries, 1)
	require.Equal(t, "Test Fleet Report", flQueries[0].Name)

	flSecrets, err := s.DS.GetEnrollSecrets(ctx, &fl.ID)
	require.NoError(t, err)
	require.Len(t, flSecrets, 1)
	require.Equal(t, "foobar", flSecrets[0].Secret)

	titles, _, _, err := s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &fl.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 1)

	// Step 2: Apply a minimal fleet config that omits policies, agent_options, controls, reports, software, settings.
	minimalFleetConfig := fmt.Sprintf(`
name: %s
`, fleetName)

	minimalFleetFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = minimalFleetFile.WriteString(minimalFleetConfig)
	require.NoError(t, err)
	require.NoError(t, minimalFleetFile.Close())

	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(), "-f", minimalFleetFile.Name(),
	}))

	// Verify policies were cleared.
	flPols, err = s.DS.ListMergedTeamPolicies(ctx, fl.ID, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, flPols, 0)

	fl, err = s.DS.TeamByName(ctx, fleetName)
	require.NoError(t, err)

	// Verify agent_options were cleared (set to null).
	require.Nil(t, fl.Config.AgentOptions)

	// Verify controls were cleared (disk encryption reverts to false).
	require.False(t, fl.Config.MDM.EnableDiskEncryption)

	// Verify features reverted to defaults (enable_host_users defaults to true).
	require.True(t, fl.Config.Features.EnableHostUsers)

	// Verify reports were cleared.
	flQueries, _, _, _, err = s.DS.ListQueries(ctx, fleet.ListQueryOptions{TeamID: &fl.ID})
	require.NoError(t, err)
	require.Len(t, flQueries, 0)

	// Verify secrets are cleared.
	flSecrets, err = s.DS.GetEnrollSecrets(ctx, &fl.ID)
	require.NoError(t, err)
	require.Len(t, flSecrets, 0)

	// Verify software was cleared.
	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &fl.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 0)

	// Verify labels are cleared.
	labels, err := s.DS.LabelsByName(ctx, []string{"Test Fleet Label"}, fleet.TeamFilter{TeamID: &fl.ID})
	require.NoError(t, err)
	require.Len(t, labels, 0)
}

// TestFMALabelsIncludeAll tests that labels_include_all is correctly applied and
// cleared for Fleet Maintained Apps via gitops, for both no-team and a specific team.
func (s *enterpriseIntegrationGitopsTestSuite) TestFMALabelsIncludeAll() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	slug := fmt.Sprintf("foo%s/darwin", t.Name())
	lblName := "Label1" + t.Name()
	const (
		globalTemplate = `
agent_options:
labels:
  - name: %s
    label_membership_type: dynamic
    query: SELECT 1
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`
		noTeamTemplate = `name: Unassigned
controls:
policies:
software:
  fleet_maintained_apps:
    - slug: %s
%s
`
		teamTemplate = `
controls:
software:
  fleet_maintained_apps:
    - slug: %s
%s
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)
	const noLabels = ""

	withLabelsIncludeAll := fmt.Sprintf(`
      labels_include_all:
        - %s
`, lblName)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(globalFile, globalTemplate, lblName)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(noTeamFile, noTeamTemplate, slug, withLabelsIncludeAll)
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = fmt.Fprintf(teamFile, teamTemplate, slug, withLabelsIncludeAll, teamName)
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	// Mock server to serve FMA installer bytes
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("foo"))
	}))
	defer installerServer.Close()

	// Mock server to serve the FMA manifest
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versions := []*ma.FMAManifestApp{
			{
				Version: "1.0",
				Queries: ma.FMAQueries{
					Exists: "SELECT 1 FROM osquery_info;",
				},
				InstallerURL:       installerServer.URL + "/foo.pkg",
				InstallScriptRef:   "fooscript",
				UninstallScriptRef: "fooscript",
				SHA256:             "no_check",
			},
		}
		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs:     map[string]string{"fooscript": "echo hello"},
		}
		err := json.NewEncoder(w).Encode(manifest)
		require.NoError(t, err)
	}))
	defer manifestServer.Close()

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL, t)

	// Insert the FMA record so gitops can resolve the slug
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
			 VALUES (?, ?, 'darwin', ?)`, "foo"+t.Name(), slug, `com.example.foo`+t.Name())
		return err
	})

	// Apply configs — dry-run first, then real run
	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
		"--dry-run",
	}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
	}))

	// Retrieve the team so we have its ID
	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)

	// Locate the FMA installer for no-team and assert labels_include_all is set
	noTeamTitles, _, _, err := s.DS.ListSoftwareTitles(ctx,
		fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, noTeamTitles, 1)
	noTeamTitleID := noTeamTitles[0].ID

	noTeamMeta, err := s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, noTeamTitleID, false)
	require.NoError(t, err)
	require.Empty(t, noTeamMeta.LabelsIncludeAny)
	require.Empty(t, noTeamMeta.LabelsExcludeAny)
	require.Len(t, noTeamMeta.LabelsIncludeAll, 1)
	require.Equal(t, lblName, noTeamMeta.LabelsIncludeAll[0].LabelName)

	// Locate the FMA installer for the team and assert labels_include_all is set
	teamTitles, _, _, err := s.DS.ListSoftwareTitles(ctx,
		fleet.SoftwareTitleListOptions{TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, teamTitles, 1)
	teamTitleID := teamTitles[0].ID

	teamMeta, err := s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.Empty(t, teamMeta.LabelsIncludeAny)
	require.Empty(t, teamMeta.LabelsExcludeAny)
	require.Len(t, teamMeta.LabelsIncludeAll, 1)
	require.Equal(t, lblName, teamMeta.LabelsIncludeAll[0].LabelName)

	// Now re-apply without labels_include_all and confirm they are cleared
	err = os.WriteFile(noTeamFilePath, fmt.Appendf(nil, noTeamTemplate, slug, noLabels), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, slug, noLabels, teamName), 0o644)
	require.NoError(t, err)

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
		"--dry-run",
	}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
	}))

	// Labels should now be empty for no-team
	noTeamMeta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, noTeamTitleID, false)
	require.NoError(t, err)
	require.Empty(t, noTeamMeta.LabelsIncludeAny)
	require.Empty(t, noTeamMeta.LabelsExcludeAny)
	require.Empty(t, noTeamMeta.LabelsIncludeAll)

	// Labels should now be empty for the team
	teamMeta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.Empty(t, teamMeta.LabelsIncludeAny)
	require.Empty(t, teamMeta.LabelsExcludeAny)
	require.Empty(t, teamMeta.LabelsIncludeAll)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestFleetGitopsDDMUnsupportedFleetVariable() {
	t := s.T()
	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	// Create a DDM declaration with an unsupported Fleet variable
	declDir := t.TempDir()
	declFile := path.Join(declDir, "decl-unsupported-var.json")
	err := os.WriteFile(declFile, []byte(`{
		"Type": "com.apple.configuration.management.test",
		"Identifier": "com.example.unsupported-var",
		"Payload": {"Value": "$FLEET_VAR_BOZO"}
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

	// Applying a DDM declaration with an unsupported Fleet variable should fail
	_, err = fleetctl.RunAppNoChecks([]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})
	require.ErrorContains(t, err, "Fleet variable $FLEET_VAR_BOZO is not supported in DDM profiles")
}

func (s *enterpriseIntegrationGitopsTestSuite) TestManagedLocalAccount() {
	t := s.T()
	ctx := context.Background()

	originalAppConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.DS.SaveAppConfig(ctx, originalAppConfig))
	})

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	t.Setenv("FLEET_URL", s.Server.URL)

	const (
		globalConfig = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

		noTeamConfig = `name: Unassigned
controls:
  setup_experience:
    enable_create_local_admin_account: true
    end_user_local_account_type: "admin"
policies:
software:
`

		teamConfig = `
controls:
  setup_experience:
    enable_create_local_admin_account: true
    end_user_local_account_type: "admin"
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
	)

	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
		return p
	}

	globalFile := write("global.yml", globalConfig)
	noTeamFile := write("unassigned.yml", noTeamConfig)
	teamName := uuid.NewString()
	teamFile := write("team.yml", fmt.Sprintf(teamConfig, teamName))

	s.assertDryRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", noTeamFile, "-f", teamFile, "--dry-run"}))
	s.assertRealRunOutput(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", noTeamFile, "-f", teamFile}))

	appConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.True(t, appConfig.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
	assert.True(t, appConfig.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
	assert.True(t, appConfig.MDM.MacOSSetup.EndUserLocalAccountType.Valid)
	assert.Equal(t, "admin", appConfig.MDM.MacOSSetup.EndUserLocalAccountType.Value)

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.True(t, team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
	assert.True(t, team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
	assert.True(t, team.Config.MDM.MacOSSetup.EndUserLocalAccountType.Valid)
	assert.Equal(t, "admin", team.Config.MDM.MacOSSetup.EndUserLocalAccountType.Value)
}

// TestDryRunMacOSSetupScriptWithManualAgentInstallConflict tests that both
// dry-run and real gitops runs fail when manual_agent_install is true and a
// macos_script is configured. Regression test for
// https://github.com/fleetdm/fleet/issues/34464
func (s *enterpriseIntegrationGitopsTestSuite) TestDryRunMacOSSetupScriptWithManualAgentInstallConflict() {
	t := s.T()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/signed.pkg")
	}))
	defer bootstrapServer.Close()

	t.Setenv("FLEET_URL", s.Server.URL)

	// Create a setup experience script file
	scriptFile, err := os.CreateTemp(t.TempDir(), "*.sh")
	require.NoError(t, err)
	_, err = scriptFile.WriteString(`echo "setup script"`)
	require.NoError(t, err)
	err = scriptFile.Close()
	require.NoError(t, err)

	// Global config
	const globalTemplate = `
agent_options:
controls:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
reports:
`

	t.Run("team", func(t *testing.T) {
		globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)
		_, err = globalFile.WriteString(globalTemplate)
		require.NoError(t, err)
		err = globalFile.Close()
		require.NoError(t, err)

		teamName := uuid.NewString()
		teamTemplate := `
controls:
  setup_experience:
    macos_bootstrap_package: %s
    macos_manual_agent_install: true
    macos_script: %s
software:
reports:
policies:
agent_options:
name: %s
settings:
  secrets: [{"secret":"enroll_secret"}]
`
		teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)
		_, err = teamFile.WriteString(fmt.Sprintf(teamTemplate, bootstrapServer.URL, scriptFile.Name(), teamName))
		require.NoError(t, err)
		err = teamFile.Close()
		require.NoError(t, err)

		// Dry-run should fail with the manual_agent_install conflict
		fleetctl.RunAppCheckErr(t, []string{
			"gitops", "--config", fleetctlConfig.Name(),
			"-f", globalFile.Name(), "-f", teamFile.Name(),
			"--dry-run",
		}, "macos_manual_agent_install")

		// Actual run should also fail with the same conflict
		fleetctl.RunAppCheckErr(t, []string{
			"gitops", "--config", fleetctlConfig.Name(),
			"-f", globalFile.Name(), "-f", teamFile.Name(),
		}, "macos_manual_agent_install")
	})

	t.Run("no team", func(t *testing.T) {
		globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)
		_, err = globalFile.WriteString(globalTemplate)
		require.NoError(t, err)
		err = globalFile.Close()
		require.NoError(t, err)

		noTeamTemplate := `name: Unassigned
policies:
controls:
  setup_experience:
    macos_manual_agent_install: true
    macos_script: %s
software:
`
		noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)
		_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, scriptFile.Name()))
		require.NoError(t, err)
		err = noTeamFile.Close()
		require.NoError(t, err)
		noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "unassigned.yml")
		err = os.Rename(noTeamFile.Name(), noTeamFilePath)
		require.NoError(t, err)

		// Dry-run should fail with the manual_agent_install conflict
		fleetctl.RunAppCheckErr(t, []string{
			"gitops", "--config", fleetctlConfig.Name(),
			"-f", globalFile.Name(), "-f", noTeamFilePath,
			"--dry-run",
		}, "macos_manual_agent_install")

		// Actual run should also fail with the same conflict
		fleetctl.RunAppCheckErr(t, []string{
			"gitops", "--config", fleetctlConfig.Name(),
			"-f", globalFile.Name(), "-f", noTeamFilePath,
		}, "macos_manual_agent_install")
	})
}
