package fleetctl

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func RunAppForTest(t *testing.T, args []string) string {
	w, err := RunAppNoChecks(args)
	require.NoError(t, err)
	return w.String()
}

func RunAppCheckErr(t *testing.T, args []string, errorMsg string) string {
	w, err := RunAppNoChecks(args)
	require.Error(t, err)
	require.Contains(t, err.Error(), errorMsg)
	return w.String()
}

func RunWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := CreateApp(nil, w, errWriter, noopExitErrHandler)
	StashRawArgs(app, args)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}

// Alias for RunApp; added rather than changing all existing calls to `RunApp`,
// to avoid confusion and in case the behavior of `RunApp` needs to diverge in the future.
func RunAppNoChecks(args []string) (*bytes.Buffer, error) {
	return RunApp(args)
}

type gitopsTestNotFoundError struct{}

func (e *gitopsTestNotFoundError) Error() string    { return "not found" }
func (e *gitopsTestNotFoundError) IsNotFound() bool { return true }

// setupEmptyGitOpsMocks sets all datastore mock functions needed by the gitops command to
// no-op defaults. Tests should override individual mocks after calling this to add tracking
// or return specific values.
func setupEmptyGitOpsMocks(ds *mock.Store) {
	// App config
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error { return nil }

	// Labels
	ds.GetLabelSpecsFunc = func(ctx context.Context, filter fleet.TeamFilter) ([]*fleet.LabelSpec, error) {
		return nil, nil
	}
	ds.ApplyLabelSpecsWithAuthorFunc = func(ctx context.Context, specs []*fleet.LabelSpec, authorID *uint) error {
		return nil
	}
	ds.SetAsideLabelsFunc = func(ctx context.Context, teamID *uint, names []string, user fleet.User) error {
		return nil
	}
	ds.LabelByNameFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) (*fleet.Label, error) {
		return &fleet.Label{ID: 1, Name: name}, nil
	}
	ds.DeleteLabelFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) error { return nil }
	ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
		return map[string]*fleet.Label{}, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]uint, error) {
		return map[string]uint{}, nil
	}

	// Secrets
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, isNew bool, teamID *uint) (bool, error) {
		return true, nil
	}

	// MDM profiles
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration, androidProfiles []*fleet.MDMAndroidConfigProfile, vars []fleet.MDMProfileIdentifierFleetVariables,
	) (fleet.MDMProfilesUpdates, error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (fleet.MDMProfilesUpdates, error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration, usesFleetVars []fleet.FleetVarName) (*fleet.MDMAppleDeclaration, error) {
		return &fleet.MDMAppleDeclaration{}, nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.SetOrUpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile) error {
		return nil
	}
	ds.DeleteMDMWindowsConfigProfileByTeamAndNameFunc = func(ctx context.Context, teamID *uint, profileName string) error {
		return nil
	}
	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMAppleConfigProfile, error) {
		return &fleet.MDMAppleConfigProfile{}, nil
	}
	ds.GetMDMAppleBootstrapPackageMetaFunc = func(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
		return &fleet.MDMAppleBootstrapPackage{}, nil
	}
	ds.DeleteMDMAppleBootstrapPackageFunc = func(ctx context.Context, teamID uint) error { return nil }
	ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
		return nil, &gitopsTestNotFoundError{}
	}
	ds.DeleteMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) error { return nil }
	ds.InsertOrReplaceMDMConfigAssetFunc = func(ctx context.Context, asset fleet.MDMConfigAsset) error {
		return nil
	}
	ds.HardDeleteMDMConfigAssetFunc = func(ctx context.Context, assetName fleet.MDMAssetName) error {
		return nil
	}
	ds.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) error {
		return nil
	}
	ds.MDMGetEULAMetadataFunc = func(ctx context.Context) (*fleet.MDMEULA, error) {
		return nil, &gitopsTestNotFoundError{}
	}
	ds.MDMInsertEULAFunc = func(ctx context.Context, eula *fleet.MDMEULA) error { return nil }
	ds.MDMDeleteEULAFunc = func(ctx context.Context, token string) error { return nil }
	ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
		return document, nil, nil
	}

	// Scripts
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		return []fleet.ScriptResponse{}, nil
	}

	// Policies and queries
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions, platform string) ([]*fleet.Policy, error) {
		return nil, nil
	}
	ds.ListTeamPoliciesFunc = func(
		ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions, automationFilter string, platform string,
	) ([]*fleet.Policy, []*fleet.Policy, error) {
		return nil, nil, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, int, int, *fleet.PaginationMetadata, error) {
		return nil, 0, 0, nil, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &gitopsTestNotFoundError{}
	}

	// Teams
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, &gitopsTestNotFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		return nil, nil
	}
	ds.TeamConflictsWithNameFunc = func(ctx context.Context, name string, excludeID uint) (*fleet.Team, error) {
		return nil, nil
	}
	ds.TeamLiteFunc = func(ctx context.Context, id uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{}, nil
	}
	ds.NewTeamFunc = func(ctx context.Context, newTeam *fleet.Team) (*fleet.Team, error) {
		newTeam.ID = 1
		return newTeam, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	ds.DefaultTeamConfigFunc = func(ctx context.Context) (*fleet.TeamConfig, error) {
		return &fleet.TeamConfig{}, nil
	}
	ds.SaveDefaultTeamConfigFunc = func(ctx context.Context, config *fleet.TeamConfig) error { return nil }
	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}

	// Software
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.BatchSetInHouseAppsInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmFilter fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.ListSoftwareAutoUpdateSchedulesFunc = func(ctx context.Context, teamID uint, source string, optionalFilter ...fleet.SoftwareAutoUpdateScheduleFilter) ([]fleet.SoftwareAutoUpdateSchedule, error) {
		return []fleet.SoftwareAutoUpdateSchedule{}, nil
	}
	ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
		return nil, nil
	}
	ds.GetTeamsWithInstallerByHashFunc = func(ctx context.Context, sha256, url string) (map[uint][]*fleet.ExistingSoftwareInstaller, error) {
		return map[uint][]*fleet.ExistingSoftwareInstaller{}, nil
	}
	ds.GetInstallerByTeamAndURLFunc = func(ctx context.Context, teamID *uint, url string) (*fleet.ExistingSoftwareInstaller, error) {
		return nil, nil
	}
	ds.DeleteIconsAssociatedWithTitlesWithoutInstallersFunc = func(ctx context.Context, teamID uint) error {
		return nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error { return nil }

	// VPP / App Store
	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam, _ map[string]uint) (bool, error) {
		return false, nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error { return nil }
	ds.GetVPPAppsFunc = func(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
		return nil, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}
	// Default to "no existing vpp_apps row" so resolveAddAnchor takes the
	// first-add path. Tests that exercise re-anchor behavior override these.
	ds.GetVPPAppByAdamIDPlatformFunc = func(ctx context.Context, adamID string, platform fleet.InstallableDevicePlatform) (*fleet.VPPApp, error) {
		return nil, vppAppNotFound{}
	}
	ds.GetVPPTokenOwningAppInCountryFunc = func(ctx context.Context, adamID string, platform fleet.InstallableDevicePlatform, country string) (*fleet.VPPTokenDB, error) {
		return nil, vppAppNotFound{}
	}

	// ABM
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}
	ds.GetABMTokenCountFunc = func(ctx context.Context) (int, error) { return 0, nil }
	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error { return nil }

	// Certificate authorities
	ds.BatchApplyCertificateAuthoritiesFunc = func(ctx context.Context, ops fleet.CertificateAuthoritiesBatchOperations) error {
		return nil
	}
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}
	ds.ListCertificateAuthoritiesFunc = func(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
		return nil, nil
	}
	ds.GetCertificateTemplatesByTeamIDFunc = func(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error) {
		return []*fleet.CertificateTemplateResponseSummary{}, &fleet.PaginationMetadata{}, nil
	}
	ds.BatchDeleteCertificateTemplatesFunc = func(ctx context.Context, ids []uint) (bool, error) {
		return false, nil
	}
	ds.CreatePendingCertificateTemplatesForExistingHostsFunc = func(ctx context.Context, certificateTemplateID uint, teamID uint) (int64, error) {
		return 0, nil
	}
	ds.SetHostCertificateTemplatesToPendingRemoveFunc = func(ctx context.Context, certTmplID uint) error {
		return nil
	}

	// Conditional access
	ds.ConditionalAccessMicrosoftGetFunc = func(ctx context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
		return &fleet.ConditionalAccessMicrosoftIntegration{}, nil
	}

	// Setup experience
	ds.TeamIDsWithSetupExperienceIdPEnabledFunc = func(ctx context.Context) ([]uint, error) {
		return nil, nil
	}

	// Jobs
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return &fleet.Job{}, nil
	}
}

type vppAppNotFound struct{}

func (vppAppNotFound) Error() string    { return "vpp app not found" }
func (vppAppNotFound) IsNotFound() bool { return true }
