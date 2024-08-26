package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	apnsCSRPath         = "fleet-mdm-csr.csr"
	bmPublicKeyCertPath = "fleet-apple-mdm-bm-public-key.crt"
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate certificates and keys required for MDM.",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Subcommands: []*cli.Command{
			generateMDMAppleCommand(),
			generateMDMAppleBMCommand(),
		},
	}
}

func generateMDMAppleCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-apple",
		Aliases: []string{"mdm_apple"},
		Usage:   "Generates certificate signing request (CSR) to turn on MDM features.",
		Flags: []cli.Flag{
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:  "csr",
				Usage: "The output path for the APNs CSR.",
				Value: apnsCSRPath,
			},
		},
		Action: func(c *cli.Context) error {
			csrPath := c.String("csr")

			// get the fleet API client first, so that any login requirement are met
			// before printing the CSR output message.
			client, err := clientFromCLI(c)
			if err != nil {
				fmt.Fprintf(c.App.ErrWriter, "client from CLI: %s\n", err)
				return ErrGeneric
			}

			csr, err := client.RequestAppleCSR()
			if err != nil {
				fmt.Fprintf(c.App.ErrWriter, "requesting APNs CSR: %s\n", err)
				return ErrGeneric
			}

			if err := os.WriteFile(csrPath, csr, defaultFileMode); err != nil {
				fmt.Fprintf(c.App.ErrWriter, "write CSR: %s\n", err)
				return ErrGeneric
			}

			appCfg, err := client.GetAppConfig()
			if err != nil {
				fmt.Fprintf(c.App.ErrWriter, "fetching app config: %s\n", err)
				return ErrGeneric
			}

			fmt.Fprintf(
				c.App.Writer,
				`Success!

Generated your certificate signing request (CSR) at %s

Go to %s/settings/integrations/mdm/apple and follow the steps.
`,
				csrPath,
				appCfg.ServerSettings.ServerURL,
			)

			return nil
		},
	}
}

func generateMDMAppleBMCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-apple-bm",
		Aliases: []string{"mdm_apple_bm"},
		Usage:   "Generate Apple Business Manager public key to enable automatic enrollment for macOS hosts.",
		Flags: []cli.Flag{
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:  "public-key",
				Usage: "The output path for the Apple Business Manager public key certificate.",
				Value: bmPublicKeyCertPath,
			},
		},
		Action: func(c *cli.Context) error {
			publicKeyPath := c.String("public-key")

			// get the fleet API client first, so that any login requirement are met
			// before printing the CSR output message.
			client, err := clientFromCLI(c)
			if err != nil {
				fmt.Fprintf(c.App.ErrWriter, "client from CLI: %s", err)
				return ErrGeneric
			}

			publicKey, err := client.RequestAppleABM()
			if err != nil {
				fmt.Fprintf(c.App.ErrWriter, "requesting ABM public key: %s", err)
				return ErrGeneric
			}

			if err := os.WriteFile(publicKeyPath, publicKey, defaultFileMode); err != nil {
				fmt.Fprintf(c.App.ErrWriter, "write public key: %s", err)
				return ErrGeneric
			}

			appCfg, err := client.GetAppConfig()
			if err != nil {
				fmt.Fprintf(c.App.ErrWriter, "fetching app config: %s", err)
				return ErrGeneric
			}

			fmt.Fprintf(
				c.App.Writer,
				`Success!

Generated your public key at %s

Go to %s/settings/integrations/automatic-enrollment/apple and follow the steps.

`,
				publicKeyPath,
				appCfg.ServerSettings.ServerURL,
			)

			return nil
		},
	}
}
