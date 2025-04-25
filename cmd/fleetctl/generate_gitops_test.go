// filepath: cmd/fleetctl/generate_gitops_test.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

type MockClient struct {
	IsFree bool
}

func (c *MockClient) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	b, err := os.ReadFile("./testdata/generateGitops/appConfig.json")
	if err != nil {
		return nil, err
	}
	var appConfig fleet.EnrichedAppConfig
	if err := json.Unmarshal(b, &appConfig); err != nil {
		return nil, err
	}
	if c.IsFree == true {
		appConfig.License.Tier = fleet.TierFree
	}
	return &appConfig, nil
}

func (MockClient) GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error) {
	spec := &fleet.EnrollSecretSpec{
		Secrets: []*fleet.EnrollSecret{
			{
				Secret: "some-secret-number-one",
			},
			{
				Secret: "some-secret-number-two",
			},
		},
	}
	return spec, nil
}

func (MockClient) ListTeams(query string) ([]fleet.Team, error) {
	teams := []fleet.Team{
		{
			ID:   1,
			Name: "Team A",
		},
	}
	return teams, nil
}

func (MockClient) ListScripts(query string) ([]*fleet.Script, error) {
	switch query {
	case "team_id=1":
		return []*fleet.Script{{
			ID:              2,
			TeamID:          ptr.Uint(1),
			Name:            "Script B.ps1",
			ScriptContentID: 2,
		}}, nil
	case "team_id=0":
		return []*fleet.Script{{
			ID:              3,
			TeamID:          ptr.Uint(0),
			Name:            "Script Z.ps1",
			ScriptContentID: 3,
		}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (MockClient) ListConfigurationProfiles(teamID *uint) ([]*fleet.MDMConfigProfilePayload, error) {
	if teamID == nil {
		return []*fleet.MDMConfigProfilePayload{
			{
				ProfileUUID: "global-macos-mobileconfig-profile-uuid",
				Name:        "Global MacOS MobileConfig Profile",
				Platform:    "darwin",
				Identifier:  "com.example.global-macos-mobileconfig-profile",
				LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{
					LabelName: "Label A",
				}, {
					LabelName: "Label B",
				}},
			},
			{
				ProfileUUID: "global-macos-json-profile-uuid",
				Name:        "Global MacOS JSON Profile",
				Platform:    "darwin",
				Identifier:  "com.example.global-macos-json-profile",
				LabelsExcludeAny: []fleet.ConfigurationProfileLabel{{
					LabelName: "Label C",
				}},
			},
			{
				ProfileUUID: "global-windows-profile-uuid",
				Name:        "Global Windows Profile",
				Platform:    "windows",
				Identifier:  "com.example.global-windows-profile",
				LabelsIncludeAny: []fleet.ConfigurationProfileLabel{{
					LabelName: "Label D",
				}},
			},
		}, nil
	}
	if *teamID == 1 {
		return []*fleet.MDMConfigProfilePayload{
			{
				ProfileUUID: "test-mobileconfig-profile-uuid",
				Name:        "Team MacOS MobileConfig Profile",
				Platform:    "darwin",
				Identifier:  "com.example.team-macos-mobileconfig-profile",
			},
		}, nil
	}
	if *teamID == 0 {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected team ID: %v", *teamID)
}

func (MockClient) GetScriptContents(scriptID uint) ([]byte, error) {
	if scriptID == 2 {
		return []byte("pop goes the weasel!"), nil
	}
	if scriptID == 3 {
		return []byte("#!/usr/bin/env pwsh\necho \"Hello from Script B!\""), nil
	}
	return nil, fmt.Errorf("script not found")
}

func (MockClient) GetProfileContents(profileID string) ([]byte, error) {
	switch profileID {
	case "global-macos-mobileconfig-profile-uuid":
		return []byte("<xml>global macos mobileconfig profile</xml>"), nil
	case "global-macos-json-profile-uuid":
		return []byte(`{"profile": "global macos json profile"}`), nil
	case "global-windows-profile-uuid":
		return []byte("<xml>global windows profile</xml>"), nil
	case "test-mobileconfig-profile-uuid":
		return []byte("<xml>test mobileconfig profile</xml>"), nil
	}
	return nil, fmt.Errorf("profile not found")
}

func (MockClient) GetTeam(teamID uint) (*fleet.Team, error) {
	if teamID == 1 {
		b, err := os.ReadFile("./testdata/generateGitops/teamConfig.json")
		if err != nil {
			return nil, err
		}
		var config fleet.TeamConfig
		if err := json.Unmarshal(b, &config); err != nil {
			return nil, err
		}
		return &fleet.Team{
			ID:     1,
			Name:   "Test Team",
			Config: config,
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "some-team-secret",
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("team not found")
}

func (MockClient) ListSoftwareTitles(query string) ([]fleet.SoftwareTitleListResult, error) {
	switch query {
	case "available_for_install=1":
		return []fleet.SoftwareTitleListResult{
			{
				ID:   1,
				Name: "Global Software",
				Versions: []fleet.SoftwareVersion{{
					ID:      1,
					Version: "1.0.0",
				}, {
					ID:      2,
					Version: "2.0.0",
				}},
				HashSHA256: ptr.String("global-software-hash"),
				SoftwarePackage: &fleet.SoftwarePackageOrApp{
					Name: "global-software.pkg",
				},
			},
		}, nil
	case "available_for_install=1&team_id=1":
		return []fleet.SoftwareTitleListResult{
			{
				ID:   2,
				Name: "Team Software",
				Versions: []fleet.SoftwareVersion{{
					ID:      3,
					Version: "5.6.7",
				}, {
					ID:      4,
					Version: "8.9.10",
				}},
				AppStoreApp: &fleet.SoftwarePackageOrApp{
					AppStoreID: "com.example.team-software",
				},
				HashSHA256: ptr.String("team-software-hash"),
			},
		}, nil
	case "available_for_install=1&team_id=0":
		return []fleet.SoftwareTitleListResult{}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (MockClient) GetPolicies(teamID *uint) ([]*fleet.Policy, error) {
	if teamID == nil {
		return []*fleet.Policy{
			{
				PolicyData: fleet.PolicyData{
					ID:          1,
					Name:        "Global Policy",
					Query:       "SELECT * FROM global_policy WHERE id = 1",
					Resolution:  ptr.String("Do a global thing"),
					Description: "This is a global policy",
					Platform:    "darwin",
					LabelsIncludeAny: []fleet.LabelIdent{{
						LabelName: "Label A",
					}, {
						LabelName: "Label B",
					}},
				},
				InstallSoftware: &fleet.PolicySoftwareTitle{
					SoftwareTitleID: 1,
				},
			},
		}, nil
	}
	return []*fleet.Policy{
		{
			PolicyData: fleet.PolicyData{
				ID:          1,
				Name:        "Team Policy",
				Query:       "SELECT * FROM team_policy WHERE id = 1",
				Resolution:  ptr.String("Do a team thing"),
				Description: "This is a team policy",
				Platform:    "linux,windows",
			},
			RunScript: &fleet.PolicyScript{
				ID: 1,
			},
		},
	}, nil
}

func (MockClient) GetQueries(teamID *uint, name *string) ([]fleet.Query, error) {
	if teamID == nil {
		return []fleet.Query{
			{
				ID:                 1,
				Name:               "Global Query",
				Query:              "SELECT * FROM global_query WHERE id = 1",
				Description:        "This is a global query",
				Platform:           "darwin",
				Interval:           3600,
				ObserverCanRun:     true,
				AutomationsEnabled: false,
				LabelsIncludeAny: []fleet.LabelIdent{{
					LabelName: "Label A",
				}, {
					LabelName: "Label B",
				}},
			},
		}, nil
	}
	return []fleet.Query{
		{
			ID:                 1,
			Name:               "Team Query",
			Query:              "SELECT * FROM team_query WHERE id = 1",
			Description:        "This is a team query",
			Platform:           "linux,windows",
			Interval:           1800,
			ObserverCanRun:     false,
			AutomationsEnabled: true,
		},
	}, nil
}

func (MockClient) GetSoftwareTitleByID(ID uint) (*fleet.SoftwareTitle, error) {
	switch ID {
	case 1:
		return &fleet.SoftwareTitle{
			ID: 1,
			SoftwarePackage: &fleet.SoftwareInstaller{
				LabelsIncludeAny: []fleet.SoftwareScopeLabel{{
					LabelName: "Label A",
				}, {
					LabelName: "Label B",
				}},
			},
		}, nil
	case 2:
		return &fleet.SoftwareTitle{
			ID: 2,
			AppStoreApp: &fleet.VPPAppStoreApp{
				LabelsExcludeAny: []fleet.SoftwareScopeLabel{{
					LabelName: "Label C",
				}, {
					LabelName: "Label D",
				}},
			},
		}, nil
	default:
		return nil, fmt.Errorf("software title not found")
	}
}

func (MockClient) GetLabels() ([]*fleet.LabelSpec, error) {
	return []*fleet.LabelSpec{{
		Name:                "Label A",
		Platform:            "linux,macos",
		Description:         "Label A description",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		Query:               "SELECT * FROM osquery_info",
	}, {
		Name:                "Label B",
		Description:         "Label B description",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		Hosts:               []string{"host1", "host2"},
	}}, nil
}

func (MockClient) Me() (*fleet.User, error) {
	return &fleet.User{
		ID:         1,
		Name:       "Test User",
		Email:      "test@example.com",
		GlobalRole: ptr.String("admin"),
	}, nil
}

func TestGenerateGitops(t *testing.T) {
	fleetClient := &MockClient{}
	action := createGenerateGitopsAction(fleetClient)
	buf := new(bytes.Buffer)
	tempDir := os.TempDir() + "/" + uuid.New().String()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("dir", tempDir, "")

	cliContext := cli.NewContext(&cli.App{
		Name:      "test",
		Usage:     "test",
		Writer:    buf,
		ErrWriter: buf,
	}, flagSet, nil)
	err := action(cliContext)
	require.NoError(t, err, buf.String())

	fmt.Println(buf.String()) // Debugging line

	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	})
}

func TestGenerateGitopsFree(t *testing.T) {
	fleetClient := &MockClient{}
	fleetClient.IsFree = true
	action := createGenerateGitopsAction(fleetClient)
	buf := new(bytes.Buffer)
	tempDir := os.TempDir() + "/" + uuid.New().String()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("dir", tempDir, "")

	cliContext := cli.NewContext(&cli.App{
		Name:      "test",
		Usage:     "test",
		Writer:    buf,
		ErrWriter: buf,
	}, flagSet, nil)
	err := action(cliContext)
	require.NoError(t, err, buf.String())

	fmt.Println(buf.String()) // Debugging line

	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	})
}

func TestGenerateOrgSettings(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	orgSettingsRaw, err := cmd.generateOrgSettings()
	require.NoError(t, err)
	require.NotNil(t, orgSettingsRaw)
	var orgSettings map[string]interface{}
	b, err := yaml.Marshal(orgSettingsRaw)
	fmt.Println("Org settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &orgSettings)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedOrgSettings.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, orgSettings)
}

func TestGenerateOrgSettingsInsecure(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()
	require.NoError(t, err)

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.Bool("insecure", true, "Output sensitive information in plaintext.")
	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, flagSet, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	orgSettingsRaw, err := cmd.generateOrgSettings()
	require.NoError(t, err)
	require.NotNil(t, orgSettingsRaw)
	var orgSettings map[string]interface{}
	b, err := yaml.Marshal(orgSettingsRaw)
	fmt.Println("Org settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &orgSettings)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedOrgSettings-insecure.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, orgSettings)
}

func TestGenerateTeamSettings(t *testing.T) {
	// Get the test team.
	fleetClient := &MockClient{}
	team, err := fleetClient.GetTeam(1)
	require.NoError(t, err)

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    nil,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	TeamSettingsRaw, err := cmd.generateTeamSettings("team.yml", team)
	require.NoError(t, err)
	require.NotNil(t, TeamSettingsRaw)
	var TeamSettings map[string]interface{}
	b, err := yaml.Marshal(TeamSettingsRaw)
	fmt.Println("Team settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &TeamSettings)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamSettings.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, TeamSettings)
}

func TestGenerateTeamSettingsInsecure(t *testing.T) {
	// Get the test team.
	fleetClient := &MockClient{}
	team, err := fleetClient.GetTeam(1)
	require.NoError(t, err)

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.Bool("insecure", true, "Output sensitive information in plaintext.")
	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(&cli.App{}, flagSet, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    nil,
	}

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	TeamSettingsRaw, err := cmd.generateTeamSettings("team.yml", team)
	require.NoError(t, err)
	require.NotNil(t, TeamSettingsRaw)
	var TeamSettings map[string]interface{}
	b, err := yaml.Marshal(TeamSettingsRaw)
	fmt.Println("Team settings raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &TeamSettings)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamSettings-insecure.yaml")
	require.NoError(t, err)
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, TeamSettings)
}

func TestGenerateControls(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		ScriptList:   make(map[uint]string),
	}

	// Generate global controls.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	mdmConfig := fleet.TeamMDM{
		EnableDiskEncryption: appConfig.MDM.EnableDiskEncryption.Value,
		MacOSUpdates:         appConfig.MDM.MacOSUpdates,
		IOSUpdates:           appConfig.MDM.IOSUpdates,
		IPadOSUpdates:        appConfig.MDM.IPadOSUpdates,
		WindowsUpdates:       appConfig.MDM.WindowsUpdates,
		MacOSSetup:           appConfig.MDM.MacOSSetup,
	}
	controlsRaw, err := cmd.generateControls(nil, "", &mdmConfig)
	require.NoError(t, err)
	require.NotNil(t, controlsRaw)
	var controls map[string]interface{}
	b, err := yaml.Marshal(controlsRaw)
	fmt.Println("Controls raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &controls)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalControls.yaml")
	require.NoError(t, err)
	var expectedControls map[string]interface{}
	err = yaml.Unmarshal(b, &expectedControls)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedControls, controls)

	// Generate controls for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	controlsRaw, err = cmd.generateControls(ptr.Uint(1), "some_team", nil)
	require.NoError(t, err)
	require.NotNil(t, controlsRaw)
	b, err = yaml.Marshal(controlsRaw)
	fmt.Println("Controls raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &controls)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamControls.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedControls)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedControls, controls)
}

func TestGenerateSoftware(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		SoftwareList: make(map[uint]Software),
	}

	softwareRaw, err := cmd.generateSoftware("default.yml", 0)
	require.NoError(t, err)
	require.NotNil(t, softwareRaw)
	var software map[string]interface{}
	b, err := yaml.Marshal(softwareRaw)
	fmt.Println("software raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &software)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalSoftware.yaml")
	require.NoError(t, err)
	var expectedSoftware map[string]interface{}
	err = yaml.Unmarshal(b, &expectedSoftware)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedSoftware, software)

	// Generate software for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	softwareRaw, err = cmd.generateSoftware("team.yml", 1)
	require.NoError(t, err)
	require.NotNil(t, softwareRaw)
	b, err = yaml.Marshal(softwareRaw)
	fmt.Println("software raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &software)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamSoftware.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedSoftware)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedSoftware, software)
}

func TestGeneratePolicies(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
		SoftwareList: map[uint]Software{
			1: {
				Hash:    "global-software-hash",
				Comment: "__GLOBAL_SOFTWARE_COMMENT_TOKEN__",
			},
		},
		ScriptList: map[uint]string{
			1: "/path/to/script1.sh",
		},
	}

	policiesRaw, err := cmd.generatePolicies("default.yml", nil)
	require.NoError(t, err)
	require.NotNil(t, policiesRaw)
	var policies []map[string]interface{}
	b, err := yaml.Marshal(policiesRaw)
	fmt.Println("policies raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &policies)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalPolicies.yaml")
	require.NoError(t, err)
	var expectedPolicies []map[string]interface{}
	err = yaml.Unmarshal(b, &expectedPolicies)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedPolicies, policies)

	// Generate policies for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	policiesRaw, err = cmd.generatePolicies("team.yml", ptr.Uint(1))
	require.NoError(t, err)
	require.NotNil(t, policiesRaw)
	b, err = yaml.Marshal(policiesRaw)
	fmt.Println("policies raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &policies)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamPolicies.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedPolicies)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedPolicies, policies)
}

func TestGenerateQueries(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	queriesRaw, err := cmd.generateQueries("default.yml", nil)
	require.NoError(t, err)
	require.NotNil(t, queriesRaw)
	var queries []map[string]interface{}
	b, err := yaml.Marshal(queriesRaw)
	fmt.Println("queries raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &queries)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedGlobalQueries.yaml")
	require.NoError(t, err)
	var expectedQueries []map[string]interface{}
	err = yaml.Unmarshal(b, &expectedQueries)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedQueries, queries)

	// Generate queries for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	queriesRaw, err = cmd.generateQueries("team.yml", ptr.Uint(1))
	require.NoError(t, err)
	require.NotNil(t, queriesRaw)
	b, err = yaml.Marshal(queriesRaw)
	fmt.Println("queries raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &queries)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamQueries.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedQueries)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedQueries, queries)
}

func TestGenerateLabels(t *testing.T) {
	// Get the test app config.
	fleetClient := &MockClient{}
	appConfig, err := fleetClient.GetAppConfig()

	// Create the command.
	cmd := &GenerateGitopsCommand{
		Client:       fleetClient,
		CLI:          cli.NewContext(cli.NewApp(), nil, nil),
		Messages:     Messages{},
		FilesToWrite: make(map[string]interface{}),
		AppConfig:    appConfig,
	}

	labelsRaw, err := cmd.generateLabels()
	require.NoError(t, err)
	require.NotNil(t, labelsRaw)
	var labels []map[string]interface{}
	b, err := yaml.Marshal(labelsRaw)
	fmt.Println("labels raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &labels)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedLabels.yaml")
	require.NoError(t, err)
	var expectedlabels []map[string]interface{}
	err = yaml.Unmarshal(b, &expectedlabels)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedlabels, labels)
}
