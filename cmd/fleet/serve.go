package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/e-dard/netbug"
	"github.com/fleetdm/fleet/v4/server"

	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/ee/server/licensing"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/launcher"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/kit/version"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/throttled/throttled/v2/store/redigostore"
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
	// Whether to enable development Fleet Basic license
	devLicense := false

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
			}

			license, err := licensing.LoadLicense(config.License.Key)
			if err != nil {
				initFatal(
					err,
					"failed to load license - for help use https://fleetdm.com/contact",
				)
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
				initFatal(errors.Errorf("%s is not a valid value for osquery_host_identifier", config.Osquery.HostIdentifier), "set host identifier")
			}

			if len(config.Server.URLPrefix) > 0 {
				// Massage provided prefix to match expected format
				config.Server.URLPrefix = strings.TrimSuffix(config.Server.URLPrefix, "/")
				if len(config.Server.URLPrefix) > 0 && !strings.HasPrefix(config.Server.URLPrefix, "/") {
					config.Server.URLPrefix = "/" + config.Server.URLPrefix
				}

				if !allowedURLPrefixRegexp.MatchString(config.Server.URLPrefix) {
					initFatal(
						errors.Errorf("prefix must match regexp \"%s\"", allowedURLPrefixRegexp.String()),
						"setting server URL prefix",
					)
				}
			}

			var ds fleet.Datastore
			var carveStore fleet.CarveStore
			mailService := mail.NewService()

			ds, err = mysql.New(config.Mysql, clock.C, mysql.Logger(logger))
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

			migrationStatus, err := ds.MigrationStatus()
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			switch migrationStatus {
			case fleet.SomeMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database is missing required migrations. This is likely to cause\n"+
					"#   errors in Fleet.\n"+
					"#\n"+
					"#   Run `%s prepare db` to perform migrations.\n"+
					"################################################################################\n",
					os.Args[0])

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

			redisPool, err := pubsub.NewRedisPool(config.Redis.Address, config.Redis.Password, config.Redis.Database, config.Redis.UseTLS)
			if err != nil {
				initFatal(err, "initialize Redis")
			}
			resultStore := pubsub.NewRedisQueryResults(redisPool, config.Redis.DuplicateResults)
			liveQueryStore := live_query.NewRedisLiveQuery(redisPool)
			ssoSessionStore := sso.NewSessionStore(redisPool)

			osqueryLogger, err := logging.New(config, logger)
			if err != nil {
				initFatal(err, "initializing osquery logging")
			}

			svc, err := service.NewService(ds, resultStore, logger, osqueryLogger, config, mailService, clock.C, ssoSessionStore, liveQueryStore, carveStore, *license)
			if err != nil {
				initFatal(err, "initializing service")
			}

			if license.Tier == fleet.TierBasic {
				svc, err = eeservice.NewService(svc, ds, logger, config, mailService, clock.C, license)
				if err != nil {
					initFatal(err, "initial Fleet Basic service")
				}
			}

			cancelBackground := runCrons(ds, kitlog.With(logger, "component", "crons"), config)

			// Flush seen hosts every second
			go func() {
				ticker := time.NewTicker(1 * time.Second)
				for {
					if err := svc.FlushSeenHosts(context.Background()); err != nil {
						level.Info(logger).Log(
							"err", err,
							"msg", "failed to update host seen times",
						)
					}
					<-ticker.C
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

			limiterStore, err := redigostore.New(redisPool, "ratelimit::", 0)
			if err != nil {
				initFatal(err, "initialize rate limit store")
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

			rootMux := http.NewServeMux()
			rootMux.Handle("/healthz", prometheus.InstrumentHandler("healthz", health.Handler(httpLogger, healthCheckers)))
			rootMux.Handle("/version", prometheus.InstrumentHandler("version", version.Handler()))
			rootMux.Handle("/assets/", prometheus.InstrumentHandler("static_assets", service.ServeStaticAssets("/assets/")))
			rootMux.Handle("/metrics", prometheus.InstrumentHandler("metrics", promhttp.Handler()))
			rootMux.Handle("/api/", apiHandler)
			rootMux.Handle("/", frontendHandler)
			rootMux.Handle("/debug/", service.MakeDebugHandler(svc, config, logger))

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

			srv := &http.Server{
				Addr:              config.Server.Address,
				Handler:           launcher.Handler(rootMux),
				ReadTimeout:       25 * time.Second,
				WriteTimeout:      40 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
				IdleTimeout:       5 * time.Minute,
				MaxHeaderBytes:    1 << 18, // 0.25 MB (262144 bytes)
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
				<-sig //block on signal
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				errs <- func() error {
					cancelBackground()
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

	return serveCmd
}

// Locker represents an object that can obtain an atomic lock on a resource
// in a non blocking manner for an owner, with an expiration time.
type Locker interface {
	// Lock tries to get an atomic lock on an instance named with `name`
	// and an `owner` identified by a random string per instance.
	// Subsequently locking the same resource name for the same owner
	// renews the lock expiration.
	// It returns true, nil if it managed to obtain a lock on the instance.
	// false and potentially an error otherwise.
	// This must not be blocking.
	Lock(name string, owner string, expiration time.Duration) (bool, error)
	// Unlock tries to unlock the lock by that `name` for the specified
	// `owner`. Unlocking when not holding the lock shouldn't error
	Unlock(name string, owner string) error
}

const (
	lockKeyLeader          = "leader"
	lockKeyVulnerabilities = "vulnerabilities"
)

func trySendStatistics(ds fleet.Datastore, frequency time.Duration, url string) error {
	ac, err := ds.AppConfig()
	if err != nil {
		return err
	}
	if !ac.EnableAnalytics {
		return nil
	}

	stats, shouldSend, err := ds.ShouldSendStatistics(frequency)
	if err != nil {
		return err
	}
	if !shouldSend {
		return nil
	}

	statsBytes, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(statsBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Error posting to %s: %d", url, resp.StatusCode)
	}
	return ds.RecordStatisticsSent()
}

func runCrons(ds fleet.Datastore, logger kitlog.Logger, config config.FleetConfig) context.CancelFunc {
	locker, ok := ds.(Locker)
	if !ok {
		initFatal(errors.New("No global locker available"), "")
	}
	ctx, cancelBackground := context.WithCancel(context.Background())

	ourIdentifier, err := server.GenerateRandomText(64)
	if err != nil {
		initFatal(errors.New("Error generating random instance identifier"), "")
	}

	go cronCleanups(ctx, ds, kitlog.With(logger, "cron", "cleanups"), locker, ourIdentifier)
	go cronVulnerabilities(
		ctx, ds, kitlog.With(logger, "cron", "vulnerabilities"), locker, ourIdentifier, config)

	return cancelBackground
}

func cronCleanups(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, locker Locker, identifier string) {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			break
		}
		if locked, err := locker.Lock(lockKeyLeader, identifier, time.Hour); err != nil || !locked {
			level.Debug(logger).Log("leader", "Not the leader. Skipping...")
			continue
		}
		_, err := ds.CleanupDistributedQueryCampaigns(time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning distributed query campaigns", "details", err)
		}
		err = ds.CleanupIncomingHosts(time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning incoming hosts", "details", err)
		}
		_, err = ds.CleanupCarves(time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning carves", "details", err)
		}
		err = ds.CleanupOrphanScheduledQueryStats()
		if err != nil {
			level.Error(logger).Log("err", "cleaning scheduled query stats", "details", err)
		}

		err = trySendStatistics(ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics")
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
	locker Locker,
	identifier string,
	config config.FleetConfig,
) {
	if config.Vulnerabilities.CurrentInstanceChecks == "no" || config.Vulnerabilities.CurrentInstanceChecks == "0" {
		level.Info(logger).Log("vulnerability scanning", "host not configured to check for vulnerabilities")
		return
	}

	appConfig, err := ds.AppConfig()
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return
	}
	if ptr.StringValueOrZero(appConfig.VulnerabilityDatabasesPath) == "" &&
		config.Vulnerabilities.DatabasesPath == "" {
		level.Info(logger).Log("vulnerability scanning", "not configured")
		return
	}

	vulnPath := ptr.StringValueOrZero(appConfig.VulnerabilityDatabasesPath)
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

	ticker := time.NewTicker(10 * time.Second)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(config.Vulnerabilities.Periodicity)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			break
		}
		if config.Vulnerabilities.CurrentInstanceChecks == "auto" {
			if locked, err := locker.Lock(lockKeyVulnerabilities, identifier, time.Hour); err != nil || !locked {
				level.Debug(logger).Log("leader", "Not the leader. Skipping...")
				continue
			}
		}

		err := vulnerabilities.TranslateSoftwareToCPE(ds, vulnPath, logger, config.Vulnerabilities.CPEDatabaseURL)
		if err != nil {
			level.Error(logger).Log("msg", "analyzing vulnerable software: Software->CPE", "err", err)
			continue
		}

		err = vulnerabilities.TranslateCPEToCVE(ctx, ds, vulnPath, logger, config.Vulnerabilities.CVEFeedPrefixURL)
		if err != nil {
			level.Error(logger).Log("msg", "analyzing vulnerable software: CPE->CVE", "err", err)
			continue
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
			errors.Errorf("%s is invalid", profile),
			"set TLS profile",
		)
	}

	return &cfg
}
