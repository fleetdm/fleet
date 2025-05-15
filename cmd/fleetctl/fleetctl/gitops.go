package fleetctl

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

const (
	filenameMaxLength           = 255
	ReapplyingTeamForVPPAppsMsg = "[!] re-applying configs for team %s -- this only happens once for new teams that have VPP apps\n"
)

type LabelUsage struct {
	Name string
	Type string
}

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
				Usage:       "The file(s) with the GitOps configuration.",
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

			// We need the controls from no-team.yml to apply them when applying the global app config.
			noTeamControls, noTeamPresent, err := extractControlsForNoTeam(flFilenames, appConfig)
			if err != nil {
				return fmt.Errorf("extracting controls from no-team.yml: %w", err)
			}

			var originalABMConfig []any
			var originalVPPConfig []any
			var teamNames []string
			var teamDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions
			var abmTeams, vppTeams, missingVPPTeams []string
			var hasMissingABMTeam, usesLegacyABMConfig bool
			type missingVPPTeamWithApps struct {
				config   *spec.GitOps
				vppApps  []*fleet.TeamSpecAppStoreApp
				filename string
			}
			var missingVPPTeamsWithApps []missingVPPTeamWithApps

			// we keep track of team software installers and scripts for correct policy application
			teamsSoftwareInstallers := make(map[string][]fleet.SoftwarePackageResponse)
			teamsVPPApps := make(map[string][]fleet.VPPAppResponse)
			teamsScripts := make(map[string][]fleet.ScriptResponse)

			// We keep track of the secrets to check if duplicates exist during dry run
			secrets := make(map[string]struct{})
			// We keep track of the environment FLEET_SECRET_* variables
			allFleetSecrets := make(map[string]string)
			// Keep track of which labels we'd have after this gitops run.
			var proposedLabelNames []string

			// Parsed config and filename pair
			type ConfigFile struct {
				Config         *spec.GitOps
				Filename       string
				IsGlobalConfig bool
			}

			// Load all configs in before processing them
			configs := make([]ConfigFile, 0, len(flFilenames.Value()))

			// We only want to have one global config loaded
			globalConfigLoaded := false

			// List of things we want to do at the end of this run
			var allPostOps []func() error

			for _, flFilename := range flFilenames.Value() {
				baseDir := filepath.Dir(flFilename)
				config, err := spec.GitOpsFromFile(flFilename, baseDir, appConfig, logf)
				if err != nil {
					return err
				}
				isGlobalConfig := config.TeamName == nil
				if isGlobalConfig {
					if globalConfigLoaded {
						return errors.New("only one global config file may be provided to fleetctl gitops")
					}
					globalConfigLoaded = true
				}
				configFile := ConfigFile{Config: config, Filename: flFilename, IsGlobalConfig: isGlobalConfig}
				if isGlobalConfig {
					// If it's a global file, put it at the beginning
					// of the array so it gets processed first
					configs = append([]ConfigFile{configFile}, configs...)
				} else {
					configs = append(configs, configFile)
				}
			}

			for _, configFile := range configs {
				config := configFile.Config
				flFilename := configFile.Filename
				isGlobalConfig := configFile.IsGlobalConfig

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

					// If config.Labels is nil, it means we plan on deleting all existing labels.
					if config.Labels == nil {
						proposedLabelNames = make([]string, 0)
					} else if len(config.Labels) > 0 {
						// If config.Labels is populated, get the names it contains.
						proposedLabelNames = make([]string, len(config.Labels))
						for i, l := range config.Labels {
							proposedLabelNames[i] = l.Name
						}
					}
				} else if !appConfig.License.IsPremium() {
					logf("[!] skipping team config %s since teams are only supported for premium Fleet users\n", flFilename)
					continue
				}

				// If we haven't populated this list yet, it means we're either doing team-level GitOps only,
				// or a global YAML was provided with no `labels:` key in it (meaning "keep existing labels").
				// In either case we'll get the list of label names from the db so we can ensure that we're
				// not attempting to apply non-existent labels to other entities.
				if proposedLabelNames == nil {
					proposedLabelNames = make([]string, 0)
					persistedLabels, err := fleetClient.GetLabels()
					if err != nil {
						return err
					}
					for _, persistedLabel := range persistedLabels {
						if persistedLabel.LabelType == fleet.LabelTypeRegular {
							proposedLabelNames = append(proposedLabelNames, persistedLabel.Name)
						}
					}
				}

				// Gather stats on where labels are used in the this gitops config,
				// so we can bail if any of the referenced labels wouldn't exist
				// after this run (either because they'd be deleted, never existed
				// in the first place).
				labelsUsed, err := getLabelUsage(config)
				if err != nil {
					return err
				}

				// Check if any used labels are not in the proposed labels list.
				// If there are, we'll bail out with helpful error messages.
				unknownLabelsUsed := false
				for labelUsed := range labelsUsed {
					if slices.Index(proposedLabelNames, labelUsed) == -1 {
						for _, labelUser := range labelsUsed[labelUsed] {
							logf("[!] Unknown label '%s' is referenced by %s '%s'\n", labelUsed, labelUser.Type, labelUser.Name)
						}
						unknownLabelsUsed = true
					}
				}
				if unknownLabelsUsed {
					return errors.New("Please create the missing labels, or update your settings to not refer to these labels.")
				}

				// Special handling for tokens is required because they link to teams (by
				// name.) Because teams can be created/deleted during the same gitops run, we
				// grab some information to help us determine allowed/restricted actions and
				// when to perform the associations.

				if isGlobalConfig && totalFilenames > 1 && !(totalFilenames == 2 && noTeamPresent) && appConfig.License.IsPremium() {
					abmTeams, hasMissingABMTeam, usesLegacyABMConfig, err = checkABMTeamAssignments(config, fleetClient)
					if err != nil {
						return err
					}

					vppTeams, missingVPPTeams, err = checkVPPTeamAssignments(config, fleetClient)
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

					if len(missingVPPTeams) > 0 {
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

				// We cannot apply a VPP app to a new team until that team gets a VPP token.
				// So, we create the team, then apply the VPP token, then apply VPP apps.
				if !isGlobalConfig && len(missingVPPTeams) > 0 && len(config.Software.AppStoreApps) > 0 {
					for _, missingTeam := range missingVPPTeams {
						if missingTeam == *config.TeamName {
							missingVPPTeamsWithApps = append(missingVPPTeamsWithApps, missingVPPTeamWithApps{
								config:   config,
								vppApps:  config.Software.AppStoreApps,
								filename: flFilename,
							})
							config.Software.AppStoreApps = nil
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

				err = fleetClient.SaveEnvSecrets(allFleetSecrets, config.FleetSecrets, flDryRun)
				if err != nil {
					return err
				}
				assumptions, postOps, err := fleetClient.DoGitOps(c.Context, config, flFilename, logf, flDryRun, teamDryRunAssumptions, appConfig,
					teamsSoftwareInstallers, teamsVPPApps, teamsScripts)
				if err != nil {
					return err
				}
				if config.TeamName != nil {
					teamNames = append(teamNames, *config.TeamName)
				} else {
					teamDryRunAssumptions = assumptions
				}
				allPostOps = append(allPostOps, postOps...)
			}

			// if there were assignments to tokens, and some of the teams were missing at that time, submit a separate patch request to set them now.
			if len(abmTeams) > 0 && hasMissingABMTeam {
				if err = applyABMTokenAssignmentIfNeeded(c, teamNames, abmTeams, originalABMConfig, usesLegacyABMConfig, flDryRun,
					fleetClient); err != nil {
					return err
				}
			}
			if len(missingVPPTeams) > 0 {
				if err = applyVPPTokenAssignmentIfNeeded(c, teamNames, vppTeams, originalVPPConfig, flDryRun, fleetClient); err != nil {
					return err
				}
			}
			// Now that VPP tokens have been assigned, we can apply VPP apps to the new team.
			// For simplicity, we simply re-apply the entire config. This only happens once when the team is created.
			for _, teamWithApps := range missingVPPTeamsWithApps {
				_, _ = fmt.Fprintf(c.App.Writer, ReapplyingTeamForVPPAppsMsg, *teamWithApps.config.TeamName)
				teamWithApps.config.Software.AppStoreApps = teamWithApps.vppApps
				_, postOps, err := fleetClient.DoGitOps(c.Context, teamWithApps.config, teamWithApps.filename, logf, flDryRun, teamDryRunAssumptions, appConfig,
					teamsSoftwareInstallers, teamsVPPApps, teamsScripts)
				if err != nil {
					return err
				}
				allPostOps = append(allPostOps, postOps...)
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

			// we only want to reset the no-team config if the global config was loaded.
			// NOTE: noTeamPresent is refering to the "No Team" team. It does not
			// mean that other teams are not present.
			if globalConfigLoaded && !noTeamPresent {
				defaultNoTeamConfig := new(spec.GitOps)
				defaultNoTeamConfig.TeamName = ptr.String(fleet.TeamNameNoTeam)
				_, postOps, err := fleetClient.DoGitOps(c.Context, defaultNoTeamConfig, "no-team.yml", logf, flDryRun, nil, appConfig,
					map[string][]fleet.SoftwarePackageResponse{}, map[string][]fleet.VPPAppResponse{}, map[string][]fleet.ScriptResponse{})
				if err != nil {
					return err
				}

				allPostOps = append(allPostOps, postOps...)
			}

			for _, postOp := range allPostOps {
				if err := postOp(); err != nil {
					return err
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

// Given a set of referenced labels and info about who is using them, update a provided usage map.
func updateLabelUsage(labels []string, ident string, usageType string, currentUsage map[string][]LabelUsage) {
	for _, label := range labels {
		var usage []LabelUsage
		if _, ok := currentUsage[label]; !ok {
			currentUsage[label] = make([]LabelUsage, 0)
		}
		usage = currentUsage[label]
		usage = append(usage, LabelUsage{
			Name: ident,
			Type: usageType,
		})
		currentUsage[label] = usage
	}
}

// Create a map of label name -> who is using that label.
// This will be used to determine if any non-existent labels are being referenced.
func getLabelUsage(config *spec.GitOps) (map[string][]LabelUsage, error) {
	result := make(map[string][]LabelUsage)

	// Get profile label usage
	for _, osSettingName := range []interface{}{config.Controls.MacOSSettings, config.Controls.WindowsSettings} {
		if osSettings, ok := getCustomSettings(osSettingName); ok {
			for _, setting := range osSettings {
				var labels []string
				err := fmt.Errorf("MDM profile '%s' has multiple label keys; please choose one of `labels_include_any`, `labels_include_all` or `labels_exclude_any`.", setting.Path)

				if len(setting.LabelsIncludeAny) > 0 {
					labels = setting.LabelsIncludeAny
				}
				if len(setting.LabelsIncludeAll) > 0 {
					if len(labels) > 0 {
						return nil, err
					}
					labels = setting.LabelsIncludeAll
				}
				if len(setting.LabelsExcludeAny) > 0 {
					if len(labels) > 0 {
						return nil, err
					}
					labels = setting.LabelsExcludeAny
				}

				updateLabelUsage(labels, setting.Path, "MDM Profile", result)
			}
		}
	}

	// Get software package installer label usage
	for _, setting := range config.Software.Packages {
		var labels []string
		if len(setting.LabelsIncludeAny) > 0 {
			labels = setting.LabelsIncludeAny
		}
		if len(setting.LabelsExcludeAny) > 0 {
			if len(labels) > 0 {
				return nil, fmt.Errorf("Software package '%s' has multiple label keys; please choose one of `labels_include_any`, `labels_exclude_any`.", setting.URL)
			}
			labels = setting.LabelsExcludeAny
		}
		updateLabelUsage(labels, setting.URL, "Software Package", result)
	}

	// Get app store app installer label usage
	for _, setting := range config.Software.AppStoreApps {
		var labels []string
		if len(setting.LabelsIncludeAny) > 0 {
			labels = setting.LabelsIncludeAny
		}
		if len(setting.LabelsExcludeAny) > 0 {
			if len(labels) > 0 {
				return nil, fmt.Errorf("App Store App '%s' has multiple label keys; please choose one of `labels_include_any`, `labels_exclude_any`.", setting.AppStoreID)
			}
			labels = setting.LabelsExcludeAny
		}
		updateLabelUsage(labels, setting.AppStoreID, "App Store App", result)
	}

	for _, setting := range config.Software.FleetMaintainedApps {
		var labels []string
		if len(setting.LabelsIncludeAny) > 0 {
			labels = setting.LabelsIncludeAny
		}
		if len(setting.LabelsExcludeAny) > 0 {
			if len(labels) > 0 {
				return nil, fmt.Errorf("Fleet Maintained App '%s' has multiple label keys; please choose one of `labels_include_any`, `labels_exclude_any`.", setting.Slug)
			}
			labels = setting.LabelsExcludeAny
		}
		updateLabelUsage(labels, setting.Slug, "Fleet Maintained App", result)
	}

	// Get query label usage
	for _, query := range config.Queries {
		updateLabelUsage(query.LabelsIncludeAny, query.Name, "Query", result)
	}

	// Get policy label usage
	for _, policy := range config.Policies {
		var labels []string
		if len(policy.LabelsIncludeAny) > 0 {
			labels = policy.LabelsIncludeAny
		}
		if len(policy.LabelsExcludeAny) > 0 {
			if len(labels) > 0 {
				return nil, fmt.Errorf("Policy '%s' has multiple label keys; please choose one of `labels_include_any`, `labels_exclude_any`.", policy.Name)
			}
			labels = policy.LabelsExcludeAny
		}
		updateLabelUsage(labels, policy.Name, "Policy", result)
	}

	return result, nil
}

func getCustomSettings(osSettings interface{}) ([]fleet.MDMProfileSpec, bool) {
	if settingsMap, ok := osSettings.(fleet.WithMDMProfileSpecs); ok {
		return settingsMap.GetMDMProfileSpecs(), true
	}
	return nil, false
}

func extractControlsForNoTeam(flFilenames cli.StringSlice, appConfig *fleet.EnrichedAppConfig) (spec.GitOpsControls, bool, error) {
	for _, flFilename := range flFilenames.Value() {
		if filepath.Base(flFilename) == "no-team.yml" {
			if !appConfig.License.IsPremium() {
				// Message is printed in the next flFilenames loop to avoid printing it multiple times
				break
			}
			baseDir := filepath.Dir(flFilename)
			config, err := spec.GitOpsFromFile(flFilename, baseDir, appConfig, func(format string, a ...interface{}) {})
			if err != nil {
				return spec.GitOpsControls{}, false, err
			}
			return config.Controls, true, nil
		}
	}
	return spec.GitOpsControls{}, false, nil
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

			abmToks, err := fleetClient.CountABMTokens()
			if err != nil {
				return nil, false, false, err
			}

			if hasLegacyConfig && abmToks > 1 {
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
	vppTeams []string, missingTeams []string, err error,
) {
	if mdm, ok := config.OrgSettings["mdm"]; ok {
		if mdmMap, ok := mdm.(map[string]any); ok {
			teams, err := fleetClient.ListTeams("")
			if err != nil {
				return nil, nil, err
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
											missingTeams = append(missingTeams, normalizedTeam)
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

	return vppTeams, missingTeams, nil
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
