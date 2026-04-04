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
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *enterpriseIntegrationGitopsTestSuite) TestDeleteMacOSSetupDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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

func (s *enterpriseIntegrationGitopsTestSuite) TestUnsetConfigurationProfileLabelsDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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
	err = os.WriteFile(globalFile.Name(), fmt.Appendf(nil, globalTemplate, profileFile.Name(), emptyLabelsIncludeAny), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, profileFile.Name(), "", teamName), 0o644)
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
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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
		withLabelsIncludeAll = `
      labels_include_all:
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

	// switch both to labels_include_all
	err = os.WriteFile(noTeamFilePath, fmt.Appendf(nil, noTeamTemplate, withLabelsIncludeAll), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, withLabelsIncludeAll, teamName), 0o644)
	require.NoError(t, err)

	// Apply configs
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t,
		[]string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name()}), true)

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
	err = os.WriteFile(noTeamFilePath, fmt.Appendf(nil, noTeamTemplate, emptyLabelsIncludeAny), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, "", teamName), 0o644)
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
	require.Len(t, meta.LabelsIncludeAll, 0)

	meta, err = s.DS.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, teamTitleID, false)
	require.NoError(t, err)
	require.NotNil(t, meta.TitleID)
	require.Equal(t, teamTitleID, *meta.TitleID)
	require.Len(t, meta.LabelsExcludeAny, 0)
	require.Len(t, meta.LabelsIncludeAny, 0)
	require.Len(t, meta.LabelsIncludeAll, 0)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestNoTeamWebhookSettingsDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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
	require.Contains(t, output, "applying webhook settings for unassigned hosts")
	require.Contains(t, output, "applied webhook settings for unassigned hosts")

	// Verify webhook settings are disabled
	verifyNoTeamWebhookSettings(ctx, t, s.DS, fleet.FailingPoliciesWebhookSettings{
		Enable: false,
	})
}

func (s *enterpriseIntegrationGitopsTestSuite) TestMacOSSetupDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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
queries:
`

		globalConfigOnly = `
agent_options:
controls:
  macos_setup:
    bootstrap_package: %s
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
    bootstrap_package: %s
    manual_agent_install: true
policies:
software:
`

		teamConfig = `
controls:
  macos_setup:
    bootstrap_package: %s
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
	_, err = noTeamFile.WriteString(fmt.Sprintf(noTeamConfig, bootstrapServer.URL))
	require.NoError(t, err)
	err = noTeamFile.Close()
	require.NoError(t, err)
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
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

func (s *enterpriseIntegrationGitopsTestSuite) TestIPASoftwareInstallersDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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
		require.Empty(t, meta.LabelsIncludeAll)
		require.Len(t, meta.LabelsIncludeAny, 1)
		require.Equal(t, lbl.ID, meta.LabelsIncludeAny[0].LabelID)
		require.Empty(t, meta.InstallScript) // install script should be ignored for ipa apps
	}
	require.ElementsMatch(t, []string{"ios_apps", "ipados_apps"}, sources)
	require.ElementsMatch(t, []string{"ios", "ipados"}, platforms)

	// update the team config to clear the label condition
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, `
      - url: ${SOFTWARE_INSTALLER_URL}/ipa_test.ipa
        labels_include_any:
