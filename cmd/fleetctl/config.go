package main

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/urfave/cli"
)

func setupCommand() cli.Command {
	var (
		flAddress string
	)
	return cli.Command{
		Name:      "setup",
		Usage:     "Setup a Kolide Fleet instance",
		UsageText: `fleetctl config login [options]`,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "address",
				Value:       "",
				Destination: &flAddress,
				Usage:       "The address of the Kolide Fleet instance",
			},
		},
		Action: func(cliCtx *cli.Context) error {
			logger := log.NewLogfmtLogger(os.Stdout)
			level.Info(logger).Log("msg", "setting up fleet")
			return nil
		},
	}
}

func loginCommand() cli.Command {
	var (
		flAddress string
	)
	return cli.Command{
		Name:      "login",
		Usage:     "Login to Kolide Fleet",
		UsageText: `fleetctl config login [options]`,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "address",
				Value:       "",
				Destination: &flAddress,
				Usage:       "The address of the Kolide Fleet instance",
			},
		},
		Action: func(cliCtx *cli.Context) error {
			logger := log.NewLogfmtLogger(os.Stdout)
			level.Info(logger).Log("msg", "logging in to fleet")
			return nil
		},
	}
}
