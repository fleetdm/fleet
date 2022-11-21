package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_ca"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func appleMDMCommand() *cli.Command {
	return &cli.Command{
		Name:  "apple-mdm",
		Usage: "Apple MDM functionality",
		// Apple MDM functionality will be merged but hidden until we release MVP publicly.
		Hidden: true,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Subcommands: []*cli.Command{
			appleMDMSetupCommand(),
			appleMDMEnrollmentProfilesCommand(),
			appleMDMEnqueueCommandCommand(),
			appleMDMDEPCommand(),
			appleMDMDevicesCommand(),
			appleMDMCommandResultsCommand(),
			appleMDMInstallersCommand(),
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
			const (
				cn = "fleet"
				// Setting validityDays to 10 in case user doing the init command
				// is different than user uploading to Apple.
				// (Though we've heard from other users that Apple doesn't really check
				// the expiration of this public key.)
				validityDays = 10
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
			//nolint:gosec // G101: no credentials, just the file name.
			tokenPath := "fleet-mdm-apple-dep.token"
			if err := os.WriteFile(tokenPath, token, defaultFileMode); err != nil {
				return fmt.Errorf("write token file: %w", err)
			}
			fmt.Printf("Successfully generated token file: %s.\n", tokenPath)
			fmt.Printf("Set FLEET_MDM_APPLE_DEP_TOKEN=$(cat %s) when running Fleet.\n", tokenPath)
			return nil
		},
	}
}

func appleMDMEnrollmentProfilesCommand() *cli.Command {
	return &cli.Command{
		Name:  "enrollment-profiles",
		Usage: "Commands to manage enrollment profiles",
		Subcommands: []*cli.Command{
			appleMDMEnrollmentProfilesCreateAutomaticCommand(),
			appleMDMEnrollmentProfilesCreateManualCommand(),
			appleMDMEnrollmentProfilesListCommand(),
		},
	}
}

func appleMDMEnrollmentProfilesCreateAutomaticCommand() *cli.Command {
	var depProfilePath string
	return &cli.Command{
		Name:  "create-automatic",
		Usage: "Create an automatic enrollment profile",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dep-profile",
				Usage:       "JSON file with fields defined in https://developer.apple.com/documentation/devicemanagement/profile",
				Destination: &depProfilePath,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			profile, err := os.ReadFile(depProfilePath)
			if err != nil {
				return fmt.Errorf("read dep profile: %w", err)
			}
			client, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			depProfile := json.RawMessage(profile)
			enrollmentProfile, err := client.CreateEnrollmentProfile(fleet.MDMAppleEnrollmentTypeAutomatic, &depProfile)
			if err != nil {
				return fmt.Errorf("create enrollment profile: %w", err)
			}
			fmt.Printf("Automatic enrollment profile created, ID: %d\n", enrollmentProfile.ID)
			return nil
		},
	}
}

func appleMDMEnrollmentProfilesCreateManualCommand() *cli.Command {
	return &cli.Command{
		Name:  "create-manual",
		Usage: "Create a manual enrollment profile",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			enrollmentProfile, err := client.CreateEnrollmentProfile(fleet.MDMAppleEnrollmentTypeManual, nil)
			if err != nil {
				return fmt.Errorf("create enrollment profile: %w", err)
			}
			fmt.Printf("Manual enrollment profile created, URL: %s.\n", enrollmentProfile.EnrollmentURL)
			return nil
		},
	}
}

func appleMDMEnrollmentProfilesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all enrollments",
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			enrollments, err := fleet.ListEnrollments()
			if err != nil {
				return fmt.Errorf("create enrollment: %w", err)
			}

			// format output as a table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetRowLine(true)
			table.SetHeader([]string{"ID", "Type", "DEP Profile", "Enrollment URL"})
			table.SetAutoWrapText(false)
			table.SetRowLine(true)

			for _, enrollment := range enrollments {
				var depProfile string
				if enrollment.DEPProfile != nil {
					depProfile = string(*enrollment.DEPProfile)
				}
				table.Append([]string{
					strconv.FormatUint(uint64(enrollment.ID), 10),
					string(enrollment.Type),
					depProfile,
					enrollment.EnrollmentURL,
				})
			}

			table.Render()
			return nil
		},
	}
}

