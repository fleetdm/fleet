package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	teamName       = "Team Test"
	fleetServerURL = "https://fleet.example.com"
	orgName        = "GitOps Test"
)

func TestBasicGlobalGitOps(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables

	_, ds := runServerWithMockedDS(t)

	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration,
	) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) error {
		return nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error { return nil }
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) { return nil, nil }
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) { return nil, nil }

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}
	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Test"
	)
	t.Setenv("FLEET_SERVER_URL", fleetServerURL)

	_, err = tmpFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: $FLEET_SERVER_URL
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: ${ORG_NAME}
  secrets:
`,
	)
	require.NoError(t, err)

	// No file
	var errWriter strings.Builder
	_, err = runAppNoChecks([]string{"gitops", tmpFile.Name()})
	require.Error(t, err)
	assert.Equal(t, `Required flag "f" not set`, err.Error())

	// Blank file
	errWriter.Reset()
	_, err = runAppNoChecks([]string{"gitops", "-f", ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file name cannot be empty")

	// Bad file
	errWriter.Reset()
	_, err = runAppNoChecks([]string{"gitops", "-f", "fileDoesNotExist.yml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")

	// Empty file
	errWriter.Reset()
	badFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = runAppNoChecks([]string{"gitops", "-f", badFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "errors occurred")

	// DoGitOps error
	t.Setenv("ORG_NAME", "")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization name must be present")

	// Dry run
	t.Setenv("ORG_NAME", orgName)
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Empty(t, enrolledSecrets)
}

func TestBasicTeamGitOps(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License: license,
		},
	)

	const secret = "TestSecret"

	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error { return nil }
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile, macDecls []*fleet.MDMAppleDeclaration,
	) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) error {
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.ListTeamPoliciesFunc = func(
		ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions,
	) (teamPolicies []*fleet.Policy, inheritedPolicies []*fleet.Policy, err error) {
		return nil, nil, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) { return nil, nil }
	team := &fleet.Team{
		ID:        1,
		CreatedAt: time.Now(),
		Name:      teamName,
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == teamName {
			return team, nil
		}
		return nil, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return team, nil
		}
		return nil, nil
	}
	var savedTeam *fleet.Team
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.ElementsMatch(t, labels, []string{fleet.BuiltinLabelMacOS14Plus})
		return map[string]uint{fleet.BuiltinLabelMacOS14Plus: 1}, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
		declaration.DeclarationUUID = uuid.NewString()
		return declaration, nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}

	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	t.Setenv("TEST_SECRET", secret)

	_, err = tmpFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
name: ${TEST_TEAM_NAME}
team_settings:
  secrets: [{"secret":"${TEST_SECRET}"}]
`,
	)
	require.NoError(t, err)

	// DoGitOps error
	t.Setenv("TEST_TEAM_NAME", "")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'name' is required")

	// Dry run
	t.Setenv("TEST_TEAM_NAME", teamName)
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	assert.Nil(t, savedTeam)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	require.Len(t, enrolledSecrets, 1)
	assert.Equal(t, secret, enrolledSecrets[0].Secret)
}

