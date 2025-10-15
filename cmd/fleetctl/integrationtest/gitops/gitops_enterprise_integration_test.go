package gitops

import (
	"context"
	"encoding/json"
	"fmt"
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

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/integrationtest"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	appleMdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-git/go-git/v5"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsEnterpriseGitops(t *testing.T) {
	testingSuite := new(enterpriseIntegrationGitopsTestSuite)
	testingSuite.WithServer.Suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type enterpriseIntegrationGitopsTestSuite struct {
	suite.Suite
	integrationtest.WithServer
	fleetCfg config.FleetConfig
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

	serverConfig := service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		FleetConfig:       &fleetCfg,
		MDMStorage:        mdmStorage,
		DEPStorage:        depStorage,
		SCEPStorage:       scepStorage,
		Pool:              redisPool,
		APNSTopic:         "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
		SCEPConfigService: eeservice.NewSCEPConfigService(kitlog.NewLogfmtLogger(os.Stdout), nil),
		DigiCertService:   digicert.NewService(),
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

	teams, err := s.DS.ListTeams(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
	require.NoError(t, err)
	for _, tm := range teams {
		err := s.DS.DeleteTeam(ctx, tm.ID)
		require.NoError(t, err)
	}

	// Clean software installers in "No team" (the others are deleted in ts.DS.DeleteTeam above).
	mysql.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM software_installers WHERE global_or_team_id = 0;`)
		return err
	})
	mysql.ExecAdhocSQL(t, s.DS, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_apps;")
		return err
	})

	lbls, err := s.DS.ListLabels(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
	require.NoError(t, err)
	for _, lbl := range lbls {
		if lbl.LabelType != fleet.LabelTypeBuiltIn {
			err := s.DS.DeleteLabel(ctx, lbl.Name)
			require.NoError(t, err)
		}
	}
}

// TestFleetGitops runs `fleetctl gitops` command on configs in https://github.com/fleetdm/fleet-gitops repo.
// Changes to that repo may cause this test to fail.
func (s *enterpriseIntegrationGitopsTestSuite) TestFleetGitops() {
	t := s.T()
	const fleetGitopsRepo = "https://github.com/fleetdm/fleet-gitops"

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
	t.Setenv("FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET", "workstations_canary_enroll_secret")
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
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"deleted_team_secret"}]
`, deletedTeamName,
		),
	)
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, s.DS)

	// Apply the team to be deleted
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", deletedTeamFile.Name()})

	// Dry run
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})
	for _, fileName := range teamFileNames {
		// When running no-teams, global config must also be provided ...
		if strings.Contains(fileName, "no-team.yml") {
			_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "-f", globalFile, "--dry-run"})
		} else {
			_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "--dry-run"})
		}
	}

	// Dry run with all the files
	args := []string{"gitops", "--config", fleetctlConfig.Name(), "--dry-run", "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = fleetctl.RunAppForTest(t, args)

	// Real run with all the files, but don't delete other teams
	args = []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = fleetctl.RunAppForTest(t, args)

	// Check that all the teams exist
	teamsJSON := fleetctl.RunAppForTest(t, []string{"get", "teams", "--config", fleetctlConfig.Name(), "--json"})
	assert.Equal(t, 3, strings.Count(teamsJSON, "team_id"))

	// Real run with all the files, and delete other teams
	args = []string{"gitops", "--config", fleetctlConfig.Name(), "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = fleetctl.RunAppForTest(t, args)

	// Check that only the right teams exist
	teamsJSON = fleetctl.RunAppForTest(t, []string{"get", "teams", "--config", fleetctlConfig.Name(), "--json"})
	assert.Equal(t, 2, strings.Count(teamsJSON, "team_id"))
	assert.NotContains(t, teamsJSON, deletedTeamName)

	// Real run with one file at a time
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
	for _, fileName := range teamFileNames {
		// When running no-teams, global config must also be provided ...
		if strings.Contains(fileName, "no-team.yml") {
			_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "-f", globalFile})
		} else {
			_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName})
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
		Name:       "GitOps User",
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
queries:
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
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"enroll_secret"}]
`, teamName,
		),
	)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()})

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
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()})

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
  macos_settings:
    custom_settings:
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
queries:
`,
		dirPath,
		digiCertServer.URL,
		scepServer.URL+"/scep",
	))
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})

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
  macos_settings:
    custom_settings:
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
queries:
`,
		dirPath,
		digiCertServer.URL,
		"digicert_upn_2", // minor modification to stored config so gitops run is not a no-op and triggers call to external digicert service
		scepServer.URL+"/scep",
	))
	require.NoError(t, err)

	// Apply configs
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})

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
  macos_settings:
    custom_settings:
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
queries:
`)
	require.NoError(t, err)

	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})

	groupedCAs, err = s.DS.GetGroupedCertificateAuthorities(t.Context(), true)
	require.NoError(t, err)
	assert.Empty(t, groupedCAs.DigiCert)
	assert.Empty(t, groupedCAs.CustomScepProxy)
}

