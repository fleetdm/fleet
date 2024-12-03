package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	teamName       = "Team Test"
	fleetServerURL = "https://fleet.example.com"
	orgName        = "GitOps Test"
)

func TestGitOpsFilenameValidation(t *testing.T) {
	filename := strings.Repeat("a", filenameMaxLength+1)
	_, err := runAppNoChecks([]string{"gitops", "-f", filename})
	assert.ErrorContains(t, err, "file name must be less than")
}

func TestGitOpsBasicGlobalFree(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables

	_, ds := runServerWithMockedDS(t)

	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		return []fleet.ScriptResponse{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
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

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
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

	// Missing controls.
	tmpFile2, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = tmpFile2.WriteString(
		`
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: https://example.com
  org_info:
    contact_url: https://example.com/contact
    org_name: Foobar
  secrets:
`,
	)
	require.NoError(t, err)
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile2.Name()})
	require.Error(t, err)
	assert.Equal(t, `'controls' must be set on global config`, err.Error())

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

func TestGitOpsBasicGlobalPremium(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:         license,
			KeyValueStore:   newMemKeyValueStore(),
			EnableSCEPProxy: true,
		},
	)

	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		return []fleet.ScriptResponse{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
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
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		return map[string]uint{labels[0]: 1}, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
		return &fleet.MDMAppleDeclaration{}, nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return &fleet.Job{}, nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Premium Test"
	)
	t.Setenv("FLEET_SERVER_URL", fleetServerURL)

	_, err = tmpFile.WriteString(
		`
controls:
  ios_updates:
    deadline: "2022-02-02"
    minimum_version: "17.6"
  ipados_updates:
    deadline: "2023-03-03"
    minimum_version: "18.0"
queries:
policies:
agent_options:
org_settings:
  integrations:
    ndes_scep_proxy:
      url: https://ndes.example.com/scep
      admin_url: https://ndes.example.com/admin
      username: ndes_user
      password: ndes_password
  server_settings:
    server_url: $FLEET_SERVER_URL
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: ${ORG_NAME}
  secrets:
software:
`,
	)
	require.NoError(t, err)

	// Dry run
	t.Setenv("ORG_NAME", orgName)
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Empty(t, enrolledSecrets)
	assert.True(t, savedAppConfig.Integrations.NDESSCEPProxy.Valid)
	assert.Equal(t, "https://ndes.example.com/scep", savedAppConfig.Integrations.NDESSCEPProxy.Value.URL)
}

func TestGitOpsBasicTeam(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:       license,
			KeyValueStore: newMemKeyValueStore(),
		},
	)

	const secret = "TestSecret"

	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
		return nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		return []fleet.ScriptResponse{}, nil
	}
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile, macDecls []*fleet.MDMAppleDeclaration,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
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
	var savedTeam *fleet.Team
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == teamName && savedTeam != nil {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		if savedTeam != nil && *savedTeam.Filename == filename {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return savedTeam, nil
		}
		return nil, nil
	}
	var enrolledTeamSecrets []*fleet.EnrollSecret
	ds.NewTeamFunc = func(ctx context.Context, newTeam *fleet.Team) (*fleet.Team, error) {
		newTeam.ID = team.ID
		savedTeam = newTeam
		enrolledTeamSecrets = newTeam.Secrets
		return newTeam, nil
	}
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return true, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Len(t, labels, 1)
		switch labels[0] {
		case fleet.BuiltinLabelMacOS14Plus:
			return map[string]uint{fleet.BuiltinLabelMacOS14Plus: 1}, nil
		case fleet.BuiltinLabelIOS:
			return map[string]uint{fleet.BuiltinLabelIOS: 2}, nil
		case fleet.BuiltinLabelIPadOS:
			return map[string]uint{fleet.BuiltinLabelIPadOS: 3}, nil
		default:
			return nil, &notFoundError{}
		}
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledTeamSecrets = secrets
		return nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
		return &fleet.MDMAppleDeclaration{}, nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return &fleet.Job{}, nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmFilter fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	t.Setenv("TEST_SECRET", "")

	_, err = tmpFile.WriteString(
		`
controls:
  ios_updates:
    deadline: "2024-10-10"
    minimum_version: "18.0"
  ipados_updates:
    deadline: "2025-11-11"
    minimum_version: "17.6"
queries:
policies:
agent_options:
name: ${TEST_TEAM_NAME}
team_settings:
  secrets: ${TEST_SECRET}
software:
`,
	)
	require.NoError(t, err)

	// DoGitOps error
	t.Setenv("TEST_TEAM_NAME", "")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'name' is required")

	// Invalid name for "No team" file (dry and real).
	t.Setenv("TEST_TEAM_NAME", "no TEam")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("file %q for 'No team' must be named 'no-team.yml'", tmpFile.Name()))
	t.Setenv("TEST_TEAM_NAME", "no TEam")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("file %q for 'No team' must be named 'no-team.yml'", tmpFile.Name()))

	t.Setenv("TEST_TEAM_NAME", "All teams")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"All teams" is a reserved team name`)

	t.Setenv("TEST_TEAM_NAME", "All TEAMS")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"All teams" is a reserved team name`)

	// Dry run
	t.Setenv("TEST_TEAM_NAME", teamName)
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	assert.Nil(t, savedTeam)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	assert.Empty(t, enrolledTeamSecrets)

	// The previous run created the team, so let's rerun with an existing team
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	assert.Empty(t, enrolledTeamSecrets)

	// Add a secret
	t.Setenv("TEST_SECRET", fmt.Sprintf("[{\"secret\":\"%s\"}]", secret))
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	require.Len(t, enrolledTeamSecrets, 1)
	assert.Equal(t, secret, enrolledTeamSecrets[0].Secret)
}

func TestGitOpsFullGlobal(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	// mdm test configuration must be set so that activating windows MDM works.
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")

	// License is not needed because we are not using any premium features in our config.
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			MDMStorage:  new(mdmmock.MDMAppleStore),
			MDMPusher:   mockPusher{},
			FleetConfig: &fleetCfg,
		},
	)

	var appliedScripts []*fleet.Script
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		appliedScripts = scripts
		var scriptResponses []fleet.ScriptResponse
		for _, script := range scripts {
			scriptResponses = append(scriptResponses, fleet.ScriptResponse{
				ID:     script.ID,
				Name:   script.Name,
				TeamID: script.TeamID,
			})
		}

		return scriptResponses, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	var appliedMacProfiles []*fleet.MDMAppleConfigProfile
	var appliedWinProfiles []*fleet.MDMWindowsConfigProfile
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile, macDecls []*fleet.MDMAppleDeclaration,
	) (updates fleet.MDMProfilesUpdates, err error) {
		appliedMacProfiles = macProfiles
		appliedWinProfiles = winProfiles
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
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
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return true, nil
	}
	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	// Needed for checking tokens
	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Test"
	)
	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)
	t.Setenv("SOFTWARE_INSTALLER_URL", fleetServerURL)
	file := "./testdata/gitops/global_config_no_paths.yml"

	// Dry run
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
	assert.Equal(t, 2000, savedAppConfig.ServerSettings.QueryReportCap)
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
	assert.True(t, savedAppConfig.WebhookSettings.ActivitiesWebhook.Enable)
	assert.Equal(t, "https://activities_webhook_url", savedAppConfig.WebhookSettings.ActivitiesWebhook.DestinationURL)
}

