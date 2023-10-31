package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	kithttp "github.com/go-kit/kit/transport/http"
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
		Name:    "run-command",
		Aliases: []string{"run_command"},
		Usage:   "Run a custom MDM command on one macOS host. Head to Apple's documentation for a list of available commands and example payloads here:  https://developer.apple.com/documentation/devicemanagement/commands_and_queries",
		Flags: []cli.Flag{
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:     "host",
				Usage:    "The host, specified by hostname, uuid, osquery_host_id or node_key, that you want to run the MDM command on.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "payload",
				Usage:    "A path to an XML file containing the raw MDM request payload.",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			// print an error if MDM is not configured
			if err := client.CheckMDMEnabled(); err != nil {
				return err
			}

			hostIdent := c.String("host")
			payloadFile := c.String("payload")
			payload, err := os.ReadFile(payloadFile)
			if err != nil {
				return fmt.Errorf("read payload: %w", err)
			}

			host, err := client.HostByIdentifier(hostIdent)
			if err != nil {
				var nfe service.NotFoundErr
				if errors.As(err, &nfe) {
					return errors.New("The host doesn't exist. Please provide a valid hostname, uuid, osquery_host_id or node_key.")
				}
				var sce kithttp.StatusCoder
				if errors.As(err, &sce) {
					if sce.StatusCode() == http.StatusForbidden {
						return fmt.Errorf("Permission denied. You don't have permission to run an MDM command on this host: %w", err)
					}
				}
				return err
			}

			// TODO(mna): this "On" check is brittle, but looks like it's the only
			// enrollment indication we have right now...
			if host.MDM.EnrollmentStatus == nil || !strings.HasPrefix(*host.MDM.EnrollmentStatus, "On") ||
				host.MDM.Name != fleet.WellKnownMDMFleet {
				return errors.New("Can't run the MDM command because the host doesn't have MDM turned on. Run the following command to see a list of hosts with MDM on: fleetctl get hosts --mdm")
			}

			result, err := client.EnqueueCommand([]string{host.UUID}, payload)
			if err != nil {
				var sce kithttp.StatusCoder
				if errors.As(err, &sce) {
					if sce.StatusCode() == http.StatusForbidden {
						return fmt.Errorf("Permission denied. You don't have permission to run an MDM command on this host: %w", err)
					}
					if sce.StatusCode() == http.StatusUnsupportedMediaType {
						return fmt.Errorf("The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file: %w", err)
					}
				}
				return err
			}

			fmt.Fprintf(c.App.Writer, `
The hosts will run the command the next time it checks into Fleet.

Copy and run this command to see results:

fleetctl get mdm-command-results --id=%v
`, result.CommandUUID)

			return nil
		},
	}
}
