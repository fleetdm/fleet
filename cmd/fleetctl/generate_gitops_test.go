// filepath: cmd/fleetctl/generate_gitops_test.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

	// Generate the org settings.
	// Note that nested keys here may be strings,
	// so we'll JSON marshal and unmarshal to a map for comparison.
	orgSettingsRaw, err := generateOrgSettings(nil, appConfig, nil)
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