func TestGitOpsFullTeam(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	// mdm test configuration must be set so that activating windows MDM works.
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")

	// License is not needed because we are not using any premium features in our config.
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:          license,
			MDMStorage:       new(mdmmock.MDMAppleStore),
			MDMPusher:        mockPusher{},
			FleetConfig:      &fleetCfg,
			NoCacheDatastore: true,
			KeyValueStore:    newMemKeyValueStore(),
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
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		appliedScripts = scripts
		var scriptResponses []fleet.ScriptResponse
		for _, script := range scripts {
			scriptResponses = append(scriptResponses, fleet.ScriptResponse{
				ID:     script.ID,
				Name:   script.Name,
				TeamID: script.TeamID,
			})
		}

		return scriptResponses, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	var appliedMacProfiles []*fleet.MDMAppleConfigProfile
	var appliedWinProfiles []*fleet.MDMWindowsConfigProfile
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile, macDecls []*fleet.MDMAppleDeclaration,
	) (updates fleet.MDMProfilesUpdates, err error) {
		appliedMacProfiles = macProfiles
		appliedWinProfiles = winProfiles
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return job, nil
	}
	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, profile fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
		return &profile, nil
	}
	ds.NewMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
		return declaration, nil
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
	var savedTeam *fleet.Team
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == "Conflict" {
			return &fleet.Team{}, nil
		}
		if savedTeam != nil && savedTeam.Name == name {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		if savedTeam != nil && *savedTeam.Filename == filename {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == savedTeam.ID {
			return savedTeam, nil
		}
		return nil, nil
	}
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return true, nil
	}
	const teamID = uint(123)
	var enrolledSecrets []*fleet.EnrollSecret
	ds.NewTeamFunc = func(ctx context.Context, newTeam *fleet.Team) (*fleet.Team, error) {
		newTeam.ID = teamID
		savedTeam = newTeam
		enrolledSecrets = newTeam.Secrets
		return newTeam, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		if team.ID == teamID {
			savedTeam = team
		} else {
			assert.Fail(t, "unexpected team ID when saving team")
		}
		return team, nil
	}

	// Policies
	policy := fleet.Policy{}
	policy.ID = 1
	policy.Name = "Policy to delete"
	policy.TeamID = ptr.Uint(teamID)
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
	query.TeamID = ptr.Uint(teamID)
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
	var appliedSoftwareInstallers []*fleet.UploadSoftwareInstallerPayload
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		appliedSoftwareInstallers = installers
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}
	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
		return nil
	}
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmFilter fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}

	startSoftwareInstallerServer(t)

	t.Setenv("TEST_TEAM_NAME", teamName)

	// Dry run
	const baseFilename = "team_config_no_paths.yml"
	gitopsFile := "./testdata/gitops/" + baseFilename
	_ = runAppForTest(t, []string{"gitops", "-f", gitopsFile, "--dry-run"})
	assert.Nil(t, savedTeam)
	assert.Len(t, enrolledSecrets, 0)
	assert.Len(t, appliedPolicySpecs, 0)
	assert.Len(t, appliedQueries, 0)
	assert.Len(t, appliedScripts, 0)
	assert.Len(t, appliedMacProfiles, 0)
	assert.Len(t, appliedWinProfiles, 0)
	assert.Empty(t, appliedSoftwareInstallers)

	// Real run
	// Setting global calendar config
	appConfig.Integrations = fleet.Integrations{
		GoogleCalendar: []*fleet.GoogleCalendarIntegration{{}},
	}
	_ = runAppForTest(t, []string{"gitops", "-f", gitopsFile})
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
	assert.Equal(t, baseFilename, *savedTeam.Filename)
	require.Len(t, appliedSoftwareInstallers, 2)
	packageID := `"ruby"`
	uninstallScriptProcessed := strings.ReplaceAll(file.GetUninstallScript("deb"), "$PACKAGE_ID", packageID)
	assert.ElementsMatch(t, []string{fmt.Sprintf("echo 'uninstall' %s\n", packageID), uninstallScriptProcessed},
		[]string{appliedSoftwareInstallers[0].UninstallScript, appliedSoftwareInstallers[1].UninstallScript})

	// Change team name
	newTeamName := "New Team Name"
	t.Setenv("TEST_TEAM_NAME", newTeamName)
	_ = runAppForTest(t, []string{"gitops", "-f", gitopsFile, "--dry-run"})
	_ = runAppForTest(t, []string{"gitops", "-f", gitopsFile})
	require.NotNil(t, savedTeam)
	assert.Equal(t, newTeamName, savedTeam.Name)
	assert.Equal(t, baseFilename, *savedTeam.Filename)

	// Try to change team name again, but this time the new name conflicts with an existing team
	t.Setenv("TEST_TEAM_NAME", "Conflict")
	_, err = runAppNoChecks([]string{"gitops", "-f", gitopsFile, "--dry-run"})
	assert.ErrorContains(t, err, "team name already exists")
	_, err = runAppNoChecks([]string{"gitops", "-f", gitopsFile})
	assert.ErrorContains(t, err, "team name already exists")

	// Now clear the settings
	t.Setenv("TEST_TEAM_NAME", newTeamName)
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
software:
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
	assert.Equal(t, newTeamName, savedTeam.Name)
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
	assert.Equal(t, filepath.Base(tmpFile.Name()), *savedTeam.Filename)
}

func createFakeITunesAndVPPServices(t *testing.T) {
	config := &appleVPPConfigSrvConf{
		Assets: []vpp.Asset{
			{
				AdamID:         "1",
				PricingParam:   "STDQ",
				AvailableCount: 12,
			},
			{
				AdamID:         "2",
				PricingParam:   "STDQ",
				AvailableCount: 3,
			},
		},
		SerialNumbers: []string{"123", "456"},
	}
	startVPPApplyServer(t, config)

	appleITunesSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a map of apps we can respond with
		db := map[string]string{
			// macos app
			"1": `{"bundleId": "a-1", "artworkUrl512": "https://example.com/images/1", "version": "1.0.0", "trackName": "App 1", "TrackID": 1}`,
			// macos, ios, ipados app
			"2": `{"bundleId": "b-2", "artworkUrl512": "https://example.com/images/2", "version": "2.0.0", "trackName": "App 2", "TrackID": 2,
				"supportedDevices": ["MacDesktop-MacDesktop", "iPhone5s-iPhone5s", "iPadAir-iPadAir"] }`,
			// ipados app
			"3": `{"bundleId": "c-3", "artworkUrl512": "https://example.com/images/3", "version": "3.0.0", "trackName": "App 3", "TrackID": 3,
				"supportedDevices": ["iPadAir-iPadAir"] }`,
		}

		adamIDString := r.URL.Query().Get("id")
		adamIDs := strings.Split(adamIDString, ",")

		var objs []string
		for _, a := range adamIDs {
			objs = append(objs, db[a])
		}

		_, _ = w.Write([]byte(fmt.Sprintf(`{"results": [%s]}`, strings.Join(objs, ","))))
	}))
	t.Setenv("FLEET_DEV_ITUNES_URL", appleITunesSrv.URL)
}

