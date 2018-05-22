package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func logoutCommand() cli.Command {
	return cli.Command{
		Name:      "logout",
		Usage:     "Logout of Kolide Fleet",
		UsageText: `fleetctl logout [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if err := fleet.Logout(); err != nil {
				return errors.Wrap(err, "error logging in")
			}

			if err := setConfigValue(c, "token", ""); err != nil {
				return errors.Wrap(err, "error setting token for the current context")
			}

			fmt.Printf("[+] Fleet logout successful and local token cleared!\n")

			return nil
		},
	}
}
