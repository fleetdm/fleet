package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/orbit/pkg/certificate"
	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/fleetdm/orbit/pkg/database"
	"github.com/fleetdm/orbit/pkg/insecure"
	"github.com/fleetdm/orbit/pkg/osquery"
	"github.com/fleetdm/orbit/pkg/update"
	"github.com/fleetdm/orbit/pkg/update/filestore"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	tufURL         = "https://tuf.fleetctl.com"
	certPath       = "/tmp/fleet.pem"
	defaultRootDir = "/var/lib/orbit"
)

func main() {
	log.Logger = log.Output(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano},
	)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	app.Commands = []*cli.Command{
		shellCommand,
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "root-dir",
			Usage:   "Root directory for Orbit state",
			Value:   defaultRootDir,
			EnvVars: []string{"ORBIT_ROOT_DIR"},
		},
		&cli.BoolFlag{
			Name:    "insecure",
			Usage:   "Disable TLS certificate verification",
			EnvVars: []string{"ORBIT_INSECURE"},
		},
		&cli.StringFlag{
			Name:    "fleet-url",
			Usage:   "URL (host:port) of Fleet server",
			EnvVars: []string{"ORBIT_FLEET_URL"},
		},
		&cli.StringFlag{
			Name:    "fleet-certificate",
			Usage:   "Path to server cerificate bundle",
			EnvVars: []string{"ORBIT_FLEET_CERTIFICATE"},
		},
		&cli.StringFlag{
			Name:    "tuf-url",
			Usage:   "URL of TUF update server",
			Value:   tufURL,
			EnvVars: []string{"ORBIT_TUF_URL"},
		},
		&cli.StringFlag{
			Name:    "enroll-secret",
			Usage:   "Enroll secret for authenticating to Fleet server",
			EnvVars: []string{"ORBIT_ENROLL_SECRET"},
		},
		&cli.StringFlag{
			Name:    "enroll-secret-path",
			Usage:   "Path to file containing enroll secret",
			EnvVars: []string{"ORBIT_ENROLL_SECRET_PATH"},
		},
		&cli.StringFlag{
			Name:    "osquery-channel",
			Usage:   "Update channel of osquery to use",
			Value:   "stable",
			EnvVars: []string{"ORBIT_OSQUERY_CHANNEL"},
		},
		&cli.StringFlag{
			Name:    "orbit-channel",
			Usage:   "Update channel of orbit to use",
			Value:   "stable",
			EnvVars: []string{"ORBI_ORBIT_CHANNEL"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enable debug logging",
			EnvVars: []string{"ORBIT_DEBUG"},
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		if c.Bool("insecure") && c.String("fleet-certificate") != "" {
			return errors.New("insecure and fleet-certificate may not be specified together")
		}

		if c.String("enroll-secret-path") != "" {
			if c.String("enroll-secret") != "" {
				return errors.New("enroll-secret and enroll-secret-path may not be specified together")
			}

			b, err := ioutil.ReadFile(c.String("enroll-secret-path"))
			if err != nil {
				return errors.Wrap(err, "read enroll secret file")
			}

			if err := c.Set("enroll-secret", strings.TrimSpace(string(b))); err != nil {
				return errors.Wrap(err, "set enroll secret from file")
			}
		}

		if err := os.MkdirAll(c.String("root-dir"), constant.DefaultDirMode); err != nil {
			return errors.Wrap(err, "initialize root dir")
		}

		db, err := database.Open(filepath.Join(c.String("root-dir"), "orbit.db"))
		if err != nil {
			return err
		}
		defer func() {
			if err := db.Close(); err != nil {
				log.Error().Err(err).Msg("Close badger")
			}
		}()

		localStore, err := filestore.New(filepath.Join(c.String("root-dir"), "tuf-metadata.json"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create local metadata store")
		}

		// Initialize updater and get expected version
		opt := update.DefaultOptions
		opt.RootDirectory = c.String("root-dir")
		opt.ServerURL = c.String("tuf-url")
		opt.LocalStore = localStore
		opt.InsecureTransport = c.Bool("insecure")
		updater, err := update.New(opt)
		if err != nil {
			return err
		}
		if err := updater.UpdateMetadata(); err != nil {
			log.Info().Err(err).Msg("failed to update metadata. using saved metadata.")
		}
		osquerydPath, err := updater.Get("osqueryd", c.String("osquery-channel"))
		if err != nil {
			return err
		}

		var g run.Group

		updateRunner, err := update.NewRunner(updater, update.RunnerOptions{
			CheckInterval: 10 * time.Second,
			Targets: map[string]string{
				"osqueryd": c.String("osquery-channel"),
				"orbit":    c.String("orbit-channel"),
			},
		})
		if err != nil {
			return err
		}
		g.Add(updateRunner.Execute, updateRunner.Interrupt)

		var options []func(*osquery.Runner) error
		options = append(options, osquery.WithDataPath(c.String("root-dir")))

		fleetURL := c.String("fleet-url")
		if !strings.HasPrefix(fleetURL, "http") {
			fleetURL = "https://" + fleetURL
		}

		enrollSecret := c.String("enroll-secret")
		if enrollSecret != "" {
			options = append(options,
				osquery.WithEnv([]string{"ENROLL_SECRET=" + enrollSecret}),
				osquery.WithFlags([]string{"--enroll_secret_env=ENROLL_SECRET"}),
			)
		}

		if fleetURL != "https://" && c.Bool("insecure") {
			proxy, err := insecure.NewTLSProxy(fleetURL)
			if err != nil {
				return errors.Wrap(err, "create TLS proxy")
			}

			g.Add(
				func() error {
					log.Info().
						Str("addr", fmt.Sprintf("localhost:%d", proxy.Port)).
						Str("target", c.String("fleet-url")).
						Msg("using insecure TLS proxy")
					err := proxy.InsecureServeTLS()
					return err
				},
				func(error) {
					if err := proxy.Close(); err != nil {
						log.Error().Err(err).Msg("close proxy")
					}
				},
			)

			// Write cert that proxy uses
			err = ioutil.WriteFile(certPath, []byte(insecure.ServerCert), os.ModePerm)
			if err != nil {
				return errors.Wrap(err, "write server cert")
			}

			// Rewrite URL to the proxy URL. Note the proxy handles any URL
			// prefix so we don't need to carry that over here.
			parsedURL := &url.URL{
				Scheme: "https",
				Host:   fmt.Sprintf("localhost:%d", proxy.Port),
			}

			// Check and log if there are any errors with TLS connection.
			pool, err := certificate.LoadPEM(certPath)
			if err != nil {
				return errors.Wrap(err, "load certificate")
			}
			if err := certificate.ValidateConnection(pool, fleetURL); err != nil {
				log.Info().Err(err).Msg("Failed to connect to Fleet server. Osquery connection may fail.")
			}

			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(parsedURL)),
				osquery.WithFlags([]string{"--tls_server_certs", certPath}),
			)
		} else if fleetURL != "https://" {
			if enrollSecret == "" {
				return errors.New("enroll secret must be specified to connect to Fleet server")
			}

			parsedURL, err := url.Parse(fleetURL)
			if err != nil {
				return errors.Wrap(err, "parse URL")
			}

			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(parsedURL)),
			)

			if certPath := c.String("fleet-certificate"); certPath != "" {
				// Check and log if there are any errors with TLS connection.
				pool, err := certificate.LoadPEM(certPath)
				if err != nil {
					return errors.Wrap(err, "load certificate")
				}
				if err := certificate.ValidateConnection(pool, fleetURL); err != nil {
					log.Info().Err(err).Msg("Failed to connect to Fleet server. Osquery connection may fail.")
				}

				options = append(options,
					osquery.WithFlags([]string{"--tls_server_certs=" + certPath}),
				)
			} else {
				// Check and log if there are any errors with TLS connection.
				pool, err := x509.SystemCertPool()
				if err != nil {
					log.Info().Err(err).Msg("Failed to retrieve system cert pool. Cannot validate Fleet server connection.")
				} else if err := certificate.ValidateConnection(pool, fleetURL); err != nil {
					log.Info().Err(err).Msg("Failed to connect to Fleet server. Osquery connection may fail. Provide certificate with --fleet-certificate.")
				}
			}
		}

		// --force is sometimes needed when an older osquery process has not
		// exited properly
		options = append(options, osquery.WithFlags([]string{"--force"}))

		if c.Bool("debug") {
			options = append(options,
				osquery.WithFlags([]string{"--verbose", "--tls_dump"}),
			)
		}

		// Handle additional args after --
		options = append(options, osquery.WithFlags(c.Args().Slice()))

		// Create an osquery runner with the provided options
		r, _ := osquery.NewRunner(osquerydPath, options...)
		g.Add(r.Execute, r.Interrupt)

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		if err := g.Run(); err != nil {
			log.Error().Err(err).Msg("unexpected exit")
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("")
	}
}
