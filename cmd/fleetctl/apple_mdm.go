package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_ca"
	"github.com/groob/plist"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func appleMDMCommand() *cli.Command {
	return &cli.Command{
		Name:  "apple-mdm",
		Usage: "Apple MDM functionality",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Subcommands: []*cli.Command{
			appleMDMSetupCommand(),
			appleMDMEnrollmentsCommand(),
			appleMDMEnqueueCommandCommand(),
			appleMDMCommandResultsCommand(),
		},
	}
}

func appleMDMSetupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Setup commands for Apple MDM",
		Subcommands: []*cli.Command{
			appleMDMSetupSCEPCommand(),
			appleMDMSetupAPNSCommand(),
			appleMDMSetupDEPCommand(),
		},
	}
}

func appleMDMSetupSCEPCommand() *cli.Command {
	// TODO(lucas): Define workflow when SCEP CA certificate expires.
	var (
		validityYears      int
		cn                 string
		organization       string
		organizationalUnit string
		country            string
	)
	return &cli.Command{
		Name:  "scep",
		Usage: "Create SCEP certificate authority",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "validity-years",
				Usage:       "Validity of the SCEP CA certificate in years",
				Required:    true,
				Destination: &validityYears,
			},
			&cli.StringFlag{
				Name:        "cn",
				Usage:       "Common name to set in the SCEP CA certificate",
				Required:    true,
				Destination: &cn,
			},
			&cli.StringFlag{
				Name:        "organization",
				Usage:       "Organization to set in the SCEP CA certificate",
				Required:    true,
				Destination: &organization,
			},
			&cli.StringFlag{
				Name:        "organizational-unit",
				Usage:       "Organizational unit to set in the SCEP CA certificate",
				Required:    true,
				Destination: &organizationalUnit,
			},
			&cli.StringFlag{
				Name:        "country",
				Usage:       "Country to set in the SCEP CA certificate",
				Required:    true,
				Destination: &country,
			},
		},
		Action: func(c *cli.Context) error {
			certPEM, keyPEM, err := scep_ca.Create(validityYears, cn, organization, organizationalUnit, country)
			if err != nil {
				return fmt.Errorf("creating SCEP CA: %w", err)
			}
			const (
				certPath = "fleet-mdm-apple-scep.crt"
				keyPath  = "fleet-mdm-apple-scep.key"
			)
			if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
				return fmt.Errorf("write %s: %w", certPath, err)
			}
			if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
				return fmt.Errorf("write %s: %w", keyPath, err)
			}
			fmt.Printf("Successfully generated SCEP CA: %s, %s.\n", certPath, keyPath)
			fmt.Printf("Set FLEET_MDM_APPLE_SCEP_CA_CERT_PEM=$(cat %s) FLEET_MDM_APPLE_SCEP_CA_KEY_PEM=$(cat %s) when running Fleet.\n", certPath, keyPath)
			return nil
		},
	}
}

func appleMDMSetupAPNSCommand() *cli.Command {
	return &cli.Command{
		Name:  "apns",
		Usage: "Commands to setup APNS certificate",
		Subcommands: []*cli.Command{
			appleMDMSetupAPNSInitCommand(),
			appleMDMSetupAPNSFinalizeCommand(),
		},
	}
}

func appleMDMSetupAPNSInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Start APNS certificate configuration",
		Action: func(c *cli.Context) error {
			// TODO(lucas): Implement command.
			fmt.Println("Not implemented yet.")
			return nil
		},
	}
}

func appleMDMSetupAPNSFinalizeCommand() *cli.Command {
	var encryptedReq string
	return &cli.Command{
		Name:  "finalize",
		Usage: "Finalize APNS certificate configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "encrypted-req",
				Usage:       "File path of the encrypted .req p7m file",
				Destination: &encryptedReq,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			// TODO(lucas): Implement command.
			fmt.Println("Not implemented yet.")
			return nil
		},
	}
}

