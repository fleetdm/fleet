package cli

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/mail"
	"github.com/kolide/kolide-ose/server/service"
	"github.com/kolide/kolide-ose/server/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func createServeCmd(configManager config.Manager) *cobra.Command {
	var devMode bool = false

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
			var (
				httpAddr = flag.String("http.addr", ":8080", "HTTP listen address")
				ctx      = context.Background()
				logger   kitlog.Logger
			)
			flag.Parse()

			config := configManager.LoadConfig()

			logger = kitlog.NewLogfmtLogger(os.Stderr)
			logger = kitlog.NewContext(logger).With("ts", kitlog.DefaultTimestampUTC)

			var mailService kolide.MailService
			if devMode {
				mailService = devMailService{}
			} else {
				mailService = mail.NewService(config.SMTP)
			}

			var ds kolide.Datastore
			var err error
			if devMode {
				fmt.Println(
					"Dev mode enabled, using in-memory DB.\n",
					"Warning: Changes will not be saved across process restarts. This should NOT be used in production.",
				)
				ds, err = datastore.New("inmem", "")
				if err != nil {
					initFatal(err, "initializing datastore")
				}
			} else {
				connString := datastore.GetMysqlConnectionString(config.Mysql)
				ds, err = datastore.New("gorm-mysql", connString)
				if err != nil {
					initFatal(err, "initializing datastore")
				}
			}

			svc, err := service.NewService(ds, logger, config, mailService, clock.C)
			if err != nil {
				initFatal(err, "initializing service")
			}

			if devMode {
				// Bootstrap a few users when using the in-memory database.
				// Each user's default password will just be their username.
				users := []kolide.User{
					{
						Name:     "Admin User",
						Username: "admin",
						Email:    "admin@kolide.co",
						Position: "Director of Security",
						Admin:    true,
						Enabled:  true,
					},
					{
						Name:     "Normal User",
						Username: "user",
						Email:    "user@kolide.co",
						Position: "Security Engineer",
						Admin:    false,
						Enabled:  true,
					},
				}

				for _, user := range users {
					user := user
					err := user.SetPassword(user.Username, config.Auth.SaltKeySize, config.Auth.BcryptCost)
					if err != nil {
						initFatal(err, "creating bootstrap user")
					}
					_, err = ds.NewUser(&user)
					if err != nil {
						initFatal(err, "creating bootstrap user")
					}
				}
				devOrgInfo := &kolide.OrgInfo{
					OrgName:    "Kolide",
					OrgLogoURL: fmt.Sprintf("%s/logo.png", config.Server.Address),
				}
				_, err := svc.NewOrgInfo(ctx, kolide.OrgInfoPayload{
					OrgName:    &devOrgInfo.OrgName,
					OrgLogoURL: &devOrgInfo.OrgLogoURL,
				})
				if err != nil {
					initFatal(err, "creating fake org info")
				}
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

			svcLogger := kitlog.NewContext(logger).With("component", "service")
			svc = service.NewLoggingService(svc, svcLogger)
			svc = service.NewMetricsService(svc, requestCount, requestLatency)

			httpLogger := kitlog.NewContext(logger).With("component", "http")

			apiHandler := service.MakeHandler(ctx, svc, config.Auth.JwtKey, httpLogger)
			http.Handle("/api/", accessControl(apiHandler))
			http.Handle("/version", version.Handler())
			http.Handle("/metrics", prometheus.Handler())
			http.Handle("/assets/", service.ServeStaticAssets("/assets/"))
			http.Handle("/", service.ServeFrontend())

			errs := make(chan error, 2)
			go func() {
				if !config.Server.TLS || (devMode && !configManager.IsSet("server.tls")) {
					logger.Log("transport", "http", "address", *httpAddr, "msg", "listening")
					errs <- http.ListenAndServe(*httpAddr, nil)
				} else {
					logger.Log("transport", "https", "address", *httpAddr, "msg", "listening")
					errs <- http.ListenAndServeTLS(
						*httpAddr,
						config.Server.Cert,
						config.Server.Key,
						nil,
					)
				}
			}()
			go func() {
				c := make(chan os.Signal)
				signal.Notify(c, syscall.SIGINT)
				errs <- fmt.Errorf("%s", <-c)
			}()

			logger.Log("terminated", <-errs)
		},
	}

	serveCmd.PersistentFlags().BoolVar(&devMode, "dev", false, "Use dev settings (in-mem DB, etc.)")

	return serveCmd
}

// used in devMode to print an email
// which would otherwise be sent via SMTP
type devMailService struct{}

func (devMailService) SendEmail(e kolide.Email) error {
	fmt.Println("---dev mode: printing email---")
	defer fmt.Println("---dev mode: email printed---")
	msg, err := e.Msg.Message()
	if err != nil {
		return err
	}
	fmt.Printf("From: %q To: %q \n", e.From, e.To)
	_, err = os.Stdout.Write(msg)
	return err

}

// cors headers
func accessControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
