package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/sethvargo/go-password/password"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	globalRoleFlagName = "global-role"
	teamFlagName       = "team"
	passwordFlagName   = "password"
	emailFlagName      = "email"
	nameFlagName       = "name"
	ssoFlagName        = "sso"
	mfaFlagName        = "mfa"
	apiOnlyFlagName    = "api-only"
	csvFlagName        = "csv"
)

func userCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage Fleet users",
		Subcommands: []*cli.Command{
			createUserCommand(),
			deleteUserCommand(),
			createBulkUsersCommand(),
			deleteBulkUsersCommand(),
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
				Name:  mfaFlagName,
				Usage: "Require email verification on login (not applicable to SSO users)",
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
			mfa := c.Bool(mfaFlagName)
			apiOnly := c.Bool(apiOnlyFlagName)
			globalRoleString := c.String(globalRoleFlagName)
			teamStrings := c.StringSlice(teamFlagName)

			if mfa && sso {
				return errors.New("email verification on login is not applicable to SSO users")
			}

			var globalRole *string
			var teams []fleet.UserTeam
			if globalRoleString != "" && len(teamStrings) > 0 { //nolint:gocritic // ignore ifElseChain
				return errors.New("Users may not have global_role and teams.")
			} else if globalRoleString == "" && len(teamStrings) == 0 {
				globalRole = ptr.String(fleet.RoleObserver)
			} else if globalRoleString != "" {
				if !fleet.ValidGlobalRole(globalRoleString) {
					return fmt.Errorf("'%s' is not a valid global role", globalRoleString)
				}
				globalRole = ptr.String(globalRoleString)
			} else {
				for _, t := range teamStrings {
					parts := strings.Split(t, ":")
					if len(parts) != 2 {
						return fmt.Errorf("Unable to parse '%s' as team_id:role", t)
					}
					teamID, err := strconv.Atoi(parts[0])
					if err != nil {
						return fmt.Errorf("Unable to parse team_id: %w", err)
					}
					if !fleet.ValidTeamRole(parts[1]) {
						return fmt.Errorf("'%s' is not a valid team role", parts[1])
					}

					teams = append(teams, fleet.UserTeam{Team: fleet.Team{ID: uint(teamID)}, Role: parts[1]}) //nolint:gosec // dismiss G115
				}
			}

			if sso && len(password) > 0 {
				return errors.New("Password may not be provided for SSO users.")
			}
			if !sso && len(password) == 0 {
				fmt.Print("Enter password for user: ")
				passBytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("Failed to read password: %w", err)
				}
				if len(passBytes) == 0 {
					return errors.New("Password may not be empty.")
				}

				fmt.Print("Enter password for user (confirm): ")
				confBytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("Failed to read confirmation: %w", err)
				}

				if !bytes.Equal(passBytes, confBytes) {
					return errors.New("Confirmation does not match")
				}

				password = string(passBytes)
			}

			// Only set the password reset flag if SSO is not enabled and user is not API-only. Otherwise
			// the user will be stuck in a bad state and not be able to log in.
			force_reset := !sso && !apiOnly

			// password requirements are validated as part of `CreateUser`
			sessionKey, err := client.CreateUser(fleet.UserPayload{
				Password:                 &password,
				Email:                    &email,
				Name:                     &name,
				SSOEnabled:               &sso,
				MFAEnabled:               &mfa,
				AdminForcedPasswordReset: &force_reset,
				APIOnly:                  &apiOnly,
				GlobalRole:               globalRole,
				Teams:                    &teams,
			})
			if err != nil {
				return fmt.Errorf("Failed to create user: %w", err)
			}

			if apiOnly && sessionKey != nil && *sessionKey != "" {
				fmt.Fprintf(c.App.Writer, "Success! The API token for your new user is: %s\n", *sessionKey)
			}

			return nil
		},
	}
}