// TestUnsetConfigurationProfileLabels tests the removal of labels associated with a
// configuration profile via gitops.
func (s *enterpriseIntegrationGitopsTestSuite) TestUnsetConfigurationProfileLabels() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)
	lbl, err := s.DS.NewLabel(ctx, &fleet.Label{Name: "Label1", Query: "SELECT 1"})
	require.NoError(t, err)
	require.NotZero(t, lbl.ID)

	profileFile, err := os.CreateTemp(t.TempDir(), "*.mobileconfig")
	require.NoError(t, err)
	_, err = profileFile.WriteString(test.GenerateMDMAppleProfile("test", "test", uuid.NewString()))
	require.NoError(t, err)
	err = profileFile.Close()
	require.NoError(t, err)

	const (
		globalTemplate = `
agent_options:
controls:
  macos_settings:
    custom_settings:
      - path: %s
%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
queries:
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
  macos_settings:
    custom_settings:
      - path: %s
%s
software:
queries:
policies:
agent_options:
name: %s
team_settings:
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
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()})

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
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()})

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
queries:
`

		noTeamTemplate = `name: No team
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
		teamTemplate = `
controls:
software:
  packages:
    - url: ${SOFTWARE_INSTALLER_URL}/ruby.deb
%s
queries:
policies:
agent_options:
name: %s
team_settings:
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
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
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
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()})

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

	// remove the label conditions
	err = os.WriteFile(noTeamFilePath, []byte(fmt.Sprintf(noTeamTemplate, emptyLabelsIncludeAny)), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), []byte(fmt.Sprintf(teamTemplate, "", teamName)), 0o644)
	require.NoError(t, err)

	// Apply configs
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()})

	// the installer is not scoped by label anymore
	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, noTeamTitleID, false)
	require.NoError(t, err)
	require.NotNil(t, meta.TitleID)
	require.Equal(t, noTeamTitleID, *meta.TitleID)
	require.Len(t, meta.LabelsExcludeAny, 0)
	require.Len(t, meta.LabelsIncludeAny, 0)

	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.NotNil(t, meta.TitleID)
	require.Equal(t, teamTitleID, *meta.TitleID)
	require.Len(t, meta.LabelsExcludeAny, 0)
	require.Len(t, meta.LabelsIncludeAny, 0)
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
queries:
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
		noTeamTemplate = `name: No team
policies:
controls:
  macos_setup:
    script: %s
software:
`
	)

	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, scriptFile.Name()))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})

	// Check script existance
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
queries:
`

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplate)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	// Create a no-team.yml file with webhook settings
	const noTeamTemplateWithWebhook = `
name: No team
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
team_settings:
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
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	// Test dry-run first
	output := fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"})

	// Check that webhook settings are mentioned in the output
	require.Contains(t, output, "applying webhook settings for 'No team'")
	require.Contains(t, output, "would've applied webhook settings for 'No team'")

	// Apply the configuration (non-dry-run)
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})

	// Verify the output mentions webhook settings were applied
	require.Contains(t, output, "applying webhook settings for 'No team'")
	require.Contains(t, output, "applied webhook settings for 'No team'")

	// Verify webhook settings were actually applied by checking the database
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable:         true,
		DestinationURL: "https://example.com/no-team-webhook",
		HostBatchSize:  50,
		PolicyIDs:      []uint{1, 2, 3},
	})

	// Test updating webhook settings
	const noTeamTemplateUpdatedWebhook = `
name: No team
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
team_settings:
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
	noTeamFilePathUpdated := filepath.Join(filepath.Dir(noTeamFileUpdated.Name()), "no-team.yml")
	err = os.Rename(noTeamFileUpdated.Name(), noTeamFilePathUpdated)
	require.NoError(t, err)

	// Apply the updated configuration
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathUpdated})

	// Verify the output still mentions webhook settings were applied
	require.Contains(t, output, "applying webhook settings for 'No team'")
	require.Contains(t, output, "applied webhook settings for 'No team'")

	// Verify webhook settings were updated
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable:         false,
		DestinationURL: "https://updated.example.com/webhook",
		HostBatchSize:  100,
		PolicyIDs:      []uint{4, 5},
	})

	// Test removing webhook settings entirely
	const noTeamTemplateNoWebhook = `
