// filepath: cmd/fleetctl/generate_gitops_test.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