func TestGitOpsBasicGlobalAndTeam(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:       license,
			KeyValueStore: newMemKeyValueStore(),
		},
	)

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appConfig := savedAppConfig.Copy()
		return appConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}

	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
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

	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return true, nil
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
	) (updates fleet.MDMProfilesUpdates, err error) {
		assert.Empty(t, macProfiles)
		assert.Empty(t, winProfiles)
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		assert.Empty(t, scripts)
		return []fleet.ScriptResponse{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		assert.Empty(t, profileUUIDs)
		return fleet.MDMProfilesUpdates{}, nil
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
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = 1
		return job, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return savedTeam, nil
		}
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == teamName && savedTeam != nil {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		if savedTeam != nil && *savedTeam.Filename == filename {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.NewTeamFunc = func(ctx context.Context, newTeam *fleet.Team) (*fleet.Team, error) {
		newTeam.ID = team.ID
		savedTeam = newTeam
		enrolledTeamSecrets = newTeam.Secrets
		return newTeam, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmFilter fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	vppToken := &fleet.VPPTokenDB{
		Location:  "Foobar",
		RenewDate: time.Now().Add(24 * 365 * time.Hour),
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{vppToken}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}
	ds.CountABMTokensFunc = func(ctx context.Context) (uint32, error) {
		return 0, nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}

	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		var teamsSummary []*fleet.TeamSummary
		if savedTeam != nil {
			teamsSummary = append(teamsSummary, &fleet.TeamSummary{
				ID:          savedTeam.ID,
				Name:        savedTeam.Name,
				Description: savedTeam.Description,
			})
		}
		return teamsSummary, nil
	}

	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
		if teamID != nil && *teamID == savedTeam.ID {
			return vppToken, nil
		}
		return nil, &notFoundError{}
	}

	ds.UpdateVPPTokenTeamsFunc = func(ctx context.Context, id uint, teams []uint) (*fleet.VPPTokenDB, error) {
		return vppToken, nil
	}

	createFakeITunesAndVPPServices(t)

	globalFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)
	t.Setenv("TEST_TEAM_NAME", teamName)
	t.Setenv("TEST_SECRET", secret)

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
  mdm:
    volume_purchasing_program:
    - location: Foobar
      teams:
      - "${TEST_TEAM_NAME}"
  secrets: [{"secret":"globalSecret"}]
software:
`,
	)
	require.NoError(t, err)

	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	_, err = teamFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
name: ${TEST_TEAM_NAME}
team_settings:
  secrets: [{"secret":"${TEST_SECRET}"}]
software:
  app_store_apps:
    - app_store_id: '1'
`,
	)
	require.NoError(t, err)

	teamFileDupSecret, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFileDupSecret.WriteString(
		`
controls:
queries:
policies:
agent_options:
name: ${TEST_TEAM_NAME}
team_settings:
  secrets: [{"secret":"${TEST_SECRET}"},{"secret":"globalSecret"}]
software:
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

	// Duplicate secret
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFile.Name(), "-f", teamFileDupSecret.Name(), "--dry-run"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate enroll secret found")

	ds.GetVPPTokenByTeamIDFuncInvoked = false

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")

	// Dry run should not attempt to get the VPP token when applying VPP apps (it may not exist).
	require.False(t, ds.GetVPPTokenByTeamIDFuncInvoked)
	ds.ListTeamsFuncInvoked = false

	// Dry run, deleting other teams
	savedAppConfig = &fleet.AppConfig{}
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run", "--delete-other-teams"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")
	assert.True(t, ds.ListTeamsFuncInvoked)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Len(t, enrolledSecrets, 1)
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	require.Len(t, enrolledTeamSecrets, 1)
	assert.Equal(t, secret, enrolledTeamSecrets[0].Secret)

	// Dry run again (after team was created by real run)
	ds.GetVPPTokenByTeamIDFuncInvoked = false
	_ = runAppForTest(t, []string{"gitops", "-f", globalFile.Name(), "-f", teamFile.Name(), "--dry-run"})
	// Dry run should not attempt to get the VPP token when applying VPP apps (it may not exist).
	require.False(t, ds.GetVPPTokenByTeamIDFuncInvoked)

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

func TestGitOpsBasicGlobalAndNoTeam(t *testing.T) {
	// Cannot run t.Parallel() because runServerWithMockedDS sets the FLEET_SERVER_ADDRESS
	// environment variable.

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:       license,
			KeyValueStore: newMemKeyValueStore(),
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
	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
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

	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return true, nil
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
	) (updates fleet.MDMProfilesUpdates, err error) {
		assert.Empty(t, macProfiles)
		assert.Empty(t, winProfiles)
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		assert.Empty(t, scripts)
		return []fleet.ScriptResponse{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		assert.Empty(t, profileUUIDs)
		return fleet.MDMProfilesUpdates{}, nil
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
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = 1
		return job, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team.ID {
			return savedTeam, nil
		}
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == teamName && savedTeam != nil {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		if savedTeam != nil && *savedTeam.Filename == filename {
			return savedTeam, nil
		}
		return nil, &notFoundError{}
	}
	ds.NewTeamFunc = func(ctx context.Context, newTeam *fleet.Team) (*fleet.Team, error) {
		newTeam.ID = team.ID
		savedTeam = newTeam
		enrolledTeamSecrets = newTeam.Secrets
		return newTeam, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeam = team
		return team, nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmFilter fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}

	globalFileBasic, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	_, err = globalFileBasic.WriteString(fmt.Sprintf(
		`
controls:
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: %s
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: %s
  secrets: [{"secret":"globalSecret"}]
software:
`, fleetServerURL, orgName),
	)
	require.NoError(t, err)

	globalFileWithSoftware, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileWithSoftware.WriteString(fmt.Sprintf(
		`
controls:
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: %s
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: %s
  secrets: [{"secret":"globalSecret"}]
software:
  packages:
    - url: https://example.com
`, fleetServerURL, orgName),
	)
	require.NoError(t, err)

	globalFileWithControls, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileWithControls.WriteString(fmt.Sprintf(
		`
controls:
  ios_updates:
    deadline: "2022-02-02"
    minimum_version: "17.6"
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: %s
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: %s
  secrets: [{"secret":"globalSecret"}]
software:
`, fleetServerURL, orgName),
	)
	require.NoError(t, err)

	globalFileWithoutControlsAndSoftwareKeys, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = globalFileWithoutControlsAndSoftwareKeys.WriteString(fmt.Sprintf(
		`
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: %s
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: %s
  secrets: [{"secret":"globalSecret"}]
`, fleetServerURL, orgName),
	)
	require.NoError(t, err)

	teamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = teamFile.WriteString(fmt.Sprintf(`
controls:
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"%s"}]
software:
`, teamName, secret),
	)
	require.NoError(t, err)

	noTeamFilePath := filepath.Join(t.TempDir(), "no-team.yml")
	noTeamFile, err := os.Create(noTeamFilePath)
	require.NoError(t, err)
	_, err = noTeamFile.WriteString(`
controls:
policies:
name: No team
software:
`)
	require.NoError(t, err)

	noTeamFilePathPoliciesCalendarPath := filepath.Join(t.TempDir(), "no-team.yml")
	noTeamFilePathPoliciesCalendar, err := os.Create(noTeamFilePathPoliciesCalendarPath)
	require.NoError(t, err)
	_, err = noTeamFilePathPoliciesCalendar.WriteString(`
controls:
policies:
  - name: Foobar
    query: SELECT 1 FROM osquery_info WHERE start_time < 0;
    calendar_events_enabled: true
name: No team
software:
`)
	require.NoError(t, err)

	noTeamFilePathWithControls := filepath.Join(t.TempDir(), "no-team.yml")
	noTeamFileWithControls, err := os.Create(noTeamFilePathWithControls)
	require.NoError(t, err)
	_, err = noTeamFileWithControls.WriteString(`
controls:
  ipados_updates:
    deadline: "2023-03-03"
    minimum_version: "18.0"
policies:
name: No team
software:
`)
	require.NoError(t, err)

	noTeamFilePathWithoutControls := filepath.Join(t.TempDir(), "no-team.yml")
	noTeamFileWithoutControls, err := os.Create(noTeamFilePathWithoutControls)
	require.NoError(t, err)
	_, err = noTeamFileWithoutControls.WriteString(`