name: No team
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
	noTeamFilePathNoWebhook := filepath.Join(filepath.Dir(noTeamFileNoWebhook.Name()), "no-team.yml")
	err = os.Rename(noTeamFileNoWebhook.Name(), noTeamFilePathNoWebhook)
	require.NoError(t, err)

	// Apply configuration without webhook settings
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathNoWebhook})

	// Verify webhook settings are mentioned as being applied (they're applied as nil to clear)
	require.Contains(t, output, "applying webhook settings for 'No team'")
	require.Contains(t, output, "applied webhook settings for 'No team'")

	// Verify webhook settings were cleared
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable: false,
	})

	// Test case: team_settings exists but webhook_settings is nil
	// First, set webhook settings again
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})
	require.Contains(t, output, "applied webhook settings for 'No team'")

	// Verify webhook was set
	webhookSettings = getNoTeamWebhookSettings(ctx, t, s.DS)
	require.True(t, webhookSettings.Enable)

	// Now apply config with team_settings but no webhook_settings
	const noTeamTemplateTeamSettingsNoWebhook = `
name: No team
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
team_settings:
`
	noTeamFileTeamNoWebhook, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFileTeamNoWebhook.WriteString(noTeamTemplateTeamSettingsNoWebhook)
	require.NoError(t, err)
	err = noTeamFileTeamNoWebhook.Close()
	require.NoError(t, err)
	noTeamFilePathTeamNoWebhook := filepath.Join(filepath.Dir(noTeamFileTeamNoWebhook.Name()), "no-team.yml")
	err = os.Rename(noTeamFileTeamNoWebhook.Name(), noTeamFilePathTeamNoWebhook)
	require.NoError(t, err)

	// Apply configuration with team_settings but no webhook_settings
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathTeamNoWebhook})

	// Verify webhook settings are cleared
	require.Contains(t, output, "applying webhook settings for 'No team'")
	require.Contains(t, output, "applied webhook settings for 'No team'")

	// Verify webhook settings are disabled
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable: false,
	})

	// Test case: webhook_settings exists but failing_policies_webhook is nil
	// First, set webhook settings again
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})
	require.Contains(t, output, "applied webhook settings for 'No team'")

	// Verify webhook was set
	webhookSettings = getNoTeamWebhookSettings(ctx, t, s.DS)
	require.True(t, webhookSettings.Enable)

	// Now apply config with webhook_settings but no failing_policies_webhook
	const noTeamTemplateWebhookNoFailing = `
name: No team
policies:
  - name: No Team Test Policy
    query: SELECT 1 FROM osquery_info WHERE version = '0.0.0';
    description: Test policy for no team
    resolution: This is a test
controls:
software:
team_settings:
  webhook_settings:
`
	noTeamFileWebhookNoFailing, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFileWebhookNoFailing.WriteString(noTeamTemplateWebhookNoFailing)
	require.NoError(t, err)
	err = noTeamFileWebhookNoFailing.Close()
	require.NoError(t, err)
	noTeamFilePathWebhookNoFailing := filepath.Join(filepath.Dir(noTeamFileWebhookNoFailing.Name()), "no-team.yml")
	err = os.Rename(noTeamFileWebhookNoFailing.Name(), noTeamFilePathWebhookNoFailing)
	require.NoError(t, err)

	// Apply configuration with webhook_settings but no failing_policies_webhook
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePathWebhookNoFailing})

	// Verify webhook settings are cleared
	require.Contains(t, output, "applying webhook settings for 'No team'")
	require.Contains(t, output, "applied webhook settings for 'No team'")

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
`
	)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(fmt.Sprintf(globalTemplateWithCustomSettings, profileFile.Name()))
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})

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
queries:
`
	)

	globalFile, err = os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFile.WriteString(globalTemplateWithoutCustomSettings)
	require.NoError(t, err)
	err = globalFile.Close()
	require.NoError(t, err)

	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()})

	// Check profile does not exist
	profiles, err = s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(profiles))
}

