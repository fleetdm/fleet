package android

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

type Suite struct {
	integrationtest.BaseSuite
	AndroidProxy *android_mock.Proxy
}

func SetUpSuite(t *testing.T, uniqueTestName string) *Suite {
	ds, redisPool, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndRedisAndService(t, uniqueTestName)
	logger := log.NewLogfmtLogger(os.Stdout)
	proxy := android_mock.Proxy{}
	proxy.InitCommonMocks()
	androidSvc, err := android_service.NewServiceWithProxy(
		logger,
		ds,
		&proxy,
		fleetSvc,
	)
	require.NoError(t, err)
	users, server := service.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierFree,
		},
		FleetConfig:   &fleetCfg,
		Pool:          redisPool,
		Logger:        logger,
		FeatureRoutes: []endpoint_utils.HandlerRoutesFunc{android_service.GetRoutes(fleetSvc, androidSvc)},
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
