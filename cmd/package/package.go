package main

import (
	"os"
	"time"

	"github.com/fleetdm/orbit/pkg/packaging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	var opt packaging.Options
	log.Logger = log.Output(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano},
	)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	app.Commands = []*cli.Command{}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "enroll-secret",
			Usage:       "Enroll secret for authenticating to Fleet server",
			Destination: &opt.EnrollSecret,
		},
		&cli.StringFlag{
			Name:        "fleet-url",
			Usage:       "URL (host:port) of Fleet server",
			Destination: &opt.FleetURL,
		},
		&cli.BoolFlag{
			Name:        "insecure",
			Usage:       "Disable TLS certificate verification",
			Destination: &opt.Insecure,
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	}
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		return nil
	}
	app.Action = func(c *cli.Context) error {
		if opt.FleetURL != "" || opt.EnrollSecret != "" {
			opt.StartService = true

			if opt.FleetURL == "" || opt.EnrollSecret == "" {
				return errors.New("--enroll-secret and --fleet-url must be provided together")
			}
		}

		return packaging.BuildDeb(opt)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("package failed")
	}
}