func (s *enterpriseIntegrationGitopsTestSuite) TestMacOSSetup() {
	t := s.T()
	ctx := context.Background()

	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

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
queries:
`

		globalConfigOnly = `
agent_options:
controls:
  macos_setup:
    manual_agent_install: %t
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
policies:
queries:
`

		noTeamConfig = `name: No team
controls:
  macos_setup:
    manual_agent_install: true
policies:
software:
`

		teamConfig = `
controls:
  macos_setup:
    manual_agent_install: %t
software:
queries:
policies:
agent_options:
name: %s
team_settings:
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
	_, err = noTeamFile.WriteString(noTeamConfig)
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	teamName := uuid.NewString()
	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(teamConfig, true, teamName))
	require.NoError(t, err)
	err = teamFile.Close()
	require.NoError(t, err)
	teamFileClear, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFileClear.WriteString(fmt.Sprintf(teamConfig, false, teamName))
	require.NoError(t, err)
	err = teamFileClear.Close()
	require.NoError(t, err)

	globalFileOnlySet, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileOnlySet.WriteString(fmt.Sprintf(globalConfigOnly, true))
	require.NoError(t, err)
	err = globalFileOnlySet.Close()
	require.NoError(t, err)
	globalFileOnlyClear, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileOnlyClear.WriteString(fmt.Sprintf(globalConfigOnly, false))
	require.NoError(t, err)
	err = globalFileOnlyClear.Close()
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)

	// Apply configs
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()})

	appConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.True(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.True(t, team.Config.MDM.MacOSSetup.ManualAgentInstall.Value)

	// Apply global configs without no-team
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlyClear.Name(), "-f", teamFileClear.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlyClear.Name(), "-f", teamFileClear.Name()})
	appConfig, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.False(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)
	team, err = s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.False(t, team.Config.MDM.MacOSSetup.ManualAgentInstall.Value)

	// Apply global configs only
	_ = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlySet.Name(), "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlySet.Name()})
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
	result, err := s.DS.LabelIDsByName(ctx, []string{nonManagedLabel.Name})
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
	const noTeamTemplate = `name: No team
policies:
controls:
  macos_setup:
    script: %s
software:
`
	noTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamTemplate, scriptFile.Name()))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
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
queries:
`)
	require.NoError(t, err)

	// Apply the configs
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"})
	_ = fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})

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
  macos_settings:
    custom_settings:
      - path: %s
queries: []
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
team_settings:
  secrets:
    - secret: test_secret
agent_options:
  config:
    decorators:
      load:
        - SELECT uuid AS host_uuid FROM system_info;
controls:
  macos_settings:
    custom_settings:
      - path: %s
queries:
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

// for https://github.com/fleetdm/fleet/issues/28107,
// scenario 3: team profile with 2 labels, one is deleted and both are removed from profile in the same apply
func (s *enterpriseIntegrationGitopsTestSuite) TestGitopsDeleteLabelAndRemoveFromProfileScenario3() {
	t := s.T()
	ctx := t.Context()
	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

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

	// create the global gitops file
	globalFile := writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
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
`, filepath.Base(labelA), filepath.Base(labelB)))

	// create the team gitops file
	teamFile := writeGitopsFile(t, gitopsDir, "team-*.yml", fmt.Sprintf(`
name: team1
team_settings:
  features:
    enable_host_users: true
    enable_software_inventory: true
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  secrets:
    - secret: $FLEET_TEAM_ENROLL_SECRET
agent_options:
  config:
    options:
      disable_distributed: false
      distributed_interval: 10
      pack_delimiter: /
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
        labels_include_any:
          - A
          - B
policies:
queries:
software:
`, filepath.Base(profile)))

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_TEAM_ENROLL_SECRET", "team_enroll_secret")

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", teamFile})

	filter := fleet.TeamFilter{User: test.UserAdmin, IncludeObserver: true}
	tms, err := s.DS.ListTeams(ctx, filter, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, tms, 1)

	tm, err := s.DS.TeamByName(ctx, "team1")
	require.NoError(t, err)

	// at this point the profile is valid and has "include any" with labels A and B
	profs, _, err := s.DS.ListMDMConfigProfiles(ctx, &tm.ID, fleet.ListOptions{})
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
	// from Fleet at the same time (so it shouldn't be "broken" in the profile as
	// it is removed from it).
	globalFile = writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
labels:
  - path: ./%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
`, filepath.Base(labelB)))

	teamFile = writeGitopsFile(t, gitopsDir, "team-*.yml", fmt.Sprintf(`
name: team1
team_settings:
  features:
    enable_host_users: true
    enable_software_inventory: true
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  secrets:
    - secret: $FLEET_TEAM_ENROLL_SECRET
