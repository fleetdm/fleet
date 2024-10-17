package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

func deleteCommand() *cli.Command {
	var flFilename string
	return &cli.Command{
		Name:      "delete",
		Usage:     "Specify files to declaratively batch delete osquery configurations",
		UsageText: `fleetctl delete [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "f",
				EnvVars:     []string{"FILENAME"},
				Value:       "",
				Destination: &flFilename,
				Usage:       "A file to apply",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if flFilename == "" {
				return errors.New("-f must be specified")
			}

			b, err := os.ReadFile(flFilename)
			if err != nil {
				return err
			}

			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			specs, err := spec.GroupFromBytes(b)
			if err != nil {
				return err
			}

			for _, query := range specs.Queries {
				fmt.Printf("[+] deleting query %q\n", query.Name)
				if err := fleet.DeleteQuery(query.Name); err != nil {
					root := ctxerr.Cause(err)
					switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
					case service.NotFoundErr:
						fmt.Printf("[!] query %q doesn't exist\n", query.Name)
						continue
					}
					return err
				}
			}

			for _, pack := range specs.Packs {
				fmt.Printf("[+] deleting pack %q\n", pack.Name)
				if err := fleet.DeletePack(pack.Name); err != nil {
					root := ctxerr.Cause(err)
					switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
					case service.NotFoundErr:
						fmt.Printf("[!] pack %q doesn't exist\n", pack.Name)
						continue
					}
					return err
				}
			}

			for _, label := range specs.Labels {
				fmt.Printf("[+] deleting label %q\n", label.Name)
				if err := fleet.DeleteLabel(label.Name); err != nil {
					root := ctxerr.Cause(err)
					switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
					case service.NotFoundErr:
						fmt.Printf("[!] label %q doesn't exist\n", label.Name)
						continue
					}
					return err
				}
			}

			return nil
		},
	}
}
