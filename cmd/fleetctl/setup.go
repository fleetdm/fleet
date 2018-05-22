package main

import (
	"fmt"

	"github.com/kolide/fleet/server/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func setupCommand() cli.Command {
	var (
		flEmail    string
		flPassword string
		flOrgName  string
	)
	return cli.Command{
		Name:      "setup",
		Usage:     "Setup a Kolide Fleet instance",
		UsageText: `fleetctl config login [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			cli.StringFlag{
				Name:        "email",
				EnvVar:      "EMAIL",
				Value:       "",
				Destination: &flEmail,
				Usage:       "Email of the admin user to create",
			},
			cli.StringFlag{
				Name:        "password",
				EnvVar:      "PASSWORD",
				Value:       "",
				Destination: &flPassword,
				Usage:       "Password for the admin user",
			},
			cli.StringFlag{
				Name:        "org-name",
				EnvVar:      "ORG_NAME",
				Value:       "",
				Destination: &flOrgName,
				Usage:       "Name of the organization",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if flEmail == "" {
				return errors.Errorf("Email of the admin user to create must be provided")
			}
			if flPassword == "" {
				return errors.Errorf("Password for the admin user to create must be provided")
			}
			token, err := fleet.Setup(flEmail, flPassword, flOrgName)
			if err != nil {
				switch err.(type) {
				case service.SetupAlreadyErr:
					return err
				}
				return errors.Wrap(err, "error setting up Fleet")
			}

			if err := setConfigValue(c, "email", flEmail); err != nil {
				return errors.Wrap(err, "error setting email for the current context")
			}

			if err := setConfigValue(c, "token", token); err != nil {
				return errors.Wrap(err, "error setting token for the current context")
			}

			fmt.Printf("[+] Fleet setup successful and context configured!\n")

			return nil
		},
	}
}
