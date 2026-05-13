package svctest

import (
	"context"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/scim"
	"github.com/fleetdm/fleet/v4/ee/server/service/condaccess"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/httpsig"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/acl/acmeacl"
	"github.com/fleetdm/fleet/v4/server/acl/activityacl"
	activity_bootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	acme_bootstrap "github.com/fleetdm/fleet/v4/server/mdm/acme/bootstrap"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

// RunServerForTestsWithDS spins up a full HTTP test server backed by ds and
// returns the seeded users plus the server.
func RunServerForTestsWithDS(t *testing.T, ds fleet.Datastore, opts ...*service.TestServerOpts) (map[string]fleet.User, *httptest.Server) {
	if len(opts) > 0 && opts[0].EnableCachedDS {
		ds = cached_mysql.New(ds)
	}
	cfg := config.TestConfig()
	if len(opts) > 0 && opts[0].FleetConfig != nil {
		cfg = *opts[0].FleetConfig
	}
	svc, ctx := NewTestService(t, ds, cfg, opts...)
	return RunServerForTestsWithServiceWithDS(t, ctx, ds, svc, opts...)
}

// RunServerForTestsWithServiceWithDS spins up an HTTP test server using the
// given service. Use this when callers need to compose the service themselves
// (e.g., to wire in custom storage or auth modules) before standing up the
// HTTP layer.
func RunServerForTestsWithServiceWithDS(t *testing.T, ctx context.Context, ds fleet.Datastore, svc fleet.Service,
	opts ...*service.TestServerOpts,
) (map[string]fleet.User, *httptest.Server) {
	var cfg config.FleetConfig
	if len(opts) > 0 && opts[0].FleetConfig != nil {
		cfg = *opts[0].FleetConfig
	} else {
		cfg = config.TestConfig()
	}
	users := map[string]fleet.User{}
	if len(opts) == 0 || (len(opts) > 0 && !opts[0].SkipCreateTestUsers) {
		users = createTestUsers(t, ds)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if len(opts) > 0 && opts[0].Logger != nil {
		logger = opts[0].Logger
	}

	if len(opts) > 0 {
		opts[0].FeatureRoutes = append(opts[0].FeatureRoutes, android_service.GetRoutes(svc, opts[0].AndroidModule))
	}

	// Activity routes. If DBConns is provided, wire the real bounded context into
	// the main handler. Otherwise, build a path-only stub from the same registration
	// code and surface it to apiendpoints.Init for catalog validation only.
	var extraInitFeatureRoutes []apiendpoints.FeatureRouteFunc
	if len(opts) > 0 && opts[0].DBConns != nil {
		legacyAuthorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)
		activityAuthorizer := authz.NewAuthorizerAdapter(legacyAuthorizer)
		activityACLAdapter := activityacl.NewFleetServiceAdapter(svc)
		activitySvc, activityRoutesFn := activity_bootstrap.New(
			opts[0].DBConns,
			activityAuthorizer,
			activityACLAdapter,
			logger,
		)
		svc.SetActivityService(activitySvc)
		activityAuthMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
			return auth.AuthenticatedUser(svc, next)
		}
		opts[0].FeatureRoutes = append(opts[0].FeatureRoutes, activityRoutesFn(activityAuthMiddleware))
	} else {
		// DBConns is not available (e.g. mock-backed tests). The activity bounded context
		// only dereferences its dependencies when an endpoint is actually served, so we
		// can pass empty conns + nil deps just to extract the route declarations.
		_, activityRoutesFn := activity_bootstrap.New(
			&common_mysql.DBConnections{},
			nil,
			nil,
			logger,
		)
		noopAuth := func(next endpoint.Endpoint) endpoint.Endpoint { return next }
		extraInitFeatureRoutes = append(extraInitFeatureRoutes, apiendpoints.FeatureRouteFunc(activityRoutesFn(noopAuth)))
	}

	var mdmPusher nanomdm_push.Pusher
	if len(opts) > 0 && opts[0].MDMPusher != nil {
		mdmPusher = opts[0].MDMPusher
	}
	rootMux := http.NewServeMux()

	memLimitStore, _ := memstore.New(0)
	var limitStore throttled.GCRAStore = memLimitStore
	var redisPool fleet.RedisPool
	if len(opts) > 0 && opts[0].Pool != nil {
		redisPool = opts[0].Pool
		limitStore = &redis.ThrottledStore{
			Pool:      opts[0].Pool,
			KeyPrefix: "ratelimit::",
		}
	} else {
		redisPool = redistest.SetupRedis(t, t.Name(), false, false, false) // We are good to initalize a redis pool here as it is only called by integration tests
	}

	// Wire real ACME service module if DBConns is provided (overrides the mock set in newTestServiceWithConfig).
	if len(opts) > 0 && opts[0].DBConns != nil {
		var acmeOpts []acme_bootstrap.ServiceOption
		if opts[0].ACMECertCA != nil && opts[0].ACMECertKey != nil {
			rootCAPool := x509.NewCertPool()
			rootCAPool.AddCert(opts[0].ACMECertCA)
			acmeOpts = append(acmeOpts, acme_bootstrap.WithTestAppleRootCAs(rootCAPool))
		}
		acmeSigner := &acmeCSRSigner{signer: depot.NewSigner(opts[0].SCEPStorage, depot.WithValidityDays(365), depot.WithAllowRenewalDays(14))}
		acmeSvc, acmeRoutes := acme_bootstrap.New(opts[0].DBConns, redisPool, acmeacl.NewFleetDatastoreAdapter(ds, acmeSigner), logger, acmeOpts...)
		svc.SetACMEService(acmeSvc)
		opts[0].FeatureRoutes = append(opts[0].FeatureRoutes, acmeRoutes(log.Logged))
	}

	if len(opts) > 0 {
		mdmStorage := opts[0].MDMStorage
		scepStorage := opts[0].SCEPStorage
		commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPusher)
		if mdmStorage != nil && scepStorage != nil {
			vppInstaller := svc.(fleet.AppleMDMVPPInstaller)
			checkInAndCommand := service.NewMDMAppleCheckinAndCommandService(ds, commander, vppInstaller, opts[0].License.IsPremium(), logger, redis_key_value.New(redisPool), svc.NewActivity)
			checkInAndCommand.RegisterResultsHandler("InstalledApplicationList", service.NewInstalledApplicationListResultsHandler(ds, commander, logger, cfg.Server.VPPVerifyTimeout, cfg.Server.VPPVerifyRequestDelay, svc.NewActivity))
			checkInAndCommand.RegisterResultsHandler(fleet.DeviceLocationCmdName, service.NewDeviceLocationResultsHandler(ds, commander, logger))
			checkInAndCommand.RegisterResultsHandler(fleet.SetRecoveryLockCmdName, service.NewSetRecoveryLockResultsHandler(ds, logger, svc.NewActivity))
			err := service.RegisterAppleMDMProtocolServices(
				rootMux,
				cfg.MDM,
				mdmStorage,
				scepStorage,
				logger,
				checkInAndCommand,
				service.NewMDMAppleDDMService(ds, logger),
				commander,
				"https://test-url.com",
				cfg,
			)
			require.NoError(t, err)
		}
		if opts[0].EnableSCEPProxy {
			var timeout *time.Duration
			if opts[0].SCEPConfigService != nil {
				scepConfig, ok := opts[0].SCEPConfigService.(*scep.SCEPConfigService)
				if ok {
					// In tests, we share the same Timeout pointer between SCEPConfigService and SCEPProxy
					timeout = scepConfig.Timeout
				}
			}
			err := service.RegisterSCEPProxy(
				rootMux,
				ds,
				logger,
				timeout,
				&cfg,
			)
			require.NoError(t, err)
		}
	}

	if len(opts) > 0 && opts[0].WithDEPWebview {
		frontendHandler := service.WithMDMEnrollmentMiddleware(svc, logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// do nothing and return 200
			w.WriteHeader(http.StatusOK)
		}))
		rootMux.Handle("/", frontendHandler)
	}

	featureRoutes := opts0FeatureRoutes(opts)
	var extra []service.ExtraHandlerOption
	extra = append(extra, service.WithLoginRateLimit(throttled.PerMin(1000)))

	if len(opts) > 0 && opts[0].HostIdentity != nil {
		require.NoError(t, hostidentity.RegisterSCEP(rootMux, opts[0].HostIdentity.SCEPStorage, ds, logger, &cfg))
		var httpSigVerifier func(http.Handler) http.Handler
		httpSigVerifier, err := httpsig.Middleware(ds, opts[0].HostIdentity.RequireHTTPMessageSignature,
			logger.With("component", "http-sig-verifier"))
		require.NoError(t, err)
		extra = append(extra, service.WithHTTPSigVerifier(httpSigVerifier))
	}

	if len(opts) > 0 && opts[0].ConditionalAccess != nil {
		require.NoError(t, condaccess.RegisterSCEP(ctx, rootMux, opts[0].ConditionalAccess.SCEPStorage, ds, logger, &cfg))
		require.NoError(t, condaccess.RegisterIdP(rootMux, ds, logger, &cfg, limitStore))
	}
	var carveStore fleet.CarveStore = ds // In tests, we use MySQL as storage for carves.
	apiHandler := service.MakeHandler(svc, cfg, logger, limitStore, redisPool, carveStore, featureRoutes, extra...)
	if err := apiendpoints.Init(apiHandler, extraInitFeatureRoutes...); err != nil {
		t.Fatalf("error initializing API endpoints: %v", err)
	}
	rootMux.Handle("/api/", apiHandler)
	var errHandler *errorstore.Handler
	ctxErrHandler := ctxerr.FromContext(ctx)
	if ctxErrHandler != nil {
		errHandler = ctxErrHandler.(*errorstore.Handler)
	}
	debugHandler := service.MakeDebugHandler(svc, cfg, logger, errHandler, ds)
	rootMux.Handle("/debug/", debugHandler)
	rootMux.Handle("/enroll", service.ServeEndUserEnrollOTA(svc, "", ds, logger, false))

	if len(opts) > 0 && opts[0].EnableSCIM {
		require.NoError(t, scim.RegisterSCIM(rootMux, ds, svc, logger, &cfg))
		rootMux.Handle("/api/v1/fleet/scim/details", apiHandler)
		rootMux.Handle("/api/latest/fleet/scim/details", apiHandler)
	}

	server := httptest.NewUnstartedServer(rootMux)
	server.Config = cfg.Server.DefaultHTTPServer(ctx, rootMux)
	// WriteTimeout is set for security purposes.
	// If we don't set it, (bugy or malignant) clients making long running
	// requests could DDOS Fleet.
	require.NotZero(t, server.Config.WriteTimeout)
	if len(opts) > 0 && opts[0].HTTPServerConfig != nil {
		server.Config = opts[0].HTTPServerConfig
		// make sure we use the application handler we just created
		server.Config.Handler = rootMux
	}
	server.Start()
	t.Cleanup(func() {
		server.Close()
	})
	return users, server
}

// opts0FeatureRoutes safely extracts FeatureRoutes from the first opt if present.
func opts0FeatureRoutes(opts []*service.TestServerOpts) []endpointer.HandlerRoutesFunc {
	if len(opts) > 0 && len(opts[0].FeatureRoutes) > 0 {
		return opts[0].FeatureRoutes
	}
	return nil
}
