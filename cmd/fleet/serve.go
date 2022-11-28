package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/e-dard/netbug"
	"github.com/fleetdm/fleet/v4/ee/server/licensing"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/server"
	configpkg "github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/launcher"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mail"
	config_apple "github.com/fleetdm/fleet/v4/server/mdm/apple/config"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/redis_policy_set"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/kit/version"
	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/push/buford"
	nanomdm_pushsvc "github.com/micromdm/nanomdm/push/service"
	"github.com/ngrok/sqlmw"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.elastic.co/apm/module/apmhttp"
	_ "go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/mysql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

var allowedURLPrefixRegexp = regexp.MustCompile("^(?:/[a-zA-Z0-9_.~-]+)+$")

type initializer interface {
	// Initialize is used to populate a datastore with
	// preloaded data
	Initialize() error
}

func createServeCmd(configManager configpkg.Manager) *cobra.Command {
	// Whether to enable the debug endpoints
	debug := false
	// Whether to enable developer options
	dev := false
	// Whether to enable development Fleet Premium license
	devLicense := false
	// Whether to enable development Fleet Premium license with an expired license
	devExpiredLicense := false

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch the Fleet server",
		Long: `
Launch the Fleet server

Use fleet serve to run the main HTTPS server. The Fleet server bundles
together all static assets and dependent libraries into a statically linked go
binary (which you're executing right now). Use the options below to customize
the way that the Fleet server works.
`,
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if dev {
				applyDevFlags(&config)
			}

			if devLicense {
				// This license key is valid for development only
				config.License.Key = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNzUxMjQxNjAwLCJzdWIiOiJkZXZlbG9wbWVudC1vbmx5IiwiZGV2aWNlcyI6MTAwLCJub3RlIjoiZm9yIGRldmVsb3BtZW50IG9ubHkiLCJ0aWVyIjoicHJlbWl1bSIsImlhdCI6MTY1NjY5NDA4N30.dvfterOvfTGdrsyeWYH9_lPnyovxggM5B7tkSl1q1qgFYk_GgOIxbaqIZ6gJlL0cQuBF9nt5NgV0AUT9RmZUaA"
			} else if devExpiredLicense {
				// An expired license key
				config.License.Key = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjI5NzYzMjAwLCJzdWIiOiJEZXYgbGljZW5zZSAoZXhwaXJlZCkiLCJkZXZpY2VzIjo1MDAwMDAsIm5vdGUiOiJUaGlzIGxpY2Vuc2UgaXMgdXNlZCB0byBmb3IgZGV2ZWxvcG1lbnQgcHVycG9zZXMuIiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjI5OTA0NzMyfQ.AOppRkl1Mlc_dYKH9zwRqaTcL0_bQzs7RM3WSmxd3PeCH9CxJREfXma8gm0Iand6uIWw8gHq5Dn0Ivtv80xKvQ"
			}

			license, err := licensing.LoadLicense(config.License.Key)
			if err != nil {
				initFatal(
					err,
					"failed to load license - for help use https://fleetdm.com/contact",
				)
			}

			if license != nil && license.IsPremium() && license.IsExpired() {
				fleet.WriteExpiredLicenseBanner(os.Stderr)
			}

			var logger kitlog.Logger
			{
				output := os.Stderr
				if config.Logging.JSON {
					logger = kitlog.NewJSONLogger(output)
				} else {
					logger = kitlog.NewLogfmtLogger(output)
				}
				if config.Logging.Debug {
					logger = level.NewFilter(logger, level.AllowDebug())
				} else {
					logger = level.NewFilter(logger, level.AllowInfo())
				}
				logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
			}

			// Init tracing
			if config.Logging.TracingEnabled {
				ctx := context.Background()
				client := otlptracegrpc.NewClient()
				otlpTraceExporter, err := otlptrace.New(ctx, client)
				if err != nil {
					initFatal(err, "Failed to initialize tracing")
				}
				batchSpanProcessor := sdktrace.NewBatchSpanProcessor(otlpTraceExporter)
				tracerProvider := sdktrace.NewTracerProvider(
					sdktrace.WithSpanProcessor(batchSpanProcessor),
				)
				otel.SetTracerProvider(tracerProvider)
			}

			allowedHostIdentifiers := map[string]bool{
				"provided": true,
				"instance": true,
				"uuid":     true,
				"hostname": true,
			}
			if !allowedHostIdentifiers[config.Osquery.HostIdentifier] {
				initFatal(fmt.Errorf("%s is not a valid value for osquery_host_identifier", config.Osquery.HostIdentifier), "set host identifier")
			}

			if len(config.Server.URLPrefix) > 0 {
				// Massage provided prefix to match expected format
				config.Server.URLPrefix = strings.TrimSuffix(config.Server.URLPrefix, "/")
				if len(config.Server.URLPrefix) > 0 && !strings.HasPrefix(config.Server.URLPrefix, "/") {
					config.Server.URLPrefix = "/" + config.Server.URLPrefix
				}

				if !allowedURLPrefixRegexp.MatchString(config.Server.URLPrefix) {
					initFatal(
						fmt.Errorf("prefix must match regexp \"%s\"", allowedURLPrefixRegexp.String()),
						"setting server URL prefix",
					)
				}
			}

			var ds fleet.Datastore
			var carveStore fleet.CarveStore
			var installerStore fleet.InstallerStore
			mailService := mail.NewService()

			opts := []mysql.DBOption{mysql.Logger(logger), mysql.WithFleetConfig(&config)}
			if config.MysqlReadReplica.Address != "" {
				opts = append(opts, mysql.Replica(&config.MysqlReadReplica))
			}
			if dev && os.Getenv("FLEET_ENABLE_DEV_SQL_INTERCEPTOR") != "" {
				opts = append(opts, mysql.WithInterceptor(&devSQLInterceptor{
					logger: kitlog.With(logger, "component", "sql-interceptor"),
				}))
			}

			if config.Logging.TracingEnabled {
				opts = append(opts, mysql.TracingEnabled(&config.Logging))
			}

			mds, err := mysql.New(config.Mysql, clock.C, opts...)
			if err != nil {
				initFatal(err, "initializing datastore")
			}
			ds = mds

			if config.S3.Bucket != "" {
				carveStore, err = s3.NewCarveStore(config.S3, ds)
				if err != nil {
					initFatal(err, "initializing S3 carvestore")
				}
			} else {
				carveStore = ds
			}

			if config.Packaging.S3.Bucket != "" {
				var err error
				installerStore, err = s3.NewInstallerStore(config.Packaging.S3)
				if err != nil {
					initFatal(err, "initializing S3 installer store")
				}
			}

			migrationStatus, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			switch migrationStatus.StatusCode {
			case fleet.AllMigrationsCompleted:
				// OK
			case fleet.UnknownMigrations:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database has unrecognized migrations. This could happen when\n"+
					"#   running an older version of Fleet on a newer migrated database.\n"+
					"#\n"+
					"#   Unknown migrations: tables=%v, data=%v.\n"+
					"################################################################################\n",
					migrationStatus.UnknownTable, migrationStatus.UnknownData)
				if dev {
					os.Exit(1)
				}
			case fleet.SomeMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database is missing required migrations. This is likely to cause\n"+
					"#   errors in Fleet.\n"+
					"#\n"+
					"#   Missing migrations: tables=%v, data=%v.\n"+
					"#\n"+
					"#   Run `%s prepare db` to perform migrations.\n"+
					"#\n"+
					"#   To run the server without performing migrations:\n"+
					"#     - Set environment variable FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS=1, or,\n"+
					"#     - Set config updates.allow_mising_migrations to true, or,\n"+
					"#     - Use command line argument --upgrades_allow_missing_migrations=true\n"+
					"################################################################################\n",
					migrationStatus.MissingTable, migrationStatus.MissingData, os.Args[0])
				if !config.Upgrades.AllowMissingMigrations {
					os.Exit(1)
				}
			case fleet.NoMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# ERROR:\n"+
					"#   Your Fleet database is not initialized. Fleet cannot start up.\n"+
					"#\n"+
					"#   Run `%s prepare db` to initialize the database.\n"+
					"################################################################################\n",
					os.Args[0])
				os.Exit(1)
			}

			if initializingDS, ok := ds.(initializer); ok {
				if err := initializingDS.Initialize(); err != nil {
					initFatal(err, "loading built in data")
				}
			}

			if config.Packaging.GlobalEnrollSecret != "" {
				secrets, err := ds.GetEnrollSecrets(cmd.Context(), nil)
				if err != nil {
					initFatal(err, "loading enroll secrets")
				}

				var globalEnrollSecret string
				for _, secret := range secrets {
					if secret.TeamID == nil {
						globalEnrollSecret = secret.Secret
						break
					}
				}

				if globalEnrollSecret != "" {
					if globalEnrollSecret != config.Packaging.GlobalEnrollSecret {
						fmt.Printf("################################################################################\n" +
							"# WARNING:\n" +
							"#  You have provided a global enroll secret config, but there's\n" +
							"#  already one set up for your application.\n" +
							"#\n" +
							"#  This is generally an error and the provided value will be\n" +
							"#  ignored, if you really need to configure an enroll secret please\n" +
							"#  remove the global enroll secret from the database manually.\n" +
							"################################################################################\n")
						os.Exit(1)
					}
				} else {
					ds.ApplyEnrollSecrets(cmd.Context(), nil, []*fleet.EnrollSecret{{Secret: config.Packaging.GlobalEnrollSecret}})
				}
			}

			redisPool, err := redis.NewPool(redis.PoolConfig{
				Server:                    config.Redis.Address,
				Password:                  config.Redis.Password,
				Database:                  config.Redis.Database,
				UseTLS:                    config.Redis.UseTLS,
				ConnTimeout:               config.Redis.ConnectTimeout,
				KeepAlive:                 config.Redis.KeepAlive,
				ConnectRetryAttempts:      config.Redis.ConnectRetryAttempts,
				ClusterFollowRedirections: config.Redis.ClusterFollowRedirections,
				ClusterReadFromReplica:    config.Redis.ClusterReadFromReplica,
				TLSCert:                   config.Redis.TLSCert,
				TLSKey:                    config.Redis.TLSKey,
				TLSCA:                     config.Redis.TLSCA,
				TLSServerName:             config.Redis.TLSServerName,
				TLSHandshakeTimeout:       config.Redis.TLSHandshakeTimeout,
				MaxIdleConns:              config.Redis.MaxIdleConns,
				MaxOpenConns:              config.Redis.MaxOpenConns,
				ConnMaxLifetime:           config.Redis.ConnMaxLifetime,
				IdleTimeout:               config.Redis.IdleTimeout,
				ConnWaitTimeout:           config.Redis.ConnWaitTimeout,
				WriteTimeout:              config.Redis.WriteTimeout,
				ReadTimeout:               config.Redis.ReadTimeout,
			})
			if err != nil {
				initFatal(err, "initialize Redis")
			}
			level.Info(logger).Log("component", "redis", "mode", redisPool.Mode())

			ds = cached_mysql.New(ds)
			var dsOpts []mysqlredis.Option
			if license.DeviceCount > 0 && config.License.EnforceHostLimit {
				dsOpts = append(dsOpts, mysqlredis.WithEnforcedHostLimit(license.DeviceCount))
			}
			redisWrapperDS := mysqlredis.New(ds, redisPool, dsOpts...)
			ds = redisWrapperDS

			resultStore := pubsub.NewRedisQueryResults(redisPool, config.Redis.DuplicateResults)
			liveQueryStore := live_query.NewRedisLiveQuery(redisPool)
			ssoSessionStore := sso.NewSessionStore(redisPool)

			osqueryLogger, err := logging.New(config, logger)
			if err != nil {
				initFatal(err, "initializing osquery logging")
			}

			failingPolicySet := redis_policy_set.NewFailing(redisPool)

			task := async.NewTask(ds, redisPool, clock.C, config.Osquery)

			if config.Sentry.Dsn != "" {
				v := version.Version()
				err = sentry.Init(sentry.ClientOptions{
					Dsn:     config.Sentry.Dsn,
					Release: fmt.Sprintf("%s_%s_%s", v.Version, v.Branch, v.Revision),
				})
				if err != nil {
					initFatal(err, "initializing sentry")
				}
				level.Info(logger).Log("msg", "sentry initialized", "dsn", config.Sentry.Dsn)

				defer sentry.Recover()
				defer sentry.Flush(2 * time.Second)
			}

			var geoIP fleet.GeoIP
			geoIP = &fleet.NoOpGeoIP{}
			if config.GeoIP.DatabasePath != "" {
				maxmind, err := fleet.NewMaxMindGeoIP(logger, config.GeoIP.DatabasePath)
				if err != nil {
					level.Error(logger).Log("msg", "failed to initialize maxmind geoip, check database path", "database_path", config.GeoIP.DatabasePath, "error", err)
				} else {
					geoIP = maxmind
				}
			}

			var (
				scepStorage      *scep_mysql.MySQLDepot
				depStorage       *mysql.NanoDEPStorage
				mdmStorage       *mysql.NanoMDMStorage
				mdmPushService   *nanomdm_pushsvc.PushService
				mdmPushCertTopic string
			)
			if config.MDMApple.Enable {
				if err := config_apple.Verify(config.MDMApple); err != nil {
					initFatal(err, "verify apple mdm config")
				}
				scepStorage, err = mds.NewMDMAppleSCEPDepot(
					[]byte(config.MDMApple.SCEP.CA.PEMCert),
					[]byte(config.MDMApple.SCEP.CA.PEMKey),
				)
				if err != nil {
					initFatal(err, "initialize mdm apple scep storage")
				}
				mdmStorage, err = mds.NewMDMAppleMDMStorage(
					[]byte(config.MDMApple.MDM.PushCert.PEMCert),
					[]byte(config.MDMApple.MDM.PushCert.PEMKey),
				)
				if err != nil {
					initFatal(err, "initialize mdm apple MySQL storage")
				}
				mdmPushCertTopic, err = cryptoutil.TopicFromPEMCert([]byte(config.MDMApple.MDM.PushCert.PEMCert))
				if err != nil {
					initFatal(err, "extract topic from mdm push certificate")
				}
				depStorage, err = mds.NewMDMAppleDEPStorage([]byte(config.MDMApple.DEP.Token))
				if err != nil {
					initFatal(err, "initialize mdm apple dep storage")
				}
				nanoMDMLogger := NewNanoMDMLogger(kitlog.With(logger, "component", "apple-mdm-push"))
				pushProviderFactory := buford.NewPushProviderFactory()
				mdmPushService = nanomdm_pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, nanoMDMLogger)
			}

			cronSchedules := fleet.NewCronSchedules()

			baseCtx := licensectx.NewContext(context.Background(), license)
			ctx, cancelFunc := context.WithCancel(baseCtx)
			defer cancelFunc() // TODO(sarah); Handle release of locks in graceful shutdown

			eh := errorstore.NewHandler(ctx, redisPool, logger, config.Logging.ErrorRetentionPeriod)
			ctx = ctxerr.NewContext(ctx, eh)
			svc, err := service.NewService(
				ctx,
				ds,
				task,
				resultStore,
				logger,
				osqueryLogger,
				config,
				mailService,
				clock.C,
				ssoSessionStore,
				liveQueryStore,
				carveStore,
				installerStore,
				failingPolicySet,
				geoIP,
				redisWrapperDS,
				depStorage,
				mdmStorage,
				mdmPushService,
				mdmPushCertTopic,
				cronSchedules,
			)
			if err != nil {
				initFatal(err, "initializing service")
			}

			if license.IsPremium() {
				svc, err = eeservice.NewService(svc, ds, logger, config, mailService, clock.C)
				if err != nil {
					initFatal(err, "initial Fleet Premium service")
				}
			}

			instanceID, err := server.GenerateRandomText(64)
			if err != nil {
				initFatal(errors.New("Error generating random instance identifier"), "")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newCleanupsAndAggregationSchedule(ctx, instanceID, ds, logger, redisWrapperDS)
			}); err != nil {
				initFatal(err, "failed to register cleanups_then_aggregations schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newSendStatsSchedule(ctx, instanceID, ds, config, license, logger)
			}); err != nil {
				initFatal(err, "failed to register stats schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newVulnerabilitiesSchedule(ctx, instanceID, ds, logger, &config.Vulnerabilities)
			}); err != nil {
				initFatal(err, "failed to register vulnerabilities schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newAutomationsSchedule(ctx, instanceID, ds, logger, 5*time.Minute, failingPolicySet)
			}); err != nil {
				initFatal(err, "failed to register automations schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newIntegrationsSchedule(ctx, instanceID, ds, logger)
			}); err != nil {
				initFatal(err, "failed to register integrations schedule")
			}

			if config.MDMApple.Enable {
				if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
					return newAppleMDMDEPProfileAssigner(ctx, instanceID, config.MDMApple.DEP.SyncPeriodicity, ds, depStorage, logger, config.Logging.Debug)
				}); err != nil {
					initFatal(err, "failed to register apple_mdm_dep_profile_assigner schedule")
				}
			}

			level.Info(logger).Log("msg", fmt.Sprintf("started cron schedules: %s", strings.Join(cronSchedules.ScheduleNames(), ", ")))

			// StartCollectors starts a goroutine per collector, using ctx to cancel.
			task.StartCollectors(ctx, kitlog.With(logger, "cron", "async_task"))

			// Flush seen hosts every second
			hostsAsyncCfg := config.Osquery.AsyncConfigForTask(configpkg.AsyncTaskHostLastSeen)
			if !hostsAsyncCfg.Enabled {
				go func() {
					for range time.Tick(time.Duration(rand.Intn(10)+1) * time.Second) {
						if err := task.FlushHostsLastSeen(baseCtx, clock.C.Now()); err != nil {
							level.Info(logger).Log(
								"err", err,
								"msg", "failed to update host seen times",
							)
						}
					}
				}()
			}

			fieldKeys := []string{"method", "error"}
			requestCount := kitprometheus.NewCounterFrom(prometheus.CounterOpts{
				Namespace: "api",
				Subsystem: "service",
				Name:      "request_count",
				Help:      "Number of requests received.",
			}, fieldKeys)
			requestLatency := kitprometheus.NewSummaryFrom(prometheus.SummaryOpts{
				Namespace: "api",
				Subsystem: "service",
				Name:      "request_latency_microseconds",
				Help:      "Total duration of requests in microseconds.",
			}, fieldKeys)

			svc = service.NewMetricsService(svc, requestCount, requestLatency)

			httpLogger := kitlog.With(logger, "component", "http")

			limiterStore := &redis.ThrottledStore{
				Pool:      redisPool,
				KeyPrefix: "ratelimit::",
			}

			var apiHandler, frontendHandler http.Handler
			{
				frontendHandler = service.PrometheusMetricsHandler(
					"get_frontend",
					service.ServeFrontend(config.Server.URLPrefix, config.Server.SandboxEnabled, httpLogger),
				)
				apiHandler = service.MakeHandler(svc, config, httpLogger, limiterStore)

				setupRequired, err := svc.SetupRequired(baseCtx)
				if err != nil {
					initFatal(err, "fetching setup requirement")
				}
				// WithSetup will check if first time setup is required
				// By performing the same check inside main, we can make server startups
				// more efficient after the first startup.
				if setupRequired {
					apiHandler = service.WithSetup(svc, logger, apiHandler)
					frontendHandler = service.RedirectLoginToSetup(svc, logger, frontendHandler, config.Server.URLPrefix)
				} else {
					frontendHandler = service.RedirectSetupToLogin(svc, logger, frontendHandler, config.Server.URLPrefix)
				}

			}

			healthCheckers := make(map[string]health.Checker)
			{
				// a list of dependencies which could affect the status of the app if unavailable.
				deps := map[string]interface{}{
					"mysql": ds,
					"redis": resultStore,
				}

				// convert all dependencies to health.Checker if they implement the healthz methods.
				for name, dep := range deps {
					if hc, ok := dep.(health.Checker); ok {
						healthCheckers[name] = hc
					} else {
						initFatal(errors.New(name+" should be a health.Checker"), "initializing health checks")
					}
				}

			}

			// Instantiate a gRPC service to handle launcher requests.
			launcher := launcher.New(svc, logger, grpc.NewServer(), healthCheckers)

			rootMux := http.NewServeMux()
			rootMux.Handle("/healthz", service.PrometheusMetricsHandler("healthz", health.Handler(httpLogger, healthCheckers)))
			rootMux.Handle("/version", service.PrometheusMetricsHandler("version", version.Handler()))
			rootMux.Handle("/assets/", service.PrometheusMetricsHandler("static_assets", service.ServeStaticAssets("/assets/")))

			if config.MDMApple.Enable {
				if err := registerAppleMDMProtocolServices(
					rootMux,
					config.MDMApple.SCEP,
					mdmStorage,
					scepStorage,
					logger,
				); err != nil {
					initFatal(err, "setup mdm apple services")
				}
			}

			if config.Prometheus.BasicAuth.Username != "" && config.Prometheus.BasicAuth.Password != "" {
				metricsHandler := basicAuthHandler(
					config.Prometheus.BasicAuth.Username,
					config.Prometheus.BasicAuth.Password,
					service.PrometheusMetricsHandler("metrics", promhttp.Handler()),
				)
				rootMux.Handle("/metrics", metricsHandler)
			} else {
				level.Info(logger).Log("msg", "metrics endpoint disabled (http basic auth credentials not set)")
			}

			rootMux.Handle("/api/", apiHandler)
			rootMux.Handle("/", frontendHandler)

			debugHandler := &debugMux{
				fleetAuthenticatedHandler: service.MakeDebugHandler(svc, config, logger, eh, ds),
			}
			rootMux.Handle("/debug/", debugHandler)

			if path, ok := os.LookupEnv("FLEET_TEST_PAGE_PATH"); ok {
				// test that we can load this
				_, err := ioutil.ReadFile(path)
				if err != nil {
					initFatal(err, "loading FLEET_TEST_PAGE_PATH")
				}
				rootMux.HandleFunc("/test", func(rw http.ResponseWriter, req *http.Request) {
					testPage, err := ioutil.ReadFile(path)
					if err != nil {
						rw.WriteHeader(http.StatusNotFound)
						return
					}
					rw.Write(testPage)
					rw.WriteHeader(http.StatusOK)
				})
			}

			if debug {
				// Add debug endpoints with a random
				// authorization token
				debugToken, err := server.GenerateRandomText(24)
				if err != nil {
					initFatal(err, "generating debug token")
				}
				debugHandler.tokenAuthenticatedHandler = http.StripPrefix("/debug/", netbug.AuthHandler(debugToken))
				fmt.Printf("*** Debug mode enabled ***\nAccess the debug endpoints at /debug/?token=%s\n", url.QueryEscape(debugToken))
			}

			if len(config.Server.URLPrefix) > 0 {
				prefixMux := http.NewServeMux()
				prefixMux.Handle(config.Server.URLPrefix+"/", http.StripPrefix(config.Server.URLPrefix, rootMux))
				rootMux = prefixMux
			}

			liveQueryRestPeriod := 90 * time.Second // default (see #1798)
			if v := os.Getenv("FLEET_LIVE_QUERY_REST_PERIOD"); v != "" {
				duration, err := time.ParseDuration(v)
				if err != nil {
					level.Error(logger).Log("live_query_rest_period_err", err)
				} else {
					liveQueryRestPeriod = duration
				}
			}

			// The "GET /api/latest/fleet/queries/run" API requires
			// WriteTimeout to be higher than the live query rest period
			// (otherwise the response is not sent back to the client).
			//
			// We add 10s to the live query rest period to allow the writing
			// of the response.
			liveQueryRestPeriod += 10 * time.Second

			// Create the handler based on whether tracing should be there
			var handler http.Handler
			if config.Logging.TracingEnabled && config.Logging.TracingType == "elasticapm" {
				handler = launcher.Handler(apmhttp.Wrap(rootMux))
			} else {
				handler = launcher.Handler(rootMux)
			}

			srv := config.Server.DefaultHTTPServer(ctx, handler)
			if liveQueryRestPeriod > srv.WriteTimeout {
				srv.WriteTimeout = liveQueryRestPeriod
			}
			srv.SetKeepAlivesEnabled(config.Server.Keepalive)
			errs := make(chan error, 2)
			go func() {
				if !config.Server.TLS {
					logger.Log("transport", "http", "address", config.Server.Address, "msg", "listening")
					errs <- srv.ListenAndServe()
				} else {
					logger.Log("transport", "https", "address", config.Server.Address, "msg", "listening")
					srv.TLSConfig = getTLSConfig(config.Server.TLSProfile)
					errs <- srv.ListenAndServeTLS(
						config.Server.Cert,
						config.Server.Key,
					)
				}
			}()
			go func() {
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
				<-sig // block on signal
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				errs <- func() error {
					cancelFunc()
					cleanupCronStatsOnShutdown(ctx, ds, logger, instanceID)
					launcher.GracefulStop()
					return srv.Shutdown(ctx)
				}()
			}()

			logger.Log("terminated", <-errs)
		},
	}

	serveCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug endpoints")
	serveCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")
	serveCmd.PersistentFlags().BoolVar(&devLicense, "dev_license", false, "Enable development license")
	serveCmd.PersistentFlags().BoolVar(&devExpiredLicense, "dev_expired_license", false, "Enable expired development license")

	return serveCmd
}