func appleMDMSetupDEPCommand() *cli.Command {
	return &cli.Command{
		Name:  "dep",
		Usage: "Configure DEP token",
		Subcommands: []*cli.Command{
			appleMDMSetDEPTokenInitCommand(),
			appleMDMSetDEPTokenFinalizeCommand(),
		},
	}
}

func appleMDMSetDEPTokenInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Start DEP token configuration",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			// TODO(lucas): Check validity days default value.
			const (
				cn           = "fleet"
				validityDays = 1
				pemCertPath  = "fleet-mdm-apple-dep.crt"
				pemKeyPath   = "fleet-mdm-apple-dep.key"
			)
			key, cert, err := tokenpki.SelfSignedRSAKeypair(cn, validityDays)
			if err != nil {
				return fmt.Errorf("generate encryption keypair: %w", err)
			}
			pemCert := tokenpki.PEMCertificate(cert.Raw)
			pemKey := tokenpki.PEMRSAPrivateKey(key)
			if err := os.WriteFile(pemCertPath, pemCert, defaultFileMode); err != nil {
				return fmt.Errorf("write certificate: %w", err)
			}
			if err := os.WriteFile(pemKeyPath, pemKey, defaultFileMode); err != nil {
				return fmt.Errorf("write private key: %w", err)
			}
			fmt.Printf("Successfully generated DEP public and private key: %s, %s\n", pemCertPath, pemKeyPath)
			fmt.Printf("Upload %s to your Apple Business MDM server. (Don't forget to click \"Save\" after uploading it.)", pemCertPath)
			return nil
		},
	}
}

func appleMDMSetDEPTokenFinalizeCommand() *cli.Command {
	var (
		pemCertPath        string
		pemKeyPath         string
		encryptedTokenPath string
	)
	return &cli.Command{
		Name:  "finalize",
		Usage: "Finalize DEP token configuration for an automatic enrollment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "certificate",
				Usage:       "Path to the certificate generated in the init step",
				Destination: &pemCertPath,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "private-key",
				Usage:       "Path to the private key file generated in the init step",
				Destination: &pemKeyPath,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "encrypted-token",
				Usage:       "Path to the encrypted token file downloaded from Apple Business (*.p7m)",
				Destination: &encryptedTokenPath,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			pemCert, err := os.ReadFile(pemCertPath)
			if err != nil {
				return fmt.Errorf("read certificate: %w", err)
			}
			depCert, err := tokenpki.CertificateFromPEM(pemCert)
			if err != nil {
				return fmt.Errorf("parse certificate: %w", err)
			}
			pemKey, err := os.ReadFile(pemKeyPath)
			if err != nil {
				return fmt.Errorf("read private key: %w", err)
			}
			depKey, err := tokenpki.RSAKeyFromPEM(pemKey)
			if err != nil {
				return fmt.Errorf("parse private key: %w", err)
			}
			encryptedToken, err := os.ReadFile(encryptedTokenPath)
			if err != nil {
				return fmt.Errorf("read encrypted token: %w", err)
			}
			token, err := tokenpki.DecryptTokenJSON(encryptedToken, depCert, depKey)
			if err != nil {
				return fmt.Errorf("decrypt token: %w", err)
			}
			tokenPath := "fleet-mdm-apple-dep.token"
			if err := os.WriteFile(tokenPath, token, defaultFileMode); err != nil {
				return fmt.Errorf("write token file: %w", err)
			}
			fmt.Printf("Successfully generated token file: %s.\n", tokenPath)
			// TODO(lucas): Delete pemCertPath, pemKeyPath and encryptedTokenPath files?
			fmt.Printf("Set FLEET_MDM_APPLE_DEP_TOKEN=$(cat %s) when running Fleet.\n", tokenPath)
			return nil
		},
	}
}

func appleMDMEnrollmentsCommand() *cli.Command {
	return &cli.Command{
		Name:  "enrollments",
		Usage: "Commands to manage enrollments",
		Subcommands: []*cli.Command{
			appleMDMEnrollmentsCreateAutomaticCommand(),
			appleMDMEnrollmentsCreateManualCommand(),
			appleMDMEnrollmentsDeleteCommand(),
			appleMDMEnrollmentsListCommand(),
		},
	}
}