func TestFullGlobalGitOps(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	// mdm test configuration must be set so that activating windows MDM works.
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, nil, "../../server/service/testdata")

	// License is not needed because we are not using any premium features in our config.
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			MDMStorage:  new(mock.MDMAppleStore),
			MDMPusher:   mockPusher{},
			FleetConfig: &fleetCfg,
		},
	)

	var appliedScripts []*fleet.Script
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error {
		appliedScripts = scripts
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	var appliedMacProfiles []*fleet.MDMAppleConfigProfile
	var appliedWinProfiles []*fleet.MDMWindowsConfigProfile
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile, macDecls []*fleet.MDMAppleDeclaration,
	) error {
		appliedMacProfiles = macProfiles
		appliedWinProfiles = winProfiles
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return job, nil
	}

	// Policies
	policy := fleet.Policy{}
	policy.ID = 1
	policy.Name = "Policy to delete"
	policyDeleted := false
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) {
		return []*fleet.Policy{&policy}, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		if slices.Contains(ids, 1) {
			return map[uint]*fleet.Policy{1: &policy}, nil
		}
		return nil, nil
	}
	ds.DeleteGlobalPoliciesFunc = func(ctx context.Context, ids []uint) ([]uint, error) {
		policyDeleted = true
		assert.Equal(t, []uint{policy.ID}, ids)
		return ids, nil
	}
	var appliedPolicySpecs []*fleet.PolicySpec
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		appliedPolicySpecs = specs
		return nil
	}

	// Queries
	query := fleet.Query{}
	query.ID = 1
	query.Name = "Query to delete"
	queryDeleted := false
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) {
		return []*fleet.Query{&query}, nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		queryDeleted = true
		assert.Equal(t, []uint{query.ID}, ids)
		return 1, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == query.ID {
			return &query, nil
		}
		return nil, nil
	}
	var appliedQueries []*fleet.Query
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &notFoundError{}
	}
	ds.ApplyQueriesFunc = func(
		ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{},
	) error {
		appliedQueries = queries
		return nil
	}

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}
	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Test"
	)
	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)
	t.Setenv("APPLE_BM_DEFAULT_TEAM", teamName)
	file := "./testdata/gitops/global_config_no_paths.yml"

	// Dry run should fail because Apple BM Default Team does not exist and premium license is not set
	_, err = runAppNoChecks([]string{"gitops", "-f", file, "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "missing or invalid license"))

	// Dry run
	t.Setenv("APPLE_BM_DEFAULT_TEAM", "")
	_ = runAppForTest(t, []string{"gitops", "-f", file, "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")
	assert.Len(t, enrolledSecrets, 0)
	assert.Len(t, appliedPolicySpecs, 0)
	assert.Len(t, appliedQueries, 0)
	assert.Len(t, appliedScripts, 0)
	assert.Len(t, appliedMacProfiles, 0)
	assert.Len(t, appliedWinProfiles, 0)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", file})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Contains(t, string(*savedAppConfig.AgentOptions), "distributed_denylist_duration")
	assert.Len(t, enrolledSecrets, 2)
	assert.True(t, policyDeleted)
	assert.Len(t, appliedPolicySpecs, 5)
	assert.True(t, queryDeleted)
	assert.Len(t, appliedQueries, 3)
	assert.Len(t, appliedScripts, 1)
	assert.Len(t, appliedMacProfiles, 1)
	assert.Len(t, appliedWinProfiles, 1)
	require.Len(t, savedAppConfig.Integrations.GoogleCalendar, 1)
	assert.Equal(t, "service@example.com", savedAppConfig.Integrations.GoogleCalendar[0].ApiKey["client_email"])
	assert.True(t, savedAppConfig.ActivityExpirySettings.ActivityExpiryEnabled)
	assert.Equal(t, 60, savedAppConfig.ActivityExpirySettings.ActivityExpiryWindow)
	assert.True(t, savedAppConfig.ServerSettings.AIFeaturesDisabled)
}

func TestFullTeamGitOps(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	// mdm test configuration must be set so that activating windows MDM works.
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, nil, "../../server/service/testdata")

	// License is not needed because we are not using any premium features in our config.
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:          license,
			MDMStorage:       new(mock.MDMAppleStore),
			MDMPusher:        mockPusher{},
			FleetConfig:      &fleetCfg,
			NoCacheDatastore: true,
		},
	)

	appConfig := fleet.AppConfig{
		// During dry run, the global calendar integration setting may not be set
		MDM: fleet.MDM{
			EnabledAndConfigured:        true,
			WindowsEnabledAndConfigured: true,
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &appConfig, nil
	}

	var appliedScripts []*fleet.Script
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error {
		appliedScripts = scripts
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	var appliedMacProfiles []*fleet.MDMAppleConfigProfile
	var appliedWinProfiles []*fleet.MDMWindowsConfigProfile
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile, macDecls []*fleet.MDMAppleDeclaration,
	) error {
		appliedMacProfiles = macProfiles
		appliedWinProfiles = winProfiles
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return job, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.ElementsMatch(t, labels, []string{fleet.BuiltinLabelMacOS14Plus})
		return map[string]uint{fleet.BuiltinLabelMacOS14Plus: 1}, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
		declaration.DeclarationUUID = uuid.NewString()
		return declaration, nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}

	// Team
	team := &fleet.Team{
		ID:        1,
		CreatedAt: time.Now(),
		Name:      teamName,
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == teamName {
			return team, nil
		}
		return nil, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return team, nil
		}
		return nil, nil
	}
	var savedTeam *fleet.Team
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}

	// Policies
	policy := fleet.Policy{}
	policy.ID = 1
	policy.Name = "Policy to delete"
	policy.TeamID = &team.ID
	policyDeleted := false
	ds.ListTeamPoliciesFunc = func(
		ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions,
	) (teamPolicies []*fleet.Policy, inheritedPolicies []*fleet.Policy, err error) {
		return []*fleet.Policy{&policy}, nil, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		if slices.Contains(ids, 1) {
			return map[uint]*fleet.Policy{1: &policy}, nil
		}
		return nil, nil
	}
	ds.DeleteTeamPoliciesFunc = func(ctx context.Context, teamID uint, IDs []uint) ([]uint, error) {
		policyDeleted = true
		assert.Equal(t, []uint{policy.ID}, IDs)
		return []uint{policy.ID}, nil
	}
	var appliedPolicySpecs []*fleet.PolicySpec
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		appliedPolicySpecs = specs
		return nil
	}

	// Queries
	query := fleet.Query{}
	query.ID = 1
	query.TeamID = &team.ID
	query.Name = "Query to delete"
	queryDeleted := false
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) {
		return []*fleet.Query{&query}, nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		queryDeleted = true
		assert.Equal(t, []uint{query.ID}, ids)
		return 1, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == query.ID {
			return &query, nil
		}
		return nil, nil
	}
	var appliedQueries []*fleet.Query
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &notFoundError{}
	}
	ds.ApplyQueriesFunc = func(
		ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{},
	) error {
		appliedQueries = queries
		return nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}

	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	t.Setenv("TEST_TEAM_NAME", teamName)

	// Dry run
	file := "./testdata/gitops/team_config_no_paths.yml"
	_ = runAppForTest(t, []string{"gitops", "-f", file, "--dry-run"})
	assert.Nil(t, savedTeam)
	assert.Len(t, enrolledSecrets, 0)
	assert.Len(t, appliedPolicySpecs, 0)
	assert.Len(t, appliedQueries, 0)
	assert.Len(t, appliedScripts, 0)
	assert.Len(t, appliedMacProfiles, 0)
	assert.Len(t, appliedWinProfiles, 0)

	// Real run
	// Setting global calendar config
	appConfig.Integrations = fleet.Integrations{
		GoogleCalendar: []*fleet.GoogleCalendarIntegration{{}},
	}
	_ = runAppForTest(t, []string{"gitops", "-f", file})
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	assert.Contains(t, string(*savedTeam.Config.AgentOptions), "distributed_denylist_duration")
	assert.True(t, savedTeam.Config.Features.EnableHostUsers)
	assert.Equal(t, 30, savedTeam.Config.HostExpirySettings.HostExpiryWindow)
	assert.True(t, savedTeam.Config.MDM.EnableDiskEncryption)
	assert.Len(t, enrolledSecrets, 2)
	assert.True(t, policyDeleted)
	assert.Len(t, appliedPolicySpecs, 5)
	assert.True(t, queryDeleted)
	assert.Len(t, appliedQueries, 3)
	assert.Len(t, appliedScripts, 1)
	assert.Len(t, appliedMacProfiles, 1)
	assert.Len(t, appliedWinProfiles, 1)
	assert.True(t, savedTeam.Config.WebhookSettings.HostStatusWebhook.Enable)
	assert.Equal(t, "https://example.com/host_status_webhook", savedTeam.Config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.NotNil(t, savedTeam.Config.Integrations.GoogleCalendar)
	assert.True(t, savedTeam.Config.Integrations.GoogleCalendar.Enable)

	// Now clear the settings
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	secret := "TestSecret"
	t.Setenv("TEST_SECRET", secret)

	_, err = tmpFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
name: ${TEST_TEAM_NAME}
team_settings:
  secrets: [{"secret":"${TEST_SECRET}"}]
`,
	)
	require.NoError(t, err)

	// Dry run
	savedTeam = nil
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	assert.Nil(t, savedTeam)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	require.Len(t, enrolledSecrets, 1)
	assert.Equal(t, secret, enrolledSecrets[0].Secret)
	assert.False(t, savedTeam.Config.WebhookSettings.HostStatusWebhook.Enable)
	assert.Equal(t, "", savedTeam.Config.WebhookSettings.HostStatusWebhook.DestinationURL)
	assert.NotNil(t, savedTeam.Config.Integrations.GoogleCalendar)
	assert.False(t, savedTeam.Config.Integrations.GoogleCalendar.Enable)
	assert.Empty(t, savedTeam.Config.Integrations.GoogleCalendar)
	assert.Empty(t, savedTeam.Config.MDM.MacOSSettings.CustomSettings)
	assert.Empty(t, savedTeam.Config.MDM.WindowsSettings.CustomSettings.Value)
	assert.Empty(t, savedTeam.Config.MDM.MacOSUpdates.Deadline.Value)
	assert.Empty(t, savedTeam.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	assert.Empty(t, savedTeam.Config.MDM.MacOSSetup.BootstrapPackage.Value)
	assert.False(t, savedTeam.Config.MDM.EnableDiskEncryption)
}

func TestBasicGlobalAndTeamGitOps(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License: license,
		},
	)

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Test"
		secret         = "TestSecret"
	)
	var enrolledSecrets []*fleet.EnrollSecret
	var enrolledTeamSecrets []*fleet.EnrollSecret
	var savedTeam *fleet.Team
	team := &fleet.Team{
		ID:        1,
		CreatedAt: time.Now(),
		Name:      teamName,
	}

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		if teamID == nil {
			enrolledSecrets = secrets
		} else {
			enrolledTeamSecrets = secrets
		}
		return nil
	}
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration,
	) error {
		return nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error { return nil }
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) error {
		return nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.ElementsMatch(t, labels, []string{fleet.BuiltinLabelMacOS14Plus})
		return map[string]uint{fleet.BuiltinLabelMacOS14Plus: 1}, nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) { return nil, nil }
	ds.ListTeamPoliciesFunc = func(
		ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions,
	) (teamPolicies []*fleet.Policy, inheritedPolicies []*fleet.Policy, err error) {
		return nil, nil, nil
	}
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return nil, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) { return nil, nil }
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = 1
		return job, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return team, nil
		}
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == teamName {
			return team, nil
		}
		return nil, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (
		*fleet.MDMAppleDeclaration, error,
	) {
		declaration.DeclarationUUID = uuid.NewString()
		return declaration, nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)

	_, err = globalFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: $FLEET_SERVER_URL
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: ${ORG_NAME}
  secrets:
`,
	)
	require.NoError(t, err)

	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	t.Setenv("TEST_TEAM_NAME", teamName)
	t.Setenv("TEST_SECRET", secret)

	_, err = teamFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
