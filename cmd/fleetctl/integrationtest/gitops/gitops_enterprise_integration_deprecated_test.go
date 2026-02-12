package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"testing"
	"text/template"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *enterpriseIntegrationGitopsTestSuite) TestDeleteMacOSSetupDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestCAIntegrationsDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}), true)

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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}), true)

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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}), true)

	groupedCAs, err = s.DS.GetGroupedCertificateAuthorities(t.Context(), true)
	require.NoError(t, err)
	assert.Empty(t, groupedCAs.DigiCert)
	assert.Empty(t, groupedCAs.CustomScepProxy)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestUnsetConfigurationProfileLabelsDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestUnsetSoftwareInstallerLabelsDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}), true)

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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}), true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestDeletingNoTeamYAMLDeprecated() {
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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath}), true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestNoTeamWebhookSettingsDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, output, true)

	// Check that webhook settings are mentioned in the output
	require.Contains(t, output, "would've applied webhook settings for 'No team'")

	// Apply the configuration (non-dry-run)
	output = fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath})
	s.assertRealRunOutputWithDeprecation(t, output, true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestRemoveCustomSettingsFromDefaultYAMLDeprecated() {
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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}), true)

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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}), true)

	// Check profile does not exist
	profiles, err = s.DS.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(profiles))
}

func (s *enterpriseIntegrationGitopsTestSuite) TestMacOSSetupDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}), true)

	appConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.True(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)

	team, err := s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.True(t, team.Config.MDM.MacOSSetup.ManualAgentInstall.Value)

	// Apply global configs without no-team
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlyClear.Name(), "-f", teamFileClear.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlyClear.Name(), "-f", teamFileClear.Name()}), true)
	appConfig, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.False(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)
	team, err = s.DS.TeamByName(ctx, teamName)
	require.NoError(t, err)
	assert.False(t, team.Config.MDM.MacOSSetup.ManualAgentInstall.Value)

	// Apply global configs only
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlySet.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFileOnlySet.Name()}), true)
	appConfig, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.True(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestMacOSSetupScriptWithFleetSecretDeprecated() {
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath}), true)

	// Verify the script was saved
	_, err = s.DS.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)

	// Verify the secret was saved
	secretVariables, err := s.DS.GetSecretVariables(ctx, []string{secretName})
	require.NoError(t, err)
	require.Equal(t, secretVariables[0].Name, secretName)
	require.Equal(t, secretVariables[0].Value, secretValue)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestAddManualLabelsDeprecated() {
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
queries:
labels:
  - name: my-label-deprecated
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name()}), true)

	// Verify the label was created and has the correct hosts
	labels, err := s.DS.LabelsByName(ctx, []string{"my-label-deprecated"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, labels, 1)
	label := labels["my-label-deprecated"]
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

func (s *enterpriseIntegrationGitopsTestSuite) TestIPASoftwareInstallersDeprecated() {
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
labels:
  - name: Label1
    label_membership_type: dynamic
    query: SELECT 1
`

		noTeamTemplate = `name: No team
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
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
	err = os.Rename(noTeamFile.Name(), noTeamFilePath)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.Server.URL)
	testing_utils.StartSoftwareInstallerServer(t)

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath}), true)

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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

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
	}
	require.ElementsMatch(t, []string{"ios_apps", "ipados_apps"}, sources)
	require.ElementsMatch(t, []string{"ios", "ipados"}, platforms)

	// update the team config to clear all installers
	err = os.WriteFile(teamFile.Name(), []byte(fmt.Sprintf(teamTemplate, "", teamName)), 0o644)
	require.NoError(t, err)

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 0)
}

// TestGitOpsSoftwareDisplayName tests that display names for software packages and VPP apps

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareDisplayNameDeprecated() {
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
queries:
`

		noTeamTemplate = `name: No team
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
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}), true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareIconsDeprecated() {
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
queries:
`

		noTeamTemplate = `name: No team
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
queries:
policies:
agent_options:
name: %s
team_settings:
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
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
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
			VALUES ('foodeprecated', 'foodeprecated/darwin', 'darwin', 'com.example.foodeprecated')`)
		return err
	})

	// Apply configs
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}), true)

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
		stmt, args, err := sqlx.In("SELECT filename FROM software_title_icons WHERE team_id = ? AND software_title_id IN (?)", 0, teamTitleIDs)
		if err != nil {
			return err
		}
		return sqlx.SelectContext(ctx, q, &teamIconFilenames, stmt, args...)
	})
	require.Len(t, teamIconFilenames, 2)
	require.Equal(t, "icon.png", teamIconFilenames[0])
	require.Equal(t, "icon.png", teamIconFilenames[1])
}

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsTeamLabelsDeprecated() {
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
queries:
labels:
  - name: global-label-one
    label_membership_type: dynamic
    query: SELECT 1
  - name: global-label-two
    label_membership_type: dynamic
    query: SELECT 1
`), 0o644))

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name()}), true)

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
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"enroll_secret"}]
labels:
  - name: team-one-label-one
    label_membership_type: dynamic
    query: SELECT 2
  - name: team-one-label-two
    label_membership_type: dynamic
    query: SELECT 3
`, teamOneName), 0o644))

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", teamOneFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", teamOneFile.Name()}), true)

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
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"enroll_secret"}]
labels:
  - name: team-one-label-one
    label_membership_type: dynamic
    query: SELECT 2
`, teamOneName), 0o644))

	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name()}), true)

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
queries:
labels:
  - name: global-label-one
    label_membership_type: dynamic
    query: SELECT 1

`), 0o644))

	require.NoError(t, os.WriteFile(teamOneFile.Name(), fmt.Appendf(nil,

		`
controls:
software:
queries:
policies:
agent_options:
name: %s
team_settings:
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
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"enroll_secret2"}]
labels:
  - name: team-one-label-one
    label_membership_type: dynamic
    query: SELECT 2
`, teamTwoName), 0o644))

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name(), "-f", teamTwoFile.Name(), "--dry-run"}), true)

	// TODO: Seems like we require two passes to achieve equilibrium?
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name(), "-f", teamTwoFile.Name()}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalFile.Name(), "-f", teamOneFile.Name(), "-f", teamTwoFile.Name()}), true)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsTeamLabelsMultipleReposDeprecated() {
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
queries:{{ .Queries }}
policies:
labels:{{ .Labels }}
agent_options:
name:{{ .Name }}
team_settings:
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

// TestGitOpsVPPAppAutoUpdate tests that auto-update settings for VPP apps (iOS/iPadOS)

func (s *enterpriseIntegrationGitopsTestSuite) TestFleetDesktopSettingsBrowserAlternativeHostDeprecated() {
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
queries:
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
				s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalCfgFile.Name(), "--dry-run"}), true)
				s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetCfg.Name(), "-f", globalCfgFile.Name()}), true)
			}

			storedCfg, err := s.DS.AppConfig(ctx)
			require.NoError(t, err)
			require.NotNil(t, storedCfg)
			require.Equal(t, testCase.Expected, storedCfg.FleetDesktop.AlternativeBrowserHost)
		})
	}
}