policies:
name: No team
software:
`)
	require.NoError(t, err)

	// Dry run, global defines software, should fail.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithSoftware.Name(), "-f", teamFile.Name(), "-f", noTeamFile.Name(), "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "'software' cannot be set on global file"))
	// Real run, global defines software, should fail.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithSoftware.Name(), "-f", teamFile.Name(), "-f", noTeamFile.Name()})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "'software' cannot be set on global file"))

	// Dry run, both global and no-team.yml define controls.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithControls.Name(), "-f", teamFile.Name(), "-f", noTeamFileWithControls.Name(), "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "'controls' cannot be set on both global config and on no-team.yml"))
	// Real run, both global and no-team.yml define controls.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithControls.Name(), "-f", teamFile.Name(), "-f", noTeamFileWithControls.Name()})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "'controls' cannot be set on both global config and on no-team.yml"))

	// Dry run, both global and no-team.yml defines policy with calendar events enabled.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithControls.Name(), "-f", teamFile.Name(), "-f", noTeamFilePathPoliciesCalendar.Name(), "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "calendar events are not supported on \"No team\" policies: \"Foobar\""), err.Error())
	// Real run, both global and no-team.yml define controls.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithControls.Name(), "-f", teamFile.Name(), "-f", noTeamFilePathPoliciesCalendar.Name()})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "calendar events are not supported on \"No team\" policies: \"Foobar\""), err.Error())

	// Dry run, controls should be defined somewhere, either in no-team.yml or global.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithoutControlsAndSoftwareKeys.Name(), "-f", teamFile.Name(), "-f", noTeamFileWithoutControls.Name(), "--dry-run"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "'controls' must be set on global config or no-team.yml"))
	// Real run, both global and no-team.yml define controls.
	_, err = runAppNoChecks([]string{"gitops", "-f", globalFileWithoutControlsAndSoftwareKeys.Name(), "-f", teamFile.Name(), "-f", noTeamFileWithoutControls.Name()})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "'controls' must be set on global config or no-team.yml"))

	// Dry run, global file without controls and software keys.
	_ = runAppForTest(t, []string{"gitops", "-f", globalFileWithoutControlsAndSoftwareKeys.Name(), "-f", teamFile.Name(), "-f", noTeamFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")

	// Real run, global file without controls and software keys.
	_ = runAppForTest(t, []string{"gitops", "-f", globalFileWithoutControlsAndSoftwareKeys.Name(), "-f", teamFile.Name(), "-f", noTeamFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Len(t, enrolledSecrets, 1)
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	require.Len(t, enrolledTeamSecrets, 1)
	assert.Equal(t, secret, enrolledTeamSecrets[0].Secret)

	// Restore to test below.
	savedAppConfig = &fleet.AppConfig{}

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFileBasic.Name(), "-f", teamFile.Name(), "-f", noTeamFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")
	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", globalFileBasic.Name(), "-f", teamFile.Name(), "-f", noTeamFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Len(t, enrolledSecrets, 1)
	require.NotNil(t, savedTeam)
	assert.Equal(t, teamName, savedTeam.Name)
	require.Len(t, enrolledTeamSecrets, 1)
	assert.Equal(t, secret, enrolledTeamSecrets[0].Secret)
}

func TestGitOpsFullGlobalAndTeam(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables
	// mdm test configuration must be set so that activating windows MDM works.
	ds, savedAppConfigPtr, savedTeams := setupFullGitOpsPremiumServer(t)
	startSoftwareInstallerServer(t)

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
		enrolledTeamSecrets = team.Secrets
		savedTeams[team.Name] = &team
		return team, nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}
	ds.CountABMTokensFunc = func(ctx context.Context) (uint32, error) {
		return 0, nil
	}

	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes()
	require.NoError(t, err)
	crt, key, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	scepCert := tokenpki.PEMCertificate(crt.Raw)
	scepKey := tokenpki.PEMRSAPrivateKey(key)

	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert:   {Value: scepCert},
			fleet.MDMAssetCAKey:    {Value: scepKey},
			fleet.MDMAssetAPNSKey:  {Value: apnsKey},
			fleet.MDMAssetAPNSCert: {Value: apnsCert},
		}, nil
	}

	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
		return nil
	}

	globalFile := "./testdata/gitops/global_config_no_paths.yml"
	teamFile := "./testdata/gitops/team_config_no_paths.yml"

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
	require.NotNil(t, *savedTeams[teamName])
	assert.Equal(t, teamName, (*savedTeams[teamName]).Name)
	require.Len(t, enrolledTeamSecrets, 2)
}

func TestGitOpsTeamSofwareInstallers(t *testing.T) {
	startSoftwareInstallerServer(t)
	startAndServeVPPServer(t)

	cases := []struct {
		file    string
		wantErr string
	}{
		{"testdata/gitops/team_software_installer_not_found.yml", "Please make sure that URLs are reachable from your Fleet server."},
		{"testdata/gitops/team_software_installer_unsupported.yml", "The file should be .pkg, .msi, .exe, .deb or .rpm."},
		// commenting out, results in the process getting killed on CI and on some machines
		// {"testdata/gitops/team_software_installer_too_large.yml", "The maximum file size is 3 GB"},
		{"testdata/gitops/team_software_installer_valid.yml", ""},
		{"testdata/gitops/team_software_installer_subdir.yml", ""},
		{"testdata/gitops/subdir/team_software_installer_valid.yml", ""},
		{"testdata/gitops/team_software_installer_valid_apply.yml", ""},
		{"testdata/gitops/team_software_installer_pre_condition_multiple_queries.yml", "should have only one query."},
		{"testdata/gitops/team_software_installer_pre_condition_multiple_queries_apply.yml", "should have only one query."},
		{"testdata/gitops/team_software_installer_pre_condition_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_install_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_uninstall_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_post_install_not_found.yml", "no such file or directory"},
		{"testdata/gitops/team_software_installer_no_url.yml", "software URL is required"},
		{"testdata/gitops/team_software_installer_invalid_self_service_value.yml", "\"packages.self_service\" must be a bool, found string"},
		// team tests for setup experience software/script
		{"testdata/gitops/team_setup_software_valid.yml", ""},
		{"testdata/gitops/team_setup_software_invalid_script.yml", "no_such_script.sh: no such file"},
		{"testdata/gitops/team_setup_software_invalid_software_package.yml", "no_such_software.yml\" does not exist for that team"},
		{"testdata/gitops/team_setup_software_invalid_vpp_app.yml", "\"no_such_app\" does not exist for that team"},
	}
	for _, c := range cases {
		t.Run(filepath.Base(c.file), func(t *testing.T) {
			ds, _, _ := setupFullGitOpsPremiumServer(t)
			tokExpire := time.Now().Add(time.Hour)
			token, err := test.CreateVPPTokenEncoded(tokExpire, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}
			ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
				return &fleet.VPPTokenDB{
					ID:        1,
					OrgName:   "Fleet",
					Location:  "Earth",
					RenewDate: tokExpire,
					Token:     string(token),
					Teams:     nil,
				}, nil
			}

			_, err = runAppNoChecks([]string{"gitops", "-f", c.file})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestGitOpsTeamSoftwareInstallersQueryEnv(t *testing.T) {
	startSoftwareInstallerServer(t)
	ds, _, _ := setupFullGitOpsPremiumServer(t)

	t.Setenv("QUERY_VAR", "IT_WORKS")

	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, tmID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		if installers[0].PreInstallQuery != "select IT_WORKS" {
			return fmt.Errorf("Missing env var, got %s", installers[0].PreInstallQuery)
		}
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}

	_, err := runAppNoChecks([]string{"gitops", "-f", "testdata/gitops/team_software_installer_valid_env_query.yml"})
	require.NoError(t, err)
}

func TestGitOpsNoTeamSoftwareInstallers(t *testing.T) {
	startSoftwareInstallerServer(t)
	startAndServeVPPServer(t)

	cases := []struct {
		noTeamFile string
		wantErr    string
	}{
		{"testdata/gitops/no_team_software_installer_not_found.yml", "Please make sure that URLs are reachable from your Fleet server."},
		{"testdata/gitops/no_team_software_installer_unsupported.yml", "The file should be .pkg, .msi, .exe, .deb or .rpm."},
		// commenting out, results in the process getting killed on CI and on some machines
		// {"testdata/gitops/no_team_software_installer_too_large.yml", "The maximum file size is 3 GB"},
		{"testdata/gitops/no_team_software_installer_valid.yml", ""},
		{"testdata/gitops/no_team_software_installer_subdir.yml", ""},
		{"testdata/gitops/subdir/no_team_software_installer_valid.yml", ""},
		{"testdata/gitops/no_team_software_installer_pre_condition_multiple_queries.yml", "should have only one query."},
		{"testdata/gitops/no_team_software_installer_pre_condition_not_found.yml", "no such file or directory"},
		{"testdata/gitops/no_team_software_installer_install_not_found.yml", "no such file or directory"},
		{"testdata/gitops/no_team_software_installer_uninstall_not_found.yml", "no such file or directory"},
		{"testdata/gitops/no_team_software_installer_post_install_not_found.yml", "no such file or directory"},
		{"testdata/gitops/no_team_software_installer_no_url.yml", "software URL is required"},
		{"testdata/gitops/no_team_software_installer_invalid_self_service_value.yml", "\"packages.self_service\" must be a bool, found string"},
		// No team tests for setup experience software/script
		{"testdata/gitops/no_team_setup_software_valid.yml", ""},
		{"testdata/gitops/no_team_setup_software_invalid_script.yml", "no_such_script.sh: no such file"},
		{"testdata/gitops/no_team_setup_software_invalid_software_package.yml", "no_such_software.yml\" does not exist for that team"},
		{"testdata/gitops/no_team_setup_software_invalid_vpp_app.yml", "\"no_such_app\" does not exist for that team"},
	}
	for _, c := range cases {
		t.Run(filepath.Base(c.noTeamFile), func(t *testing.T) {
			ds, _, _ := setupFullGitOpsPremiumServer(t)
			tokExpire := time.Now().Add(time.Hour)
			token, err := test.CreateVPPTokenEncoded(tokExpire, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}
			ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
				return &fleet.VPPTokenDB{
					ID:        1,
					OrgName:   "Fleet",
					Location:  "Earth",
					RenewDate: tokExpire,
					Token:     string(token),
					Teams:     nil,
				}, nil
			}

			t.Setenv("APPLE_BM_DEFAULT_TEAM", "")
			globalFile := "./testdata/gitops/global_config_no_paths.yml"
			if strings.HasPrefix(filepath.Base(c.noTeamFile), "no_team_setup_software") {
				// the controls section is in the no-team test file, so use a global file without that section
				globalFile = "./testdata/gitops/global_config_no_paths_no_controls.yml"
			}
			dstPath := filepath.Join(filepath.Dir(c.noTeamFile), "no-team.yml")
			t.Cleanup(func() {
				os.Remove(dstPath)
			})
			err = file.Copy(c.noTeamFile, dstPath, 0o755)
			require.NoError(t, err)
			_, err = runAppNoChecks([]string{"gitops", "-f", globalFile, "-f", dstPath})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestGitOpsTeamVPPApps(t *testing.T) {
	startAndServeVPPServer(t)

	cases := []struct {
		file            string
		wantErr         string
		tokenExpiration time.Time
	}{
		{"testdata/gitops/team_vpp_valid_app.yml", "", time.Now().Add(24 * time.Hour)},
		{"testdata/gitops/team_vpp_valid_app_self_service.yml", "", time.Now().Add(24 * time.Hour)},
		{"testdata/gitops/team_vpp_valid_empty.yml", "", time.Now().Add(24 * time.Hour)},
		{"testdata/gitops/team_vpp_valid_empty.yml", "", time.Now().Add(-24 * time.Hour)},
		{"testdata/gitops/team_vpp_valid_app.yml", "VPP token expired", time.Now().Add(-24 * time.Hour)},
		{"testdata/gitops/team_vpp_invalid_app.yml", "app not available on vpp account", time.Now().Add(24 * time.Hour)},
		{"testdata/gitops/team_vpp_incorrect_type.yml", "\"app_store_apps.app_store_id\" must be a string, found number", time.Now().Add(24 * time.Hour)},
		{"testdata/gitops/team_vpp_empty_adamid.yml", "software app store id required", time.Now().Add(24 * time.Hour)},
	}

	for _, c := range cases {
		t.Run(filepath.Base(c.file), func(t *testing.T) {
			ds, _, _ := setupFullGitOpsPremiumServer(t)
			token, err := test.CreateVPPTokenEncoded(c.tokenExpiration, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}

			ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
				return &fleet.VPPTokenDB{
					ID:        1,
					OrgName:   "Fleet",
					Location:  "Earth",
					RenewDate: c.tokenExpiration,
					Token:     string(token),
					Teams:     nil,
				}, nil
			}

			_, err = runAppNoChecks([]string{"gitops", "-f", c.file})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestGitOpsCustomSettings(t *testing.T) {
	cases := []struct {
		file    string
		wantErr string
	}{
		{"testdata/gitops/global_macos_windows_custom_settings_valid.yml", ""},
		{"testdata/gitops/global_macos_custom_settings_valid_deprecated.yml", ""},
		{"testdata/gitops/global_windows_custom_settings_invalid_label_mix.yml", `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included`},
		{"testdata/gitops/global_windows_custom_settings_invalid_label_mix_2.yml", `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included`},
		{"testdata/gitops/global_windows_custom_settings_unknown_label.yml", `some or all the labels provided don't exist`},
		{"testdata/gitops/team_macos_windows_custom_settings_valid.yml", ""},
		{"testdata/gitops/team_macos_custom_settings_valid_deprecated.yml", ""},
		{"testdata/gitops/team_macos_windows_custom_settings_invalid_labels_mix.yml", `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included`},
		{"testdata/gitops/team_macos_windows_custom_settings_invalid_labels_mix_2.yml", `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included`},
		{"testdata/gitops/team_macos_windows_custom_settings_unknown_label.yml", `some or all the labels provided don't exist`},
	}
	for _, c := range cases {
		t.Run(filepath.Base(c.file), func(t *testing.T) {
			ds, appCfgPtr, _ := setupFullGitOpsPremiumServer(t)
			(*appCfgPtr).MDM.EnabledAndConfigured = true
			(*appCfgPtr).MDM.WindowsEnabledAndConfigured = true
			labelToIDs := map[string]uint{
				fleet.BuiltinLabelMacOS14Plus: 1,
				"A":                           2,
				"B":                           3,
				"C":                           4,
			}
			ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
				// for this test, recognize labels A, B and C (as well as the built-in macos 14+ one)
				ret := make(map[string]uint)
				for _, lbl := range labels {
					id, ok := labelToIDs[lbl]
					if ok {
						ret[lbl] = id
					}
				}
				return ret, nil
			}
			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}

			_, err := runAppNoChecks([]string{"gitops", "-f", c.file})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func startSoftwareInstallerServer(t *testing.T) {
	// start the web server that will serve the installer
	b, err := os.ReadFile(filepath.Join("..", "..", "server", "service", "testdata", "software-installers", "ruby.deb"))
	require.NoError(t, err)

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
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
					for sz < 3000*1024*1024 {
						n, _ := w.Write(b)
						sz += n
					}
				default:
					w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
					_, _ = w.Write(b)
				}
			},
		),
	)
	t.Cleanup(srv.Close)
	t.Setenv("SOFTWARE_INSTALLER_URL", srv.URL)
}

