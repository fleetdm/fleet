package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/unicode/norm"
)

const filenameMaxLength = 255

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
				Usage:       "The file(s) with the GitOps configuration. If multiple files are provided, the first file must be the global configuration and the rest must be team configurations.",
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
			totalFilenames := len(flFilenames.Value())
			if totalFilenames == 0 {
				return errors.New("-f must be specified")
			}
			for _, flFilename := range flFilenames.Value() {
				if strings.TrimSpace(flFilename) == "" {
					return errors.New("file name cannot be empty")
				}
				if len(filepath.Base(flFilename)) > filenameMaxLength {
					return fmt.Errorf("file name must be less than %d characters: %s", filenameMaxLength, filepath.Base(flFilename))
				}
			}

			// Check license
			fleetClient, err := clientFromCLI(c)
			if err != nil {
				return err
			}
			appConfig, err := fleetClient.GetAppConfig()
			if err != nil {
				return err
			}
			if appConfig.License == nil {
				return errors.New("no license struct found in app config")
			}

			var appleBMDefaultTeam string
			var appleBMDefaultTeamFound bool
			var teamNames []string
			var firstFileMustBeGlobal *bool
			var teamDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions
			if totalFilenames > 1 {
				firstFileMustBeGlobal = ptr.Bool(true)
			}
			// We keep track of the secrets to check if duplicates exist during dry run
			secrets := make(map[string]struct{})
			for _, flFilename := range flFilenames.Value() {
				baseDir := filepath.Dir(flFilename)
				config, err := spec.GitOpsFromFile(flFilename, baseDir, appConfig)
				if err != nil {
					return err
				}
				isGlobalConfig := config.TeamName == nil
				if firstFileMustBeGlobal != nil {
					switch {
					case *firstFileMustBeGlobal && !isGlobalConfig:
						return fmt.Errorf("first file %s must be the global config", flFilename)
					case !*firstFileMustBeGlobal && isGlobalConfig:
						return fmt.Errorf(
							"the file %s cannot be the global config, only the first file can be the global config", flFilename,
						)
					}
					firstFileMustBeGlobal = ptr.Bool(false)
				}
				if isGlobalConfig && totalFilenames > 1 {
					// Check if Apple BM default team already exists
					appleBMDefaultTeam, appleBMDefaultTeamFound, err = checkAppleBMDefaultTeam(config, fleetClient)
					if err != nil {
						return err
					}
				}
				logf := func(format string, a ...interface{}) {
					_, _ = fmt.Fprintf(c.App.Writer, format, a...)
				}
				if flDryRun {
					incomingSecrets := fleetClient.GetGitOpsSecrets(config)
					for _, secret := range incomingSecrets {
						if _, ok := secrets[secret]; ok {
							return fmt.Errorf("duplicate enroll secret found in %s", flFilename)
						}
						secrets[secret] = struct{}{}
					}
				}
				assumptions, err := fleetClient.DoGitOps(c.Context, config, flFilename, logf, flDryRun, teamDryRunAssumptions, appConfig)
				if err != nil {
					return err
				}
				if config.TeamName != nil {
					teamNames = append(teamNames, *config.TeamName)
				} else {
					teamDryRunAssumptions = assumptions
				}
			}
			if appleBMDefaultTeam != "" && !appleBMDefaultTeamFound {
				// If the Apple BM default team did not exist earlier, check again and apply it if needed
				err = applyAppleBMDefaultTeamIfNeeded(c, teamNames, appleBMDefaultTeam, flDryRun, fleetClient)
				if err != nil {
					return err
				}
			}
			if flDeleteOtherTeams {
				teams, err := fleetClient.ListTeams("")
				if err != nil {
					return err
				}
				for _, team := range teams {
					if !slices.Contains(teamNames, team.Name) {
						if appleBMDefaultTeam == team.Name {
							return fmt.Errorf("apple_bm_default_team %s cannot be deleted", appleBMDefaultTeam)
						}
						if flDryRun {
							_, _ = fmt.Fprintf(c.App.Writer, "[!] would delete team %s\n", team.Name)
						} else {
							_, _ = fmt.Fprintf(c.App.Writer, "[-] deleting team %s\n", team.Name)
							if err := fleetClient.DeleteTeam(team.ID); err != nil {
								return err
							}
						}
					}
				}
			}

			if flDryRun {
				_, _ = fmt.Fprintf(c.App.Writer, "[!] gitops dry run succeeded\n")
			} else {
				_, _ = fmt.Fprintf(c.App.Writer, "[!] gitops succeeded\n")
			}
			return nil
		},
	}
}

func checkAppleBMDefaultTeam(config *spec.GitOps, fleetClient *service.Client) (
	appleBMDefaultTeam string, appleBMDefaultTeamFound bool, err error,
) {
	if mdm, ok := config.OrgSettings["mdm"]; ok {
		if mdmMap, ok := mdm.(map[string]interface{}); ok {
			if appleBMDT, ok := mdmMap["apple_bm_default_team"]; ok {
				if appleBMDefaultTeam, ok = appleBMDT.(string); ok {
					teams, err := fleetClient.ListTeams("")
					if err != nil {
						return "", false, err
					}
					// Normalize AppleBMDefaultTeam for Unicode support
					appleBMDefaultTeam = norm.NFC.String(appleBMDefaultTeam)
					for _, team := range teams {
						if team.Name == appleBMDefaultTeam {
							appleBMDefaultTeamFound = true
							break
						}
					}
					if !appleBMDefaultTeamFound {
						// If team is not found, we need to remove the AppleBMDefaultTeam from the global config, and then apply it after teams are processed
						mdmMap["apple_bm_default_team"] = ""
					}
				}
			}
		}
	}
	return appleBMDefaultTeam, appleBMDefaultTeamFound, nil
}

func applyAppleBMDefaultTeamIfNeeded(
	ctx *cli.Context, teamNames []string, appleBMDefaultTeam string, flDryRun bool, fleetClient *service.Client,
) error {
	if !slices.Contains(teamNames, appleBMDefaultTeam) {
		return fmt.Errorf("apple_bm_default_team %s not found in team configs", appleBMDefaultTeam)
	}
	appConfigUpdate := map[string]map[string]interface{}{
		"mdm": {
			"apple_bm_default_team": appleBMDefaultTeam,
		},
	}
	if flDryRun {
		_, _ = fmt.Fprintf(ctx.App.Writer, "[!] would apply apple_bm_default_team %s\n", appleBMDefaultTeam)
	} else {
		_, _ = fmt.Fprintf(ctx.App.Writer, "[+] applying apple_bm_default_team %s\n", appleBMDefaultTeam)
		if err := fleetClient.ApplyAppConfig(appConfigUpdate, fleet.ApplySpecOptions{}); err != nil {
			return fmt.Errorf("applying fleet config: %w", err)
		}
	}
	return nil
}