func appleMDMEnqueueCommandCommand() *cli.Command {
	return &cli.Command{
		Name:  "enqueue-command",
		Usage: "Enqueue an MDM command. See the results using the command-results command and passing the command UUID that is returned from this command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.StringFlag{
				Name:  "command-payload",
				Usage: "A plist file containing the raw MDM command payload. Note that a new CommandUUID will be generated automatically. See https://developer.apple.com/documentation/devicemanagement/commands_and_queries for available commands.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			payloadFilename := c.String("command-payload")
			if payloadFilename == "" {
				return errors.New("must provide a command payload file")
			}
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
			appleMDMEnqueueCommandSimpleCommand("ProfileList"),
			appleMDMEnqueueCommandRemoveProfileCommand(),
			appleMDMEnqueueCommandInstallEnterpriseApplicationCommand(),
			appleMDMEnqueueCommandSimpleCommand("ProvisioningProfileList"),
			appleMDMEnqueueCommandSimpleCommand("CertificateList"),
			appleMDMEnqueueCommandSimpleCommand("SecurityInfo"),
			appleMDMEnqueueCommandSimpleCommand("RestartDevice"),
			appleMDMEnqueueCommandSimpleCommand("ShutdownDevice"),
			appleMDMEnqueueCommandSimpleCommand("StopMirroring"),
			appleMDMEnqueueCommandSimpleCommand("ClearRestrictionsPassword"),
			appleMDMEnqueueCommandSimpleCommand("UserList"),
			appleMDMEnqueueCommandSimpleCommand("LogOutUser"),
			appleMDMEnqueueCommandSimpleCommand("PlayLostModeSound"),
			appleMDMEnqueueCommandSimpleCommand("DisableLostMode"),
			appleMDMEnqueueCommandSimpleCommand("DeviceLocation"),
			appleMDMEnqueueCommandSimpleCommand("ManagedMediaList"),
			appleMDMEnqueueCommandSimpleCommand("DeviceConfigured"),
			appleMDMEnqueueCommandSimpleCommand("AvailableOSUpdates"),
			appleMDMEnqueueCommandSimpleCommand("NSExtensionMappings"),
			appleMDMEnqueueCommandSimpleCommand("OSUpdateStatus"),
			appleMDMEnqueueCommandSimpleCommand("EnableRemoteDesktop"),
			appleMDMEnqueueCommandSimpleCommand("DisableRemoteDesktop"),
			appleMDMEnqueueCommandSimpleCommand("ActivationLockBypassCode"),
			appleMDMEnqueueCommandSimpleCommand("ScheduleOSUpdateScan"),
			appleMDMEnqueueCommandEraseDeviceCommand(),
			appleMDMEnqueueCommandDeviceLockCommand(),
			appleMDMEnqueueCommandDeviceInformationCommand(),
		},
	}
}

func appleMDMEnqueueCommandInstallProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "InstallProfile",
		Usage: "Enqueue the InstallProfile MDM command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.StringFlag{
				Name:  "mobileconfig",
				Usage: "The mobileconfig file containing the profile to install.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			profilePayloadFilename := c.String("mobileconfig")
			if profilePayloadFilename == "" {
				return errors.New("must provide a mobileprofile payload")
			}
			profilePayloadBytes, err := os.ReadFile(profilePayloadFilename)
			if err != nil {
				return fmt.Errorf("read payload: %w", err)
			}

			payload := &mdm.CommandPayload{
				Command: &mdm.Command{
					RequestType: "InstallProfile",
					InstallProfile: &mdm.InstallProfile{
						Payload: profilePayloadBytes,
					},
				},
			}

			return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
		},
	}
}

func appleMDMEnqueueCommandRemoveProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "RemoveProfile",
		Usage: "Enqueue the RemoveProfile MDM command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.StringFlag{
				Name:  "identifier",
				Usage: "The PayloadIdentifier value for the profile to remove eg cis.macOSBenchmark.section2.SecureKeyboard.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			identifier := c.String("identifier")
			if identifier == "" {
				return errors.New("must provide the identifier of the profile")
			}

			payload := &mdm.CommandPayload{
				Command: &mdm.Command{
					RequestType: "RemoveProfile",
					RemoveProfile: &mdm.RemoveProfile{
						Identifier: identifier,
					},
				},
			}

			return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
		},
	}
}

func appleMDMEnqueueCommandSimpleCommand(name string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: fmt.Sprintf("Enqueue the %s MDM command.", name),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
		},
		Action: func(c *cli.Context) error {
			return runSimpleCommand(c, name)
		},
	}
}

