// filepath: cmd/fleetctl/generate_gitops_test.go
package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	err := action(nil)
	if err != nil {
		t.Fatalf("Error generating GitOps configuration: %v", err)
	}
}
