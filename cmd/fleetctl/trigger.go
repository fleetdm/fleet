package main

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

func triggerCommand() *cli.Command {
	var name string
	return &cli.Command{
		Name:      "trigger",
		Usage:     "Trigger an ad hoc run of all jobs in a specified cron schedule",
		UsageText: `fleetctl trigger [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				EnvVars:     []string{"NAME"},
				Value:       "",
				Destination: &name,
				Usage:       "Name of the cron schedule to trigger",
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

			if err := client.TriggerCronSchedule(name); err != nil {
				root := ctxerr.Cause(err)
				switch root.(type) {
				case service.NotFoundErr, service.ConflictErr:
					fmt.Printf("[!] %s\n", formatTriggerErrMsg(name, root.Error()))
					return nil
				default:
					return err
				}
			}

			fmt.Printf("[+] Sent request to trigger %s schedule\n", name)
			return nil
		},
	}
}

func formatTriggerErrMsg(name string, msg string) string {
	formatted := msg
	if name == "" {
		formatted = strings.Replace(strings.ToLower(msg), "invalid name", "name must be specified", 1)
	}
	if len(formatted) >= 1 {
		formatted = strings.ToUpper(formatted[:1]) + formatted[1:]
	}
	return formatted
}
