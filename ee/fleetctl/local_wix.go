package eefleetctl

import "github.com/urfave/cli/v2"

func LocalWixDirFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "local-wix-dir",
		Usage:       "Deprecated and ignored: MSI installers are now built in pure Go and WiX is no longer used.",
		Destination: dest,
	}
}
