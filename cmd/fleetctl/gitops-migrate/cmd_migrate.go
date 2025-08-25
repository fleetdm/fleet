package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

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
	log := LoggerFromContext(ctx)

	// Create a temp directory to which we'll write the backup archive.
	tmpDir, err := mkBackupDir()
	if err != nil {
		return err
	}

	// Backup the provided migration target.
	archivePath, err := backup(ctx, args.From, tmpDir)
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
	success := new(atomic.Int32)
	failed := new(atomic.Int32)
	for item, err := range fsEnum(args.From) {
		// Handle any iterator errors.
		if err != nil {
			log.Error("encountered error in file system enumeration", "error", err)
			failed.Add(1)
			continue
		}

		// Ignore directories.
		if item.Stats.IsDir() {
			log.Debug("ignoring directory", "path", item.Path)
			continue
		}

		// Ignore non-YAML files.
		if !strings.HasSuffix(item.Path, ".yml") &&
			!strings.HasSuffix(item.Path, ".yaml") {
			log.Debug("ignoring non-YAML file", "path", item.Path)
			continue
		}

		log := log.With("team_file", item.Path)

		// Get a read-writable handle to the input file.
		teamFile, err := os.OpenFile(item.Path, fileFlagsReadWrite, 0)
		if err != nil {
			log.Error("failed to get a read-writable handle to file", "error", err)
			failed.Add(1)
			continue
		}

		// Unmarshal the file content.
		team := make(map[string]any)
		err = yaml.NewDecoder(teamFile).Decode(&team)
		if err != nil {
			log.Error("failed to unmarshal file", "error", err)
			failed.Add(1)
			continue
		}

		// Look for a 'software' key.
		software, ok := team[keySoftware].(map[string]any)
		if !ok {
			log.Warn("team file contains no software")
			continue
		}

		// Look for a 'packages' key.
		packagesObjects, ok := software[keyPackages].([]any)
		if !ok {
			log.Warn("team file's software object contains no packages")
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
				log.Warn("software->packages object is nil")
				continue
			}

			// Look for a 'path' key in the package map, assert it as a 'string'.
			packagePath, ok := pkg[keyPath].(string)
			if !ok || packagePath == "" {
				log.Error(
					"team YAML file has package with no 'path' key",
					"package_index", i,
				)
				failed.Add(1)
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
					"failed to construct absolute path to referenced package package",
					"package_path", packagePath,
				)
				failed.Add(1)
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
						"failed to get a readable handle to package file",
						"error", err,
					)
					failed.Add(1)
					continue
				}
				log := log.With("package_file", absPath)

				// Decode the package file.
				pkg := make(map[string]any)
				err = yaml.NewDecoder(packageFile).Decode(pkg)
				if err != nil {
					log.Error(
						"failed to decode package file",
						"error", err,
					)
					failed.Add(1)
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
						"failed to seek to file start for re-encode",
						"error", err,
					)
					failed.Add(1)
					continue
				}

				// Serialize the package file back to disk.
				err = yaml.NewEncoder(packageFile).Encode(pkg)
				if err != nil {
					log.Error(
						"failed to re-encode package file",
						"error", err,
					)
					failed.Add(1)
					continue
				}

				// Seek to identify the number of bytes we wrote during the YAML encode.
				n, err := packageFile.Seek(0, io.SeekCurrent)
				if err != nil {
					log.Error(
						"failed to seek after package file re-encode",
						"error", err,
					)
					failed.Add(1)
					continue
				}

				// Truncate at the number of bytes we just wrote.
				err = packageFile.Truncate(n)
				if err != nil {
					log.Error(
						"failed to truncate package file after re-encode",
						"error", err,
					)
					failed.Add(1)
					continue
				}

				// Close the package file.
				err = packageFile.Close()
				if err != nil {
					log.Error(
						"failed to close package file after re-encode",
						"error", err,
					)
					failed.Add(1)
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
					"failed to seek to team file start for YAML encode",
					"error", err,
				)
				failed.Add(1)
				continue
			}

			// Serialize the team file back to disk.
			err = yaml.NewEncoder(teamFile).Encode(team)
			if err != nil {
				log.Error(
					"failed to YAML-encode updated team file back to disk",
					"error", err,
				)
				failed.Add(1)
				continue
			}

			// Identify the number of bytes we just wrote.
			n, err := teamFile.Seek(0, io.SeekCurrent)
			if err != nil {
				log.Error(
					"failed to identify number of bytes written following team file "+
						"YAML-encode",
					"error", err,
				)
				failed.Add(1)
				continue
			}

			// Truncate the file at the number of bytes we wrote during the
			// YAML-encode.
			err = teamFile.Truncate(n)
			if err != nil {
				log.Error(
					"failed to truncate team file following YAML-encode",
					"error", err,
				)
				failed.Add(1)
				continue
			}
		}

		// Close the team YAML file.
		err = teamFile.Close()
		if err != nil {
			log.Error(
				"failed to close team YAML file following YAML-encode",
				"error", err,
			)
			failed.Add(1)
			continue
		}

		// Success!
		if changeCount > 0 {
			log.Info(
				"successfully applied transforms to team file",
				"count", changeCount,
			)
			success.Add(1)
		}
	}

	log.Info(
		"migration complete",
		"successful", success.Load(),
		"failed", failed.Load(),
	)

	if f := failed.Load(); f > 0 {
		return errors.New("encountered failures during attempted GitOps migration")
	}

	return nil
}
