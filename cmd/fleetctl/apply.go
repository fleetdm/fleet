package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/urfave/cli/v2"
)

func applyCommand() *cli.Command {
	var flFilename string
	return &cli.Command{
		Name:      "apply",
		Usage:     "Apply files to declaratively manage osquery configurations",
		UsageText: `fleetctl apply [options]`,
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
			fleetClient, err := clientFromCLI(c)
			if err != nil {
				return err
			}
			specs, err := spec.GroupFromBytes(b)
			if err != nil {
				return err
			}
			logf := func(format string, a ...interface{}) {
				fmt.Fprintf(c.App.Writer, format, a...)
			}
			err = fleetClient.ApplyGroup(c.Context, specs, logf)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
