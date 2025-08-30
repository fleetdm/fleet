package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"
	"gopkg.in/yaml.v3"
)

const (
	cmdMigrate = "migrate"

	// Define some well-known map keys we'll use further down.
	keyPath            = "path"
	keyPackages        = "packages"
	keySoftware        = "software"
	keyCategories      = "categories"
	keySelfService     = "self_service"
	keyLabelsExclude   = "labels_exclude_any"
	keyLabelsInclude   = "labels_include_any"
	keySetupExperience = "setup_experience"
)

func cmdMigrateExec(ctx context.Context, args Args) error {
	if len(args.Commands) < 1 {
		showUsageAndExit(
			1,
			"expected positional argument specifying the path to your Fleet GitOps "+
				"YAML files",
		)
	}
	from := args.Commands[0]

	// Create a temp directory to which we'll write the backup archive.
	tmpDir, err := mkBackupDir()
	if err != nil {
		return err
	}

	// Backup the provided migration target.
	archivePath, err := backup(ctx, from, tmpDir)
	if err != nil {
		return err
	}
	// TODO: use a named return for 'error' and put this archive back if we fail
	// the migration ('err != nil').
	_ = archivePath

	// Packages can belong to >1 team file. So, to avoid unnecessary i/o we
	// capture the state we mutate(read: delete) for each package file here and
	// refer back to it if we encounter the same item again.
	known := make(map[string]map[string]any)
	// Track successful/failed conversions.
	success := 0
	failed := 0
	for item, err := range fsEnum(from) {
		// Handle any iterator errors.
		if err != nil {
			log.Errorf("Encountered error in file system enumeration: %s.", err)
			failed += 1
			continue
		}

		// Ignore directories.
		if item.Stats.IsDir() {
			log.Debugf("Ignoring directory: %s.", item.Path)
			continue
		}

		// Ignore non-YAML files.
		if !strings.HasSuffix(item.Path, ".yml") &&
			!strings.HasSuffix(item.Path, ".yaml") {
			log.Debugf("Ignoring non-YAML file: %s.", item.Path)
			continue
		}

		// Get a read-writable handle to the input file.
		teamFile, err := os.OpenFile(item.Path, fileFlagsReadWrite, 0)
		if err != nil {
			log.Error(
				"Failed to get a read-writable handle to file.",
				"File Path", item.Path,
				"Error", err,
			)
			failed += 1
			continue
		}

		// Unmarshal the file content.
		team := make(map[string]any)
		err = yaml.NewDecoder(teamFile).Decode(&team)
		if err != nil {
			log.Error(
				"Failed to unmarshal file.",
				"Team File", item.Path,
				"Error", err,
			)
			failed += 1
			continue
		}

		// Look for a 'software' key.
		software, ok := team[keySoftware].(map[string]any)
		if !ok {
			log.Debug("Skipping non-team YAML file.", "File", item.Path)
			continue
		}

		// Look for a 'packages' key.
		packagesObjects, ok := software[keyPackages].([]any)
		if !ok {
			log.Debug(
				"Team file's software object contains no packages.",
				"Team File", item.Path,
			)
			continue
		}

		// Iterate each 'packages' item.
		//
		// To avoid unnecessarily serializing a file back to disk when no changes
		// have been performed, since this would blow away all comments and manual
		// spacing, we need to count the number of changes which we actaully
		// apply. If this count is zero, we simply skip the re-encode step.
		changeCount := 0
		for i, packagesObject := range packagesObjects {
			// Attempt to assert the 'packages' YAML-array item as a map.
			pkg, ok := packagesObject.(map[string]any)
			if !ok {
				log.Warn(
					"Software->packages object is nil.",
					"Team File", item.Path,
				)
				continue
			}

			// Look for a 'path' key in the package map, assert it as a 'string'.
			packagePath, ok := pkg[keyPath].(string)
			if !ok || packagePath == "" {
				log.Debugf(
					"The software package at index [%d] has no 'path' key, skipping.",
					i,
				)
				continue
			}

			// Construct an absolute path from the 'path' key's value.
			absPath := filepath.Join(
				filepath.Dir(item.Path),
				packagePath,
			)
			absPath, err = filepath.Abs(absPath)
			if err != nil {
				log.Error(
					"Failed to construct absolute path to referenced package package.",
					"File Path", item.Path,
					"Package Path", packagePath,
				)
				failed += 1
				continue
			}

			// Check if this is a package file we've processed previously. If not
			// unmarshal the file, record & delete the fields we're relocating and
			// serialize the updated package representation back to disk.
			var state map[string]any
			if state, ok = known[absPath]; !ok {
				state = make(map[string]any)
				// Get a read-writable handle to the package file.
				packageFile, err := os.OpenFile(absPath, fileFlagsReadWrite, 0)
				if err != nil {
					log.Error(
						"Failed to get a read-writable handle to package file.",
						"Team File", item.Path,
						"Error", err,
					)
					failed += 1
					continue
				}

				// Decode the package file.
				pkg := make(map[string]any)
				err = yaml.NewDecoder(packageFile).Decode(pkg)
				if err != nil {
					log.Error(
						"failed to decode package file",
						"Package File",
						"Error", err,
					)
					failed += 1
					continue
				}

				// Record and delete the fields we care about.

				if v, ok := pkg[keySelfService]; ok {
					if b, ok := v.(bool); ok {
						state[keySelfService] = b
					}
					delete(pkg, keySelfService)
				}

				if v, ok := pkg[keySetupExperience]; ok {
					if v, ok := v.(bool); ok {
						state[keySetupExperience] = v
					}
					delete(pkg, keySetupExperience)
				}

				if v, ok := pkg[keyLabelsInclude]; ok {
					if v, ok := v.([]any); ok && len(v) > 0 {
						includes := make([]string, 0, len(v))
						for i := range len(v) {
							if s, ok := v[i].(string); ok {
								includes = append(includes, s)
							}
						}
						if len(includes) > 0 {
							state[keyLabelsInclude] = includes
						}
					}
					delete(pkg, keyLabelsInclude)
				}

				if v, ok := pkg[keyLabelsExclude]; ok {
					if v, ok := v.([]any); ok && len(v) > 0 {
						excludes := make([]string, 0, len(v))
						for i := range len(v) {
							if s, ok := v[i].(string); ok {
								excludes = append(excludes, s)
							}
						}
						if len(excludes) > 0 {
							state[keyLabelsExclude] = excludes
						}
					}
					delete(pkg, keyLabelsExclude)
				}

				if v, ok := pkg[keyCategories]; ok {
					if v, ok := v.([]any); ok && len(v) > 0 {
						categories := make([]string, 0, len(v))
						for i := range len(v) {
							if s, ok := v[i].(string); ok {
								categories = append(categories, s)
							}
						}
						if len(categories) > 0 {
							state[keyCategories] = categories
						}
					}
					delete(pkg, keyCategories)
				}

				// Seek to file start for re-encode.
				_, err = packageFile.Seek(0, io.SeekStart)
				if err != nil {
					log.Error(
						"Failed to seek to file start for re-encode.",
						"Team File", item.Path,
						"Package File", packagePath,
						"Error", err,
					)
					failed += 1
					continue
				}

				// Serialize the package file back to disk.
				enc := yaml.NewEncoder(packageFile)
				enc.SetIndent(2)
				err = enc.Encode(pkg)
				if err != nil {
					log.Error(
						"Failed to re-encode package file.",
						"Team File", item.Path,
						"Package File", packagePath,
						"Error", err,
					)
					failed += 1
					continue
				}

				// Seek to identify the number of bytes we wrote during the YAML encode.
				n, err := packageFile.Seek(0, io.SeekCurrent)
				if err != nil {
					log.Error(
						"Failed to seek after package file re-encode.",
						"Team File", item.Path,
						"Package File", packagePath,
						"Error", err,
					)
					failed += 1
					continue
				}

				// Truncate at the number of bytes we just wrote.
				err = packageFile.Truncate(n)
				if err != nil {
					log.Error(
						"Failed to truncate package file after re-encode.",
						"Team File", item.Path,
						"Package File", packagePath,
						"Error", err,
					)
					failed += 1
					continue
				}

				// Close the package file.
				err = packageFile.Close()
				if err != nil {
					log.Error(
						"Failed to close package file after re-encode.",
						"Team File", item.Path,
						"Package File", packagePath,
						"Error", err,
					)
					failed += 1
					continue
				}

				// Store the package state.
				known[absPath] = state
			}

			// Relocate any items we removed from the package file to this package
			// entry.

			if v, ok := state[keySelfService]; ok {
				changeCount += 1
				pkg[keySelfService] = v
			}

			if v, ok := state[keySetupExperience]; ok {
				changeCount += 1
				pkg[keySetupExperience] = v
			}

			if v, ok := state[keyCategories]; ok {
				changeCount += 1
				pkg[keyCategories] = v
			}

			if v, ok := state[keyLabelsInclude]; ok {
				changeCount += 1
				pkg[keyLabelsInclude] = v
			}

			if v, ok := state[keyLabelsExclude]; ok {
				changeCount += 1
				pkg[keyLabelsExclude] = v
			}
		}

		// Only re-encode the file if we actually changed something.
		if changeCount > 0 {
			// Seek to the file start for the YAML-encode.
			_, err = teamFile.Seek(0, io.SeekStart)
			if err != nil {
				log.Error(
					"Failed to seek to team file start for YAML encode.",
					"Team File", item.Path,
					"Error", err,
				)
				failed += 1
				continue
			}

			// Serialize the team file back to disk.
			enc := yaml.NewEncoder(teamFile)
			enc.SetIndent(2)
			err = enc.Encode(team)
			if err != nil {
				log.Error(
					"Failed to YAML-encode updated team file back to disk.",
					"Team File", item.Path,
					"Error", err,
				)
				failed += 1
				continue
			}

			// Identify the number of bytes we just wrote.
			n, err := teamFile.Seek(0, io.SeekCurrent)
			if err != nil {
				log.Error(
					"Failed to identify number of bytes written following team file "+
						"YAML-encode.",
					"Team File", item.Path,
					"Error", err,
				)
				failed += 1
				continue
			}

			// Truncate the file at the number of bytes we wrote during the
			// YAML-encode.
			err = teamFile.Truncate(n)
			if err != nil {
				log.Error(
					"Failed to truncate team file following YAML-encode.",
					"Team File", item.Path,
					"Error", err,
				)
				failed += 1
				continue
			}
		}

		// Close the team YAML file.
		err = teamFile.Close()
		if err != nil {
			log.Error(
				"Failed to close team YAML file following YAML-encode.",
				"Team File", item.Path,
				"Error", err,
			)
			failed += 1
			continue
		}

		// Success!
		if changeCount > 0 {
			log.Info(
				"Successfully applied transforms to team file.",
				"Team File", item.Path,
				"Count", changeCount,
			)
			success += 1
		}
	}

	log.Info(
		"Migration complete.",
		"Successful", success,
		"Failed", failed,
	)

	if f := failed; f > 0 {
		return errors.New("encountered failures during attempted GitOps migration")
	}

	return nil
}
