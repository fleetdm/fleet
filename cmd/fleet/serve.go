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
	"syscall"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/e-dard/netbug"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/datastore/mysql"
	"github.com/kolide/fleet/server/health"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/launcher"
	"github.com/kolide/fleet/server/mail"
	"github.com/kolide/fleet/server/pubsub"
	"github.com/kolide/fleet/server/service"
	"github.com/kolide/fleet/server/sso"
	"github.com/kolide/kit/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type initializer interface {
	// Initialize is used to populate a datastore with
	// preloaded data
	Initialize() error
}

func createServeCmd(configManager config.Manager) *cobra.Command {
	// Whether to enable the debug endpoints
	debug := false

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

			var logger kitlog.Logger
			{
				output := os.Stderr
				if config.Logging.JSON {
					logger = kitlog.NewJSONLogger(output)
				} else {
					logger = kitlog.NewLogfmtLogger(output)
				}
				logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
			}

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

			var ds kolide.Datastore
			var err error
			mailService := mail.NewService()

			ds, err = mysql.New(config.Mysql, clock.C, mysql.Logger(logger))
			if err != nil {
				initFatal(err, "initializing datastore")
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

			if config.Auth.JwtKey == "" {
				jwtKey, err := kolide.RandomText(24)
				if err != nil {
					initFatal(err, "generating sample jwt key")
				}
				fmt.Printf("################################################################################\n"+
					"# ERROR:\n"+
					"#   A value must be supplied for --auth_jwt_key. This value is used to create\n"+
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

			var resultStore kolide.QueryResultStore
			redisPool := pubsub.NewRedisPool(config.Redis.Address, config.Redis.Password)
			resultStore = pubsub.NewRedisQueryResults(redisPool)
			ssoSessionStore := sso.NewSessionStore(redisPool)

			svc, err := service.NewService(ds, resultStore, logger, config, mailService, clock.C, ssoSessionStore)
			if err != nil {
				initFatal(err, "initializing service")
			}

			go func() {
				ticker := time.NewTicker(1 * time.Hour)
				for {
					ds.CleanupDistributedQueryCampaigns(time.Now())
					ds.CleanupIncomingHosts(time.Now())
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

			var apiHandler, frontendHandler http.Handler
			{
				frontendHandler = prometheus.InstrumentHandler("get_frontend", service.ServeFrontend(httpLogger))
				apiHandler = service.MakeHandler(svc, config.Auth.JwtKey, httpLogger)

				setupRequired, err := service.RequireSetup(svc)
				if err != nil {
					initFatal(err, "fetching setup requirement")
				}
				// WithSetup will check if first time setup is required
				// By performing the same check inside main, we can make server startups
				// more efficient after the first startup.
				if setupRequired {
					apiHandler = service.WithSetup(svc, logger, apiHandler)
					frontendHandler = service.RedirectLoginToSetup(svc, logger, frontendHandler)
				} else {
					frontendHandler = service.RedirectSetupToLogin(svc, logger, frontendHandler)
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

			r := http.NewServeMux()

			r.Handle("/healthz", prometheus.InstrumentHandler("healthz", health.Handler(httpLogger, healthCheckers)))
			r.Handle("/version", prometheus.InstrumentHandler("version", version.Handler()))
			r.Handle("/assets/", prometheus.InstrumentHandler("static_assets", service.ServeStaticAssets("/assets/")))
			r.Handle("/metrics", prometheus.InstrumentHandler("metrics", promhttp.Handler()))
			r.Handle("/api/", apiHandler)
			r.Handle("/", frontendHandler)

			if path, ok := os.LookupEnv("KOLIDE_TEST_PAGE_PATH"); ok {
				// test that we can load this
				_, err := ioutil.ReadFile(path)
				if err != nil {
					initFatal(err, "loading KOLIDE_TEST_PAGE_PATH")
				}
				r.HandleFunc("/test", func(rw http.ResponseWriter, req *http.Request) {
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
				r.Handle("/debug/", http.StripPrefix("/debug/", netbug.AuthHandler(debugToken)))
				fmt.Printf("*** Debug mode enabled ***\nAccess the debug endpoints at /debug/?token=%s\n", url.QueryEscape(debugToken))
			}

			srv := &http.Server{
				Addr:              config.Server.Address,
				Handler:           launcher.Handler(r),
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

	return serveCmd
}

// Support for TLS security profiles, we set up the TLS configuation based on
// value supplied to server_tls_compatibility command line flag. The default
// profile is 'modern'.
// See https://wiki.mozilla.org/Security/Server_Side_TLS
func getTLSConfig(profile string) *tls.Config {
	cfg := tls.Config{
		PreferServerCipherSuites: true,
	}

	switch profile {
	case config.TLSProfileModern:
		cfg.MinVersion = tls.VersionTLS12
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
			tls.X25519,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		)
	case config.TLSProfileIntermediate:
		cfg.MinVersion = tls.VersionTLS10
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
			tls.X25519,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_RC4_128_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		)
	case config.TLSProfileOld:
		cfg.MinVersion = tls.VersionSSL30
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
			tls.X25519,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_RC4_128_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		)
	default:
		panic("invalid tls profile " + profile)
	}

	return &cfg
}
