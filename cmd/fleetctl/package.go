package main

import (
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/lib"
	"github.com/urfave/cli/v2"
)

var (
	config lib.PackageConfig
)

func packageCommand() *cli.Command {
	return &cli.Command{
		Name:        "package",
		Aliases:     nil,
		Usage:       "Create an Orbit installer package",
		Description: "An easy way to create fully boot-strapped installer packages for Windows, macOS, or Linux",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "type",
				Usage:       "Type of package to build",
				Destination: &config.Type,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "enroll-secret",
				Usage:       "Enroll secret for authenticating to Fleet server",
				Destination: &config.EnrollSecret,
			},
			&cli.StringFlag{
				Name:        "fleet-url",
				Usage:       "URL (host:port) of Fleet server",
				Destination: &config.FleetURL,
			},
			&cli.StringFlag{
				Name:        "fleet-certificate",
				Usage:       "Path to server certificate chain",
				Destination: &config.FleetCertificate,
			},
			&cli.StringFlag{
				Name:        "identifier",
				Usage:       "Identifier for package product",
				Value:       "com.fleetdm.orbit",
				Destination: &config.Identifier,
			},
			&cli.StringFlag{
				Name:        "version",
				Usage:       "Version for package product",
				Destination: &config.Version,
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "Disable TLS certificate verification",
				Destination: &config.Insecure,
			},
			&cli.BoolFlag{
				Name:        "service",
				Usage:       "Install orbit/osquery with a persistence service (launchd, systemd, etc.)",
				Value:       true,
				Destination: &config.StartService,
			},
			&cli.StringFlag{
				Name:        "sign-identity",
				Usage:       "Identity to use for macOS codesigning",
				Destination: &config.SignIdentity,
			},
			&cli.BoolFlag{
				Name:        "notarize",
				Usage:       "Whether to notarize macOS packages",
				Destination: &config.Notarize,
			},
			&cli.StringFlag{
				Name:        "osqueryd-channel",
				Usage:       "Update channel of osqueryd to use",
				Value:       "stable",
				Destination: &config.OsquerydChannel,
			},
			&cli.StringFlag{
				Name:        "desktop-channel",
				Usage:       "Update channel of desktop to use",
				Value:       "stable",
				Destination: &config.DesktopChannel,
			},
			&cli.StringFlag{
				Name:        "orbit-channel",
				Usage:       "Update channel of Orbit to use",
				Value:       "stable",
				Destination: &config.OrbitChannel,
			},
			&cli.BoolFlag{
				Name:        "disable-updates",
				Usage:       "Disable auto updates on the generated package",
				Destination: &config.DisableUpdates,
			},
			&cli.StringFlag{
				Name:        "update-url",
				Usage:       "URL for update server",
				Value:       "https://tuf.fleetctl.com",
				Destination: &config.UpdateURL,
			},
			&cli.StringFlag{
				Name:        "update-roots",
				Usage:       "Root key JSON metadata for update server (from fleetctl updates roots)",
				Destination: &config.UpdateRoots,
			},
			&cli.StringFlag{
				Name:        "osquery-flagfile",
				Usage:       "Flagfile to package and provide to osquery",
				Destination: &config.OsqueryFlagfile,
			},
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Enable debug logging in orbit",
				Destination: &config.Debug,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "Log detailed information when building the package",
				Destination: &config.Verbose,
			},
			&cli.BoolFlag{
				Name:        "fleet-desktop",
				Usage:       "Include the Fleet Desktop Application in the package",
				Destination: &config.Desktop,
			},
			&cli.DurationFlag{
				Name:        "update-interval",
				Usage:       "Interval that Orbit will use to check for new updates (10s, 1h, etc.)",
				Value:       15 * time.Minute,
				Destination: &config.OrbitUpdateInterval,
			},
			&cli.BoolFlag{
				Name:        "disable-open-folder",
				Usage:       "Disable opening the folder at the end",
				Destination: &config.DisableOpenFolder,
			},
			&cli.BoolFlag{
				Name:        "native-tooling",
				Usage:       "Build the package using native tooling (only available in Linux)",
				EnvVars:     []string{"FLEETCTL_NATIVE_TOOLING"},
				Destination: &config.NativeTooling,
			},
		},
		Action: func(c *cli.Context) error {
			return lib.PackageAction(config)
		},
	}
}
