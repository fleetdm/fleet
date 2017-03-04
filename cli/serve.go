package cli

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/mysql"
	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/license"
	"github.com/kolide/kolide/server/mail"
	"github.com/kolide/kolide/server/pubsub"
	"github.com/kolide/kolide/server/service"
	"github.com/kolide/kolide/server/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type initializer interface {
	// Initialize is used to populate a datastore with
	// preloaded data
	Initialize() error
}

func createServeCmd(configManager config.Manager) *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch the kolide server",
		Long: `
Launch the kolide server

Use kolide serve to run the main HTTPS server. The Kolide server bundles
together all static assets and dependent libraries into a statically linked go
binary (which you're executing right now). Use the options below to customize
the way that the kolide server works.
`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			config := configManager.LoadConfig()

			var logger kitlog.Logger
			{
				output := os.Stderr
				if config.Logging.JSON {
					logger = kitlog.NewJSONLogger(output)
				} else {
					logger = kitlog.NewLogfmtLogger(output)
				}
				logger = kitlog.NewContext(logger).With("ts", kitlog.DefaultTimestampUTC)
			}

			var ds kolide.Datastore
			var err error
			mailService := mail.NewService()

			ds, err = mysql.New(config.Mysql, clock.C, mysql.Logger(logger))
			if err != nil {
				initFatal(err, "initializing datastore")
			}

			if initializingDS, ok := ds.(initializer); ok {
				if err := initializingDS.Initialize(); err != nil {
					initFatal(err, "loading built in data")
				}
			}

			licenseService := license.NewChecker(
				ds,
				"https://kolide.co/api/v0/licenses",
				license.Logger(logger),
			)

			err = licenseService.Start()
			if err != nil {
				initFatal(err, "initializing license service")
			}

			var resultStore kolide.QueryResultStore
			redisPool := pubsub.NewRedisPool(config.Redis.Address, config.Redis.Password)
			resultStore = pubsub.NewRedisQueryResults(redisPool)

			svc, err := service.NewService(ds, resultStore, logger, config, mailService, clock.C, licenseService)
			if err != nil {
				initFatal(err, "initializing service")
			}

			go func() {
				ticker := time.NewTicker(1 * time.Hour)
				for {
					ds.CleanupDistributedQueryCampaigns(time.Now())
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

			svcLogger := kitlog.NewContext(logger).With("component", "service")
			svc = service.NewLoggingService(svc, svcLogger)
			svc = service.NewMetricsService(svc, requestCount, requestLatency)

			httpLogger := kitlog.NewContext(logger).With("component", "http")

			var apiHandler, frontendHandler http.Handler
			{
				frontendHandler = prometheus.InstrumentHandler("get_frontend", service.ServeFrontend())
				apiHandler = service.MakeHandler(ctx, svc, config.Auth.JwtKey, httpLogger)

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

			// a list of dependencies which could affect the status of the app if unavailable
			healthCheckers := map[string]interface{}{
				"datastore":          ds,
				"query_result_store": resultStore,
			}

			r := http.NewServeMux()
			r.Handle("/healthz", prometheus.InstrumentHandler("healthz", healthz(httpLogger, healthCheckers)))
			r.Handle("/version", prometheus.InstrumentHandler("version", version.Handler()))
			r.Handle("/assets/", prometheus.InstrumentHandler("static_assets", service.ServeStaticAssets("/assets/")))
			r.Handle("/metrics", prometheus.InstrumentHandler("metrics", promhttp.Handler()))
			r.Handle("/api/", apiHandler)
			r.Handle("/", frontendHandler)

			srv := &http.Server{
				Addr:    config.Server.Address,
				Handler: r,
			}
			errs := make(chan error, 2)
			go func() {
				if !config.Server.TLS {
					logger.Log("transport", "http", "address", config.Server.Address, "msg", "listening")
					errs <- srv.ListenAndServe()
				} else {
					logger.Log("transport", "https", "address", config.Server.Address, "msg", "listening")
					errs <- srv.ListenAndServeTLS(
						config.Server.Cert,
						config.Server.Key,
					)
				}
			}()
			go func() {
				sig := make(chan os.Signal)
				signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
				<-sig //block on signal
				ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)
				errs <- srv.Shutdown(ctx)
			}()

			logger.Log("terminated", <-errs)
			licenseService.Stop()
		},
	}

	return serveCmd
}

// healthz is an http handler which responds with either
// 200 OK if the server can successfuly communicate with it's backends or
// 500 if any of the backends are reporting an issue.
func healthz(logger kitlog.Logger, deps map[string]interface{}) http.HandlerFunc {
	type healthChecker interface {
		HealthCheck() error
	}

	healthy := true
	return func(w http.ResponseWriter, r *http.Request) {
		for name, dep := range deps {
			if hc, ok := dep.(healthChecker); ok {
				err := hc.HealthCheck()
				if err != nil {
					logger.Log("err", err, "health-checker", name)
					healthy = false
				}
			}
		}

		if !healthy {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
