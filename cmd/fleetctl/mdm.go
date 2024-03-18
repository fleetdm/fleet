package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
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
			mdmLockCommand(),
			mdmUnlockCommand(),
			mdmWipeCommand(),
		},
	}
}

func mdmRunCommand() *cli.Command {
	return &cli.Command{
		Name:    "run-command",
		Aliases: []string{"run_command"},
		Usage:   "Run a custom MDM command on macOS and Windows hosts.",
		Flags: []cli.Flag{
			contextFlag(),
			debugFlag(),
			&cli.StringSliceFlag{
				Name:     "hosts",
				Usage:    "Hosts specified by hostname, serial number, uuid, osquery_host_id or node_key that you want to target.",
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
			if err := client.CheckAnyMDMEnabled(); err != nil {
				return err
			}

			// dedupe and remove any empty host identifier
			hostIdents := c.StringSlice("hosts")
			slices.Sort(hostIdents)
			hostIdents = slices.Compact(hostIdents)
			if len(hostIdents) > 0 && hostIdents[0] == "" {
				// because it is sorted, an empty ident can only be at the start
				hostIdents = hostIdents[1:]
			}
			if len(hostIdents) == 0 {
				return errors.New(`Required flag "hosts" not set`)
			}

			payloadFile := c.String("payload")
			payload, err := os.ReadFile(payloadFile)
			if err != nil {
				return fmt.Errorf("read payload: %w", err)
			}

			// fetch all specified hosts by their identifier
			//
			// Note that this retrieves all hosts sequentially as it assumes the
			// number of hosts will typically be small. If we need to improve on
			// this, we could use something like the errgroup package to retrieve
			// them concurrently or better yet add a batch endpoint to get hosts by
			// the loose "identifier".
			var (
				hostUUIDs     []string
				notFoundCount int
				platform      string
			)
			for _, ident := range hostIdents {
				host, err := client.HostByIdentifier(ident)
				if err != nil {
					var nfe service.NotFoundErr
					if errors.As(err, &nfe) {
						notFoundCount++
						continue
					}

					var sce kithttp.StatusCoder
					if errors.As(err, &sce) {
						if sce.StatusCode() == http.StatusForbidden {
							return fmt.Errorf("You don't have permission to run an MDM command on one or more specified hosts: %w", err)
						}
					}
					return err
				}

				if host.Platform != platform && platform != "" {
					return errors.New(`Command can't run on hosts with different platforms. Make sure the hosts specified in the "hosts" flag are either all macOS or all Windows hosts.`)
				}
				platform = host.Platform

				// TODO(mna): this "On" check is brittle, but looks like it's the only
				// enrollment indication we have right now...
				if host.MDM.EnrollmentStatus == nil || !strings.HasPrefix(*host.MDM.EnrollmentStatus, "On") ||
					host.MDM.Name != fleet.WellKnownMDMFleet {
					return errors.New(`Can't run the MDM command because one or more hosts have MDM turned off. Run the following command to see a list of hosts with MDM on: fleetctl get hosts --mdm.`)
				}

				hostUUIDs = append(hostUUIDs, host.UUID)
			}

			if len(hostUUIDs) == 0 {
				// all hosts were not found
				return errors.New("No hosts targeted. Make sure you provide a valid hostname, UUID, osquery host ID, or node key.")
			}
			if notFoundCount > 0 {
				// at least one was not found
				return errors.New("One or more targeted hosts don't exist. Make sure you provide a valid hostname, UUID, osquery host ID, or node key.")
			}

			result, err := client.RunMDMCommand(hostUUIDs, payload, platform)
			if err != nil {
				if errors.Is(err, service.ErrMissingLicense) && platform == "windows" {
					return errors.New(fleet.WindowsMDMRequiresPremiumCmdMessage)
				}

				var sce kithttp.StatusCoder
				if errors.As(err, &sce) {
					if sce.StatusCode() == http.StatusUnsupportedMediaType && platform == "darwin" {
						return fmt.Errorf("The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file: %w", err)
					}
					// this condition needs to be repeated here: maybe the user has
					// access to fetch the host when calling HostByIdentifier above, but
					// they don't have access to execute a command on it.
					if sce.StatusCode() == http.StatusForbidden {
						return fmt.Errorf("You don't have permission to run an MDM command on one or more specified hosts: %w", err)
					}
				}
				return err
			}

			fmt.Fprintf(c.App.Writer, `
Hosts will run the command the next time they check into Fleet.

Copy and run this command to see results:

fleetctl get mdm-command-results --id=%v
`, result.CommandUUID)

			return nil
		},
	}
}

func mdmLockCommand() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "Lock a host when it needs to be returned to your organization.",
		Flags: []cli.Flag{contextFlag(), debugFlag(), &cli.StringFlag{
			Name:     "host",
			Usage:    "The host, specified by identifier, that you want to lock.",
			Required: true,
		}},
		Action: func(c *cli.Context) error {
			hostIdent := c.String("host")

			client, host, err := hostMdmActionSetup(c, hostIdent, "lock")
			if err != nil {
				return err
			}

			if err := client.MDMLockHost(host.ID); err != nil {
				return fmt.Errorf("Failed to lock host: %w", err)
			}

			fmt.Fprintf(c.App.Writer, `
The host will lock when it comes online.

Copy and run this command to see lock status:

fleetctl get host %s

When you're ready to unlock the host, copy and run this command:

fleetctl mdm unlock --host=%s

`, hostIdent, hostIdent)

			return nil
		},
	}
}

