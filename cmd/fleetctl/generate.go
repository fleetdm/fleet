package main

import (
	"fmt"
	"os"

	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/urfave/cli/v2"
)

const (
	apnsKeyPath         = "fleet-mdm-apple-apns.key"
	scepCACertPath      = "fleet-mdm-apple-scep.crt"
	scepCAKeyPath       = "fleet-mdm-apple-scep.key"
	bmPublicKeyCertPath = "fleet-apple-mdm-bm-public-key.crt"
	bmPrivateKeyPath    = "fleet-apple-mdm-bm-private.key"
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate certificates and keys required for MDM",
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
		Usage:   "Generates certificate signing request (CSR) and key for Apple Push Notification Service (APNs) and certificate and key for Simple Certificate Enrollment Protocol (SCEP) to turn on MDM features.",
		Flags: []cli.Flag{
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:     "email",
				Usage:    "The email address to send the signed APNS csr to.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "org",
				Usage:    "The organization requesting the signed APNS csr.",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "apns-key",
				Usage: "The output path for the APNs private key.",
				Value: apnsKeyPath,
			},
			&cli.StringFlag{
				Name:  "scep-cert",
				Usage: "The output path for the SCEP CA certificate.",
				Value: scepCACertPath,
			},
			&cli.StringFlag{
				Name:  "scep-key",
				Usage: "The output path for the SCEP CA private key.",
				Value: scepCAKeyPath,
			},
		},
		Action: func(c *cli.Context) error {
			email := c.String("email")
			org := c.String("org")
			apnsKeyPath := c.String("apns-key")
			scepCACertPath := c.String("scep-cert")
			scepCAKeyPath := c.String("scep-key")

			// get the fleet API client first, so that any login requirement are met
			// before printing the CSR output message.
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			fmt.Fprintf(
				c.App.Writer,
				`Sending certificate signing request (CSR) for Apple Push Notification service (APNs) to %s...
Generating APNs key, Simple Certificate Enrollment Protocol (SCEP) certificate, and SCEP key...

`,
				email,
			)

			csr, err := client.RequestAppleCSR(email, org)
			if err != nil {
				return err
			}

			if err := os.WriteFile(apnsKeyPath, csr.APNsKey, defaultFileMode); err != nil {
				return fmt.Errorf("failed to write APNs private key: %w", err)
			}
			if err := os.WriteFile(scepCACertPath, csr.SCEPCert, defaultFileMode); err != nil {
				return fmt.Errorf("failed to write SCEP CA certificate: %w", err)
			}
			if err := os.WriteFile(scepCAKeyPath, csr.SCEPKey, defaultFileMode); err != nil {
				return fmt.Errorf("failed to write SCEP CA private key: %w", err)
			}

			fmt.Fprintf(
				c.App.Writer,
				`Success!

Generated your APNs key at %s

Generated your SCEP certificate at %s

Generated your SCEP key at %s

Go to your email to download a CSR from Fleet. Then, visit https://identity.apple.com/pushcert to upload the CSR. You should receive an APNs certificate in return from Apple.

Next, use the generated certificates to deploy Fleet with `+"`mdm`"+` configuration: https://fleetdm.com/docs/deploying/configuration#mobile-device-management-mdm
`,
				apnsKeyPath,
				scepCACertPath,
				scepCAKeyPath,
			)

			return nil
		},
	}
}

func generateMDMAppleBMCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-apple-bm",
		Aliases: []string{"mdm_apple_bm"},
		Usage:   "Generate Apple Business Manager public and private keys to enable automatic enrollment for macOS hosts.",
		Flags: []cli.Flag{
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:  "public-key",
				Usage: "The output path for the Apple Business Manager public key certificate.",
				Value: bmPublicKeyCertPath,
			},
			&cli.StringFlag{
				Name:  "private-key",
				Usage: "The output path for the Apple Business Manager private key.",
				Value: bmPrivateKeyPath,
			},
		},
		Action: func(c *cli.Context) error {
			publicKeyPath := c.String("public-key")
			privateKeyPath := c.String("private-key")

			publicKeyPEM, privateKeyPEM, err := apple_mdm.NewDEPKeyPairPEM()
			if err != nil {
				return fmt.Errorf("generate key pair: %w", err)
			}

			if err := os.WriteFile(publicKeyPath, publicKeyPEM, defaultFileMode); err != nil {
				return fmt.Errorf("write public key: %w", err)
			}

			if err := os.WriteFile(privateKeyPath, privateKeyPEM, defaultFileMode); err != nil {
				return fmt.Errorf("write private key: %w", err)
			}

			fmt.Fprintf(
				c.App.Writer,
				`Success!

Generated your public key at %s

Generated your private key at %s

Visit https://business.apple.com/ and create a new MDM server with the public key. Then, download the new MDM server's token.

Next, deploy Fleet with with `+"`mdm`"+` configuration: https://fleetdm.com/docs/deploying/configuration#mobile-device-management-mdm
`,
				publicKeyPath,
				privateKeyPath,
			)

			return nil
		},
	}
}
