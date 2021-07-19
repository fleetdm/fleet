package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	adminFlagName      = "admin"
	globalRoleFlagName = "global-role"
	teamFlagName       = "team"
	passwordFlagName   = "password"
	emailFlagName      = "email"
	nameFlagName       = "name"
	ssoFlagName        = "sso"
	apiOnlyFlagName    = "api-only"
)

func userCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage Fleet users",
		Subcommands: []*cli.Command{
			createUserCommand(),
			deleteUserCommand(),
		},
	}
}

func createUserCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new user",
		UsageText: `This command will create a new user in Fleet. By default, the user will authenticate with a password and will be a global observer.

   If a password is required and not provided by flag, the command will prompt for password input through stdin.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     emailFlagName,
				Usage:    "Email for new user (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     nameFlagName,
				Usage:    "User's full name or nickname (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  passwordFlagName,
				Usage: "Password for new user",
			},
			&cli.BoolFlag{
				Name:  ssoFlagName,
				Usage: "Enable user login via SSO",
			},
			&cli.BoolFlag{
				Name:  apiOnlyFlagName,
				Usage: "Make \"API-only\" user",
			},
			&cli.StringFlag{
				Name:  globalRoleFlagName,
				Usage: "Global role to assign to user (default \"observer\")",
			},
			&cli.StringSliceFlag{
				Name:    "team",
				Aliases: []string{"t"},
				Usage:   "Team assignments in team_id:role pairs (multiple may be specified)",
			},
			configFlag(),
			contextFlag(),
			yamlFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			password := c.String(passwordFlagName)
			email := c.String(emailFlagName)
			name := c.String(nameFlagName)
			sso := c.Bool(ssoFlagName)
			apiOnly := c.Bool(apiOnlyFlagName)
			globalRoleString := c.String(globalRoleFlagName)
			teamStrings := c.StringSlice(teamFlagName)

			var globalRole *string
			var teams []fleet.UserTeam
			if globalRoleString != "" && len(teamStrings) > 0 {
				return errors.New("Users may not have global_role and teams.")
			} else if globalRoleString == "" && len(teamStrings) == 0 {
				globalRole = ptr.String(fleet.RoleObserver)
			} else if globalRoleString != "" {
				if !fleet.ValidGlobalRole(globalRoleString) {
					return errors.Errorf("'%s' is not a valid team role", globalRoleString)
				}
				globalRole = ptr.String(globalRoleString)
			} else {
				for _, t := range teamStrings {
					parts := strings.Split(t, ":")
					if len(parts) != 2 {
						return errors.Errorf("Unable to parse '%s' as team_id:role", t)
					}
					teamID, err := strconv.Atoi(parts[0])
					if err != nil {
						return errors.Wrap(err, "Unable to parse team_id")
					}
					if !fleet.ValidTeamRole(parts[1]) {
						return errors.Errorf("'%s' is not a valid team role", parts[1])
					}

					teams = append(teams, fleet.UserTeam{Team: fleet.Team{ID: uint(teamID)}, Role: parts[1]})
				}
			}

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
			err = client.CreateUser(fleet.UserPayload{
				Password:                 &password,
				Email:                    &email,
				Name:                     &name,
				SSOEnabled:               &sso,
				AdminForcedPasswordReset: &force_reset,
				APIOnly:                  &apiOnly,
				GlobalRole:               globalRole,
				Teams:                    &teams,
			})
			if err != nil {
				return errors.Wrap(err, "Failed to create user")
			}

			return nil
		},
	}
}

func deleteUserCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a user",
		UsageText: `This command will delete a user specified by their email in Fleet.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     emailFlagName,
				Usage:    "Email for user (required)",
				Required: true,
			},
			configFlag(),
			contextFlag(),
			yamlFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			email := c.String(emailFlagName)
			return client.DeleteUser(email)
		},
	}
}