func appleMDMEnrollmentsCreateAutomaticCommand() *cli.Command {
	var (
		enrollmentName string
		depConfigPath  string
	)
	return &cli.Command{
		Name:  "create-automatic",
		Usage: "Create a new automatic enrollment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of the automatic enrollment",
				Destination: &enrollmentName,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "profile",
				Usage:       "JSON file with fields defined in https://developer.apple.com/documentation/devicemanagement/profile",
				Destination: &depConfigPath,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			profile, err := os.ReadFile(depConfigPath)
			if err != nil {
				return fmt.Errorf("read dep profile: %w", err)
			}
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			depProfile := json.RawMessage(profile)
			enrollment, url, err := fleet.CreateEnrollment(enrollmentName, &depProfile)
			if err != nil {
				return fmt.Errorf("create enrollment: %w", err)
			}
			fmt.Printf("Automatic enrollment created, URL: %s, id: %d\n", url, enrollment.ID)
			return nil
		},
	}
}

func appleMDMEnrollmentsCreateManualCommand() *cli.Command {
	var enrollmentName string
	return &cli.Command{
		Name:  "create-manual",
		Usage: "Create a new manual enrollment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of the manual enrollment",
				Destination: &enrollmentName,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			enrollment, url, err := fleet.CreateEnrollment(enrollmentName, nil)
			if err != nil {
				return fmt.Errorf("create enrollment: %w", err)
			}
			fmt.Printf("Manual enrollment created, URL: %s, id: %d\n", url, enrollment.ID)
			return nil
		},
	}
}

func appleMDMEnrollmentsDeleteCommand() *cli.Command {
	var enrollmentID uint
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an enrollment",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:        "id",
				Usage:       "Identifier of the enrollment",
				Destination: &enrollmentID,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			// TODO(lucas): Implement command.
			fmt.Println("Not implemented yet.")
			return nil
		},
	}
}

func appleMDMEnrollmentsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all enrollments",
		Action: func(c *cli.Context) error {
			// TODO(lucas): Implement command.
			fmt.Println("Not implemented yet.")
			return nil
		},
	}
}

func appleMDMEnqueueCommandCommand() *cli.Command {
	return &cli.Command{
		Name:  "enqueue-command",
		Usage: "Enqueue an MDM command.",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "device-ids",
				Usage:    "The device IDs of the devices to send the MDM command to. This is the same as the hardware UUID.",
				Required: true,
			},
			&cli.StringFlag{
				Name:      "command-payload",
				Usage:     "A plist file containing the raw MDM command payload. Note that a new CommandUUID will be generated automatically. See https://developer.apple.com/documentation/devicemanagement/commands_and_queries for available commands.",
				TakesFile: true,
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := c.StringSlice("device-ids")
			if len(deviceIDs) == 0 {
				return fmt.Errorf("must provide at least one device ID")
			}

			payloadFilename := c.String("command-payload")
			payloadBytes, err := os.ReadFile(payloadFilename)
			if err != nil {
				return fmt.Errorf("read payload: %w", err)
			}

			result, err := fleet.EnqueueCommand(deviceIDs, payloadBytes)
			if err != nil {
				return err
			}

			commandUUID := result.CommandUUID
			fmt.Printf("Command UUID: %s\n", commandUUID)

			return nil
		},
		Subcommands: []*cli.Command{
			appleMDMEnqueueCommandInstallProfileCommand(),
			appleMDMEnqueueCommandRemoveProfileCommand(),
			appleMDMEnqueueCommandProfileListCommand(),
		},
	}
}

