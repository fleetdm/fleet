package integrationtest

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

type Suite struct {
	Logger       log.Logger
	FleetCfg     config.FleetConfig
	Server       *httptest.Server
	DS           *mysql.Datastore
	Users        map[string]fleet.User
	Token        string
	AndroidProxy *android_mock.Proxy

	cachedAdminToken string
}

var testUsers = map[string]struct {
	Email             string
	PlaintextPassword string
	GlobalRole        *string
}{
	"admin1": {
		PlaintextPassword: test.GoodPassword,
		Email:             "admin1@example.com",
		GlobalRole:        ptr.String(fleet.RoleAdmin),
	},
	"user1": {
		PlaintextPassword: test.GoodPassword,
		Email:             "user1@example.com",
		GlobalRole:        ptr.String(fleet.RoleMaintainer),
	},
	"user2": {
		PlaintextPassword: test.GoodPassword,
		Email:             "user2@example.com",
		GlobalRole:        ptr.String(fleet.RoleObserver),
	},
}

func (s *Suite) GetTestAdminToken(t *testing.T) string {
	testUser := testUsers["admin1"]

	// because the login endpoint is rate-limited, use the cached admin token
	// if available (if for some reason a test needs to logout the admin user,
	// then set cachedAdminToken = "" so that a new token is retrieved).
	if s.cachedAdminToken == "" {
		s.cachedAdminToken = s.GetTestToken(t, testUser.Email, testUser.PlaintextPassword)
	}
	return s.cachedAdminToken
}

func (s *Suite) GetTestToken(t *testing.T, email string, password string) string {
	return service.GetToken(t, email, password, s.Server.URL)
}

func SetUpSuite(t *testing.T, uniqueTestName string) *Suite {
	ds := mysql.CreateMySQLDS(t)
	test.AddAllHostsLabel(t, ds)

	// Set up the required fields on AppConfig
	appConf, err := ds.AppConfig(testContext())
	require.NoError(t, err)
	appConf.OrgInfo.OrgName = "FleetTest"
	appConf.ServerSettings.ServerURL = "https://example.org"
	err = ds.SaveAppConfig(testContext(), appConf)
	require.NoError(t, err)

	redisPool := redistest.SetupRedis(t, uniqueTestName, false, false, false)

	fleetCfg := config.TestConfig()
	logger := log.NewLogfmtLogger(os.Stdout)
	fleetSvc, ctx := service.NewTestService(t, ds, fleetCfg)
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
		Logger:       logger,
		DS:           ds,
		FleetCfg:     fleetCfg,
		Users:        users,
		Server:       server,
		AndroidProxy: &proxy,
	}

	appConf, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	appConf.ServerSettings.ServerURL = server.URL
	err = ds.SaveAppConfig(ctx, appConf)
	require.NoError(t, err)

	s.Token = s.GetTestAdminToken(t)
	return s
}

func testContext() context.Context {
	return context.Background()
}
