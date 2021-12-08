package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
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
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/launcher"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/kit/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var allowedURLPrefixRegexp = regexp.MustCompile("^(?:/[a-zA-Z0-9_.~-]+)+$")

type initializer interface {
	// Initialize is used to populate a datastore with
	// preloaded data
	Initialize() error
}

func createServeCmd(configManager config.Manager) *cobra.Command {
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
				config.License.Key = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjQwOTk1MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjIyNDI2NTg2fQ.WmZ0kG4seW3IrNvULCHUPBSfFdqj38A_eiXdV_DFunMHechjHbkwtfkf1J6JQJoDyqn8raXpgbdhafDwv3rmDw"
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
			mailService := mail.NewService()

			var replicaOpt mysql.DBOption
			if config.MysqlReadReplica.Address != "" {
				replicaOpt = mysql.Replica(&config.MysqlReadReplica)
			}
			ds, err = mysql.New(config.Mysql, clock.C, mysql.Logger(logger), replicaOpt)
			if err != nil {
				initFatal(err, "initializing datastore")
			}

			if config.S3.Bucket != "" {
				carveStore, err = s3.New(config.S3, ds)
				if err != nil {
					initFatal(err, "initializing S3 carvestore")
				}
			} else {
				carveStore = ds
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
			})
			if err != nil {
				initFatal(err, "initialize Redis")
			}
			level.Info(logger).Log("component", "redis", "mode", redisPool.Mode())

			ds = cached_mysql.New(ds)
			resultStore := pubsub.NewRedisQueryResults(redisPool, config.Redis.DuplicateResults)
			liveQueryStore := live_query.NewRedisLiveQuery(redisPool)
			if err := liveQueryStore.MigrateKeys(); err != nil {
				level.Info(logger).Log(
					"err", err,
					"msg", "failed to migrate live query redis keys",
				)
			}
			ssoSessionStore := sso.NewSessionStore(redisPool)

			osqueryLogger, err := logging.New(config, logger)
			if err != nil {
				initFatal(err, "initializing osquery logging")
			}

			task := &async.Task{
				Datastore:          ds,
				Pool:               redisPool,
				AsyncEnabled:       config.Osquery.EnableAsyncHostProcessing,
				LockTimeout:        config.Osquery.AsyncHostCollectLockTimeout,
				LogStatsInterval:   config.Osquery.AsyncHostCollectLogStatsInterval,
				InsertBatch:        config.Osquery.AsyncHostInsertBatch,
				DeleteBatch:        config.Osquery.AsyncHostDeleteBatch,
				UpdateBatch:        config.Osquery.AsyncHostUpdateBatch,
				RedisPopCount:      config.Osquery.AsyncHostRedisPopCount,
				RedisScanKeysCount: config.Osquery.AsyncHostRedisScanKeysCount,
			}
			svc, err := service.NewService(ds, task, resultStore, logger, osqueryLogger, config, mailService, clock.C, ssoSessionStore, liveQueryStore, carveStore, *license)
			if err != nil {
				initFatal(err, "initializing service")
			}

			if license.IsPremium() {
				svc, err = eeservice.NewService(svc, ds, logger, config, mailService, clock.C, license)
				if err != nil {
					initFatal(err, "initial Fleet Premium service")
				}
			}

			cancelBackground := runCrons(ds, task, kitlog.With(logger, "component", "crons"), config, license)

			// Flush seen hosts every second
			go func() {
				for range time.Tick(time.Duration(rand.Intn(10)+1) * time.Second) {
					if err := svc.FlushSeenHosts(context.Background()); err != nil {
						level.Info(logger).Log(
							"err", err,
							"msg", "failed to update host seen times",
						)
					}
				}
			}()

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
				frontendHandler = prometheus.InstrumentHandler("get_frontend", service.ServeFrontend(config.Server.URLPrefix, httpLogger))
				apiHandler = service.MakeHandler(svc, config, httpLogger, limiterStore)

				setupRequired, err := svc.SetupRequired(context.Background())
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
					"datastore":          ds,
					"query_result_store": resultStore,
				}

				// convert all dependencies to health.Checker if they implement the healthz methods.
				for name, dep := range deps {
					if hc, ok := dep.(health.Checker); ok {
						healthCheckers[name] = hc
					}
				}

			}

			// Instantiate a gRPC service to handle launcher requests.
			launcher := launcher.New(svc, logger, grpc.NewServer(), healthCheckers)

			// TODO: gather all the different contexts and use just one
			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()
			eh := errorstore.NewHandler(ctx, redisPool, logger, config.Logging.ErrorRetentionPeriod)

			rootMux := http.NewServeMux()
			rootMux.Handle("/healthz", prometheus.InstrumentHandler("healthz", health.Handler(httpLogger, healthCheckers)))
			rootMux.Handle("/version", prometheus.InstrumentHandler("version", version.Handler()))
			rootMux.Handle("/assets/", prometheus.InstrumentHandler("static_assets", service.ServeStaticAssets("/assets/")))
			rootMux.Handle("/metrics", prometheus.InstrumentHandler("metrics", promhttp.Handler()))
			rootMux.Handle("/api/", apiHandler)
			rootMux.Handle("/", frontendHandler)
			rootMux.Handle("/debug/", service.MakeDebugHandler(svc, config, logger, eh, ds))

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
				rootMux.Handle("/debug/", http.StripPrefix("/debug/", netbug.AuthHandler(debugToken)))
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

			defaultWritetimeout := 40 * time.Second
			writeTimeout := defaultWritetimeout
			// The "GET /api/v1/fleet/queries/run" API requires
			// WriteTimeout to be higher than the live query rest period
			// (otherwise the response is not sent back to the client).
			//
			// We add 10s to the live query rest period to allow the writing
			// of the response.
			liveQueryRestPeriod += 10 * time.Second
			if liveQueryRestPeriod > writeTimeout {
				writeTimeout = liveQueryRestPeriod
			}

			httpSrvCtx := ctxerr.NewContext(ctx, eh)
			srv := &http.Server{
				Addr:              config.Server.Address,
				Handler:           launcher.Handler(rootMux),
				ReadTimeout:       25 * time.Second,
				WriteTimeout:      writeTimeout,
				ReadHeaderTimeout: 5 * time.Second,
				IdleTimeout:       5 * time.Minute,
				MaxHeaderBytes:    1 << 18, // 0.25 MB (262144 bytes)
				BaseContext: func(l net.Listener) context.Context {
					return httpSrvCtx
				},
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
					cancelBackground()
					cancelFunc()
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

const (
	lockKeyLeader          = "leader"
	lockKeyVulnerabilities = "vulnerabilities"
	lockKeyWebhooks        = "webhooks"
)

func trySendStatistics(ctx context.Context, ds fleet.Datastore, frequency time.Duration, url string, license *fleet.LicenseInfo) error {
	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.ServerSettings.EnableAnalytics {
		return nil
	}

	stats, shouldSend, err := ds.ShouldSendStatistics(ctx, frequency, license)
	if err != nil {
		return err
	}
	if !shouldSend {
		return nil
	}

	err = server.PostJSONWithTimeout(ctx, url, stats)
	if err != nil {
		return err
	}
	return ds.RecordStatisticsSent(ctx)
}

func runCrons(ds fleet.Datastore, task *async.Task, logger kitlog.Logger, config config.FleetConfig, license *fleet.LicenseInfo) context.CancelFunc {
	ctx, cancelBackground := context.WithCancel(context.Background())

	ourIdentifier, err := server.GenerateRandomText(64)
	if err != nil {
		initFatal(errors.New("Error generating random instance identifier"), "")
	}

	// StartCollectors starts a goroutine per collector, using ctx to cancel.
	task.StartCollectors(ctx, config.Osquery.AsyncHostCollectInterval,
		config.Osquery.AsyncHostCollectMaxJitterPercent, kitlog.With(logger, "cron", "async_task"))

	go cronCleanups(ctx, ds, kitlog.With(logger, "cron", "cleanups"), ourIdentifier, license)
	go cronVulnerabilities(
		ctx, ds, kitlog.With(logger, "cron", "vulnerabilities"), ourIdentifier, config)
	go cronWebhooks(ctx, ds, kitlog.With(logger, "cron", "webhooks"), ourIdentifier)

	return cancelBackground
}

func cronCleanups(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, identifier string, license *fleet.LicenseInfo) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(1 * time.Hour)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}
		if locked, err := ds.Lock(ctx, lockKeyLeader, identifier, time.Hour); err != nil || !locked {
			level.Debug(logger).Log("leader", "Not the leader. Skipping...")
			continue
		}
		_, err := ds.CleanupDistributedQueryCampaigns(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning distributed query campaigns", "details", err)
		}
		err = ds.CleanupIncomingHosts(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning incoming hosts", "details", err)
		}
		_, err = ds.CleanupCarves(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning carves", "details", err)
		}
		err = ds.CleanupOrphanScheduledQueryStats(ctx)
		if err != nil {
			level.Error(logger).Log("err", "cleaning scheduled query stats", "details", err)
		}
		err = ds.CleanupOrphanLabelMembership(ctx)
		if err != nil {
			level.Error(logger).Log("err", "cleaning label_membership", "details", err)
		}
		err = ds.UpdateQueryAggregatedStats(ctx)
		if err != nil {
			level.Error(logger).Log("err", "aggregating query stats", "details", err)
		}
		err = ds.UpdateScheduledQueryAggregatedStats(ctx)
		if err != nil {
			level.Error(logger).Log("err", "aggregating scheduled query stats", "details", err)
		}
		err = ds.CleanupExpiredHosts(ctx)
		if err != nil {
			level.Error(logger).Log("err", "cleaning expired hosts", "details", err)
		}

		err = trySendStatistics(ctx, ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics", license)
		if err != nil {
			level.Error(logger).Log("err", "sending statistics", "details", err)
		}
		level.Debug(logger).Log("loop", "done")
	}
}

func cronVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
	config config.FleetConfig,
) {
	if config.Vulnerabilities.CurrentInstanceChecks == "no" || config.Vulnerabilities.CurrentInstanceChecks == "0" {
		level.Info(logger).Log("vulnerability scanning", "host not configured to check for vulnerabilities")
		return
	}

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return
	}
	if appConfig.VulnerabilitySettings.DatabasesPath == "" &&
		config.Vulnerabilities.DatabasesPath == "" {
		level.Info(logger).Log("vulnerability scanning", "not configured")
		return
	}
	if !appConfig.HostSettings.EnableSoftwareInventory {
		level.Info(logger).Log("software inventory", "not configured")
		return
	}

	vulnPath := appConfig.VulnerabilitySettings.DatabasesPath
	if vulnPath == "" {
		vulnPath = config.Vulnerabilities.DatabasesPath
	}
	if config.Vulnerabilities.DatabasesPath != "" && config.Vulnerabilities.DatabasesPath != vulnPath {
		vulnPath = config.Vulnerabilities.DatabasesPath
		level.Info(logger).Log(
			"databases_path", "fleet config takes precedence over app config when both are configured",
			"result", vulnPath)
	}

	level.Info(logger).Log("databases-path", vulnPath)
	level.Info(logger).Log("periodicity", config.Vulnerabilities.Periodicity)

	if config.Vulnerabilities.CurrentInstanceChecks == "auto" {
		level.Debug(logger).Log("current instance checks", "auto", "trying to create databases-path", vulnPath)
		err := os.MkdirAll(vulnPath, 0o755)
		if err != nil {
			level.Error(logger).Log("databases-path", "creation failed, returning", "err", err)
			return
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(config.Vulnerabilities.Periodicity)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}
		if config.Vulnerabilities.CurrentInstanceChecks == "auto" {
			if locked, err := ds.Lock(ctx, lockKeyVulnerabilities, identifier, time.Hour); err != nil || !locked {
				level.Debug(logger).Log("leader", "Not the leader. Skipping...")
				continue
			}
		}

		err := vulnerabilities.TranslateSoftwareToCPE(ctx, ds, vulnPath, logger, config)
		if err != nil {
			level.Error(logger).Log("msg", "analyzing vulnerable software: Software->CPE", "err", err)
			continue
		}

		err = vulnerabilities.TranslateCPEToCVE(ctx, ds, vulnPath, logger, config)
		if err != nil {
			level.Error(logger).Log("msg", "analyzing vulnerable software: CPE->CVE", "err", err)
			continue
		}

		level.Debug(logger).Log("loop", "done")
	}
}

func cronWebhooks(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, identifier string) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return
	}

	interval := appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour)
	level.Debug(logger).Log("interval", interval.String())
	ticker := time.NewTicker(interval)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}
		if locked, err := ds.Lock(ctx, lockKeyWebhooks, identifier, interval); err != nil || !locked {
			level.Debug(logger).Log("leader", "Not the leader. Skipping...")
			continue
		}

		err := webhooks.TriggerHostStatusWebhook(
			ctx, ds, kitlog.With(logger, "webhook", "host_status"), appConfig)
		if err != nil {
			level.Error(logger).Log("err", "triggering host status webhook", "details", err)
		}

		// Reread app config to be able to change interval somewhat on the fly
		appConfig, err = ds.AppConfig(ctx)
		if err != nil {
			level.Error(logger).Log("config", "couldn't read app config", "err", err)
		} else {
			interval = appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour)
			ticker.Reset(interval)
		}

		level.Debug(logger).Log("loop", "done")
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
	case config.TLSProfileModern:
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
	case config.TLSProfileIntermediate:
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
