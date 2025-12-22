package fleetctl

import (
	"errors"
	"io"

	eefleetctl "github.com/fleetdm/fleet/v4/ee/fleetctl"
	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/urfave/cli/v2"
)

const (
	defaultFileMode = 0o600
)

func CreateApp(
	reader io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	exitErrHandler cli.ExitErrHandlerFunc,
) *cli.App {
	app := cli.NewApp()
	app.Name = "fleetctl"
	app.Usage = "CLI for operating Fleet"
	app.Version = version.Version().Version
	app.ExitErrHandler = exitErrHandler
	cli.VersionPrinter = func(c *cli.Context) {
		version.PrintFull()
	}
	app.Reader = reader
	app.Writer = stdout
	app.ErrWriter = stderr
	app.DisableSliceFlagSeparator = true

	app.Commands = []*cli.Command{
		apiCommand(),
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
		generateCommand(),
		{
			// It's become common for folks to unintentionally install fleetctl when they actually
			// need the Fleet server. This is hopefully a more helpful error message.
			Name:  "prepare",
			Usage: "This is not the binary you're looking for. Please use the fleet server binary for prepare commands.",
			Action: func(c *cli.Context) error {
				return errors.New("This is not the binary you're looking for. Please use the fleet server binary for prepare commands.")
			},
		},
		triggerCommand(),
		mdmCommand(),
		upgradePacksCommand(),
		runScriptCommand(),
		gitopsCommand(),
		generateGitopsCommand(),
	}
	return app
}