type appleVPPConfigSrvConf struct {
	Assets        []vpp.Asset
	SerialNumbers []string
}

func startVPPApplyServer(t *testing.T, config *appleVPPConfigSrvConf) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "associate") {
			var associations vpp.AssociateAssetsRequest

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&associations); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if len(associations.Assets) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				res := vpp.ErrorResponse{
					ErrorNumber:  9718,
					ErrorMessage: "This request doesn't contain an asset, which is a required argument. Change the request to provide an asset.",
				}
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
				return
			}

			if len(associations.SerialNumbers) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				res := vpp.ErrorResponse{
					ErrorNumber:  9719,
					ErrorMessage: "Either clientUserIds or serialNumbers are required arguments. Change the request to provide assignable users and devices.",
				}
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
				return
			}

			var badAssets []vpp.Asset
			for _, reqAsset := range associations.Assets {
				var found bool
				for _, goodAsset := range config.Assets {
					if reqAsset == goodAsset {
						found = true
					}
				}
				if !found {
					badAssets = append(badAssets, reqAsset)
				}
			}

			var badSerials []string
			for _, reqSerial := range associations.SerialNumbers {
				var found bool
				for _, goodSerial := range config.SerialNumbers {
					if reqSerial == goodSerial {
						found = true
					}
				}
				if !found {
					badSerials = append(badSerials, reqSerial)
				}
			}

			if len(badAssets) != 0 || len(badSerials) != 0 {
				errMsg := "error associating assets."
				if len(badAssets) > 0 {
					var badAdamIds []string
					for _, asset := range badAssets {
						badAdamIds = append(badAdamIds, asset.AdamID)
					}
					errMsg += fmt.Sprintf(" assets don't exist on account: %s.", strings.Join(badAdamIds, ", "))
				}
				if len(badSerials) > 0 {
					errMsg += fmt.Sprintf(" bad serials: %s.", strings.Join(badSerials, ", "))
				}
				res := vpp.ErrorResponse{
					ErrorInfo: vpp.ResponseErrorInfo{
						Assets:        badAssets,
						ClientUserIds: []string{"something"},
						SerialNumbers: badSerials,
					},
					// Not sure what error should be returned on each
					// error type
					ErrorNumber:  1,
					ErrorMessage: errMsg,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
			}
			return
		}

		if strings.Contains(r.URL.Path, "assets") {
			// Then we're responding to GetAssets
			w.Header().Set("Content-Type", "application/json")
			encoder := json.NewEncoder(w)
			err := encoder.Encode(map[string][]vpp.Asset{"assets": config.Assets})
			if err != nil {
				panic(err)
			}
			return
		}

		resp := []byte(`{"locationName": "Fleet Location One"}`)
		if strings.Contains(r.URL.RawQuery, "invalidToken") {
			// This replicates the response sent back from Apple's VPP endpoints when an invalid
			// token is passed. For more details see:
			// https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			// https://developer.apple.com/documentation/devicemanagement/client_config
			// https://developer.apple.com/documentation/devicemanagement/errorresponse
			// Note that the Apple server returns 200 in this case.
			resp = []byte(`{"errorNumber": 9622,"errorMessage": "Invalid authentication token"}`)
		}

		if strings.Contains(r.URL.RawQuery, "serverError") {
			resp = []byte(`{"errorNumber": 9603,"errorMessage": "Internal server error"}`)
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, _ = w.Write(resp)
	}))

	t.Setenv("FLEET_DEV_VPP_URL", srv.URL)
	t.Cleanup(srv.Close)
}

