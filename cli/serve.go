package cli

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
				mailService = createDevMailService(config)
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
				var dbOption []datastore.DBOption
				gormLogger := log.New(os.Stderr, "", 0)
				gormLogger.SetOutput(kitlog.NewStdlibAdapter(logger))
				dbOption = append(dbOption, datastore.Logger(gormLogger))
				if config.Logging.Debug {
					dbOption = append(dbOption, datastore.Debug())
				}
				connString := datastore.GetMysqlConnectionString(config.Mysql)
				ds, err = datastore.New("gorm-mysql", connString, dbOption...)
				if err != nil {
					initFatal(err, "initializing datastore")
				}
			}

			svc, err := service.NewService(ds, logger, config, mailService, clock.C)
			if err != nil {
				initFatal(err, "initializing service")
			}

			if devMode {
				createDevUsers(ds, config)
				createDevHosts(ds, config)
				createDevQueries(ds, config)
				createDevLabels(ds, config)
				createDevOrgInfo(svc, config)
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
			http.Handle("/api/", apiHandler)
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

// Creates the mail service to be used with the --dev flag.
// If the user provides SMTP settings, then an actual MailService will be returned,
// otherwise return the mock devMailService which prints the contents of an email to stdout.
func createDevMailService(config config.KolideConfig) kolide.MailService {
	smtp := config.SMTP
	if smtp.Server != "" &&
		smtp.Username != "" &&
		smtp.Password != "" {
		return mail.NewService(config.SMTP)
	}
	return devMailService{}
}

// Bootstrap a few users when using the in-memory database.
// Each user's default password will just be their username.
func createDevUsers(ds kolide.Datastore, config config.KolideConfig) {
	users := []kolide.User{
		{
			CreatedAt: time.Date(2016, time.October, 27, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2016, time.October, 27, 10, 0, 0, 0, time.UTC),
			Name:      "Admin User",
			Username:  "admin",
			Email:     "admin@kolide.co",
			Position:  "Director of Security",
			Admin:     true,
			Enabled:   true,
		},
		{
			CreatedAt: time.Now().Add(-3 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
			Name:      "Normal User",
			Username:  "user",
			Email:     "user@kolide.co",
			Position:  "Security Engineer",
			Admin:     false,
			Enabled:   true,
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
}

// Bootstrap a few hosts when using the in-memory database.
func createDevHosts(ds kolide.Datastore, config config.KolideConfig) {
	hosts := []kolide.Host{
		{
			CreatedAt:        time.Date(2016, time.October, 27, 10, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Now().Add(-20 * time.Minute),
			NodeKey:          "totally-legit",
			HostName:         "jmeller-mbp.local",
			UUID:             "1234-5678-9101",
			Platform:         "darwin",
			OsqueryVersion:   "2.0.0",
			OSVersion:        "Mac OS X 10.11.6",
			Uptime:           60 * time.Minute,
			PhysicalMemory:   4145483776,
			PrimaryMAC:       "C0:11:1B:13:3E:15",
			PrimaryIP:        "192.168.1.10",
			DetailUpdateTime: time.Now().Add(-20 * time.Minute),
		},
		{
			CreatedAt:        time.Now().Add(-1 * time.Hour),
			UpdatedAt:        time.Now().Add(-20 * time.Minute),
			NodeKey:          "definitely-legit",
			HostName:         "marpaia.local",
			UUID:             "1234-5678-9102",
			Platform:         "windows",
			OsqueryVersion:   "2.0.0",
			OSVersion:        "Windows 10.0.0",
			Uptime:           60 * time.Minute,
			PhysicalMemory:   17179869184,
			PrimaryMAC:       "7e:5c:be:ef:b4:df",
			PrimaryIP:        "192.168.1.11",
			DetailUpdateTime: time.Now().Add(-10 * time.Second),
		},
	}

	for _, host := range hosts {
		host := host
		_, err := ds.NewHost(&host)
		if err != nil {
			initFatal(err, "creating bootstrap host")
		}
	}
}

func createDevOrgInfo(svc kolide.Service, config config.KolideConfig) {
	devOrgInfo := &kolide.OrgInfo{
		OrgName:    "Kolide",
		OrgLogoURL: fmt.Sprintf("%s/logo.png", config.Server.Address),
	}
	_, err := svc.NewOrgInfo(context.Background(), kolide.OrgInfoPayload{
		OrgName:    &devOrgInfo.OrgName,
		OrgLogoURL: &devOrgInfo.OrgLogoURL,
	})
	if err != nil {
		initFatal(err, "creating fake org info")
	}
}

func createDevQueries(ds kolide.Datastore, config config.KolideConfig) {
	queries := []kolide.Query{
		{
			CreatedAt: time.Date(2016, time.October, 17, 7, 6, 0, 0, time.UTC),
			UpdatedAt: time.Date(2016, time.October, 17, 7, 6, 0, 0, time.UTC),
			Name:      "dev_query_1",
			Query:     "select * from processes",
		},
		{
			CreatedAt: time.Date(2016, time.October, 27, 4, 3, 10, 0, time.UTC),
			UpdatedAt: time.Date(2016, time.October, 27, 4, 3, 10, 0, time.UTC),
			Name:      "dev_query_2",
			Query:     "select * from time",
		},
		{
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-17 * time.Hour),
			Name:      "dev_query_3",
			Query:     "select * from cpuid",
		},
		{
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-30 * time.Minute),
			Name:      "dev_query_4",
			Query:     "select 1 from processes where name like '%Apache%'",
		},
		{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      "dev_query_5",
			Query:     "select 1 from osquery_info where build_platform='darwin'",
		},
	}

	for _, query := range queries {
		query := query
		_, err := ds.NewQuery(&query)
		if err != nil {
			initFatal(err, "creating bootstrap query")
		}
	}
}

func createDevLabels(ds kolide.Datastore, config config.KolideConfig) {
	labels := []kolide.Label{
		{
			CreatedAt: time.Date(2016, time.October, 27, 8, 31, 16, 0, time.UTC),
			UpdatedAt: time.Date(2016, time.October, 27, 8, 31, 16, 0, time.UTC),
			Name:      "dev_label_apache",
			QueryID:   4,
		},
		{
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now(),
			Name:      "dev_label_darwin",
			QueryID:   5,
		},
	}

	for _, label := range labels {
		label := label
		_, err := ds.NewLabel(&label)
		if err != nil {
			initFatal(err, "creating bootstrap label")
		}
	}
}
