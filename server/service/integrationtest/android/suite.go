package android

import (
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"log/slog"
	"os"
	"testing"

	ee_android "github.com/fleetdm/fleet/v4/ee/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest"
	"github.com/fleetdm/fleet/v4/server/service/svctest"
	"github.com/stretchr/testify/require"
)

type Suite struct {
	integrationtest.BaseSuite
	AndroidProxy  *android_mock.Client
	KeyValueStore fleet.KeyValueStore
}

type suiteOptions struct {
	licenseTier   string
	wrapEEService bool
}

type SuiteOption func(*suiteOptions)

// WithEEAndroidService wraps the core Android service with the premium one, as
// cmd/fleet/serve.go does, and serves requests with the given license tier.
func WithEEAndroidService(licenseTier string) SuiteOption {
	return func(o *suiteOptions) {
		o.licenseTier = licenseTier
		o.wrapEEService = true
	}
}

func SetUpSuite(t *testing.T, uniqueTestName string, opts ...SuiteOption) *Suite {
	options := suiteOptions{licenseTier: fleet.TierFree}
	for _, opt := range opts {
		opt(&options)
	}

	license := &fleet.LicenseInfo{Tier: options.licenseTier}
	ds, redisPool, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndRedisAndService(t, uniqueTestName, &service.TestServerOpts{License: license})
	keyValueStore := redis_key_value.New(redisPool)
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	proxy := android_mock.Client{}
	proxy.InitCommonMocks()
	coreAndroidSvc, err := android_service.NewServiceWithClient(
		slogLogger,
		ds,
		&proxy,
		"test-private-key",
		ds,
		fleetSvc.NewActivity,
		config.AndroidAgentConfig{
			Package:       "com.fleetdm.agent",
			SigningSHA256: "abc123def456",
		},
		android_service.WithKeyValueStore(keyValueStore),
	)
	require.NoError(t, err)
	coreAndroidSvc.(*android_service.Service).AllowLocalhostServerURL = true
	androidSvc := coreAndroidSvc
	if options.wrapEEService {
		eeAndroidSvc, err := ee_android.NewService(coreAndroidSvc, ds, ds, &proxy, slogLogger)
		require.NoError(t, err)
		androidSvc = eeAndroidSvc
	}
	dbConns := mysqltest.TestDBConnections(t, ds)
	users, server := svctest.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License:       license,
		FleetConfig:   &fleetCfg,
		Pool:          redisPool,
		Logger:        slogLogger,
		FeatureRoutes: []endpointer.HandlerRoutesFunc{android_service.GetRoutes(fleetSvc, androidSvc)},
		DBConns:       dbConns,
	})

	s := &Suite{
		BaseSuite: integrationtest.BaseSuite{
			Logger:   slogLogger,
			DS:       ds,
			FleetCfg: fleetCfg,
			Users:    users,
			Server:   server,
		},
		AndroidProxy:  &proxy,
		KeyValueStore: keyValueStore,
	}

	integrationtest.SetUpServerURL(t, ds, server)

	s.Token = s.GetTestAdminToken(t)
	return s
}