name: ${TEST_TEAM_NAME}
team_settings:
  secrets: [{"secret":"${TEST_SECRET}"}]
`,
	)
	require.NoError(t, err)

	// Files out of order
	_, err = runAppNoChecks([]string{"gitops", "-f", teamFile.Name(), "-f", globalFile.Name(), "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "must be the global config"))

	// Global file specified multiple times
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "-f", globalFile.Name(), "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "only the first file can be the global config"))

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")

	// Dry run, deleting other teams
	assert.False(t, ds.ListTeamsFuncInvoked)
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run", "--delete-other-teams"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")
	assert.True(t, ds.ListTeamsFuncInvoked)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Empty(t, enrolledSecrets)
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	require.Len(t, enrolledTeamSecrets, 1)
	assert.Equal(t, secret, enrolledTeamSecrets[0].Secret)

	// Now, set  up a team to delete
	teamToDeleteID := uint(999)
	teamToDelete := &fleet.Team{
		ID:        teamToDeleteID,
		CreatedAt: time.Now(),
		Name:      "Team to delete",
	}
	ds.ListTeamsFuncInvoked = false
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return []*fleet.Team{teamToDelete, team}, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		switch tid {
		case team.ID:
			return team, nil
		case teamToDeleteID:
			return teamToDelete, nil
		}
		assert.Fail(t, fmt.Sprintf("unexpected team ID %d", tid))
		return teamToDelete, nil
	}
	ds.DeleteTeamFunc = func(ctx context.Context, tid uint) error {
		assert.Equal(t, teamToDeleteID, tid)
		return nil
	}
	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}

	// Real run, deleting other teams
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "--delete-other-teams"})
	assert.True(t, ds.ListTeamsFuncInvoked)
	assert.True(t, ds.DeleteTeamFuncInvoked)

}

func TestFullGlobalAndTeamGitOps(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	// mdm test configuration must be set so that activating windows MDM works.
	ds, savedAppConfigPtr, savedTeamPtr := setupFullGitOpsPremiumServer(t)

	var enrolledSecrets []*fleet.EnrollSecret
	var enrolledTeamSecrets []*fleet.EnrollSecret
	var appliedPolicySpecs []*fleet.PolicySpec
	var appliedQueries []*fleet.Query

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		if teamID == nil {
			enrolledSecrets = secrets
		} else {
			enrolledTeamSecrets = secrets
		}
		return nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		appliedPolicySpecs = specs
		return nil
	}
	ds.ApplyQueriesFunc = func(
		ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{},
	) error {
		appliedQueries = queries
		return nil
	}
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		team.ID = 1
		*savedTeamPtr = team
		enrolledTeamSecrets = team.Secrets
		return *savedTeamPtr, nil
	}

	globalFile := "./testdata/gitops/global_config_no_paths.yml"
	teamFile := "./testdata/gitops/team_config_no_paths.yml"

	// Dry run on global file should fail because Apple BM Default Team does not exist (and has not been provided)
	_, err := runAppNoChecks([]string{"gitops", "-f", globalFile, "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "team name not found"))

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile, "-f", teamFile, "--dry-run", "--delete-other-teams"})
	assert.False(t, ds.SaveAppConfigFuncInvoked)
	assert.Len(t, enrolledSecrets, 0)
	assert.Len(t, enrolledTeamSecrets, 0)
	assert.Len(t, appliedPolicySpecs, 0)
	assert.Len(t, appliedQueries, 0)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile, "-f", teamFile, "--delete-other-teams"})
	assert.Equal(t, orgName, (*savedAppConfigPtr).OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, (*savedAppConfigPtr).ServerSettings.ServerURL)
	assert.Len(t, enrolledSecrets, 2)
	require.NotNil(t, *savedTeamPtr)
	assert.Equal(t, teamName, (*savedTeamPtr).Name)
	require.Len(t, enrolledTeamSecrets, 2)
}

func TestTeamSofwareInstallersGitOps(t *testing.T) {
	// start the web server that will serve the installer
	b, err := os.ReadFile(filepath.Join("..", "..", "server", "service", "testdata", "software-installers", "ruby.deb"))
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "notfound"):
			w.WriteHeader(http.StatusNotFound)
			return
		case strings.HasSuffix(r.URL.Path, ".txt"):
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(`a simple text file`))
			return
		case strings.Contains(r.URL.Path, "toolarge"):
			w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
			var sz int
			for sz < 500*1024*1024 {
				n, _ := w.Write(b)
				sz += n
			}
		default:
			w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
			_, _ = w.Write(b)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("SOFTWARE_INSTALLER_URL", srv.URL)

	cases := []struct {
		file    string
		wantErr string
	}{
		{"testdata/gitops/team_software_installer_not_found.yml", "Please make sure that URLs are publicy accessible to the internet."},
		{"testdata/gitops/team_software_installer_unsupported.yml", "The file should be .pkg, .msi, .exe or .deb."},
		{"testdata/gitops/team_software_installer_too_large.yml", "The maximum file size is 500 MB"},
		{"testdata/gitops/team_software_installer_valid.yml", ""},
		{"testdata/gitops/team_software_installer_pre_condition_multiple_queries.yml", "should have only one query."},
		{"testdata/gitops/team_software_installer_pre_condition_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_install_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_post_install_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_no_url.yml", "software URL is required"},
	}
	for _, c := range cases {
		t.Run(filepath.Base(c.file), func(t *testing.T) {
			setupFullGitOpsPremiumServer(t)

			_, err := runAppNoChecks([]string{"gitops", "-f", c.file})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}

}

func setupFullGitOpsPremiumServer(t *testing.T) (*mock.Store, **fleet.AppConfig, **fleet.Team) {
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, nil, "../../server/service/testdata")

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			MDMStorage:       new(mock.MDMAppleStore),
			MDMPusher:        mockPusher{},
			FleetConfig:      &fleetCfg,
			License:          license,
			NoCacheDatastore: true,
		},
	)

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{
		MDM: fleet.MDM{
			EnabledAndConfigured: true,
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appConfigCopy := *savedAppConfig
		return &appConfigCopy, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		appConfigCopy := *config
		savedAppConfig = &appConfigCopy
		return nil
	}

	var savedTeam *fleet.Team

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		return nil
	}
	ds.ApplyQueriesFunc = func(
		ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{},
	) error {
		return nil
	}
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration,
	) error {
		return nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error { return nil }
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) error {
		return nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.ElementsMatch(t, labels, []string{fleet.BuiltinLabelMacOS14Plus})
		return map[string]uint{fleet.BuiltinLabelMacOS14Plus: 1}, nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) { return nil, nil }
	ds.ListTeamPoliciesFunc = func(
		ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions,
	) (teamPolicies []*fleet.Policy, inheritedPolicies []*fleet.Policy, err error) {
		return nil, nil, nil
	}
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		if savedTeam != nil {
			return []*fleet.Team{savedTeam}, nil
		}
		return nil, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) { return nil, nil }
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
		return nil, nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = 1
		return job, nil
	}
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		team.ID = 1
		savedTeam = team
		return savedTeam, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &notFoundError{}
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if savedTeam != nil && tid == savedTeam.ID {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if savedTeam != nil && name == teamName {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (
		*fleet.MDMAppleDeclaration, error,
	) {
		declaration.DeclarationUUID = uuid.NewString()
		return declaration, nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}

	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)
	t.Setenv("TEST_TEAM_NAME", teamName)
	t.Setenv("APPLE_BM_DEFAULT_TEAM", teamName)

	return ds, &savedAppConfig, &savedTeam
}
