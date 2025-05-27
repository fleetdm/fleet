package fleetctl

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
)

func lintCommand() *cli.Command {
	var flFilenames cli.StringSlice
	return &cli.Command{
		Name:      "lint-yaml",
		Usage:     "Validate GitOps YAML files",
		UsageText: `fleetctl lint-yaml [options]`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "f",
				Required:    true,
				EnvVars:     []string{"FILENAME"},
				Destination: &flFilenames,
				Usage:       "The file(s) with the GitOps configuration.",
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

			appConfig := &fleet.EnrichedAppConfig{}
			appConfig.License = &fleet.LicenseInfo{Tier: fleet.TierPremium}
			logf := func(format string, a ...interface{}) {
				_, _ = fmt.Fprintf(c.App.Writer, format, a...)
			}

			// We need the controls from no-team.yml to apply them when applying the global app config.
			noTeamControls, _, err := extractControlsForNoTeam(flFilenames, appConfig)
			if err != nil {
				return fmt.Errorf("extracting controls from no-team.yml: %w", err)
			}

			var teamNames []string

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

				if config.TeamName != nil {
					teamNames = append(teamNames, *config.TeamName)
				}
			}

			_, _ = fmt.Fprintf(c.App.Writer, "[!] linting succeeded\n")

			return nil
		},
	}
}
