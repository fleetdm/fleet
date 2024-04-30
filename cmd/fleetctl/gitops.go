package main

import (
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strings"
)

func gitopsCommand() *cli.Command {
	var (
		flFilenames        cli.StringSlice
		flDryRun           bool
		flDeleteOtherTeams bool
	)
	return &cli.Command{
		Name:      "gitops",
		Usage:     "Synchronize Fleet configuration with provided file. This command is intended to be used in a GitOps workflow.",
		UsageText: `fleetctl gitops [options]`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "f",
				Required:    true,
				EnvVars:     []string{"FILENAME"},
				Destination: &flFilenames,
				Usage:       "The `FILE` with the GitOps configuration",
			},
			&cli.BoolFlag{
				Name:        "delete-other-teams",
				EnvVars:     []string{"DELETE_OTHER_TEAMS"},
				Destination: &flDeleteOtherTeams,
				Usage:       "Delete other teams not present in the GitOps configuration",
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				EnvVars:     []string{"DRY_RUN"},
				Destination: &flDryRun,
				Usage:       "Do not apply the file(s), just validate",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if len(flFilenames.Value()) == 0 {
				return errors.New("-f must be specified")
			}
			for _, flFilename := range flFilenames.Value() {
				if strings.TrimSpace(flFilename) == "" {
					return errors.New("filename cannot be empty")
				}
			}
			for _, flFilename := range flFilenames.Value() {
				b, err := os.ReadFile(flFilename)
				if err != nil {
					return err
				}
				fleetClient, err := clientFromCLI(c)
				if err != nil {
					return err
				}
				baseDir := filepath.Dir(flFilename)
				config, err := spec.GitOpsFromBytes(b, baseDir)
				if err != nil {
					return err
				}
				logf := func(format string, a ...interface{}) {
					_, _ = fmt.Fprintf(c.App.Writer, format, a...)
				}
				appConfig, err := fleetClient.GetAppConfig()
				if err != nil {
					return err
				}
				if appConfig.License == nil {
					return errors.New("no license struct found in app config")
				}
				err = fleetClient.DoGitOps(c.Context, config, baseDir, logf, flDryRun, appConfig)
				if err != nil {
					return err
				}
				if flDryRun {
					_, _ = fmt.Fprintf(c.App.Writer, "[!] gitops dry run succeeded\n")
				} else {
					_, _ = fmt.Fprintf(c.App.Writer, "[!] gitops succeeded\n")
				}
			}
			return nil
		},
	}
}