// runSimpleCommand runs commands that do not have any extra arguments, like RestartDevice.
func runSimpleCommand(c *cli.Context, name string) error {
	fleet, err := clientFromCLI(c)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	deviceIDs := strings.Split(c.String("device-ids"), ",")
	if len(deviceIDs) == 0 {
		return errors.New("must provide at least one device ID")
	}

	payload := &mdm.CommandPayload{
		Command: &mdm.Command{
			RequestType: name,
		},
	}
	return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
}

func enqueueCommandAndPrintHelp(fleet *service.Client, deviceIDs []string, payload *mdm.CommandPayload) error {
	// convert to xml using tabs for indentation
	payloadBytes, err := plist.MarshalIndent(payload, "	")
	if err != nil {
		return fmt.Errorf("marshal command payload plist: %w", err)
	}

	result, err := fleet.EnqueueCommand(deviceIDs, payloadBytes)
	if err != nil {
		return fmt.Errorf("enqueue command: %w", err)
	}

	commandUUID := result.CommandUUID
	fmt.Printf("Command UUID: %s\n", commandUUID)
	fmt.Printf("Use `fleetctl apple-mdm command-results --command-uuid %s` to get results.\n", commandUUID)
	return nil
}

func appleMDMEnqueueCommandEraseDeviceCommand() *cli.Command {
	return &cli.Command{
		Name:  "EraseDevice",
		Usage: "Enqueue the EraseDevice MDM command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.StringFlag{
				Name:  "pin",
				Usage: "The six-character PIN for Find My.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			pin := c.String("pin")
			if len(pin) != 6 {
				return errors.New("must provide a six-character PIN for Find My")
			}

			payload := &mdm.CommandPayload{
				Command: &mdm.Command{
					RequestType: "EraseDevice",
					EraseDevice: &mdm.EraseDevice{
						PIN: pin,
					},
				},
			}

			return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
		},
	}
}

func appleMDMEnqueueCommandDeviceLockCommand() *cli.Command {
	return &cli.Command{
		Name:  "DeviceLock",
		Usage: "Enqueue the DeviceLock MDM command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.StringFlag{
				Name:  "pin",
				Usage: "The six-character PIN for Find My.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			pin := c.String("pin")
			if len(pin) != 6 {
				return errors.New("must provide a six-character PIN for Find My")
			}

			payload := &mdm.CommandPayload{
				Command: &mdm.Command{
					RequestType: "DeviceLock",
					DeviceLock: &mdm.DeviceLock{
						PIN: pin,
					},
				},
			}

			return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
		},
	}
}

func appleMDMEnqueueCommandDeviceInformationCommand() *cli.Command {
	return &cli.Command{
		Name:  "DeviceInformation",
		Usage: "Enqueue the DeviceInformation MDM command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.StringFlag{
				Name:  "queries",
				Usage: "An array of query dictionaries to get information about a device. See https://developer.apple.com/documentation/devicemanagement/deviceinformationcommand/command/queries.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			queries := strings.Split(c.String("queries"), ",")
			if len(queries) == 0 {
				return errors.New("must provide queries for the device")
			}

			payload := &mdm.CommandPayload{
				Command: &mdm.Command{
					RequestType: "DeviceInformation",
					DeviceInformation: &mdm.DeviceInformation{
						Queries: queries,
					},
				},
			}

			return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
		},
	}
}

func appleMDMEnqueueCommandInstallEnterpriseApplicationCommand() *cli.Command {
	return &cli.Command{
		Name:  "InstallEnterpriseApplication",
		Usage: "Enqueue the InstallEnterpriseApplication MDM command.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "device-ids",
				Usage: "Comma separated device IDs to send the MDM command to. This is the same as the hardware UUID.",
			},
			&cli.UintFlag{
				Name:  "installer-id",
				Usage: "ID of the installer to install on the target devices.",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			deviceIDs := strings.Split(c.String("device-ids"), ",")
			if len(deviceIDs) == 0 {
				return errors.New("must provide at least one device ID")
			}

			installerID := c.Uint("installer-id")
			if installerID == 0 {
				return errors.New("must provide an installer ID")
			}
			installer, err := fleet.MDMAppleGetInstallerDetails(installerID)
			if err != nil {
				return fmt.Errorf("get installer: %w", err)
			}

			var m appmanifest.Manifest
			if err := plist.NewDecoder(bytes.NewReader([]byte(installer.Manifest))).Decode(&m); err != nil {
				return fmt.Errorf("decode manifest: %w", err)
			}

			payload := &mdm.CommandPayload{
				Command: &mdm.Command{
					RequestType: "InstallEnterpriseApplication",
					InstallEnterpriseApplication: &mdm.InstallEnterpriseApplication{
						Manifest: &m,
					},
				},
			}

			return enqueueCommandAndPrintHelp(fleet, deviceIDs, payload)
		},
	}
}

func appleMDMDevicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "devices",
		Usage: "Inspect enrolled devices",
		Subcommands: []*cli.Command{
			appleMDMDevicesListCommand(),
		},
	}
}

func appleMDMDevicesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all devices",
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			devices, err := fleet.MDMAppleListDevices()
			if err != nil {
				return err
			}

			// format output as a table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetRowLine(true)
			table.SetHeader([]string{"Device ID", "Serial Number", "Enrolled"})
			table.SetAutoWrapText(false)
			table.SetRowLine(true)

			for _, device := range devices {
				table.Append([]string{
					device.ID,
					device.SerialNumber,
					strconv.FormatBool(device.Enabled),
				})
			}

			table.Render()

			return nil
		},
	}
}

func appleMDMDEPCommand() *cli.Command {
	return &cli.Command{
		Name:  "dep",
		Usage: "Device Enrollment Program commands",
		Subcommands: []*cli.Command{
			appleMDMDEPListCommand(),
		},
	}
}

func appleMDMDEPListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all DEP devices from the linked MDM server in Apple Business Manager",
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			devices, err := fleet.DEPListDevices()
			if err != nil {
				return err
			}

			// format output as a table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetRowLine(true)
			table.SetHeader([]string{
				"Serial Number",
				"OS",
				"Family",
				"Model",
				"Description",
				"Color",
				"Profile Status",
				"Profile UUID",
				"Profile Assign Time",
				"Profile Push Time",
				"Device Assigned Date",
				"Assigned By",
			})
			table.SetAutoWrapText(false)
			table.SetRowLine(true)

			const timeFmt = "2006-01-02T15:04:05Z"
			for _, device := range devices {
				table.Append([]string{
					device.SerialNumber,
					device.OS,
					device.DeviceFamily,
					device.Model,
					device.Description,
					device.Color,
					device.ProfileStatus,
					device.ProfileUUID,
					device.ProfileAssignTime.Format(timeFmt),
					device.ProfilePushTime.Format(timeFmt),
					device.DeviceAssignedDate.Format(timeFmt),
					device.DeviceAssignedBy,
				})
			}

			table.Render()

			return nil
		},
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
				xml := bytes.ReplaceAll(result.Result, []byte{'\t'}, []byte{' '})
				table.Append([]string{deviceID, result.Status, string(xml)})
			}

			table.Render()

			return nil
		},
	}
}

func appleMDMInstallersCommand() *cli.Command {
	return &cli.Command{
		Name:  "installers",
		Usage: "Commands to manage macOS installers",
		Subcommands: []*cli.Command{
			appleMDMInstallersUploadCommand(),
			appleMDMInstallersListCommand(),
			appleMDMInstallersDeleteCommand(),
		},
	}
}

func appleMDMInstallersUploadCommand() *cli.Command {
	var path string
	return &cli.Command{
		Name:  "upload",
		Usage: "Upload an Apple installer to Fleet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "path",
				Usage:       "Path to the installer",
				Destination: &path,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			fp, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("open path %q: %w", path, err)
			}
			defer fp.Close()
			installerID, err := fleet.UploadMDMAppleInstaller(c.Context, filepath.Base(path), fp)
			if err != nil {
				return fmt.Errorf("upload installer: %w", err)
			}
			fmt.Printf("Installer uploaded successfully, id=%d", installerID)
			return nil
		},
	}
}

func appleMDMInstallersListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all Apple installers",
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			installers, err := fleet.ListMDMAppleInstallers()
			if err != nil {
				return fmt.Errorf("list installers: %w", err)
			}
			// format output as a table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetRowLine(true)
			table.SetHeader([]string{"ID", "Name", "Manifest", "URL"})
			table.SetAutoWrapText(false)
			table.SetRowLine(true)

			for _, installer := range installers {
				manifest := strings.ReplaceAll(installer.Manifest, "\t", " ")
				table.Append([]string{strconv.FormatUint(uint64(installer.ID), 10), installer.Name, manifest, installer.URL})
			}

			table.Render()
			return nil
		},
	}
}

func appleMDMInstallersDeleteCommand() *cli.Command {
	var installerID uint
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an Apple installer",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:        "id",
				Usage:       "Identifier of the installer",
				Destination: &installerID,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			if err := fleet.MDMDeleteAppleInstaller(installerID); err != nil {
				return fmt.Errorf("delete installer: %w", err)
			}
			return nil
		},
	}
}
