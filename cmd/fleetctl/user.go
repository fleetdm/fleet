package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	adminFlagName    = "admin"
	usernameFlagName = "username"
	passwordFlagName = "password"
	emailFlagName    = "email"
	ssoFlagName      = "sso"
)

func userCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage Fleet users",
		Subcommands: []*cli.Command{
			createUserCommand(),
		},
	}
}

func createUserCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new user",
		UsageText: `This command will create a new user in Fleet. By default, the user will authenticate with a password and will not have admin privileges.

   If a password is required and not provided by flag, the command will prompt for password input through stdin.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     usernameFlagName,
				Usage:    "Username for new user (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     emailFlagName,
				Usage:    "Email for new user (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  passwordFlagName,
				Usage: "Password for new user",
			},
			&cli.BoolFlag{
				Name:  adminFlagName,
				Usage: "Grant admin privileges to created user (default false)",
			},
			&cli.BoolFlag{
				Name:  ssoFlagName,
				Usage: "Enable user login via SSO (default false)",
			},
			configFlag(),
			contextFlag(),
			yamlFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			username := c.String(usernameFlagName)
			password := c.String(passwordFlagName)
			email := c.String(emailFlagName)
			admin := c.Bool(adminFlagName)
			sso := c.Bool(ssoFlagName)

			if sso && len(password) > 0 {
				return fmt.Errorf("Password may not be provided for SSO users.")
			}
			if !sso && len(password) == 0 {
				fmt.Print("Enter password for user: ")
				passBytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return errors.Wrap(err, "Failed to read password")
				}
				if len(passBytes) == 0 {
					return fmt.Errorf("Password may not be empty.")
				}

				fmt.Print("Enter password for user (confirm): ")
				confBytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return errors.Wrap(err, "Failed to read confirmation")
				}

				if !bytes.Equal(passBytes, confBytes) {
					return fmt.Errorf("Confirmation does not match")
				}

				password = string(passBytes)
			}

			// Only set the password reset flag if SSO is not enabled. Otherwise
			// the user will be stuck in a bad state and not be able to log in.
			force_reset := !sso
			err = fleet.CreateUser(kolide.UserPayload{
				Username:                 &username,
				Password:                 &password,
				Email:                    &email,
				Admin:                    &admin,
				SSOEnabled:               &sso,
				AdminForcedPasswordReset: &force_reset,
			})
			if err != nil {
				return errors.Wrap(err, "Failed to create user")
			}

			return nil
		},
	}
}