`, teamName), 0o644)
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
		require.Empty(t, meta.LabelsIncludeAll)
	}
	require.ElementsMatch(t, []string{"ios_apps", "ipados_apps"}, sources)
	require.ElementsMatch(t, []string{"ios", "ipados"}, platforms)

	// update the team config to clear all installers
	err = os.WriteFile(teamFile.Name(), fmt.Appendf(nil, teamTemplate, "", teamName), 0o644)
	require.NoError(t, err)

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile.Name(), "-f", teamFile.Name()}), true)

	titles, _, _, err = s.DS.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{AvailableForInstall: true, TeamID: &team.ID},
		fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, titles, 0)
}

// TestGitOpsSoftwareDisplayNameDeprecated tests that display names for software packages and VPP apps
// are properly applied via GitOps.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareDisplayNameDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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

// TestGitOpsSoftwareIconsDeprecated tests that custom icons for software packages
// and fleet maintained apps are properly applied via GitOps.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsSoftwareIconsDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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
    - slug: foodeprecated/darwin
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
    - slug: foodeprecated/darwin
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

func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsTeamLabelsDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

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

// TestGitOpsTeamLabelsMultipleReposDeprecated tests a gitops setup where every team runs from an independent repo.
// Multiple repos are simulated by copying over the example repository multiple times.
func (s *enterpriseIntegrationGitopsTestSuite) TestGitOpsTeamLabelsMultipleReposDeprecated() {
	t := s.T()
	t.Setenv("FLEET_ENABLE_LOG_TOPICS", logging.DeprecatedFieldTopic)

	ctx := context.Background()

	type repoSetup struct {
		user    fleet.User
		cfg     *os.File
		repoDir string
	}
	setups := make([]repoSetup, 0, 2)

	for range 2 {
		user := s.createGitOpsUser(t)
		cfg := s.createFleetctlConfig(t, user)

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
		setups = append(setups, repoSetup{
			user:    user,
			cfg:     cfg,
			repoDir: repoDir,
		})
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
	for i, setup := range setups {
		globalFile := path.Join(setup.repoDir, "default.yml")

		newTeamCfgFile, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)

		require.NoError(t, teamCfgTmpl.Execute(newTeamCfgFile, tmplParams{
			Name:    fmt.Sprintf(" team-%d", i),
			Queries: fmt.Sprintf("\n  - name: query-%d\n    query: SELECT 1", i),
			Labels:  fmt.Sprintf("\n  - name: label-%d\n    label_membership_type: dynamic\n    query: SELECT 1", i),
		}))

		args := []string{"gitops", "--config", setup.cfg.Name(), "-f", globalFile, "-f", newTeamCfgFile.Name()}
		s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)
	}

	for i, setup := range setups {
		user := setup.user
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
	for i, setup := range setups {
		globalFile := path.Join(setup.repoDir, "default.yml")

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

		args := []string{"gitops", "--config", setup.cfg.Name(), "-f", globalFile, "-f", newTeamCfgFile.Name()}
		s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, args), true)
	}

	for i, setup := range setups {
		user := setup.user
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

func (s *enterpriseIntegrationGitopsTestSuite) TestFleetGitopsDeprecated() {
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
		if strings.Contains(fileName, "no-team.yml") {
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
		if strings.Contains(fileName, "no-team.yml") {
			s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "-f", globalFile}), true)
		} else {
			s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName}), true)
		}
	}
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
reports:
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
	require.Equal(t, secretName, secretVariables[0].Name)
	require.Equal(t, secretValue, secretVariables[0].Value)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestSpecialCaseTeamsVPPAppsDeprecated() {
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
			teamName:    "No team",
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
				noTeamFilePath := filepath.Join(filepath.Dir(teamFile.Name()), "no-team.yml")
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

func (s *enterpriseIntegrationGitopsTestSuite) TestDisallowSoftwareSetupExperienceDeprecated() {
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
  macos_setup:
    bootstrap_package: %s
    manual_agent_install: true
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
  macos_setup:
    bootstrap_package: %s
    manual_agent_install: true
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
			teamName:     "No team",
			teamTemplate: testVPP,
			errContains:  ptr.String("Couldn't edit software."),
		},
		{
			testName:     "No team Installers",
			VPPTeam:      "No team",
			teamName:     "No team",
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
				noTeamFilePath := filepath.Join(filepath.Dir(teamFile.Name()), "no-team.yml")
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

func (s *enterpriseIntegrationGitopsTestSuite) TestOmittedTopLevelKeysGlobalDeprecated() {
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
	policies, err := s.DS.ListGlobalPolicies(ctx, fleet.ListOptions{})
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
	policies, err = s.DS.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, policies)

	appCfg, err = s.DS.AppConfig(ctx)
	require.NoError(t, err)

	// Verify agent_options were cleared (set to null).
	require.Nil(t, appCfg.AgentOptions)

	// Verify controls were cleared (disk encryption reverts to false).
	require.False(t, appCfg.MDM.EnableDiskEncryption.Value)

	// Verify reports were cleared.
	queries, _, _, _, err = s.DS.ListQueries(ctx, fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Empty(t, queries)

	// Verify secrets are cleared.
	globalSecrets, err = s.DS.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, globalSecrets)

	// Verify labels are cleared.
	labels, err = s.DS.LabelsByName(ctx, []string{"Test Global Label"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Empty(t, labels)
}

func (s *enterpriseIntegrationGitopsTestSuite) TestFMALabelsIncludeAllDeprecated() {
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
		noTeamTemplate = `name: No team
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
	noTeamFilePath := filepath.Join(filepath.Dir(noTeamFile.Name()), "no-team.yml")
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
		assert.NoError(t, err)
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
	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
		"--dry-run",
	}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
	}), true)

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

	s.assertDryRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
		"--dry-run",
	}), true)
	s.assertRealRunOutputWithDeprecation(t, fleetctl.RunAppForTest(t, []string{
		"gitops", "--config", fleetctlConfig.Name(),
		"-f", globalFile.Name(), "-f", noTeamFilePath, "-f", teamFile.Name(),
	}), true)

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
