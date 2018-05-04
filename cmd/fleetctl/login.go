package main

import (
	"fmt"

	"github.com/kolide/fleet/server/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func loginCommand() cli.Command {
	var (
		flEmail    string
		flPassword string
	)
	return cli.Command{
		Name:      "login",
		Usage:     "Login to Kolide Fleet",
		UsageText: `fleetctl login [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			cli.StringFlag{
				Name:        "email",
				EnvVar:      "EMAIL",
				Value:       "",
				Destination: &flEmail,
				Usage:       "The email to use to login",
			},
			cli.StringFlag{
				Name:        "password",
				EnvVar:      "PASSWORD",
				Value:       "",
				Destination: &flPassword,
				Usage:       "The password to use to login",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			token, err := fleet.Login(flEmail, flPassword)
			if err != nil {
				switch err.(type) {
				case service.InvalidLoginErr:
					return err
				case service.NotSetupErr:
					return err
				}
				return errors.Wrap(err, "error logging in")
			}

			if err := setConfigValue(c, "email", flEmail); err != nil {
				return errors.Wrap(err, "error setting email for the current context")
			}

			if err := setConfigValue(c, "token", token); err != nil {
				return errors.Wrap(err, "error setting token for the current context")
			}

			fmt.Printf("[+] Fleet login successful and context configured!\n")

			return nil
		},
	}
}
