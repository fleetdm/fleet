package main

import (
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

func loginCommand() *cli.Command {
	var (
		flEmail    string
		flPassword string
	)
	return &cli.Command{
		Name:  "login",
		Usage: "Login to Fleet",
		UsageText: `
fleetctl login [options]

Interactively prompts for email and password if not specified in the flags or environment variables.
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "email",
				EnvVars:     []string{"EMAIL"},
				Value:       "",
				Destination: &flEmail,
				Usage:       "Email to use to log in",
			},
			&cli.StringFlag{
				Name:        "password",
				EnvVars:     []string{"PASSWORD"},
				Value:       "",
				Destination: &flPassword,
				Usage:       "Password to use to log in (recommended to use interactive entry)",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := unauthenticatedClientFromCLI(c)
			if err != nil {
				return err
			}

			// Allow interactive entry to discourage passwords in
			// CLI history.
			if flEmail == "" {
				fmt.Println("Log in using the standard Fleet credentials.")
				fmt.Print("Email: ")
				_, err := fmt.Scanln(&flEmail)
				if err != nil {
					return errors.Wrap(err, "error reading email")
				}
			}
			if flPassword == "" {
				fmt.Print("Password: ")
				passBytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return errors.Wrap(err, "error reading password")
				}
				fmt.Println()
				flPassword = string(passBytes)
			}

			token, err := fleet.Login(flEmail, flPassword)
			if err != nil {
				switch err.(type) {
				case service.NotSetupErr:
					return err
				}
				return errors.Wrap(err, "Login failed")
			}

			configPath, context := c.String("config"), c.String("context")

			if err := setConfigValue(configPath, context, "email", flEmail); err != nil {
				return errors.Wrap(err, "error setting email for the current context")
			}

			if err := setConfigValue(configPath, context, "token", token); err != nil {
				return errors.Wrap(err, "error setting token for the current context")
			}

			fmt.Printf("[+] Fleet login successful and context configured!\n")

			return nil
		},
	}
}
