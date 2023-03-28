package main

import (
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

func mdmCommand() *cli.Command {
	return &cli.Command{
		Name:  "mdm",
		Usage: "Run MDM commands against your hosts",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Subcommands: []*cli.Command{
			mdmRunCommand(),
		},
	}
}

func mdmRunCommand() *cli.Command {
	return &cli.Command{
		Name:  "run-command",
		Usage: "Run a custom MDM command on one macOS host. Head to Apple's documentation for a list of available commands and example payloads here:  https://developer.apple.com/documentation/devicemanagement/commands_and_queries",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Usage: "The host, specified by hostname, that you want to run the MDM command on.",
			},
			&cli.StringFlag{
				Name:  "payload",
				Usage: "A path to an XML file containing the raw MDM request payload.",
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			// print an error if MDM is not configured
			if err := checkMDMEnabled(client); err != nil {
				return err
			}
			fmt.Println("Running a command...")
			/*
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
			*/

			return nil
		},
	}
}

func checkMDMEnabled(client *service.Client) error {
	appCfg, err := client.GetAppConfig()
	if err != nil {
		return err
	}
	if !appCfg.MDM.EnabledAndConfigured {
		return errors.New("MDM features aren't turned on. Use `fleetctl generate mdm-apple` and then `fleet serve` with `mdm` configuration to turn on MDM features.")
	}
	return nil
}
