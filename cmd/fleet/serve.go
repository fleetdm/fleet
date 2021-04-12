package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
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
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/mysql"
	"github.com/fleetdm/fleet/server/datastore/s3"
	"github.com/fleetdm/fleet/server/health"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/launcher"
	"github.com/fleetdm/fleet/server/live_query"
	"github.com/fleetdm/fleet/server/mail"
	"github.com/fleetdm/fleet/server/pubsub"
	"github.com/fleetdm/fleet/server/service"
	"github.com/fleetdm/fleet/server/sso"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/kit/version"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/throttled/throttled/store/redigostore"
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

			// Check for deprecated config options.
			if config.Osquery.StatusLogFile != "" {
				level.Info(logger).Log(
					"DEPRECATED", "use filesystem.status_log_file.",
					"msg", "using osquery.status_log_file value for filesystem.status_log_file",
				)
				config.Filesystem.StatusLogFile = config.Osquery.StatusLogFile
			}
			if config.Osquery.ResultLogFile != "" {
				level.Info(logger).Log(
					"DEPRECATED", "use filesystem.result_log_file.",
					"msg", "using osquery.result_log_file value for filesystem.result_log_file",
				)
				config.Filesystem.ResultLogFile = config.Osquery.ResultLogFile
			}
			if config.Osquery.EnableLogRotation != false {
				level.Info(logger).Log(
					"DEPRECATED", "use filesystem.enable_log_rotation.",
					"msg", "using osquery.enable_log_rotation value for filesystem.result_log_file",
				)
				config.Filesystem.EnableLogRotation = config.Osquery.EnableLogRotation
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

			var ds kolide.Datastore
			var carveStore kolide.CarveStore
			var err error
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
			case kolide.SomeMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database is missing required migrations. This is likely to cause\n"+
					"#   errors in Fleet.\n"+
					"#\n"+
					"#   Run `%s prepare db` to perform migrations.\n"+
					"################################################################################\n",
					os.Args[0])

			case kolide.NoMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# ERROR:\n"+
					"#   Your Fleet database is not initialized. Fleet cannot start up.\n"+
					"#\n"+
					"#   Run `%s prepare db` to initialize the database.\n"+
					"################################################################################\n",
					os.Args[0])
				os.Exit(1)
			}

			if config.Auth.JwtKey != "" && config.Auth.JwtKeyPath != "" {
				initFatal(err, "A JWT key and a JWT key file were provided - please specify only one")
			}

			if config.Auth.JwtKeyPath != "" {
				fileContents, err := ioutil.ReadFile(config.Auth.JwtKeyPath)
				if err != nil {
					initFatal(err, "Could not read the JWT Key file provided")
				}
				config.Auth.JwtKey = strings.TrimSpace(string(fileContents))
			}

			if config.Auth.JwtKey == "" && config.Auth.JwtKeyPath == "" {
				jwtKey, err := kolide.RandomText(24)
				if err != nil {
					initFatal(err, "generating sample jwt key")
				}
				fmt.Printf("################################################################################\n"+
					"# ERROR:\n"+
					"#   A value must be supplied for --auth_jwt_key or --auth_jwt_key_path. This value is used to create\n"+
					"#   session tokens for users.\n"+
					"#\n"+
					"#   Consider using the following randomly generated key:\n"+
					"#   %s\n"+
					"################################################################################\n",
					jwtKey)
				os.Exit(1)
			}

			if initializingDS, ok := ds.(initializer); ok {
				if err := initializingDS.Initialize(); err != nil {
					initFatal(err, "loading built in data")
				}
			}

			redisPool := pubsub.NewRedisPool(config.Redis.Address, config.Redis.Password, config.Redis.Database, config.Redis.UseTLS)
			resultStore := pubsub.NewRedisQueryResults(redisPool)
			liveQueryStore := live_query.NewRedisLiveQuery(redisPool)
			ssoSessionStore := sso.NewSessionStore(redisPool)

			svc, err := service.NewService(ds, resultStore, logger, config, mailService, clock.C, ssoSessionStore, liveQueryStore, carveStore)
			if err != nil {
				initFatal(err, "initializing service")
			}

			go func() {
				ticker := time.NewTicker(1 * time.Hour)
				for {
					ds.CleanupDistributedQueryCampaigns(time.Now())
					ds.CleanupIncomingHosts(time.Now())
					ds.CleanupCarves(time.Now())
					<-ticker.C
				}
			}()

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

			svcLogger := kitlog.With(logger, "component", "service")

			svc = service.NewLoggingService(svc, svcLogger)
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

				setupRequired, err := service.RequireSetup(svc)
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
				debugToken, err := kolide.RandomText(24)
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
					launcher.GracefulStop()
					return srv.Shutdown(ctx)
				}()
			}()

			logger.Log("terminated", <-errs)
		},
	}

	serveCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug endpoints")
	serveCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	return serveCmd
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
