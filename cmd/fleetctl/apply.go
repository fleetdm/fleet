package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
)

func applyCommand() *cli.Command {
	var (
		flFilename string
		flForce    bool
		flDryRun   bool
	)
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
			&cli.BoolFlag{
				Name:        "force",
				EnvVars:     []string{"FORCE"},
				Destination: &flForce,
				Usage:       "Force applying the file even if it raises validation errors (only supported for 'config' and 'team' specs)",
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				EnvVars:     []string{"DRY_RUN"},
				Destination: &flDryRun,
				Usage:       "Do not apply the file, just validate it (only supported for 'config' and 'team' specs)",
			},
			&cli.StringFlag{
				Name:  "policies-team",
				Usage: "A team's name, this flag is only used on policies specs (overrides 'team' key in the policies file). This allows to easily import a group of policies to a team.",
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

			opts := fleet.ApplySpecOptions{
				Force:  flForce,
				DryRun: flDryRun,
			}
			if policiesTeamName := c.String("policies-team"); policiesTeamName != "" {
				opts.TeamForPolicies = policiesTeamName
			}
			baseDir := filepath.Dir(flFilename)
			_, err = fleetClient.ApplyGroup(c.Context, specs, baseDir, logf, opts)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
