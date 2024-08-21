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

			var originalMDMConfig []map[string]string
			var teamNames []string
			var firstFileMustBeGlobal *bool
			var teamDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions
			var abmTeams []string
			var hasMissingABMTeam, usesLegacyABMConfig bool
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
					abmTeams, hasMissingABMTeam, usesLegacyABMConfig, err = checkAppleBMDefaultTeam(config, fleetClient)
					if err != nil {
						return err
					}

					if hasMissingABMTeam {
						if mdm, ok := config.OrgSettings["mdm"]; ok {
							if mdmMap, ok := mdm.(map[string]any); ok {
								if appleBM, ok := mdmMap["apple_business_manager"]; ok {
									if bmSettings, ok := appleBM.([]any); ok {
										for _, item := range bmSettings {
											if bmConfig, ok := item.(map[string]any); ok {
												convertedConfig := make(map[string]string)
												for k, v := range bmConfig {
													if strVal, ok := v.(string); ok {
														convertedConfig[k] = strVal
													}
												}
												originalMDMConfig = append(originalMDMConfig, convertedConfig)
											}
										}
									}
								}

								// If team is not found, we need to remove the AppleBMDefaultTeam from the global config, and then apply it after teams are processed
								mdmMap["apple_business_manager"] = nil
								mdmMap["apple_bm_default_team"] = ""
							}
						}
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

			if len(abmTeams) > 0 && hasMissingABMTeam {
				if err = applyAppleBMDefaultTeamIfNeeded(c, teamNames, abmTeams, originalMDMConfig, usesLegacyABMConfig, flDryRun, fleetClient); err != nil {
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
						if slices.Contains(abmTeams, team.Name) {
							if usesLegacyABMConfig {
								return fmt.Errorf("apple_bm_default_team %s cannot be deleted", team.Name)
							}
							return fmt.Errorf("apple_business_manager team %s cannot be deleted", team.Name)
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
	abmTeams []string, missingTeam bool, usesLegacyConfig bool, err error,
) {
	if mdm, ok := config.OrgSettings["mdm"]; ok {
		if mdmMap, ok := mdm.(map[string]any); ok {
			appleBMDT, hasLegacyConfig := mdmMap["apple_bm_default_team"]
			appleBM, hasNewConfig := mdmMap["apple_business_manager"]

			if hasLegacyConfig && hasNewConfig {
				return nil, false, false, errors.New("mdm.apple_bm_default_team has been deprecated. Please use the new mdm.apple_business_manager key documented here: https://fleetdm.com/learn-more-about/apple-business-manager-gitops")
			}

			if !hasLegacyConfig && !hasNewConfig {
				return nil, false, false, nil
			}

			teams, err := fleetClient.ListTeams("")
			if err != nil {
				return nil, false, false, err
			}
			teamNames := map[string]struct{}{}
			for _, tm := range teams {
				teamNames[tm.Name] = struct{}{}
			}

			if hasLegacyConfig {
				if appleBMDefaultTeam, ok := appleBMDT.(string); ok {
					// Normalize AppleBMDefaultTeam for Unicode support
					appleBMDefaultTeam = norm.NFC.String(appleBMDefaultTeam)
					abmTeams = append(abmTeams, appleBMDefaultTeam)
					usesLegacyConfig = true
					if _, ok = teamNames[appleBMDefaultTeam]; !ok {
						missingTeam = true
					}
				}
			}

			if hasNewConfig {
				if settingMap, ok := appleBM.([]any); ok {
					for _, item := range settingMap {
						if cfg, ok := item.(map[string]any); ok {
							for _, teamConfigKey := range []string{"macos_team", "ios_team", "ipados_team"} {
								if team, ok := cfg[teamConfigKey].(string); ok && team != "" {
									team = norm.NFC.String(team)
									abmTeams = append(abmTeams, team)
									if _, ok := teamNames[team]; !ok {
										missingTeam = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return abmTeams, missingTeam, usesLegacyConfig, nil
}

func applyAppleBMDefaultTeamIfNeeded(
	ctx *cli.Context,
	teamNames []string,
	abmTeamNames []string,
	originalMDMConfig []map[string]string,
	usesLegacyConfig bool,
	flDryRun bool,
	fleetClient *service.Client,
) error {
	if usesLegacyConfig && len(abmTeamNames) > 1 {
		return errors.New("mdm.apple_bm_default_team has been deprecated. Please use the new mdm.apple_business_manager key documented here: https://fleetdm.com/learn-more-about/apple-business-manager-gitops")
	}

	if usesLegacyConfig && len(abmTeamNames) == 0 {
		return errors.New("using legacy config without any ABM teams defined")
	}

	var appConfigUpdate map[string]map[string]any
	if usesLegacyConfig {
		appleBMDefaultTeam := abmTeamNames[0]
		if !slices.Contains(teamNames, appleBMDefaultTeam) {
			return fmt.Errorf("apple_bm_default_team %s not found in team configs", appleBMDefaultTeam)
		}
		appConfigUpdate = map[string]map[string]any{
			"mdm": {
				"apple_bm_default_team": appleBMDefaultTeam,
			},
		}
	} else {
		for _, abmTeam := range abmTeamNames {
			if !slices.Contains(teamNames, abmTeam) {
				return fmt.Errorf("apple_business_manager team %s not found in team configs", abmTeam)
			}
		}

		appConfigUpdate = map[string]map[string]any{
			"mdm": {
				"apple_business_manager": originalMDMConfig,
			},
		}
	}

	if flDryRun {
		_, _ = fmt.Fprint(ctx.App.Writer, "[!] would apply ABM teams\n")
		return nil
	}
	_, _ = fmt.Fprintf(ctx.App.Writer, "[+] applying ABM teams\n")
	if err := fleetClient.ApplyAppConfig(appConfigUpdate, fleet.ApplySpecOptions{}); err != nil {
		return fmt.Errorf("applying fleet config: %w", err)
	}
	return nil
}
