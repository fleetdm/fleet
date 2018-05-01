package main

import (
	"math/rand"
	"time"

	"github.com/kolide/kit/version"
	"github.com/urfave/cli"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	app := cli.NewApp()
	app.Name = "fleetctl"
	app.Usage = "The CLI for operating Kolide Fleet"
	app.Version = version.Version().Version
	cli.VersionPrinter = func(c *cli.Context) {
		version.PrintFull()
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:        "query",
			Usage:       "run a query across your fleet",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:        "apply",
			Usage:       "apply a set of osquery configurations",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:        "edit",
			Usage:       "edit your complete configuration in an ephemeral editor",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:  "config",
			Usage: "modify how and which Fleet server to connect to",
			Subcommands: []cli.Command{
				loginCommand(),
				setupCommand(),
			},
		},
	}

	app.RunAndExitOnError()
}
