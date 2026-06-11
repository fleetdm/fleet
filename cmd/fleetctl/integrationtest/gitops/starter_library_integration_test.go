package gitops

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/fleetctltest"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/integrationtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/svctest"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsStarterLibrary(t *testing.T) {
	testingSuite := new(starterLibraryIntegrationTestSuite)
	testingSuite.WithServer.Suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type starterLibraryIntegrationTestSuite struct {
	suite.Suite
	integrationtest.WithServer
}

func (s *starterLibraryIntegrationTestSuite) SetupSuite() {
	s.WithDS.SetupSuite("starterLibraryIntegrationTestSuite")

	appConf, err := s.DS.AppConfig(context.Background())
	s.Require().NoError(err)
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	s.Require().NoError(err)

	fleetCfg := config.TestConfig()
	fleetCfg.Osquery.EnrollCooldown = 0

	redisPool := redistest.SetupRedis(s.T(), "starter_library", false, false, false)

	serverConfig := service.TestServerOpts{
		FleetConfig: &fleetCfg,
		Pool:        redisPool,
	}

	users, server := svctest.RunServerForTestsWithDS(s.T(), s.DS, &serverConfig)
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL)
	s.Server = server
	s.Users = users

	appConf, err = s.DS.AppConfig(context.Background())
	s.Require().NoError(err)
	appConf.ServerSettings.ServerURL = server.URL
	appConf.OrgInfo.OrgName = "Test Org"
	appConf.GitOpsConfig.Exceptions = fleet.GitOpsExceptions{}
	err = s.DS.SaveAppConfig(context.Background(), appConf)
	s.Require().NoError(err)
}

// TestApplyStarterLibraryFree verifies that ApplyStarterLibrary applies only
// the global config (no teams) when using a free license.
func (s *starterLibraryIntegrationTestSuite) TestApplyStarterLibraryFree() {
	t := s.T()
	ctx := context.Background()

	token := s.GetTestToken("admin1@example.com", test.GoodPassword)
	logger := slog.New(slog.DiscardHandler)

	err := service.ApplyStarterLibrary(
		ctx,
		s.Server.URL,
		token,
		logger,
		func(args []string) error {
			_, err := fleetctltest.RunAppNoChecks(args)
			return err
		},
	)
	require.NoError(t, err)

	// Verify the org name was applied.
	appConfig, err := s.DS.AppConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Test Org", appConfig.OrgInfo.OrgName)

	// Verify that no teams were created for a free license.
	teams, err := s.DS.ListTeams(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String("admin")}}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, teams)

	// Verify labels were created (global labels, team_id=0).
	labelSpecs, err := s.DS.GetLabelSpecs(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String("admin")}})
	require.NoError(t, err)
	var customLabelNames []string
	for _, l := range labelSpecs {
		if l.LabelType != fleet.LabelTypeBuiltIn {
			customLabelNames = append(customLabelNames, l.Name)
		}
	}
	assert.Contains(t, customLabelNames, "Apple Silicon macOS hosts")
	assert.Contains(t, customLabelNames, "ARM-based Windows hosts")
	assert.Contains(t, customLabelNames, "Debian-based Linux hosts")
	assert.Contains(t, customLabelNames, "x86-based Windows hosts")
}