func appleMDMEnqueueCommandInstallProfileCommand() *cli.Command {
	return &cli.Command{
		Name: "InstallProfile",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "mobileconfig",
				Usage:    "The mobileconfig file containing the profile to install.",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := c.StringSlice("device-ids")
			if len(deviceIDs) == 0 {
				return fmt.Errorf("must provide at least one device ID")
			}

			profilePayloadFilename := c.String("mobileconfig")
			profilePayloadBytes, err := os.ReadFile(profilePayloadFilename)
			if err != nil {
				return fmt.Errorf("read payload: %w", err)
			}

			payload := &apple.CommandPayload{
				Command: apple.InstallProfile{
					RequestType: "InstallProfile",
					Payload:     profilePayloadBytes,
				},
			}

			// convert to xml using tabs for indentation
			payloadBytes, err := plist.MarshalIndent(payload, plist.XMLFormat, "	")
			if err != nil {
				return fmt.Errorf("marshal command payload plist: %w", err)
			}

			result, err := fleet.EnqueueCommand(deviceIDs, payloadBytes)
			if err != nil {
				return err
			}

			commandUUID := result.CommandUUID
			fmt.Printf("Command UUID: %s\n", commandUUID)

			return nil
		},
		Subcommands: []*cli.Command{},
	}
}

func appleMDMEnqueueCommandRemoveProfileCommand() *cli.Command {
	return &cli.Command{
		Name: "RemoveProfile",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "identifier",
				Usage:    "The PayloadIdentifier value for the profile to remove eg cis.macOSBenchmark.section2.SecureKeyboard.",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := c.StringSlice("device-ids")
			if len(deviceIDs) == 0 {
				return fmt.Errorf("must provide at least one device ID")
			}

			identifier := c.String("identifier")

			payload := &apple.CommandPayload{
				Command: apple.RemoveProfile{
					RequestType: "RemoveProfile",
					Identifier:  identifier,
				},
			}

			// convert to xml using tabs for indentation
			payloadBytes, err := plist.MarshalIndent(payload, plist.XMLFormat, "	")
			if err != nil {
				return fmt.Errorf("marshal command payload plist: %w", err)
			}
			fmt.Println(string(payloadBytes))

			result, err := fleet.EnqueueCommand(deviceIDs, payloadBytes)
			if err != nil {
				return err
			}

			commandUUID := result.CommandUUID
			fmt.Printf("Command UUID: %s\n", commandUUID)

			return nil
		},
		Subcommands: []*cli.Command{},
	}
}

func appleMDMEnqueueCommandProfileListCommand() *cli.Command {
	return &cli.Command{
		Name: "ProfileList",
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := c.StringSlice("device-ids")
			if len(deviceIDs) == 0 {
				return fmt.Errorf("must provide at least one device ID")
			}

			payload := &apple.CommandPayload{
				Command: apple.ProfileList{
					RequestType: "ProfileList",
				},
			}

			// convert to xml using tabs for indentation
			payloadBytes, err := plist.MarshalIndent(payload, plist.XMLFormat, "	")
			if err != nil {
				return fmt.Errorf("marshal command payload plist: %w", err)
			}

			result, err := fleet.EnqueueCommand(deviceIDs, payloadBytes)
			if err != nil {
				return err
			}

			commandUUID := result.CommandUUID
			fmt.Printf("Command UUID: %s\n", commandUUID)

			return nil
		},
		Subcommands: []*cli.Command{},
	}
}

func appleMDMCommandResultsCommand() *cli.Command {
	return &cli.Command{
		Name:  "command-results",
		Usage: "Get MDM command results",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "command-uuid",
				Usage:    "The command uuid.",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			commandUUID := c.String("command-uuid")

			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			results, err := fleet.MDMAppleGetCommandResults(commandUUID)
			if err != nil {
				return err
			}

			// format output as a table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetRowLine(true)
			table.SetHeader([]string{"Device ID", "Status", "Result"})
			table.SetAutoWrapText(false)
			table.SetRowLine(true)

			for deviceID, result := range results {
				table.Append([]string{deviceID, result.Status, string(result.Result)})
			}

			table.Render()

			return nil
		},
	}
}
