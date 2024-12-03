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
			logf := func(format string, a ...interface{}) {
				_, _ = fmt.Fprintf(c.App.Writer, format, a...)
			}

			// We need to extract the controls from no-team.yml to be able to apply them when applying the global app config.
			var (
				noTeamControls spec.Controls
				noTeamPresent  bool
			)
			isPremium := appConfig.License.IsPremium()
			for _, flFilename := range flFilenames.Value() {
				if filepath.Base(flFilename) == "no-team.yml" {
					if !isPremium {
						// Message is printed in the next flFilenames loop to avoid printing it multiple times
						break
					}
					baseDir := filepath.Dir(flFilename)
					config, err := spec.GitOpsFromFile(flFilename, baseDir, appConfig, func(format string, a ...interface{}) {})
					if err != nil {
						return err
					}
					noTeamControls = config.Controls
					noTeamPresent = true
					break
				}
			}

			var originalABMConfig []any
			var originalVPPConfig []any
			var teamNames []string
			var firstFileMustBeGlobal *bool
			var teamDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions
			var abmTeams, vppTeams []string
			var hasMissingABMTeam, hasMissingVPPTeam, usesLegacyABMConfig bool
			if totalFilenames > 1 {
				firstFileMustBeGlobal = ptr.Bool(true)
			}

			// we keep track of team software installers and scripts for correct policy application
			teamsSoftwareInstallers := make(map[string][]fleet.SoftwarePackageResponse)
			teamsScripts := make(map[string][]fleet.ScriptResponse)

			// We keep track of the secrets to check if duplicates exist during dry run
			secrets := make(map[string]struct{})
			for _, flFilename := range flFilenames.Value() {
				baseDir := filepath.Dir(flFilename)
				config, err := spec.GitOpsFromFile(flFilename, baseDir, appConfig, logf)
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

				if isGlobalConfig {
					if noTeamControls.Set() && config.Controls.Set() {
						return errors.New("'controls' cannot be set on both global config and on no-team.yml")
					}
					if !noTeamControls.Defined && !config.Controls.Defined {
						if appConfig.License.IsPremium() {
							return errors.New("'controls' must be set on global config or no-team.yml")
						}
						return errors.New("'controls' must be set on global config")
					}
					if !config.Controls.Set() {
						config.Controls = noTeamControls
					}
				} else if !isPremium {
					logf("[!] skipping team config %s since teams are only supported for premium Fleet users\n", flFilename)
					continue
				}

				// Special handling for tokens is required because they link to teams (by
				// name.) Because teams can be created/deleted during the same gitops run, we
				// grab some information to help us determine allowed/restricted actions and
				// when to perform the associations.

				if isGlobalConfig && totalFilenames > 1 && !(totalFilenames == 2 && noTeamPresent) && isPremium {
					abmTeams, hasMissingABMTeam, usesLegacyABMConfig, err = checkABMTeamAssignments(config, fleetClient)
					if err != nil {
						return err
					}

					vppTeams, hasMissingVPPTeam, err = checkVPPTeamAssignments(config, fleetClient)
					if err != nil {
						return err
					}

					// if one of the teams assigned to an ABM token doesn't exist yet, we need to
					// submit the configs without the ABM default team set. We'll set those
					// separately later when the teams are already created.
					if hasMissingABMTeam {
						if mdm, ok := config.OrgSettings["mdm"]; ok {
							if mdmMap, ok := mdm.(map[string]any); ok {
								if appleBM, ok := mdmMap["apple_business_manager"]; ok {
									if bmSettings, ok := appleBM.([]any); ok {
										originalABMConfig = bmSettings
									}
								}

								// If team is not found, we need to remove the AppleBMDefaultTeam from
								// the global config, and then apply it after teams are processed
								mdmMap["apple_business_manager"] = nil
								mdmMap["apple_bm_default_team"] = ""
							}
						}
					}

					if hasMissingVPPTeam {
						if mdm, ok := config.OrgSettings["mdm"]; ok {
							if mdmMap, ok := mdm.(map[string]any); ok {
								if vpp, ok := mdmMap["volume_purchasing_program"]; ok {
									if vppSettings, ok := vpp.([]any); ok {
										originalVPPConfig = vppSettings
									}
								}

								// If team is not found, we need to remove the VPP config from
								// the global config, and then apply it after teams are processed
								mdmMap["volume_purchasing_program"] = nil
							}
						}
					}
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

				assumptions, err := fleetClient.DoGitOps(c.Context, config, flFilename, logf, flDryRun, teamDryRunAssumptions, appConfig, teamsSoftwareInstallers, teamsScripts)
				if err != nil {
					return err
				}
				if config.TeamName != nil {
					teamNames = append(teamNames, *config.TeamName)
				} else {
					teamDryRunAssumptions = assumptions
				}
			}

			// if there were assignments to tokens, and some of the teams were missing at that time, submit a separate patch request to set them now.
			if len(abmTeams) > 0 && hasMissingABMTeam {
				if err = applyABMTokenAssignmentIfNeeded(c, teamNames, abmTeams, originalABMConfig, usesLegacyABMConfig, flDryRun, fleetClient); err != nil {
					return err
				}
			}
			if len(vppTeams) > 0 && hasMissingVPPTeam {
				if err = applyVPPTokenAssignmentIfNeeded(c, teamNames, vppTeams, originalVPPConfig, flDryRun, fleetClient); err != nil {
					return err
				}
			}
			if flDeleteOtherTeams && appConfig.License.IsPremium() { // skip team deletion for non-premium users
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
						if slices.Contains(vppTeams, team.Name) {
							return fmt.Errorf("volume_purchasing_program team %s cannot be deleted", team.Name)
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

// checkABMTeamAssignments validates the spec, and finds if:
//
// 1. The user is using the legacy apple_bm_default_team config.
// 2. All teams assigned to ABM tokens already exist.
// 3. Performs validations according to the spec for both the new and the
// deprecated key used for this setting.
func checkABMTeamAssignments(config *spec.GitOps, fleetClient *service.Client) (
	abmTeams []string, missingTeam bool, usesLegacyConfig bool, err error,
) {
	if mdm, ok := config.OrgSettings["mdm"]; ok {
		if mdmMap, ok := mdm.(map[string]any); ok {
			appleBMDT, hasLegacyConfig := mdmMap["apple_bm_default_team"]
			appleBM, hasNewConfig := mdmMap["apple_business_manager"]

			if hasLegacyConfig && hasNewConfig {
				return nil, false, false, errors.New(fleet.AppleABMDefaultTeamDeprecatedMessage)
			}

			abmToks, err := fleetClient.ListABMTokens()
			if err != nil {
				return nil, false, false, err
			}

			if hasLegacyConfig && len(abmToks) > 1 {
				return nil, false, false, errors.New(fleet.AppleABMDefaultTeamDeprecatedMessage)
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
					// normalize for Unicode support
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
									// normalize for Unicode support
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

func applyABMTokenAssignmentIfNeeded(
	ctx *cli.Context,
	teamNames []string,
	abmTeamNames []string,
	originalMDMConfig []any,
	usesLegacyConfig bool,
	flDryRun bool,
	fleetClient *service.Client,
) error {
	if usesLegacyConfig && len(abmTeamNames) > 1 {
		return errors.New(fleet.AppleABMDefaultTeamDeprecatedMessage)
	}

	if usesLegacyConfig && len(abmTeamNames) == 0 {
		return errors.New("using legacy config without any ABM teams defined")
	}

	var appConfigUpdate map[string]map[string]any
	if usesLegacyConfig {
		appleBMDefaultTeam := abmTeamNames[0]
		if !slices.Contains(teamNames, appleBMDefaultTeam) {
			return fmt.Errorf("apple_bm_default_team team %q not found in team configs", appleBMDefaultTeam)
		}
		appConfigUpdate = map[string]map[string]any{
			"mdm": {
				"apple_bm_default_team": appleBMDefaultTeam,
			},
		}
	} else {
		for _, abmTeam := range abmTeamNames {
			if !slices.Contains(teamNames, abmTeam) {
				return fmt.Errorf("apple_business_manager team %q not found in team configs", abmTeam)
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

func checkVPPTeamAssignments(config *spec.GitOps, fleetClient *service.Client) (
	vppTeams []string, missingTeam bool, err error,
) {
	if mdm, ok := config.OrgSettings["mdm"]; ok {
		if mdmMap, ok := mdm.(map[string]any); ok {
			teams, err := fleetClient.ListTeams("")
			if err != nil {
				return nil, false, err
			}
			teamNames := map[string]struct{}{}
			for _, tm := range teams {
				teamNames[tm.Name] = struct{}{}
			}

			if vpp, ok := mdmMap["volume_purchasing_program"]; ok {
				if vppInterfaces, ok := vpp.([]any); ok {
					for _, item := range vppInterfaces {
						if itemMap, ok := item.(map[string]any); ok {
							if teams, ok := itemMap["teams"].([]any); ok {
								for _, team := range teams {
									if teamStr, ok := team.(string); ok {
										// normalize for Unicode support
										normalizedTeam := norm.NFC.String(teamStr)
										vppTeams = append(vppTeams, normalizedTeam)
										if _, ok := teamNames[normalizedTeam]; !ok {
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
	}

	return vppTeams, missingTeam, nil
}

func applyVPPTokenAssignmentIfNeeded(
	ctx *cli.Context,
	teamNames []string,
	vppTeamNames []string,
	originalVPPConfig []any,
	flDryRun bool,
	fleetClient *service.Client,
) error {
	var appConfigUpdate map[string]map[string]any
	for _, vppTeam := range vppTeamNames {
		if !fleet.IsReservedTeamName(vppTeam) && !slices.Contains(teamNames, vppTeam) {
			return fmt.Errorf("volume_purchasing_program team %s not found in team configs", vppTeam)
		}
	}

	appConfigUpdate = map[string]map[string]any{
		"mdm": {
			"volume_purchasing_program": originalVPPConfig,
		},
	}

	if flDryRun {
		_, _ = fmt.Fprint(ctx.App.Writer, "[!] would apply volume_purchasing_program teams\n")
		return nil
	}
	_, _ = fmt.Fprintf(ctx.App.Writer, "[+] applying volume_purchasing_program teams\n")
	if err := fleetClient.ApplyAppConfig(appConfigUpdate, fleet.ApplySpecOptions{}); err != nil {
		return fmt.Errorf("applying fleet config for volume_purchasing_program teams: %w", err)
	}
	return nil
}