func mdmUnlockCommand() *cli.Command {
	return &cli.Command{
		Name:  "unlock",
		Usage: "Unlock a host when it needs to be returned to your organization.",
		Flags: []cli.Flag{contextFlag(), debugFlag(), &cli.StringFlag{
			Name:     "host",
			Usage:    "The host, specified by identifier, that you want to unlock.",
			Required: true,
		}},
		Action: func(c *cli.Context) error {
			hostIdent := c.String("host")

			client, host, err := hostMdmActionSetup(c, hostIdent, "unlock")
			if err != nil {
				return err
			}

			pin, err := client.MDMUnlockHost(host.ID)
			if err != nil {
				return fmt.Errorf("Failed to unlock host: %w", err)
			}

			if host.Platform == "darwin" {
				fmt.Fprintf(c.App.Writer, `
Use this 6 digit PIN to unlock the host:

%s

`, pin)

				return nil
			}

			fmt.Fprintf(c.App.Writer, `
The host will unlock when it comes online.

Copy and run this command to see results:

fleetctl get host %s

`, hostIdent)

			return nil
		},
	}
}

// create a mdm command to wipe the device
func mdmWipeCommand() *cli.Command {
	return &cli.Command{
		Name:  "wipe",
		Usage: "Wipe a host to erase all content on a workstation.",
		Flags: []cli.Flag{contextFlag(), debugFlag(), &cli.StringFlag{
			Name:     "host",
			Usage:    "The host, specified by identifier, that you want to wipe.",
			Required: true,
		}},
		Action: func(c *cli.Context) error {
			hostIdent := c.String("host")

			client, host, err := hostMdmActionSetup(c, hostIdent, "wipe")
			if err != nil {
				return err
			}

			config, err := client.GetAppConfig()
			if err != nil {
				return err
			}

			// linux hosts need scripts to be enabled in the org settings to wipe.
			if host.Platform == "linux" && config.ServerSettings.ScriptsDisabled {
				return errors.New("Can't wipe host because running scripts is disabled in organization settings.")
			}

			if err := client.MDMWipeHost(host.ID); err != nil {
				return fmt.Errorf("Failed to wipe host: %w", err)
			}

			fmt.Fprintf(c.App.Writer, `
The host will wipe when it comes online.

Copy and run this command to see results:

fleetctl get host %s`, hostIdent)

			return nil
		},
	}
}

// Does some common setup for the host mdm actions such as validating the host,
// creating the client, getting the desired host, checking permissions, and
// ensuring MDM is turned on for the host.
func hostMdmActionSetup(c *cli.Context, hostIdent string, actionType string) (client *service.Client, host *service.HostDetailResponse, err error) {
	if len(hostIdent) == 0 {
		return nil, nil, errors.New("No host targeted. Please provide --host.")
	}

	client, err = clientFromCLI(c)
	if err != nil {
		return nil, nil, fmt.Errorf("create client: %w", err)
	}

	host, err = client.HostByIdentifier(hostIdent)
	if err != nil {
		var nfe service.NotFoundErr
		if errors.As(err, &nfe) {
			fmt.Println(hostIdent)
			return nil, nil, errors.New("The host doesn't exist. Please provide a valid host identifier.")
		}

		var sce kithttp.StatusCoder
		if errors.As(err, &sce) {
			if sce.StatusCode() == http.StatusForbidden {
				return nil, nil, fmt.Errorf("Permission denied. You don't have permission to %s this host.", actionType)
			}
		}
		return nil, nil, err
	}

	// check mdm is on for the host
	if host.Platform == "windows" || host.Platform == "darwin" {
		if host.MDM.EnrollmentStatus == nil || !strings.HasPrefix(*host.MDM.EnrollmentStatus, "On") ||
			host.MDM.Name != fleet.WellKnownMDMFleet {
			return nil, nil, fmt.Errorf("Can't %s the host because it doesn't have MDM turned on.", actionType)
		}
	}

	return client, host, nil
}
