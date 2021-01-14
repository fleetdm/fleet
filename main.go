package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/fleetdm/orbit/pkg/database"
	"github.com/fleetdm/orbit/pkg/insecure"
	"github.com/fleetdm/orbit/pkg/osquery"
	"github.com/fleetdm/orbit/pkg/update"
	"github.com/fleetdm/orbit/pkg/update/badgerstore"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	notaryURL = "https://tuf.fleetctl.com"
	certPath  = "/tmp/fleet.pem"
)

func main() {

	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	defaultRootDir := "/usr/local/fleet"
	app.Commands = []*cli.Command{
		shellCommand,
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "root-dir",
			Usage: "Root directory for Orbit state",
			Value: defaultRootDir,
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Disable TLS certificate verification",
		},
		&cli.StringFlag{
			Name:  "fleet-url",
			Usage: "URL (host:port) to Fleet server",
		},
		&cli.StringFlag{
			Name:  "notary-url",
			Usage: "URL to Notary update server",
			Value: notaryURL,
		},
		&cli.StringFlag{
			Name:  "enroll-secret",
			Usage: "Enroll secret for authenticating to Fleet server",
		},
		&cli.StringFlag{
			Name:  "osqueryd-version",
			Usage: "Version of osqueryd to use",
			Value: "stable",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	}
	app.Action = func(c *cli.Context) error {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
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

		// Initialize updater and get expected version
		opt := update.DefaultOptions
		opt.RootDirectory = c.String("root-dir")
		opt.ServerURL = c.String("notary-url")
		opt.LocalStore = badgerstore.New(db.DB)
		updater, err := update.New(opt)
		if err != nil {
			return err
		}
		if err := updater.UpdateMetadata(); err != nil {
			return err
		}
		osquerydPath, err := updater.Get(
			"osqueryd",
			constant.PlatformName,
			c.String("osqueryd-version"),
		)
		if err != nil {
			return err
		}

		var g run.Group
		var options []func(*osquery.Runner) error

		fleetURL := c.String("fleet-url")

		if c.Bool("insecure") {
			proxy, err := insecure.NewTLSProxy(fleetURL)
			if err != nil {
				return errors.Wrap(err, "create TLS proxy")
			}

			g.Add(
				func() error {
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

			// Rewrite URL to the proxy URL
			fleetURL = fmt.Sprintf("localhost:%d", proxy.Port)

			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(fleetURL)),
				osquery.WithFlags([]string{"--tls_server_certs", certPath}),
			)
		}

		if enrollSecret := c.String("enroll-secret"); enrollSecret != "" {
			options = append(options,
				osquery.WithEnv([]string{"ENROLL_SECRET=" + enrollSecret}),
				osquery.WithFlags([]string{"--enroll_secret_env", "ENROLL_SECRET"}),
			)
		}

		if fleetURL != "" {
			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(fleetURL)),
			)
		}

		options = append(options,
			osquery.WithFlags([]string{"--verbose"}),
		)

		options = append(options, osquery.WithPath(osquerydPath))
		options = append(options, osquery.WithShell())
		options = append(options, osquery.WithFlags([]string(c.Args().Slice())))

		// Create an osquery runner with the provided options
		r, _ := osquery.NewRunner(options...)
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

var shellCommand = &cli.Command{
	Name:    "shell",
	Aliases: []string{"osqueryi"},
	Usage:   "Run the osqueryi shell",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "osqueryd-version",
			Usage: "Version of osqueryd to use",
			Value: "stable",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	},
	Action: func(c *cli.Context) error {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
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
				log.Error().Err(err).Msg("close badger")
			}
		}()

		// Initialize updater and get expected version
		opt := update.DefaultOptions
		opt.RootDirectory = c.String("root-dir")
		opt.ServerURL = c.String("notary-url")
		opt.LocalStore = badgerstore.New(db.DB)
		updater, err := update.New(opt)
		if err != nil {
			return err
		}
		if err := updater.UpdateMetadata(); err != nil {
			return err
		}
		osquerydPath, err := updater.Get(
			"osqueryd",
			constant.PlatformName,
			c.String("osqueryd-version"),
		)
		if err != nil {
			return err
		}

		var g run.Group

		// Create an osquery runner with the provided options
		r, _ := osquery.NewRunner(
			osquery.WithShell(),
			osquery.WithPath(osquerydPath),
			osquery.WithFlags([]string(c.Args().Slice())),
		)
		g.Add(r.Execute, r.Interrupt)

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		if err := g.Run(); err != nil {
			log.Error().Err(err).Msg("unexpected exit")
		}

		return nil
	},
}
