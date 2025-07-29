package gitops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	teamName = "Team Test"
)

func TestGitOpsTeamSofwareInstallers(t *testing.T) {
	testing_utils.StartSoftwareInstallerServer(t)
	testing_utils.StartAndServeVPPServer(t)

	cases := []struct {
		file    string
		wantErr string
	}{
		{"testdata/gitops/team_software_installer_not_found.yml", "Please make sure that URLs are reachable from your Fleet server."},
		{"testdata/gitops/team_software_installer_unsupported.yml", "The file should be .pkg, .msi, .exe, .deb, .rpm, or .tar.gz."},
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
		{"testdata/gitops/team_software_installer_no_url.yml", "at least one of hash_sha256 or url is required for each software package"},
		{"testdata/gitops/team_software_installer_invalid_self_service_value.yml",
			"Couldn't edit \"../../fleetctl/testdata/gitops/team_software_installer_invalid_self_service_value.yml\" at \"software.packages.self_service\", expected type bool but got string"},
		{"testdata/gitops/team_software_installer_invalid_both_include_exclude.yml",
			`only one of "labels_exclude_any" or "labels_include_any" can be specified`},
		{"testdata/gitops/team_software_installer_valid_include.yml", ""},
		{"testdata/gitops/team_software_installer_valid_exclude.yml", ""},
		{"testdata/gitops/team_software_installer_invalid_unknown_label.yml",
			"Please create the missing labels, or update your settings to not refer to these labels."},
		// team tests for setup experience software/script
		{"testdata/gitops/team_setup_software_valid.yml", ""},
		{"testdata/gitops/team_setup_software_invalid_script.yml", "no_such_script.sh: no such file"},
		{"testdata/gitops/team_setup_software_invalid_software_package.yml", "no_such_software.yml\" does not exist for that team"},
		{"testdata/gitops/team_setup_software_invalid_vpp_app.yml", "\"no_such_app\" does not exist for that team"},
	}
	for _, c := range cases {
		c.file = filepath.Join("../../fleetctl", c.file)
		t.Run(filepath.Base(c.file), func(t *testing.T) {
			ds, _, _ := testing_utils.SetupFullGitOpsPremiumServer(t)
			tokExpire := time.Now().Add(time.Hour)
			token, err := test.CreateVPPTokenEncoded(tokExpire, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.GetVPPAppsFunc = func(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
				return []fleet.VPPAppResponse{}, nil
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

			ds.GetLabelSpecsFunc = func(ctx context.Context) ([]*fleet.LabelSpec, error) {
				return []*fleet.LabelSpec{
					{
						Name:                "a",
						Description:         "A global label",
						LabelMembershipType: fleet.LabelMembershipTypeManual,
						Hosts:               []string{"host2", "host3"},
					},
					{
						Name:                "b",
						Description:         "Another label",
						LabelMembershipType: fleet.LabelMembershipTypeDynamic,
						Query:               "SELECT 1 from osquery_info",
					},
				}, nil
			}

			labelToIDs := map[string]uint{
				fleet.BuiltinLabelMacOS14Plus: 1,
				"a":                           2,
				"b":                           3,
			}
			ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
				// for this test, recognize labels a and b (as well as the built-in macos 14+ one)
				ret := make(map[string]uint)
				for _, lbl := range labels {
					id, ok := labelToIDs[lbl]
					if ok {
						ret[lbl] = id
					}
				}
				return ret, nil
			}
			ds.GetTeamsWithInstallerByHashFunc = func(ctx context.Context, sha256, url string) (map[uint]*fleet.ExistingSoftwareInstaller, error) {
				return map[uint]*fleet.ExistingSoftwareInstaller{}, nil
			}
			ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
				return []uint{}, nil
			}

			_, err = fleetctl.RunAppNoChecks([]string{"gitops", "-f", c.file})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestGitOpsTeamSoftwareInstallersQueryEnv(t *testing.T) {
	testing_utils.StartSoftwareInstallerServer(t)
	ds, _, _ := testing_utils.SetupFullGitOpsPremiumServer(t)

	t.Setenv("QUERY_VAR", "IT_WORKS")

	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, tmID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		if len(installers) != 0 && installers[0].PreInstallQuery != "select IT_WORKS" {
			return fmt.Errorf("Missing env var, got %s", installers[0].PreInstallQuery)
		}
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}
	ds.GetTeamsWithInstallerByHashFunc = func(ctx context.Context, sha256, url string) (map[uint]*fleet.ExistingSoftwareInstaller, error) {
		return map[uint]*fleet.ExistingSoftwareInstaller{}, nil
	}
	ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
		return []uint{}, nil
	}

	_, err := fleetctl.RunAppNoChecks([]string{"gitops", "-f", "../../fleetctl/testdata/gitops/team_software_installer_valid_env_query.yml"})
	require.NoError(t, err)
}

func TestGitOpsNoTeamVPPPolicies(t *testing.T) {
	testing_utils.StartAndServeVPPServer(t)

	cases := []struct {
		noTeamFile string
		wantErr    string
		vppApps    []fleet.VPPAppResponse
	}{
		{
			noTeamFile: "testdata/gitops/subdir/no_team_vpp_policies_valid.yml",
			vppApps: []fleet.VPPAppResponse{
				{ // for more test coverage
					Platform: fleet.MacOSPlatform,
				},
				{ // for more test coverage
					TitleID:  ptr.Uint(122),
					Platform: fleet.MacOSPlatform,
				},
				{
					TeamID:     ptr.Uint(0),
					TitleID:    ptr.Uint(123),
					AppStoreID: "1",
					Platform:   fleet.IOSPlatform,
				},
				{
					TeamID:     ptr.Uint(0),
					TitleID:    ptr.Uint(124),
					AppStoreID: "1",
					Platform:   fleet.MacOSPlatform,
				},
				{
					TeamID:     ptr.Uint(0),
					TitleID:    ptr.Uint(125),
					AppStoreID: "1",
					Platform:   fleet.IPadOSPlatform,
				},
			},
		},
	}
	for _, c := range cases {
		c.noTeamFile = filepath.Join("../../fleetctl", c.noTeamFile)
		t.Run(filepath.Base(c.noTeamFile), func(t *testing.T) {
			ds, _, _ := testing_utils.SetupFullGitOpsPremiumServer(t)
			tokExpire := time.Now().Add(time.Hour)
			token, err := test.CreateVPPTokenEncoded(tokExpire, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}
			ds.GetVPPAppsFunc = func(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
				return c.vppApps, nil
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
			labelToIDs := map[string]uint{
				fleet.BuiltinLabelMacOS14Plus: 1,
				"a":                           2,
				"b":                           3,
			}
			ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
				// for this test, recognize labels a and b (as well as the built-in macos 14+ one)
				ret := make(map[string]uint)
				for _, lbl := range labels {
					id, ok := labelToIDs[lbl]
					if ok {
						ret[lbl] = id
					}
				}
				return ret, nil
			}
			ds.LabelsByNameFunc = func(ctx context.Context, names []string) (map[string]*fleet.Label, error) {
				return map[string]*fleet.Label{
					"a": {
						ID:   1,
						Name: "a",
					},
					"b": {
						ID:   2,
						Name: "b",
					},
				}, nil
			}
			ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
				return []uint{}, nil
			}

			t.Setenv("APPLE_BM_DEFAULT_TEAM", "")
			globalFile := "../../fleetctl/testdata/gitops/global_config_no_paths.yml"
			dstPath := filepath.Join(filepath.Dir(c.noTeamFile), "no-team.yml")
			t.Cleanup(func() {
				os.Remove(dstPath)
			})
			err = file.Copy(c.noTeamFile, dstPath, 0o755)
			require.NoError(t, err)
			_, err = fleetctl.RunAppNoChecks([]string{"gitops", "-f", globalFile, "-f", dstPath})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestGitOpsNoTeamSoftwareInstallers(t *testing.T) {
	testing_utils.StartSoftwareInstallerServer(t)
	testing_utils.StartAndServeVPPServer(t)

	cases := []struct {
		noTeamFile string
		wantErr    string
	}{
		{"testdata/gitops/no_team_software_installer_not_found.yml", "Please make sure that URLs are reachable from your Fleet server."},
		{"testdata/gitops/no_team_software_installer_unsupported.yml", "The file should be .pkg, .msi, .exe, .deb, .rpm, or .tar.gz."},
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
		{"testdata/gitops/no_team_software_installer_no_url.yml", "at least one of hash_sha256 or url is required for each software package"},
		{"testdata/gitops/no_team_software_installer_invalid_self_service_value.yml",
			"Couldn't edit \"../../fleetctl/testdata/gitops/no-team.yml\" at \"software.packages.self_service\", expected type bool but got string"},
		{"testdata/gitops/no_team_software_installer_invalid_both_include_exclude.yml",
			`only one of "labels_exclude_any" or "labels_include_any" can be specified`},
		{"testdata/gitops/no_team_software_installer_valid_include.yml", ""},
		{"testdata/gitops/no_team_software_installer_valid_exclude.yml", ""},
		{"testdata/gitops/no_team_software_installer_invalid_unknown_label.yml",
			"Please create the missing labels, or update your settings to not refer to these labels."},
		// No team tests for setup experience software/script
		{"testdata/gitops/no_team_setup_software_valid.yml", ""},
		{"testdata/gitops/no_team_setup_software_invalid_script.yml", "no_such_script.sh: no such file"},
		{"testdata/gitops/no_team_setup_software_invalid_software_package.yml", "no_such_software.yml\" does not exist for that team"},
		{"testdata/gitops/no_team_setup_software_invalid_vpp_app.yml", "\"no_such_app\" does not exist for that team"},
	}
	for _, c := range cases {
		c.noTeamFile = filepath.Join("../../fleetctl", c.noTeamFile)
		t.Run(filepath.Base(c.noTeamFile), func(t *testing.T) {
			ds, _, _ := testing_utils.SetupFullGitOpsPremiumServer(t)
			tokExpire := time.Now().Add(time.Hour)
			token, err := test.CreateVPPTokenEncoded(tokExpire, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}
			ds.GetVPPAppsFunc = func(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
				return []fleet.VPPAppResponse{}, nil
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
			ds.GetLabelSpecsFunc = func(ctx context.Context) ([]*fleet.LabelSpec, error) {
				return []*fleet.LabelSpec{
					{
						Name:                "a",
						Description:         "A global label",
						LabelMembershipType: fleet.LabelMembershipTypeManual,
						Hosts:               []string{"host2", "host3"},
					},
					{
						Name:                "b",
						Description:         "Another label",
						LabelMembershipType: fleet.LabelMembershipTypeDynamic,
						Query:               "SELECT 1 from osquery_info",
					},
				}, nil
			}
			labelToIDs := map[string]uint{
				fleet.BuiltinLabelMacOS14Plus: 1,
				"a":                           2,
				"b":                           3,
			}
			ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
				// for this test, recognize labels a and b (as well as the built-in macos 14+ one)
				ret := make(map[string]uint)
				for _, lbl := range labels {
					id, ok := labelToIDs[lbl]
					if ok {
						ret[lbl] = id
					}
				}
				return ret, nil
			}
			ds.GetTeamsWithInstallerByHashFunc = func(ctx context.Context, sha256, url string) (map[uint]*fleet.ExistingSoftwareInstaller, error) {
				return map[uint]*fleet.ExistingSoftwareInstaller{}, nil
			}
			ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
				return []uint{}, nil
			}

			t.Setenv("APPLE_BM_DEFAULT_TEAM", "")
			globalFile := "../../fleetctl/testdata/gitops/global_config_no_paths.yml"
			if strings.HasPrefix(filepath.Base(c.noTeamFile), "no_team_setup_software") {
				// the controls section is in the no-team test file, so use a global file without that section
				globalFile = "../../fleetctl/testdata/gitops/global_config_no_paths_no_controls.yml"
			}
			dstPath := filepath.Join(filepath.Dir(c.noTeamFile), "no-team.yml")
			t.Cleanup(func() {
				os.Remove(dstPath)
			})
			err = file.Copy(c.noTeamFile, dstPath, 0o755)
			require.NoError(t, err)
			_, err = fleetctl.RunAppNoChecks([]string{"gitops", "-f", globalFile, "-f", dstPath})
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

func TestGitOpsTeamVPPApps(t *testing.T) {
	testing_utils.StartAndServeVPPServer(t)

	cases := []struct {
		file            string
		wantErr         string
		tokenExpiration time.Time
		expectedLabels  map[string]uint
	}{
		{"testdata/gitops/team_vpp_valid_app.yml", "", time.Now().Add(24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_valid_app_self_service.yml", "", time.Now().Add(24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_valid_empty.yml", "", time.Now().Add(24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_valid_empty.yml", "", time.Now().Add(-24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_valid_app.yml", "VPP token expired", time.Now().Add(-24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_invalid_app.yml", "app not available on vpp account", time.Now().Add(24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_incorrect_type.yml", "Couldn't edit \"../../fleetctl/testdata/gitops/team_vpp_incorrect_type.yml\" at \"software.app_store_apps.app_store_id\", expected type string but got number",
			time.Now().Add(24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_empty_adamid.yml", "software app store id required", time.Now().Add(24 * time.Hour), map[string]uint{}},
		{"testdata/gitops/team_vpp_valid_app_labels_exclude_any.yml", "", time.Now().Add(24 * time.Hour),
			map[string]uint{"label 1": 1, "label 2": 2}},
		{"testdata/gitops/team_vpp_valid_app_labels_include_any.yml", "", time.Now().Add(24 * time.Hour),
			map[string]uint{"label 1": 1, "label 2": 2}},
		{"testdata/gitops/team_vpp_invalid_app_labels_exclude_any.yml",
			"Please create the missing labels, or update your settings to not refer to these labels.", time.Now().Add(24 * time.Hour),
			map[string]uint{"label 1": 1, "label 2": 2}},
		{"testdata/gitops/team_vpp_invalid_app_labels_include_any.yml",
			"Please create the missing labels, or update your settings to not refer to these labels.", time.Now().Add(24 * time.Hour),
			map[string]uint{"label 1": 1, "label 2": 2}},
		{"testdata/gitops/team_vpp_invalid_app_labels_both.yml",
			`only one of "labels_exclude_any" or "labels_include_any" can be specified for app store app`, time.Now().Add(24 * time.Hour),
			map[string]uint{}},
	}

	for _, c := range cases {
		c.file = filepath.Join("../../fleetctl", c.file)
		t.Run(filepath.Base(c.file), func(t *testing.T) {
			ds, _, _ := testing_utils.SetupFullGitOpsPremiumServer(t)
			token, err := test.CreateVPPTokenEncoded(c.tokenExpiration, "fleet", "ca")
			require.NoError(t, err)

			ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
				return nil
			}
			ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
				return nil
			}
			ds.GetVPPAppsFunc = func(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
				return []fleet.VPPAppResponse{}, nil
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

			ds.GetLabelSpecsFunc = func(ctx context.Context) ([]*fleet.LabelSpec, error) {
				return []*fleet.LabelSpec{
					{
						Name:                "label 1",
						Description:         "A global label",
						LabelMembershipType: fleet.LabelMembershipTypeManual,
						Hosts:               []string{"host2", "host3"},
					},
					{
						Name:                "label 2",
						Description:         "Another label",
						LabelMembershipType: fleet.LabelMembershipTypeDynamic,
						Query:               "SELECT 1 from osquery_info",
					},
				}, nil
			}
			ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
				return []uint{}, nil
			}

			found := make(map[string]uint)
			ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
				for _, l := range labels {
					if id, ok := c.expectedLabels[l]; ok {
						found[l] = id
					}
				}
				return found, nil
			}

			_, err = fleetctl.RunAppNoChecks([]string{"gitops", "-f", c.file})

			if c.wantErr == "" {
				require.NoError(t, err)
				if len(c.expectedLabels) > 0 {
					require.True(t, ds.LabelIDsByNameFuncInvoked)
				}

				require.Equal(t, c.expectedLabels, found)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}
		})
	}
}

// TestGitOpsTeamVPPAndApp tests the flow where a new team is created with VPP apps.
// GitOps must first create the team, then assign VPP token to it, and only then add VPP apps.
func TestGitOpsTeamVPPAndApp(t *testing.T) {
	testing_utils.StartAndServeVPPServer(t)
	ds, _, _ := testing_utils.SetupFullGitOpsPremiumServer(t)
	renewDate := time.Now().Add(24 * time.Hour)
	token, err := test.CreateVPPTokenEncoded(renewDate, "fleet", "ca")
	require.NoError(t, err)

	ds.GetVPPAppsFunc = func(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
		return []fleet.VPPAppResponse{}, nil
	}
	ds.GetABMTokenCountFunc = func(ctx context.Context) (int, error) {
		return 0, nil
	}

	// The following mocks are key to this test.
	vppToken := &fleet.VPPTokenDB{
		ID:        1,
		OrgName:   "Fleet",
		Location:  "Earth",
		RenewDate: renewDate,
		Token:     string(token),
		Teams:     nil,
	}
	tokensByTeams := make(map[uint]*fleet.VPPTokenDB)
	ds.UpdateVPPTokenTeamsFunc = func(ctx context.Context, id uint, teams []uint) (*fleet.VPPTokenDB, error) {
		for _, teamID := range teams {
			tokensByTeams[teamID] = vppToken
		}
		return vppToken, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{vppToken}, nil
	}
	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
		if teamID == nil {
			return vppToken, nil
		}
		token, ok := tokensByTeams[*teamID]
		if !ok {
			return nil, sql.ErrNoRows
		}
		return token, nil
	}
	ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
		return []uint{}, nil
	}

	buf, err := fleetctl.RunAppNoChecks([]string{"gitops", "-f", "../../fleetctl/testdata/gitops/global_config_vpp.yml", "-f",
		"../../fleetctl/testdata/gitops/team_vpp_valid_app.yml"})
	require.NoError(t, err)
	assert.True(t, ds.UpdateVPPTokenTeamsFuncInvoked)
	assert.True(t, ds.GetVPPTokenByTeamIDFuncInvoked)
	assert.True(t, ds.SetTeamVPPAppsFuncInvoked)
	assert.Contains(t, buf.String(), fmt.Sprintf(fleetctl.ReapplyingTeamForVPPAppsMsg, teamName))
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
			ds, savedAppConfigPtr, savedTeams := testing_utils.SetupFullGitOpsPremiumServer(t)

			ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
				return []*fleet.VPPTokenDB{{Location: "Fleet Device Management Inc."}, {Location: "Acme Inc."}}, nil
			}

			ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
				return []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}, {OrganizationName: "Foo Inc."}}, nil
			}
			ds.GetABMTokenCountFunc = func(ctx context.Context) (int, error) {
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
			out, err := fleetctl.RunAppNoChecks(append(args, "--dry-run"))
			tt.dryRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)
			if t.Failed() {
				t.FailNow()
			}

			// Real run
			out, err = fleetctl.RunAppNoChecks(args)
			tt.realRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)

			// Second real run, now that all the teams are saved
			out, err = fleetctl.RunAppNoChecks(args)
			tt.realRunAssertion(t, *savedAppConfigPtr, ds, out.String(), err)
		})
	}
}
