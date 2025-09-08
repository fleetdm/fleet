package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
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
	keyControls        = "controls"
	keyMacosSetup      = "macos_setup"
	keyPackagePath     = "package_path"
)

func cmdMigrateExec(ctx context.Context, args Args) error {
	if len(args.Commands) < 1 {
		showUsageAndExit(
			1,
			"please specify the path to your Fleet GitOps YAML files",
		)
	}
	from, err := filepath.Abs(args.Commands[0])
	if err != nil {
		return fmt.Errorf(
			"failed to derive absolute input path(%s): %w",
			args.Commands[0], err,
		)
	}

	// Create a temp directory to which we'll write the backup archive.
	tmpDir, err := mkBackupDir()
	if err != nil {
		return err
	}

	// Backup the provided migration target.
	_, err = backup(ctx, from, tmpDir)
	if err != nil {
		return err
	}

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
			if isYAMLUnmarshalSeqError(err) {
				log.Debugf(
					"Skipping file with array item(s) at the root: %s.",
					item.Path,
				)
				continue
			}
			log.Error(
				"Failed to unmarshal file.",
				"Team File", item.Path,
				"Error", err,
			)
			failed += 1
			continue
		}

		// To avoid unnecessarily serializing a file back to disk when no changes
		// have been performed, since this would blow away all comments and manual
		// spacing, we need to count the number of changes which we actaully
		// apply. If this count is zero, we simply skip the re-encode step.
		teamChangeCount := 0

		// NOTE(max): The below works but was de-scoped.
		//
		// Relocate '.controls.software.packages[].package_path' to
		// '.software.packages[].setup_experience=true'.
		//
		// For any '.controls.software.packages' objects which hold a 'package_path'
		// key:
		// - Resolve the absolute path for the software package file referenced and
		// record this absolute path to 'pkgsWithSetupExp'.
		// - Delete the 'package_path' key.
		// - If the 'package_path' key is the only key in this 'packages' array
		// object, drop that array index.
		// NOTE(max): The below works but was de-scoped.
		//
		// pkgsWithSetupExp := map[string]struct{}{}
		// Retrieve the 'controls' key, assert as 'map[string]any'.
		// if controls, ok := team[keyControls].(map[string]any); ok {
		// 	// Retrieve the 'macos_setup' key, assert as 'map[string]any'.
		// 	if macosSetup, ok := controls[keyMacosSetup].(map[string]any); ok {
		// 		// Retrieve the 'software' key, assert as '[]any'.
		// 		if pkgs, ok := macosSetup[keySoftware].([]any); ok {
		// 			pkgChangeCount := 0
		// 			// We iterate the slice backward so we can mutate it as we go.
		// 			for i, pkg := range slices.Backward(pkgs) {
		// 				// Assert the package as 'map[string]any'.
		// 				if pkg, ok := pkg.(map[string]any); ok {
		// 					// Retrieve the 'package_path' key, assert as 'string'.
		// 					if pkgPath, ok := pkg[keyPackagePath].(string); ok {
		// 						// Resolve the absolute path to the referenced package file.
		// 						pkgPathAbs, err := resolvePackagePath(item.Path, pkgPath)
		// 						if err != nil {
		// 							log.Errorf("%s.", err)
		// 							failed += 1
		// 							continue
		// 						}
		// 						// Capture the absolute path as a key in our map.
		// 						pkgsWithSetupExp[pkgPathAbs] = struct{}{}
		// 						// Delete this map key.
		// 						delete(pkg, keyPackagePath)
		// 						// If 'package_path' was the final map key, delete the entire
		// 						// slice item.
		// 						if len(pkg) == 0 {
		// 							pkgs = slices.Delete(pkgs, i, i+1)
		// 						}
		// 						// Signal mutation.
		// 						pkgChangeCount += 1
		// 					}
		// 				}
		// 			}
		// 			// If we mutated 'pkgs', update the 'macosSetup' map key.
		// 			if pkgChangeCount > 0 {
		// 				if len(pkgs) == 0 {
		// 					// If the resulting 'pkgs' slice is empty, just delete the key.
		// 					delete(macosSetup, keySoftware)
		// 				} else {
		// 					// Otherwise, insert the updated slice.
		// 					macosSetup[keySoftware] = pkgs
		// 				}
		// 			}
		// 		}
		// 	}
		// }

		// Relocate keys 'self_service', 'labels_include', 'labels_exclude' and
		// 'categories' from software package files to the '.software.packages'
		// array object in the team file.
		//
		// For any '.software.packages' array items which contain a 'path' key:
		// - Resolve the absolute path for the software package file referenced by
		//   the 'path' key's value.
		// - Check the 'known' map for existence of this absolute path, if not found:
		//   - Read and unmarshal the software package file.
		//   - If the software package file contains any of the above keys:
		//     - Record the key's value.
		//     - Delete the key.
		//       * Increment the 'swPkgChangeCount' counter for each key deleted.
		//   - If the 'swPkgChangeCount' counter is > 0, serialize the software
		//     package file back to disk.
		//   - Apply any values we recorded from the software package file to this
		//     software package array object.
		//     * Increment the 'teamChangeCount' counter for each value added.
		//   - If the 'teamChangeCount' is > 0, serialize the team file back to
		//     disk.

		// Retrieve the 'software' key, assert as 'map[string]any'.
		if software, ok := team[keySoftware].(map[string]any); ok {
			// Retrieve the 'packages' key, assert as '[]any'.
			if pkgs, ok := software[keyPackages].([]any); ok {
				// Iterate all packages.
				for _, pkg := range pkgs {
					// Assert the package as 'map[string]any'.
					if pkg, ok := pkg.(map[string]any); ok {
						// Retrieve the 'path' key, assert as 'string'.
						if pkgPath, ok := pkg[keyPath].(string); ok {
							// Construct an absolute path from the 'path' key's value.
							pkgPath, err := resolvePackagePath(item.Path, pkgPath)
							if err != nil {
								log.Errorf("%s.", err)
								failed += 1
								continue
							}

							// Check if this is a package file we've processed previously.
							//
							// If not unmarshal the file, record & delete the fields we're
							// relocating and serialize the updated package representation
							// back to disk.
							var state map[string]any
							if state, ok = known[pkgPath]; !ok {
								state = make(map[string]any)
								// Track mutations to the package file.
								//
								// If we reach the serialization point and this count is zero we skip
								// the re-encode to file to preserve formatting + comments where we can.
								var pkgChangeCount int

								// Get a read-writable handle to the package file.
								packageFile, err := os.OpenFile(pkgPath, fileFlagsReadWrite, 0)
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
										"Failed to decode package file.",
										"Package File", item.Path,
										"Error", err,
									)
									failed += 1
									continue
								}

								// Record and delete the fields we're migrating.

								// Look for the 'self_service' value.
								if v, ok := pkg[keySelfService]; ok {
									if b, ok := v.(bool); ok {
										state[keySelfService] = b
									}
									delete(pkg, keySelfService)
									pkgChangeCount += 1
								}

								// Look for 'labels_include' items.
								if v, ok := pkg[keyLabelsInclude]; ok {
									if v, ok := v.([]any); ok {
										labelIncludes := make([]string, 0, len(v))
										for i := range len(v) {
											if labelInclude, ok := v[i].(string); ok {
												labelIncludes = append(labelIncludes, labelInclude)
											}
										}
										if len(labelIncludes) > 0 {
											state[keyLabelsInclude] = labelIncludes
										}
									}
									delete(pkg, keyLabelsInclude)
									pkgChangeCount += 1
								}

								// Look for 'labels_exclude' items.
								if v, ok := pkg[keyLabelsExclude]; ok {
									if v, ok := v.([]any); ok {
										labelExcludes := make([]string, 0, len(v))
										for i := range len(v) {
											if labelExclude, ok := v[i].(string); ok {
												labelExcludes = append(labelExcludes, labelExclude)
											}
										}
										if len(labelExcludes) > 0 {
											state[keyLabelsExclude] = labelExcludes
										}
									}
									delete(pkg, keyLabelsExclude)
									pkgChangeCount += 1
								}

								// Look for 'categories' items.
								if v, ok := pkg[keyCategories]; ok {
									if v, ok := v.([]any); ok {
										categories := make([]string, 0, len(v))
										for i := range len(v) {
											if category, ok := v[i].(string); ok {
												categories = append(categories, category)
											}
										}
										if len(categories) > 0 {
											state[keyCategories] = categories
										}
									}
									delete(pkg, keyCategories)
									pkgChangeCount += 1
								}

								// Only re-encode if we actually mutated something.
								if pkgChangeCount > 0 {
									// Seek to file start for re-encode.
									_, err = packageFile.Seek(0, io.SeekStart)
									if err != nil {
										log.Error(
											"Failed to seek to file start for re-encode.",
											"Team File", item.Path,
											"Package File", pkgPath,
											"Error", err,
										)
										failed += 1
										continue
									}

									// Re-encode the package file.
									enc := yaml.NewEncoder(packageFile)
									enc.SetIndent(2)
									err = enc.Encode(pkg)
									if err != nil {
										log.Error(
											"Failed to re-encode package file.",
											"Team File", item.Path,
											"Package File", pkgPath,
											"Error", err,
										)
										failed += 1
										continue
									}

									// Seek to identify the number of bytes we wrote during the
									// YAML encode.
									n, err := packageFile.Seek(0, io.SeekCurrent)
									if err != nil {
										log.Error(
											"Failed to seek after package file re-encode.",
											"Team File", item.Path,
											"Package File", pkgPath,
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
											"Package File", pkgPath,
											"Error", err,
										)
										failed += 1
										continue
									}
								}

								// Close the package file.
								err = packageFile.Close()
								if err != nil {
									log.Error(
										"Failed to close package file after re-encode.",
										"Team File", item.Path,
										"Package File", pkgPath,
										"Error", err,
									)
									failed += 1
									continue
								}

								// Store the package state.
								known[pkgPath] = state
							}

							// Relocate any items we removed from the package file to this
							// package entry.

							// Key: 'self_service'.
							if v, ok := state[keySelfService]; ok {
								teamChangeCount += 1
								pkg[keySelfService] = v
							}

							// Key: 'categories'.
							if v, ok := state[keyCategories]; ok {
								teamChangeCount += 1
								pkg[keyCategories] = v
							}

							// Key: 'labels_include'.
							if v, ok := state[keyLabelsInclude]; ok {
								teamChangeCount += 1
								pkg[keyLabelsInclude] = v
							}

							// Key: 'labels_exclude'.
							if v, ok := state[keyLabelsExclude]; ok {
								teamChangeCount += 1
								pkg[keyLabelsExclude] = v
							}

							// NOTE(max): The below works but was de-scoped.
							//
							// If we found+removed a 'package_path' key for this item earlier,
							// add the 'setup_experience' key with a value of 'true'.
							//
							// if _, ok := pkgsWithSetupExp[pkgPath]; ok {
							// 	teamChangeCount += 1
							// 	pkg[keySetupExperience] = true
							// }
						}
					}
				}
			}
		}

		// Only re-encode the team file if we actually changed something.
		if teamChangeCount > 0 {
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
		if teamChangeCount > 0 {
			log.Info(
				"Successfully applied transforms to team file.",
				"Team File", item.Path,
				"Migrated Fields", teamChangeCount,
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

// resolvePackagePath resolves an absolute file path to the software package at
// 'pkgPath' relative to the parent directory of the 'teamFilePath'.
//
// Example:
// Absolute path to Fleet GitOps file root : /home/fleet
// teamFilePath                            : teams/workstations.yml
// pkgPath                                 : ../software/firefox.yml
// Output                                  : /home/fleet/software/firefox.yml
func resolvePackagePath(teamFilePath, pkgPathRaw string) (string, error) {
	// Concat the package path (usually relative) to the parent directory of the
	// team file path.
	//
	// If 'teamFile' is _not_ a file, skip the 'filepath.Dir' call.
	if filepath.Ext(teamFilePath) != "" {
		teamFilePath = filepath.Dir(teamFilePath)
	}
	pkgPathRel := filepath.Join(teamFilePath, pkgPathRaw)

	// Resolve the absolute path for this software package file.
	pkgPathAbs, err := filepath.Abs(pkgPathRel)
	if err != nil {
		return "", fmt.Errorf(
			"failed to resolve absolute software package file path from team file "+
				"[%s] and software package path [%s]",
			teamFilePath, pkgPathRaw,
		)
	}

	return pkgPathAbs, nil
}

// isYAMLUnmarshalSeqError simply reports whether the provided error (returned
// by 'yaml.NewDecoder().Decode' or 'yaml.Unmarshal') is in fact a
// 'yaml.TypeError' which contains a single message regarding a failed unmarshal
// of a sequence (array) to a map.
//
// Certain GitOps YAML files (ex: 'it-and-security\lib\windows\queries\all-x86-hosts.yml')
// use an array as the outermost data type. These cases are rare but exist, and
// this will break the YAML unmarshal since we expect objects at the outermost
// level for the purposes of this tool. This function just reports whether an
// encountered error is one such case.
func isYAMLUnmarshalSeqError(err error) bool {
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		// This sure is an ugly way to check error conditions, but in the case of
		// this package it builds a slice of errors as strings so we have no choice.
		if len(typeErr.Errors) == 1 && strings.Contains(
			typeErr.Errors[0], "cannot unmarshal !!seq into map[string]interface",
		) {
			return true
		}
	}
	return false
}
