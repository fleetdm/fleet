package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestEnterpriseNoMdmIntegrationsGitops(t *testing.T) {
	testingSuite := new(enterpriseNoMdmIntegrationGitopsTestSuite)
	testingSuite.suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type enterpriseNoMdmIntegrationGitopsTestSuite struct {
	suite.Suite
	withServer
	fleetCfg config.FleetConfig
}

func (s *enterpriseNoMdmIntegrationGitopsTestSuite) SetupSuite() {
	s.withDS.SetupSuite("enterpriseNoMdmIntegrationGitopsTestSuite")

	fleetCfg := config.TestConfig()
	fleetCfg.Osquery.EnrollCooldown = 0

	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)

	serverConfig := service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		FleetConfig: &fleetCfg,
		Pool:        redisPool,
	}
	users, server := service.RunServerForTestsWithDS(s.T(), s.ds, &serverConfig)
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL) // fleetctl always uses this env var in tests
	s.server = server
	s.users = users
	s.fleetCfg = fleetCfg

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

// TestFleetGitops runs `fleetctl gitops` command on configs in https://github.com/fleetdm/fleet-gitops repo.
// Changes to that repo may cause this test to fail.
func (s *enterpriseNoMdmIntegrationGitopsTestSuite) TestFleetGitops() {
	t := s.T()
	const fleetGitopsRepo = "https://github.com/fleetdm/fleet-gitops"

	// Create GitOps user
	user := fleet.User{
		Name:       "GitOps User",
		Email:      "fleetctl-gitops@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(context.Background(), &user)
	require.NoError(t, err)

	// Create a temporary fleetctl config file
	fleetctlConfig, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	token := s.getTestToken(user.Email, test.GoodPassword)
	configStr := fmt.Sprintf(
		`
contexts:
  default:
    address: %s
    tls-skip-verify: true
    token: %s
`, s.server.URL, token,
	)
	_, err = fleetctlConfig.WriteString(configStr)
	require.NoError(t, err)

	// Clone git repo
	repoDir := t.TempDir()
	_, err = git.PlainClone(
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
	t.Setenv("FLEET_SSO_METADATA", "sso_metadata")
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_ENROLL_SECRET", "workstations_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET", "workstations_canary_enroll_secret")
	// Read the global file
	globalFile := path.Join(repoDir, "default.yml")
	s.removeControls(globalFile)
	teamsDir := path.Join(repoDir, "teams")
	teamFiles, err := os.ReadDir(teamsDir)
	require.NoError(t, err)
	teamFileNames := make([]string, 0, len(teamFiles))
	for _, file := range teamFiles {
		if filepath.Ext(file.Name()) == ".yml" {
			teamsFile := path.Join(teamsDir, file.Name())
			s.removeControls(teamsFile)
			teamFileNames = append(teamFileNames, teamsFile)
		}
	}

	// Dry run with all the files
	args := []string{"gitops", "--config", fleetctlConfig.Name(), "--dry-run", "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = runAppForTest(t, args)

	// Dry run with one file at a time
	_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile, "--dry-run"})
	for _, fileName := range teamFileNames {
		_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName, "--dry-run"})
	}

	args = []string{"gitops", "--config", fleetctlConfig.Name(), "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = runAppForTest(t, args)

	// Check that only the teams exist
	teamsJSON := runAppForTest(t, []string{"get", "teams", "--config", fleetctlConfig.Name(), "--json"})
	assert.Equal(t, 2, strings.Count(teamsJSON, "team_id"))

	// Real run with one file at a time
	_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", globalFile})
	for _, fileName := range teamFileNames {
		_ = runAppForTest(t, []string{"gitops", "--config", fleetctlConfig.Name(), "-f", fileName})
	}

}

// removeControls removes MDM settings (controls) from the gitops YAML file
func (s *enterpriseNoMdmIntegrationGitopsTestSuite) removeControls(file string) {
	t := s.T()
	b, err := os.ReadFile(file)
	require.NoError(t, err)
	var top map[string]json.RawMessage
	err = yaml.Unmarshal(b, &top)
	require.NoError(t, err)
	// Remove MDM settings (controls)
	top["controls"] = []byte(`null`)
	dataToWrite, err := yaml.Marshal(top)
	require.NoError(t, err)
	err = os.WriteFile(file, dataToWrite, os.ModePerm)
	require.NoError(t, err)
}