agent_options:
  config:
    options:
      disable_distributed: false
      distributed_interval: 10
      pack_delimiter: /
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
policies:
queries:
software:
`, filepath.Base(profile)))

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", teamFile})

	tms, err = s.DS.ListTeams(ctx, filter, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, tms, 1)

	profs, _, err = s.DS.ListMDMConfigProfiles(ctx, &tm.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, "prof1", profs[0].Name)
	// TODO: the following line should fail, it should show as 2 labels and 1
	// broken, but it doesn't... maybe it's a race?
	require.Len(t, profs[0].LabelsIncludeAny, 0)
}

// for https://github.com/fleetdm/fleet/issues/28107,
// scenario 4: create team's profile with labels before gitops run (not via
// gitops), then in gitops delete label and remove from profile in the same
// apply
func (s *enterpriseIntegrationGitopsTestSuite) TestGitopsDeleteLabelAndRemoveFromProfileScenario4() {
	t := s.T()
	ctx := t.Context()
	user := s.createGitOpsUser(t)
	fleetctlConfig := s.createFleetctlConfig(t, user)

	tm, err := s.DS.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	lblA, err := s.DS.NewLabel(ctx, &fleet.Label{
		Name:                "A",
		Query:               "SELECT 1",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		LabelType:           fleet.LabelTypeRegular,
	})
	require.NoError(t, err)
	lblB, err := s.DS.NewLabel(ctx, &fleet.Label{
		Name:                "B",
		Query:               "SELECT 2",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		LabelType:           fleet.LabelTypeRegular,
	})
	require.NoError(t, err)

	rawProfile := `<?xml version="1.0" encoding="UTF-8"?>
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
`
	cp, err := s.DS.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{
		Identifier:   "com.fleet.1",
		Name:         "prof1",
		TeamID:       &tm.ID,
		Scope:        fleet.PayloadScopeSystem,
		Mobileconfig: []byte(rawProfile),
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblA.ID, LabelName: lblA.Name},
			{LabelID: lblB.ID, LabelName: lblB.Name},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, cp)

	gitopsDir := t.TempDir()

	// create the label file (only for "B", "A" will be deleted)
	labelB := writeGitopsFile(t, gitopsDir, "labelB-*.yml", `
- name: B
  query: SELECT 2;
  label_membership_type: dynamic
`)

	// create the profile file
	profile := writeGitopsFile(t, gitopsDir, "profile-*.mobileconfig", rawProfile)

	// create the global gitops file
	globalFile := writeGitopsFile(t, gitopsDir, "default.yml", fmt.Sprintf(`
policies:
queries:
agent_options:
controls:
labels:
  - path: ./%s
org_settings:
  server_settings:
    server_url: $FLEET_URL
  org_info:
    org_name: Fleet
  secrets:
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
`, filepath.Base(labelB)))

	// create the team gitops file
	teamFile := writeGitopsFile(t, gitopsDir, "team-*.yml", fmt.Sprintf(`
name: team1
team_settings:
  features:
    enable_host_users: true
    enable_software_inventory: true
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  secrets:
    - secret: $FLEET_TEAM_ENROLL_SECRET
agent_options:
  config:
    options:
      disable_distributed: false
      distributed_interval: 10
      pack_delimiter: /
controls:
  macos_settings:
    custom_settings:
      - path: ./%s
policies:
queries:
software:
`, filepath.Base(profile)))

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_TEAM_ENROLL_SECRET", "team_enroll_secret")

	// apply this config
	fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "-f", teamFile})

	filter := fleet.TeamFilter{User: test.UserAdmin, IncludeObserver: true}
	tms, err := s.DS.ListTeams(ctx, filter, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, tms, 1)

	profs, _, err := s.DS.ListMDMConfigProfiles(ctx, &tm.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, "prof1", profs[0].Name)
	// TODO: the following line should fail, it should show as 2 labels and 1
	// broken, but it doesn't... maybe it's a race?
	require.Len(t, profs[0].LabelsIncludeAny, 0)
}

// if dir is empty, t.TempDir() is used. Returns the path to the file.
func writeGitopsFile(t *testing.T, dir, pattern, content string) string {
	t.Helper()

	if dir == "" {
		dir = t.TempDir()
	}
	var f *os.File
	var err error
	if strings.Contains(pattern, "*") {
		f, err = os.CreateTemp(dir, pattern)
		require.NoError(t, err)
	} else {
		// use exact name
		f, err = os.Create(filepath.Join(dir, pattern))
		require.NoError(t, err)
	}
	_, err = f.WriteString(content)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	return f.Name()
}