func startAndServeVPPServer(t *testing.T) {
	config := &appleVPPConfigSrvConf{
		Assets: []vpp.Asset{
			{
				AdamID:         "1",
				PricingParam:   "STDQ",
				AvailableCount: 12,
			},
			{
				AdamID:         "2",
				PricingParam:   "STDQ",
				AvailableCount: 3,
			},
		},
		SerialNumbers: []string{"123", "456"},
	}

	startVPPApplyServer(t, config)

	appleITunesSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a map of apps we can respond with
		db := map[string]string{
			// macos app
			"1": `{"bundleId": "a-1", "artworkUrl512": "https://example.com/images/1", "version": "1.0.0", "trackName": "App 1", "TrackID": 1}`,
			// macos, ios, ipados app
			"2": `{"bundleId": "b-2", "artworkUrl512": "https://example.com/images/2", "version": "2.0.0", "trackName": "App 2", "TrackID": 2,
				"supportedDevices": ["MacDesktop-MacDesktop", "iPhone5s-iPhone5s", "iPadAir-iPadAir"] }`,
			// ipados app
			"3": `{"bundleId": "c-3", "artworkUrl512": "https://example.com/images/3", "version": "3.0.0", "trackName": "App 3", "TrackID": 3,
				"supportedDevices": ["iPadAir-iPadAir"] }`,
		}

		adamIDString := r.URL.Query().Get("id")
		adamIDs := strings.Split(adamIDString, ",")

		var objs []string
		for _, a := range adamIDs {
			objs = append(objs, db[a])
		}

		_, _ = w.Write([]byte(fmt.Sprintf(`{"results": [%s]}`, strings.Join(objs, ","))))
	}))
	t.Setenv("FLEET_DEV_ITUNES_URL", appleITunesSrv.URL)
}

func setupFullGitOpsPremiumServer(t *testing.T) (*mock.Store, **fleet.AppConfig, map[string]**fleet.Team) {
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			MDMStorage:       new(mdmmock.MDMAppleStore),
			MDMPusher:        mockPusher{},
			FleetConfig:      &fleetCfg,
			License:          license,
			NoCacheDatastore: true,
			KeyValueStore:    newMemKeyValueStore(),
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
	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
		return nil
	}

	savedTeams := map[string]**fleet.Team{}

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
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		return []fleet.ScriptResponse{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return true, nil
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
		if savedTeams != nil {
			var result []*fleet.Team
			for _, t := range savedTeams {
				result = append(result, *t)
			}
			return result, nil
		}
		return nil, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) { return nil, nil }
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
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
		team.ID = uint(len(savedTeams) + 1) //nolint:gosec // dismiss G115
		savedTeams[team.Name] = &team
		return team, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &notFoundError{}
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		for _, tm := range savedTeams {
			if (*tm).ID == tid {
				return *tm, nil
			}
		}
		return nil, &notFoundError{}
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		for _, tm := range savedTeams {
			if (*tm).Name == name {
				return *tm, nil
			}
		}
		return nil, &notFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		for _, tm := range savedTeams {
			if *(*tm).Filename == filename {
				return *tm, nil
			}
		}
		return nil, &notFoundError{}
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeams[team.Name] = &team
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
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}

	ds.InsertVPPTokenFunc = func(ctx context.Context, tok *fleet.VPPTokenData) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, nil
	}
	ds.GetVPPTokenFunc = func(ctx context.Context, tokenID uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, err
	}
	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return nil, nil
	}
	ds.UpdateVPPTokenTeamsFunc = func(ctx context.Context, id uint, teams []uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}}, nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmFilter fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}
	ds.SetSetupExperienceScriptFunc = func(ctx context.Context, script *fleet.Script) error {
		return nil
	}

	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)
	t.Setenv("TEST_TEAM_NAME", teamName)
	t.Setenv("APPLE_BM_DEFAULT_TEAM", teamName)

	return ds, &savedAppConfig, savedTeams
}

