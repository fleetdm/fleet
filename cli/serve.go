package cli

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/kolide/kolide-ose/mail"
	"github.com/kolide/kolide-ose/server"
	"github.com/kolide/kolide-ose/version"
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

			svc, err := server.NewService(ds, logger, config, mailService)
			if err != nil {
				initFatal(err, "initializing service")
			}

			if devMode {
				// bootstrap an admin user when using the
				// in-memory database
				var (
					name     = "Admin User"
					username = "admin"
					password = "admin"
					email    = "admin@kolide.co"
					enabled  = true
					isAdmin  = true
				)
				admin := kolide.UserPayload{
					Name:     &name,
					Username: &username,
					Password: &password,
					Email:    &email,
					Enabled:  &enabled,
					Admin:    &isAdmin,
				}
				_, err := svc.NewUser(ctx, admin)
				if err != nil {
					initFatal(err, "creating bootstrap user")
				}
			}

			svcLogger := kitlog.NewContext(logger).With("component", "service")
			svc = server.NewLoggingService(svc, svcLogger)

			httpLogger := kitlog.NewContext(logger).With("component", "http")

			apiHandler := server.MakeHandler(ctx, svc, config.Auth.JwtKey, ds, httpLogger)
			http.Handle("/api/", accessControl(apiHandler))
			http.Handle("/version", version.Handler())
			http.Handle("/assets/", server.ServeStaticAssets("/assets/"))
			http.Handle("/", server.ServeFrontend())

			errs := make(chan error, 2)
			go func() {
				logger.Log("transport", "http", "address", *httpAddr, "msg", "listening")
				errs <- http.ListenAndServe(*httpAddr, nil)
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
