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
			Name:        "create",
			Usage:       "Create resources",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:        "get",
			Usage:       "Get and list resources",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:        "put",
			Usage:       "Create or update resources",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:        "delete",
			Usage:       "Delete resources",
			Subcommands: []cli.Command{},
		},
		cli.Command{
			Name:        "ensure",
			Usage:       "Ensures the state of resources",
			Subcommands: []cli.Command{},
		},
	}

	app.RunAndExitOnError()
}
