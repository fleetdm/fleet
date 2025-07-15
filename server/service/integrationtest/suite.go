package integrationtest

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

type BaseSuite struct {
	Logger   log.Logger
	FleetCfg config.FleetConfig
	Server   *httptest.Server
	DS       *mysql.Datastore
	Users    map[string]fleet.User
	Token    string

	cachedAdminToken string
}

func (s *BaseSuite) GetTestAdminToken(t *testing.T) string {
	// because the login endpoint is rate-limited, use the cached admin token
	// if available (if for some reason a test needs to logout the admin user,
	// then set cachedAdminToken = "" so that a new token is retrieved).
	if s.cachedAdminToken == "" {
		s.cachedAdminToken = s.GetTestToken(t, service.TestAdminUserEmail, test.GoodPassword)
	}
	return s.cachedAdminToken
}

func (s *BaseSuite) GetTestToken(t *testing.T, email string, password string) string {
	return service.GetToken(t, email, password, s.Server.URL)
}

func SetUpServerURL(t *testing.T, ds *mysql.Datastore, server *httptest.Server) {
	appConf, err := ds.AppConfig(t.Context())
	require.NoError(t, err)
	appConf.ServerSettings.ServerURL = server.URL
	err = ds.SaveAppConfig(t.Context(), appConf)
	require.NoError(t, err)
}

func SetUpMySQLAndService(t *testing.T, uniqueTestName string, opts ...*service.TestServerOpts) (
	*mysql.Datastore,
	config.FleetConfig,
	fleet.Service, context.Context,
) {
	ds := mysql.CreateMySQLDSWithOptions(t, &testing_utils.DatastoreTestOptions{
		UniqueTestName: uniqueTestName,
	})
	test.AddAllHostsLabel(t, ds)

	// Set up the required fields on AppConfig
	appConf, err := ds.AppConfig(testContext())
	require.NoError(t, err)
	appConf.OrgInfo.OrgName = "FleetTest"
	appConf.ServerSettings.ServerURL = "https://example.org"
	err = ds.SaveAppConfig(testContext(), appConf)
	require.NoError(t, err)

	fleetCfg := config.TestConfig()
	fleetSvc, ctx := service.NewTestService(t, ds, fleetCfg, opts...)
	return ds, fleetCfg, fleetSvc, ctx
}

func SetUpMySQLAndRedisAndService(t *testing.T, uniqueTestName string, opts ...*service.TestServerOpts) (*mysql.Datastore, fleet.RedisPool,
	config.FleetConfig,
	fleet.Service, context.Context,
) {
	redisPool := redistest.SetupRedis(t, uniqueTestName, false, false, false)
	ds, fleetCfg, fleetSvc, ctx := SetUpMySQLAndService(t, uniqueTestName, opts...)
	return ds, redisPool, fleetCfg, fleetSvc, ctx
}

func testContext() context.Context {
	return context.Background()
}
