package fleetctl

import (
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/service"
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

If SSO is enabled on the Fleet server, a warning will be displayed. You may still attempt to
log in with email and password, but it will only succeed if your account is not SSO-enabled.

Learn how to authenticate with fleetctl for SSO-enabled accounts:
https://fleetdm.com/guides/fleetctl#users-with-single-sign-on-sso-or-email-two-factor-authentication-2-fa
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

			if ssoSettings, ssoErr := fleet.SSOSettings(); ssoErr == nil && ssoSettings != nil && ssoSettings.SSOEnabled {
				fmt.Fprintf(os.Stderr, "Warning: %s\n\n", ssoAuthInstructions)
			}

			definedAsEnvOnly := func(flagName, envName string) bool {
				cliArgPresent := false
				for _, arg := range os.Args {
					if arg == flagName {
						cliArgPresent = true
					}
				}
				return os.Getenv(envName) != "" && !cliArgPresent
			}

			// Allow interactive entry to discourage passwords in
			// CLI history.
			if flEmail == "" {
				fmt.Println("Log in using the standard Fleet credentials.")
				fmt.Print("Email: ")
				_, err := fmt.Scanln(&flEmail)
				if err != nil {
					return fmt.Errorf("error reading email: %w", err)
				}
			} else if definedAsEnvOnly("--email", "EMAIL") {
				fmt.Printf("Using value of environment variable $EMAIL as email.\n")
			}
			if flPassword == "" {
				fmt.Print("Password: ")
				passBytes, err := terminal.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return fmt.Errorf("error reading password: %w", err)
				}
				fmt.Println()
				flPassword = string(passBytes)
			} else if definedAsEnvOnly("--password", "PASSWORD") {
				fmt.Printf("Using value of environment variable $PASSWORD as password.\n")
			}

			token, err := fleet.Login(flEmail, flPassword)
			if err != nil {
				root := ctxerr.Cause(err)
				switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
				case service.NotSetupErr:
					return err
				}
				fmt.Fprintf(os.Stderr, "\n%s\n", mfaAuthInstructions)
				return fmt.Errorf("Login failed: %w", err)
			}

			configPath, context := c.String("config"), c.String("context")

			if err := setConfigValue(configPath, context, "email", flEmail); err != nil {
				return fmt.Errorf("error setting email for the current context: %w", err)
			}

			if err := setConfigValue(configPath, context, "token", token); err != nil {
				return fmt.Errorf("error setting token for the current context: %w", err)
			}

			fmt.Printf("[+] Fleet login successful and context configured!\n")

			return nil
		},
	}
}
