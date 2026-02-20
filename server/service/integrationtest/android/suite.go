package android

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest"
	"github.com/stretchr/testify/require"
)

// noopActivityModule implements activities.ActivityModule with a no-op for tests.
type noopActivityModule struct{}

func (n *noopActivityModule) NewActivity(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
	return nil
}

type Suite struct {
	integrationtest.BaseSuite
	AndroidProxy *android_mock.Client
}

func SetUpSuite(t *testing.T, uniqueTestName string) *Suite {
	ds, redisPool, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndRedisAndService(t, uniqueTestName)
	logger := logging.NewLogfmtLogger(os.Stdout)
	proxy := android_mock.Client{}
	proxy.InitCommonMocks()
	activityModule := &noopActivityModule{}
	androidSvc, err := android_service.NewServiceWithClient(
		logger.SlogLogger(),
		ds,
		&proxy,
		"test-private-key",
		ds,
		activityModule,
		config.AndroidAgentConfig{
			Package:       "com.fleetdm.agent",
			SigningSHA256: "abc123def456",
		},
	)
	require.NoError(t, err)
	androidSvc.(*android_service.Service).AllowLocalhostServerURL = true
	users, server := service.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierFree,
		},
		FleetConfig:   &fleetCfg,
		Pool:          redisPool,
		Logger:        logger,
		FeatureRoutes: []endpointer.HandlerRoutesFunc{android_service.GetRoutes(fleetSvc, androidSvc)},
	})

	s := &Suite{
		BaseSuite: integrationtest.BaseSuite{
			Logger:   logger,
			DS:       ds,
			FleetCfg: fleetCfg,
			Users:    users,
			Server:   server,
		},
		AndroidProxy: &proxy,
	}

	integrationtest.SetUpServerURL(t, ds, server)

	s.Token = s.GetTestAdminToken(t)
	return s
}
