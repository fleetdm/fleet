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
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

type MockClient struct{}

func (MockClient) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	cwd, _ := os.Getwd()
	println("Current working directory:", cwd) // Debugging line

	b, err := os.ReadFile("./testdata/generateGitops/appConfig.json")
	if err != nil {
		return nil, err
	}
	var appConfig fleet.EnrichedAppConfig
	if err := json.Unmarshal(b, &appConfig); err != nil {
		return nil, err
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
	case "":
		return []*fleet.Script{{
			ID:              1,
			TeamID:          nil,
			Name:            "Script A.sh",
			ScriptContentID: 1,
		}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (MockClient) ListConfigurationProfiles(teamID *uint) ([]*fleet.MDMConfigProfilePayload, error) {
	switch *teamID {
	case 0:
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
	case 1:
		return []*fleet.MDMConfigProfilePayload{
			{
				ProfileUUID: "test-mobileconfig-profile-uuid",
				Name:        "Team MacOS MobileConfig Profile",
				Platform:    "darwin",
				Identifier:  "com.example.team-macos-mobileconfig-profile",
			},
		}, nil
	default:
		return nil, fmt.Errorf("unexpected team ID: %v", teamID)
	}
}

func (MockClient) GetScriptContents(scriptID uint) ([]byte, error) {
	if scriptID == 1 {
		return []byte("pop goes the weasel!"), nil
	}
	if scriptID == 2 {
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
		return &fleet.Team{
			ID:   1,
			Name: "Team A",
		}, nil
	}

	return nil, fmt.Errorf("team not found")
}

func TestGenerateGitops(t *testing.T) {
	fleetClient := &MockClient{}
	action := createGenerateGitopsAction(fleetClient)
	buf := new(bytes.Buffer)
	cliContext := cli.NewContext(&cli.App{
		Name:   "test",
		Usage:  "test",
		Writer: buf,
	}, nil, nil)
	err := action(cliContext)
	require.NoError(t, err)

	fmt.Println(buf.String()) // Debugging line
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
	controlsRaw, err := cmd.generateControls(0, "", &mdmConfig)
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
	var expectedAppConfig map[string]interface{}
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, controls)

	// Generate controls for a team.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	controlsRaw, err = cmd.generateControls(1, "some_team", nil)
	require.NoError(t, err)
	require.NotNil(t, controlsRaw)
	b, err = yaml.Marshal(controlsRaw)
	fmt.Println("Controls raw:\n", string(b)) // Debugging line
	err = yaml.Unmarshal(b, &controls)
	require.NoError(t, err)

	// Get the expected org settings YAML.
	b, err = os.ReadFile("./testdata/generateGitops/expectedTeamControls.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(b, &expectedAppConfig)
	require.NoError(t, err)

	// Compare.
	require.Equal(t, expectedAppConfig, controls)
}