// basicAuthHandler wraps the given handler behind HTTP Basic Auth.
func basicAuthHandler(username, password string, next http.Handler) http.HandlerFunc {
	hashFn := func(s string) []byte {
		h := sha256.Sum256([]byte(s))
		return h[:]
	}
	expectedUsernameHash := hashFn(username)
	expectedPasswordHash := hashFn(password)

	return func(w http.ResponseWriter, r *http.Request) {
		recvUsername, recvPassword, ok := r.BasicAuth()
		if ok {
			usernameMatch := subtle.ConstantTimeCompare(hashFn(recvUsername), expectedUsernameHash) == 1
			passwordMatch := subtle.ConstantTimeCompare(hashFn(recvPassword), expectedPasswordHash) == 1

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

// Support for TLS security profiles, we set up the TLS configuation based on
// value supplied to server_tls_compatibility command line flag. The default
// profile is 'modern'.
// See https://wiki.mozilla.org/index.php?title=Security/Server_Side_TLS&oldid=1229478
func getTLSConfig(profile string) *tls.Config {
	cfg := tls.Config{
		PreferServerCipherSuites: true,
	}

	switch profile {
	case configpkg.TLSProfileModern:
		cfg.MinVersion = tls.VersionTLS13
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			// These cipher suites not explicitly listed by Mozilla, but
			// required by Go's HTTP/2 implementation
			// See: https://go-review.googlesource.com/c/net/+/200317/
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		)
	case configpkg.TLSProfileIntermediate:
		cfg.MinVersion = tls.VersionTLS12
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		)
	default:
		initFatal(
			fmt.Errorf("%s is invalid", profile),
			"set TLS profile",
		)
	}

	return &cfg
}

// devSQLInterceptor is a sql interceptor to be used for development purposes.
type devSQLInterceptor struct {
	sqlmw.NullInterceptor

	logger kitlog.Logger
}

func (in *devSQLInterceptor) StmtQueryContext(ctx context.Context, stmt driver.StmtQueryContext, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	rows, err := stmt.QueryContext(ctx, args)
	in.logQuery(start, query, args, err)
	return rows, err
}

func (in *devSQLInterceptor) StmtExecContext(ctx context.Context, stmt driver.StmtExecContext, query string, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	result, err := stmt.ExecContext(ctx, args)
	in.logQuery(start, query, args, err)
	return result, err
}

var spaceRegex = regexp.MustCompile(`\s+`)

func (in *devSQLInterceptor) logQuery(start time.Time, query string, args []driver.NamedValue, err error) {
	logLevel := level.Debug
	if err != nil {
		logLevel = level.Error
	}
	query = strings.TrimSpace(spaceRegex.ReplaceAllString(query, " "))
	logLevel(in.logger).Log("duration", time.Since(start), "query", query, "args", argsToString(args), "err", err)
}

func argsToString(args []driver.NamedValue) string {
	var allArgs strings.Builder
	allArgs.WriteString("{")
	for i, arg := range args {
		if i > 0 {
			allArgs.WriteString(", ")
		}
		if arg.Name != "" {
			allArgs.WriteString(fmt.Sprintf("%s=", arg.Name))
		}
		allArgs.WriteString(fmt.Sprintf("%v", arg.Value))
	}
	allArgs.WriteString("}")
	return allArgs.String()
}

// The debugMux directs the request to either the fleet-authenticated handler,
// which is the standard handler for debug endpoints (using a Fleet
// authorization bearer token), or to the token-authenticated handler if a
// query-string token is provided and such a handler is set. The only wayt to
// set this handler is if the --debug flag was provided to the fleet serve
// command.
type debugMux struct {
	fleetAuthenticatedHandler http.Handler
	tokenAuthenticatedHandler http.Handler
}

func (m *debugMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("token") && m.tokenAuthenticatedHandler != nil {
		m.tokenAuthenticatedHandler.ServeHTTP(w, r)
		return
	}
	m.fleetAuthenticatedHandler.ServeHTTP(w, r)
}
