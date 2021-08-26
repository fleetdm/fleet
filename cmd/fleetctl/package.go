package main

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var opt packaging.Options

func packageCommand() *cli.Command {
	return &cli.Command{
		Name:        "package",
		Aliases:     nil,
		Usage:       "Create an Orbit installer package",
		Description: "An easy way to create fully boot-strapped installer packages for Windows, macOS, or Linux",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "type",
				Usage:    "Type of package to build",
				Required: true,
			},
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
			&cli.StringFlag{
				Name:        "fleet-certificate",
				Usage:       "Path to server cerificate bundle",
				Destination: &opt.FleetCertificate,
			},
			&cli.StringFlag{
				Name:        "identifier",
				Usage:       "Identifier for package product",
				Value:       "com.fleetdm.orbit",
				Destination: &opt.Identifier,
			},
			&cli.StringFlag{
				Name:        "version",
				Usage:       "Version for package product",
				Value:       "0.0.3",
				Destination: &opt.Version,
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "Disable TLS certificate verification",
				Destination: &opt.Insecure,
			},
			&cli.BoolFlag{
				Name:        "service",
				Usage:       "Install orbit/osquery with a persistence service (launchd, systemd, etc.)",
				Value:       true,
				Destination: &opt.StartService,
			},
			&cli.StringFlag{
				Name:        "sign-identity",
				Usage:       "Identity to use for macOS codesigning",
				Destination: &opt.SignIdentity,
			},
			&cli.BoolFlag{
				Name:        "notarize",
				Usage:       "Whether to notarize macOS packages",
				Destination: &opt.Notarize,
			},
			&cli.StringFlag{
				Name:        "osqueryd-channel",
				Usage:       "Update channel of osqueryd to use",
				Value:       "stable",
				Destination: &opt.OsquerydChannel,
			},
			&cli.StringFlag{
				Name:        "orbit-channel",
				Usage:       "Update channel of Orbit to use",
				Value:       "stable",
				Destination: &opt.OrbitChannel,
			},
			&cli.StringFlag{
				Name:        "update-url",
				Usage:       "URL for update server",
				Value:       "https://tuf.fleetctl.com",
				Destination: &opt.UpdateURL,
			},
			&cli.StringFlag{
				Name:        "update-roots",
				Usage:       "Root key JSON metadata for update server (from fleetctl updates roots)",
				Destination: &opt.UpdateRoots,
			},
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Enable debug logging",
				Destination: &opt.Debug,
			},
		},
		Action: func(c *cli.Context) error {
			if opt.FleetURL != "" || opt.EnrollSecret != "" {
				if opt.FleetURL == "" || opt.EnrollSecret == "" {
					return errors.New("--enroll-secret and --fleet-url must be provided together")
				}
			}

			if opt.Insecure && opt.FleetCertificate != "" {
				return errors.New("--insecure and --fleet-certificate may not be provided together")
			}

			switch c.String("type") {
			case "pkg":
				return packaging.BuildPkg(opt)
			case "deb":
				return packaging.BuildDeb(opt)
			case "rpm":
				return packaging.BuildRPM(opt)
			case "msi":
				return packaging.BuildMSI(opt)
			default:
				return errors.New("type must be one of ('pkg', 'deb', 'rpm', 'msi')")
			}
		},
	}
}