func createBulkUsersCommand() *cli.Command {
	return &cli.Command{
		Name:      "create-users",
		Usage:     "Create bulk users",
		UsageText: `This command will create a set of users in Fleet by importing a CSV file. Expected columns are: Name,Email,SSO,API Only,Global Role,Teams. Created Users by default get random password and Observer Role.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     csvFlagName,
				Usage:    "csv file with all the users (required)",
				Required: true,
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			csvFilePath := c.String(csvFlagName)

			csvFile, err := os.Open(csvFilePath)
			if err != nil {
				return err
			}
			defer csvFile.Close()
			csvLines, err := csv.NewReader(csvFile).ReadAll()
			if err != nil {
				return err
			}
			users := []fleet.UserPayload{}
			for _, record := range csvLines[1:] {
				name := record[0]
				email := record[1]
				password, passErr := generateRandomPassword()
				sso, ssoErr := strconv.ParseBool(record[2])
				apiOnly, apiErr := strconv.ParseBool(record[3])
				globalRoleString := record[4]
				teamStrings := strings.Split(record[5], " ")
				if ssoErr != nil {
					return fmt.Errorf("SSO is not a vailed Boolean value: %w", err)
				}
				if apiErr != nil {
					return fmt.Errorf("API Only is not a vailed Boolean value: %w", err)
				}
				if passErr != nil {
					return fmt.Errorf("not able to generate a random password: %w", err)
				}

				var globalRole *string
				var teams []fleet.UserTeam

				if globalRoleString != "" && len(teamStrings) > 0 && teamStrings[0] != "" { //nolint:gocritic // ignore ifElseChain
					return errors.New("Users may not have global_role and teams.")
				} else if globalRoleString == "" && (len(teamStrings) == 0 || teamStrings[0] == "") {
					globalRole = ptr.String(fleet.RoleObserver)
				} else if globalRoleString != "" {
					if !fleet.ValidGlobalRole(globalRoleString) {
						return fmt.Errorf("'%s' is not a valid team role", globalRoleString)
					}
					globalRole = ptr.String(globalRoleString)
				} else {
					for _, t := range teamStrings {
						parts := strings.Split(t, ":")
						if len(parts) != 2 {
							return fmt.Errorf("Unable to parse '%s' as team_id:role", t)
						}
						teamID, err := strconv.Atoi(parts[0])
						if err != nil {
							return fmt.Errorf("Unable to parse team_id: %w", err)
						}
						if !fleet.ValidTeamRole(parts[1]) {
							return fmt.Errorf("'%s' is not a valid team role", parts[1])
						}

						teams = append(teams,
							fleet.UserTeam{Team: fleet.Team{ID: uint(teamID)}, Role: parts[1]}) //nolint:gosec // dismiss G115
					}
				}

				if sso && len(password) > 0 {
					password = ""
				}
				force_reset := !sso
				users = append(users, fleet.UserPayload{
					Password:                 &password,
					Email:                    &email,
					Name:                     &name,
					SSOEnabled:               &sso,
					AdminForcedPasswordReset: &force_reset,
					APIOnly:                  &apiOnly,
					GlobalRole:               globalRole,
					Teams:                    &teams,
				})
			}

			for _, user := range users {
				_, err = client.CreateUser(user)
				if err != nil {
					return fmt.Errorf("Failed to create user: %w", err)
				}
				if *user.SSOEnabled {
					fmt.Printf("Email: %v SSO: %v\n", *user.Email, *user.SSOEnabled)
				} else {
					fmt.Printf("Email: %v Generated password: %v\n", *user.Email, *user.Password)
				}

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

func deleteBulkUsersCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete-users",
		Usage:     "Delete a list of user",
		UsageText: `This command will delete a list of users by importing a CSV file containing a list of emails. Expected columns are:Email`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     csvFlagName,
				Usage:    "csv file with all the users (required)",
				Required: true,
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			csvFilePath := c.String(csvFlagName)

			csvFile, err := os.Open(csvFilePath)
			if err != nil {
				return err
			}
			defer csvFile.Close()
			csvLines, err := csv.NewReader(csvFile).ReadAll()
			if err != nil {
				return err
			}
			for _, user := range csvLines[1:] {
				email := user[0]
				if err := client.DeleteUser(email); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func generateRandomPassword() (string, error) {
	password, err := password.Generate(20, 2, 2, false, true)
	if err != nil {
		return "", err
	}
	return password, nil
}