func TestGitOpsABM(t *testing.T) {
	global := func(mdm string) string {
		return fmt.Sprintf(`
controls:
queries:
policies:
agent_options:
software:
org_settings:
  server_settings:
    server_url: "https://foo.example.com"
  org_info:
    org_name: GitOps Test
  secrets:
    - secret: "global"
  mdm:
    %s
 `, mdm)
	}

	team := func(name string) string {
		return fmt.Sprintf(`
name: %s
team_settings:
  secrets:
    - secret: "%s-secret"
agent_options:
controls:
policies:
queries:
software:
`, name, name)
	}

	workstations := team("💻 Workstations")
	iosTeam := team("📱🏢 Company-owned iPhones")
	ipadTeam := team("🔳🏢 Company-owned iPads")

	cases := []struct {
		name             string
		cfgs             []string
		dryRunAssertion  func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error)
		realRunAssertion func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error)
		tokens           []*fleet.ABMToken
	}{
		{
			name: "backwards compat",
			cfgs: []string{
				global("apple_bm_default_team: 💻 Workstations"),
				workstations,
			},
			tokens: []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Equal(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam, "💻 Workstations")
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "deprecated config with two tokens in the db fails",
			cfgs: []string{
				global("apple_bm_default_team: 💻 Workstations"),
				workstations,
			},
			tokens: []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}, {OrganizationName: "Second Token LLC"}},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				t.Logf("got: %s", out)
				require.ErrorContains(t, err, "mdm.apple_bm_default_team has been deprecated")
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.NotContains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				require.ErrorContains(t, err, "mdm.apple_bm_default_team has been deprecated")
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.NotContains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "new key all valid",
			cfgs: []string{
				global(`
                                  apple_business_manager:
                                    - organization_name: Fleet Device Management Inc.
                                      macos_team: "💻 Workstations"
                                      ios_team: "📱🏢 Company-owned iPhones"
                                      ipados_team: "🔳🏢 Company-owned iPads"`),
				workstations,
				iosTeam,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.ElementsMatch(
					t,
					appCfg.MDM.AppleBusinessManager.Value,
					[]fleet.MDMAppleABMAssignmentInfo{
						{
							OrganizationName: "Fleet Device Management Inc.",
							MacOSTeam:        "💻 Workstations",
							IOSTeam:          "📱🏢 Company-owned iPhones",
							IpadOSTeam:       "🔳🏢 Company-owned iPads",
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "new key multiple elements",
			cfgs: []string{
				global(`
                                  apple_business_manager:
                                    - organization_name: Foo Inc.
                                      macos_team: "💻 Workstations"
                                      ios_team: "📱🏢 Company-owned iPhones"
                                      ipados_team: "🔳🏢 Company-owned iPads"
                                    - organization_name: Fleet Device Management Inc.
                                      macos_team: "💻 Workstations"
                                      ios_team: "📱🏢 Company-owned iPhones"
                                      ipados_team: "🔳🏢 Company-owned iPads"`),
				workstations,
				iosTeam,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.ElementsMatch(
					t,
					appCfg.MDM.AppleBusinessManager.Value,
					[]fleet.MDMAppleABMAssignmentInfo{
						{
							OrganizationName: "Fleet Device Management Inc.",
							MacOSTeam:        "💻 Workstations",
							IOSTeam:          "📱🏢 Company-owned iPhones",
							IpadOSTeam:       "🔳🏢 Company-owned iPads",
						},
						{
							OrganizationName: "Foo Inc.",
							MacOSTeam:        "💻 Workstations",
							IOSTeam:          "📱🏢 Company-owned iPhones",
							IpadOSTeam:       "🔳🏢 Company-owned iPads",
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "both keys errors",
			cfgs: []string{
				global(`
                                  apple_bm_default_team: "💻 Workstations"
                                  apple_business_manager:
                                    - organization_name: Fleet Device Management Inc.
                                      macos_team: "💻 Workstations"
                                      ios_team: "📱🏢 Company-owned iPhones"
                                      ipados_team: "🔳🏢 Company-owned iPads"`),
				workstations,
				iosTeam,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				require.ErrorContains(t, err, "mdm.apple_bm_default_team has been deprecated")
				assert.NotContains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				require.ErrorContains(t, err, "mdm.apple_bm_default_team has been deprecated")
				assert.NotContains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "using an undefined team errors",
			cfgs: []string{
				global(`
                                  apple_business_manager:
                                    - organization_name: Fleet Device Management Inc.
                                      macos_team: "💻 Workstations"
                                      ios_team: "📱🏢 Company-owned iPhones"
                                      ipados_team: "🔳🏢 Company-owned iPads"`),
				workstations,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "apple_business_manager team \"📱🏢 Company-owned iPhones\" not found in team configs")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "apple_business_manager team \"📱🏢 Company-owned iPhones\" not found in team configs")
			},
		},
		{
			name: "no team is supported",
			cfgs: []string{
				global(`
                                  apple_business_manager:
                                    - organization_name: Fleet Device Management Inc.
                                      macos_team: "No team"
                                      ios_team: "No team"
                                      ipados_team: "No team"`),
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.ElementsMatch(
					t,
					appCfg.MDM.AppleBusinessManager.Value,
					[]fleet.MDMAppleABMAssignmentInfo{
						{
							OrganizationName: "Fleet Device Management Inc.",
							MacOSTeam:        "No team",
							IOSTeam:          "No team",
							IpadOSTeam:       "No team",
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "not provided teams defaults to no team",
			cfgs: []string{
				global(`
                                  apple_business_manager:
                                    - organization_name: Fleet Device Management Inc.
                                      macos_team: "No team"
                                      ios_team: ""`),
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.ElementsMatch(
					t,
					appCfg.MDM.AppleBusinessManager.Value,
					[]fleet.MDMAppleABMAssignmentInfo{
						{
							OrganizationName: "Fleet Device Management Inc.",
							MacOSTeam:        "No team",
							IOSTeam:          "",
							IpadOSTeam:       "",
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "non existent org name fails",
			cfgs: []string{
				global(`
                                  apple_business_manager:
                                    - organization_name: Does not exist
                                      macos_team: "No team"`),
			},
			tokens: []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "token with organization name Does not exist doesn't exist")
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.NotContains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "token with organization name Does not exist doesn't exist")
				assert.Empty(t, appCfg.MDM.AppleBusinessManager.Value)
				assert.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
				assert.NotContains(t, out, "[!] gitops dry run succeeded")
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ds, savedAppConfigPtr, savedTeams := setupFullGitOpsPremiumServer(t)

			ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
				if len(tt.tokens) > 0 {
					return tt.tokens, nil
				}
				return []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}, {OrganizationName: "Foo Inc."}}, nil
			}
			ds.CountABMTokensFunc = func(ctx context.Context) (uint32, error) {
				return uint32(len(tt.tokens)), nil // nolint:gosec // dismiss G115
			}

			ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
				var res []*fleet.TeamSummary
				for _, tm := range savedTeams {
					res = append(res, &fleet.TeamSummary{Name: (*tm).Name, ID: (*tm).ID})
				}
				return res, nil
			}

			ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
				return nil
			}

			args := []string{"gitops"}
			for _, cfg := range tt.cfgs {
				if cfg != "" {
					tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString(cfg)
					require.NoError(t, err)
					args = append(args, "-f", tmpFile.Name())
				}
			}

			// Dry run
			out, err := runAppNoChecks(append(args, "--dry-run"))
			tt.dryRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)
			if t.Failed() {
				t.FailNow()
			}

			// Real run
			out, err = runAppNoChecks(args)
			tt.realRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)

			// Second real run, now that all the teams are saved
			out, err = runAppNoChecks(args)
			tt.realRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)
		})
	}
}

func TestGitOpsVPP(t *testing.T) {
	global := func(mdm string) string {
		return fmt.Sprintf(`
controls:
queries:
policies:
agent_options:
software:
org_settings:
  server_settings:
    server_url: "https://foo.example.com"
  org_info:
    org_name: GitOps Test
  secrets:
    - secret: "global"
  mdm:
    %s
 `, mdm)
	}

	team := func(name string) string {
		return fmt.Sprintf(`
name: %s
team_settings:
  secrets:
    - secret: "%s-secret"
agent_options:
controls:
policies:
queries:
software:
`, name, name)
	}

	workstations := team("💻 Workstations")
	iosTeam := team("📱🏢 Company-owned iPhones")
	ipadTeam := team("🔳🏢 Company-owned iPads")

	cases := []struct {
		name             string
		cfgs             []string
		dryRunAssertion  func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error)
		realRunAssertion func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error)
	}{
		{
			name: "new key all valid",
			cfgs: []string{
				global(`
                                  volume_purchasing_program:
                                    - location: Fleet Device Management Inc.
                                      teams:
                                        - "💻 Workstations"
                                        - "📱🏢 Company-owned iPhones"
                                        - "🔳🏢 Company-owned iPads"`),
				workstations,
				iosTeam,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.ElementsMatch(
					t,
					appCfg.MDM.VolumePurchasingProgram.Value,
					[]fleet.MDMAppleVolumePurchasingProgramInfo{
						{
							Location: "Fleet Device Management Inc.",
							Teams: []string{
								"💻 Workstations",
								"📱🏢 Company-owned iPhones",
								"🔳🏢 Company-owned iPads",
							},
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "new key multiple elements",
			cfgs: []string{
				global(`
                                  volume_purchasing_program:
                                    - location: Acme Inc.
                                      teams:
                                        - "💻 Workstations"
                                    - location: Fleet Device Management Inc.
                                      teams:
                                        - "📱🏢 Company-owned iPhones"
                                        - "🔳🏢 Company-owned iPads"`),
				workstations,
				iosTeam,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.ElementsMatch(
					t,
					appCfg.MDM.VolumePurchasingProgram.Value,
					[]fleet.MDMAppleVolumePurchasingProgramInfo{
						{
							Location: "Acme Inc.",
							Teams: []string{
								"💻 Workstations",
							},
						},
						{
							Location: "Fleet Device Management Inc.",
							Teams: []string{
								"📱🏢 Company-owned iPhones",
								"🔳🏢 Company-owned iPads",
							},
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "using an undefined team errors",
			cfgs: []string{
				global(`
                                  volume_purchasing_program:
                                    - location: Fleet Device Management Inc.
                                      teams:
                                        - "💻 Workstations"
                                        - "📱🏢 Company-owned iPhones"
                                        - "🔳🏢 Company-owned iPads"`),
				workstations,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "volume_purchasing_program team 📱🏢 Company-owned iPhones not found in team configs")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "volume_purchasing_program team 📱🏢 Company-owned iPhones not found in team configs")
			},
		},
		{
			name: "no team is supported",
			cfgs: []string{
				global(`
                                  volume_purchasing_program:
                                    - location: Fleet Device Management Inc.
                                      teams:
                                        - "💻 Workstations"
                                        - "📱🏢 Company-owned iPhones"
                                        - "No team"`),
				workstations,
				iosTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.ElementsMatch(
					t,
					appCfg.MDM.VolumePurchasingProgram.Value,
					[]fleet.MDMAppleVolumePurchasingProgramInfo{
						{
							Location: "Fleet Device Management Inc.",
							Teams: []string{
								"💻 Workstations",
								"📱🏢 Company-owned iPhones",
								"No team",
							},
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "all teams is supported",
			cfgs: []string{
				global(`
                        volume_purchasing_program:
                          - location: Fleet Device Management Inc.
                            teams:
                              - "All teams"`),
				workstations,
				iosTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.ElementsMatch(
					t,
					appCfg.MDM.VolumePurchasingProgram.Value,
					[]fleet.MDMAppleVolumePurchasingProgramInfo{
						{
							Location: "Fleet Device Management Inc.",
							Teams: []string{
								"All teams",
							},
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "not provided teams defaults to no team",
			cfgs: []string{
				global(`
                                  volume_purchasing_program:
                                    - location: Fleet Device Management Inc.
                                      teams:`),
				workstations,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.Contains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.NoError(t, err)
				assert.ElementsMatch(
					t,
					appCfg.MDM.VolumePurchasingProgram.Value,
					[]fleet.MDMAppleVolumePurchasingProgramInfo{
						{
							Location: "Fleet Device Management Inc.",
							Teams:    nil,
						},
					},
				)
				assert.Contains(t, out, "[!] gitops succeeded")
			},
		},
		{
			name: "non existent location fails",
			cfgs: []string{
				global(`
                                  volume_purchasing_program:
                                    - location: Does not exist
                                      teams:`),
				workstations,
				ipadTeam,
			},
			dryRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "token with location Does not exist doesn't exist")
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.NotContains(t, out, "[!] gitops dry run succeeded")
			},
			realRunAssertion: func(t *testing.T, appCfg *fleet.AppConfig, ds fleet.Datastore, out string, err error) {
				assert.ErrorContains(t, err, "token with location Does not exist doesn't exist")
				assert.Empty(t, appCfg.MDM.VolumePurchasingProgram.Value)
				assert.NotContains(t, out, "[!] gitops dry run succeeded")
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ds, savedAppConfigPtr, savedTeams := setupFullGitOpsPremiumServer(t)

			ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
				return []*fleet.VPPTokenDB{{Location: "Fleet Device Management Inc."}, {Location: "Acme Inc."}}, nil
			}

			ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
				return []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}, {OrganizationName: "Foo Inc."}}, nil
			}
			ds.CountABMTokensFunc = func(ctx context.Context) (uint32, error) {
				return 1, nil
			}

			ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
				var res []*fleet.TeamSummary
				for _, tm := range savedTeams {
					res = append(res, &fleet.TeamSummary{Name: (*tm).Name, ID: (*tm).ID})
				}
				return res, nil
			}

			ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
				return nil
			}

			args := []string{"gitops"}
			for _, cfg := range tt.cfgs {
				if cfg != "" {
					tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString(cfg)
					require.NoError(t, err)
					args = append(args, "-f", tmpFile.Name())
				}
			}

			// Dry run
			out, err := runAppNoChecks(append(args, "--dry-run"))
			tt.dryRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)
			if t.Failed() {
				t.FailNow()
			}

			// Real run
			out, err = runAppNoChecks(args)
			tt.realRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)

			// Second real run, now that all the teams are saved
			out, err = runAppNoChecks(args)
			tt.realRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)
		})
	}
}

type memKeyValueStore struct {
	m sync.Map
}

func newMemKeyValueStore() *memKeyValueStore {
	return &memKeyValueStore{}
}

func (m *memKeyValueStore) Set(ctx context.Context, key string, value string, expireTime time.Duration) error {
	m.m.Store(key, value)
	return nil
}

func (m *memKeyValueStore) Get(ctx context.Context, key string) (*string, error) {
	v, ok := m.m.Load(key)
	if !ok {
		return nil, nil
	}
	vAsString := v.(string)
	return &vAsString, nil
}
