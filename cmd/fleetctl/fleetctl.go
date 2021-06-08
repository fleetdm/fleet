package main

import (
	"math/rand"
	"time"

	eefleetctl "github.com/fleetdm/fleet/ee/fleetctl"
	"github.com/kolide/kit/version"
	"github.com/urfave/cli/v2"
)

const (
	defaultFileMode = 0600
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	app := cli.NewApp()
	app.Name = "fleetctl"
	app.Usage = "CLI for operating Fleet"
	app.Version = version.Version().Version
	cli.VersionPrinter = func(c *cli.Context) {
		version.PrintFull()
	}

	app.Commands = []*cli.Command{
		applyCommand(),
		deleteCommand(),
		setupCommand(),
		loginCommand(),
		logoutCommand(),
		queryCommand(),
		getCommand(),
		&cli.Command{
			Name:  "config",
			Usage: "Modify Fleet server connection settings",
			Subcommands: []*cli.Command{
				configSetCommand(),
				configGetCommand(),
			},
		},
		convertCommand(),
		goqueryCommand(),
		userCommand(),
		debugCommand(),
		previewCommand(),
		eefleetctl.UpdatesCommand(),
	}

	app.RunAndExitOnError()
}
