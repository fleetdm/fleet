package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	eefleetctl "github.com/fleetdm/fleet/v4/ee/fleetctl"
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
	app := createApp(os.Stdin, os.Stdout, exitErrHandler)
	app.Run(os.Args)
}

// exitErrHandler implements cli.ExitErrHandlerFunc. If there is an error, prints it to stderr and exits with status 1.
func exitErrHandler(c *cli.Context, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(c.App.ErrWriter, "Error: %+v\n", err)
	cli.OsExiter(1)
}

func createApp(reader io.Reader, writer io.Writer, exitErrHandler cli.ExitErrHandlerFunc) *cli.App {
	app := cli.NewApp()
	app.Name = "fleetctl"
	app.Usage = "CLI for operating Fleet"
	app.Version = version.Version().Version
	app.ExitErrHandler = exitErrHandler
	cli.VersionPrinter = func(c *cli.Context) {
		version.PrintFull()
	}
	app.Reader = reader
	app.Writer = writer
	app.ErrWriter = writer

	app.Commands = []*cli.Command{
		applyCommand(),
		deleteCommand(),
		setupCommand(),
		loginCommand(),
		logoutCommand(),
		queryCommand(),
		getCommand(),
		{
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
		hostsCommand(),
		vulnerabilityDataStreamCommand(),
		packageCommand(),
	}
	return app
}
