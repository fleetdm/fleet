package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:      "logout",
		Usage:     "Log out of Fleet",
		UsageText: `fleetctl logout [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if err := fleet.Logout(); err != nil {
				return errors.Wrap(err, "error logging out")
			}

			configPath, context := c.String("config"), c.String("context")

			if err := setConfigValue(configPath, context, "token", ""); err != nil {
				return errors.Wrap(err, "error setting token for the current context")
			}

			fmt.Printf("[+] Fleet logout successful and local token cleared!\n")

			return nil
		},
	}
}
