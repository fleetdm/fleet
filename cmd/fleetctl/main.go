package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"runtime"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/goquerycmd"
	"github.com/urfave/cli/v2"
)

func main() {
	// Register the goquery subcommand here so that the
	// github.com/AbGuthrie/goquery/v2 dependency (and its init function) are
	// only linked into the fleetctl binary, not into other binaries that
	// import the fleetctl package (e.g. fleet server).
	fleetctl.SetGoqueryRunner(goquerycmd.Run)

	app := fleetctl.CreateApp(os.Stdin, os.Stdout, os.Stderr, exitErrHandler)
	fleetctl.StashRawArgs(app, os.Args)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "Error: %+v\n", err)
		os.Exit(1)
	}
}

// exitErrHandler implements cli.ExitErrHandlerFunc. If there is an error, prints it to stderr and exits with status 1.
func exitErrHandler(c *cli.Context, err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(c.App.ErrWriter, "Error: %+v\n", err)

	if errors.Is(err, fs.ErrPermission) {
		switch runtime.GOOS {
		case "darwin", "linux":
			fmt.Fprintf(c.App.ErrWriter, "\nThis error can usually be resolved by fixing the permissions on the %s directory, or re-running this command with sudo.\n", path.Dir(c.String("config")))
		case "windows":
			fmt.Fprintf(c.App.ErrWriter, "\nThis error can usually be resolved by fixing the permissions on the %s directory, or re-running this command with 'Run as administrator'.\n", path.Dir(c.String("config")))
		}
	}
	cli.OsExiter(1)
}
