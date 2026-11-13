package spec

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
)

// DeprecatedKeyMapping defines a mapping from an old YAML key path to a new one.
// Paths use dot notation for nested keys, with [] to indicate "all array elements".
// Examples:
//   - "controls.macos_settings" -> "controls.apple_settings"
//   - "queries" -> "reports"
//   - "org_settings.mdm.apple_business_manager" -> "org_settings.mdm.apple_business"
type DeprecatedKeyMapping struct {
	OldPath string
	NewPath string
}

// DeprecatedGitOpsKeyMappings is the single source of truth for all deprecated key renames.
// It serves two purposes:
//  1. ApplyDeprecatedKeyMappings uses the full paths to migrate deprecated keys in gitops YAML input.
//  2. buildAliasRules (in generate_gitops.go) extracts leaf key names to rename keys in
//     serialized output for generate_gitops, fleetctl get, and fleetctl apply.
//
// When adding new deprecations, add them here. The team-related keys that Fleet 5 removed are in RemovedGitOpsKeys instead.
var DeprecatedGitOpsKeyMappings = []DeprecatedKeyMapping{
	// Top-level gitops keys
	{"queries", "reports"},

	// Controls: macos_settings -> apple_settings (parent first, then children)
	{"controls.macos_settings", "controls.apple_settings"},
	{"controls.apple_settings.custom_settings", "controls.apple_settings.configuration_profiles"},

	// Controls: windows_settings children
	{"controls.windows_settings.custom_settings", "controls.windows_settings.configuration_profiles"},

	// Controls: android_settings children
	{"controls.android_settings.custom_settings", "controls.android_settings.configuration_profiles"},

	// Controls: macos_setup -> setup_experience (parent first, then children)
	{"controls.setup_experience.enable_managed_local_account", "controls.setup_experience.enable_create_local_admin_account"},

	// Org settings: server_settings
	{"org_settings.server_settings.live_query_disabled", "org_settings.server_settings.live_reporting_disabled"},
	{"org_settings.server_settings.query_reports_disabled", "org_settings.server_settings.discard_reports_data"},
	{"org_settings.server_settings.query_report_cap", "org_settings.server_settings.report_cap"},

	// Org settings: org_info logo URL fields renamed to mode-aware variants.
	{"org_settings.org_info.org_logo_url", "org_settings.org_info.org_logo_url_dark_mode"},
	{"org_settings.org_info.org_logo_url_light_background", "org_settings.org_info.org_logo_url_light_mode"},

	// org_settings.mdm.apple_business_manager -> apple_business (parent rename
	// runs before the nested children below so they resolve against the new
	// parent name).
	{"org_settings.mdm.apple_business_manager", "org_settings.mdm.apple_business"},

	// The following entries are renameto tags on struct fields that appear in serialized
	// API output (fleetctl get and fleetctl apply) but are not gitops input keys. They are
	// included here so that buildAliasRules can derive them. The paths are leaf-only (no dots)
	// since they don't participate in ApplyDeprecatedKeyMappings traversal.
	{"available_teams", "available_fleets"},
	{"default_team", "default_fleet"},
	{"host_team_id", "host_fleet_id"},
	{"inherited_query_count", "inherited_report_count"},
	{"ios_team_id", "ios_fleet_id"},
	{"ipados_team_id", "ipados_fleet_id"},
	{"live_query_results", "live_report_results"},
	{"macos_team_id", "macos_fleet_id"},
	{"query_count", "report_count"},
	{"query_id", "report_id"},
	{"query_ids", "report_ids"},
	{"query_name", "report_name"},
	{"query_stats", "report_stats"},
	{"scheduled_query_id", "scheduled_report_id"},
	{"scheduled_query_name", "scheduled_report_name"},
	{"team", "fleet"},
	{"team_id", "fleet_id"},
	{"team_ids_by_name", "fleet_ids_by_name"},
	{"team_ids", "fleet_ids"},
	{"team_name", "fleet_name"},
}

// DeprecatedGitOpsKeyScopes confines a leaf rename to the objects it belongs to, keyed by the leaf name buildAliasRules
// indexes on. ApplyDeprecatedKeyMappings doesn't need this (it matches full paths), but the serialization rules derived
// from the leaf names do: without a scope, a name that is deprecated in one object gets rewritten in every other object
// that happens to use it.
//
// Keep in sync with the `renamescope` struct tags, which do the same job for the API request/response layer.
var DeprecatedGitOpsKeyScopes = map[string][]string{
	// Deprecated spelling of the Apple toggle under setup_experience, but the canonical name of the Windows toggle under
	// windows_settings.
	"enable_managed_local_account": {"setup_experience"},
}

// RemovedGitOpsKeys are the team-related keys that Fleet 5 removed from GitOps files. They are rejected, not migrated
// (RejectRemovedKeys), and buildAliasRules also reads them, because the structs that fleetctl serializes still carry the old names.
// Every path has an entry in pkg/removednames, which supplies the error message.
var RemovedGitOpsKeys = []DeprecatedKeyMapping{
	{"team_settings", "settings"},
	{"org_settings.mdm.apple_bm_default_team", "org_settings.mdm.apple_business"},
	{"org_settings.mdm.apple_business[].macos_team", "org_settings.mdm.apple_business[].macos_fleet"},
	{"org_settings.mdm.apple_business[].ios_team", "org_settings.mdm.apple_business[].ios_fleet"},
	{"org_settings.mdm.apple_business[].ipados_team", "org_settings.mdm.apple_business[].ipados_fleet"},
	{"org_settings.mdm.apple_business[].byod_team", "org_settings.mdm.apple_business[].byod_fleet"},
	{"org_settings.mdm.volume_purchasing_program[].teams", "org_settings.mdm.volume_purchasing_program[].fleets"},
}

// removedKeyParents are the old spellings of a parent key in RemovedGitOpsKeys that ApplyDeprecatedKeyMappings renames, so a removed key
// is found under either spelling, including in a file that ApplyDeprecatedKeyMappings never sees (a file-backed org_settings).
var removedKeyParents = map[string][]string{"apple_business": {"apple_business", "apple_business_manager"}}

// RejectRemovedKeys returns an error naming the file and the removed key for every key of RemovedGitOpsKeys found in the YAML data map.
func RejectRemovedKeys(data map[string]any, filePath string) error {
	var errs []error
	for _, removed := range RemovedGitOpsKeys {
		if hasKeyPath(data, strings.Split(removed.OldPath, ".")) {
			message := fmt.Sprintf("%q is not available anymore", removed.OldPath)
			if entry, ok := removednames.Find(removednames.KindYAMLKey, removed.OldPath, ""); ok {
				message = entry.Message()
			}
			errs = append(errs, fmt.Errorf("%s: %s", filePath, message))
		}
	}
	return errors.Join(errs...)
}

// hasKeyPath reports whether the dot-separated path, with [] for every element of an array, exists in data.
func hasKeyPath(data map[string]any, parts []string) bool {
	name, isArray := strings.CutSuffix(parts[0], "[]")
	spellings, ok := removedKeyParents[name]
	if !ok {
		spellings = []string{name}
	}
	for _, key := range spellings {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch {
		case len(parts) == 1 && !isArray:
			return true
		case isArray:
			elements, _ := value.([]any)
			for _, element := range elements {
				if elementMap, ok := element.(map[string]any); ok && len(parts) > 1 && hasKeyPath(elementMap, parts[1:]) {
					return true
				}
			}
		default:
			if nested, ok := value.(map[string]any); ok && hasKeyPath(nested, parts[1:]) {
				return true
			}
		}
	}
	return false
}

// ApplyDeprecatedKeyMappings walks the YAML data map and migrates deprecated keys to their new names.
// It logs warnings for each deprecated key found and returns an error if both old and new keys are specified.
// After this function returns successfully, only the new key names will be present in the data.
func ApplyDeprecatedKeyMappings(data map[string]any, logFn Logf) error {
	for _, mapping := range DeprecatedGitOpsKeyMappings {
		if err := migrateKeyPath(data, mapping.OldPath, mapping.NewPath, logFn); err != nil {
			return err
		}
	}
	return nil
}

// migrateKeyPath migrates a single deprecated key path to its new path.
// Paths are dot-separated, with [] indicating iteration over array elements.
func migrateKeyPath(data map[string]any, oldPath, newPath string, logFn Logf) error {
	oldParts := strings.Split(oldPath, ".")
	newParts := strings.Split(newPath, ".")

	return migrateKeyPathRecursive(data, oldParts, newParts, oldPath, newPath, logFn)
}

func migrateKeyPathRecursive(data map[string]any, oldParts, newParts []string, fullOldPath, fullNewPath string, logFn Logf) error {
	if len(oldParts) == 0 || len(newParts) == 0 {
		return nil
	}

	oldKey := oldParts[0]
	newKey := newParts[0]

	// Check if this key references an array (e.g. "apple_business_manager[]").
	if trimmed, ok := strings.CutSuffix(oldKey, "[]"); ok {
		oldKey = trimmed
		arr, ok := data[oldKey]
		if !ok {
			return nil // Key doesn't exist, nothing to migrate
		}

		arrSlice, ok := arr.([]any)
		if !ok {
			return nil // Not an array, skip
		}

		// Recurse into each array element with the remaining path parts.
		for i, elem := range arrSlice {
			elemMap, ok := elem.(map[string]any)
			if !ok {
				continue // Not a map, skip
			}

			if err := migrateKeyPathRecursive(elemMap, oldParts[1:], newParts[1:], fullOldPath, fullNewPath, logFn); err != nil {
				return fmt.Errorf("in array element %d: %w", i, err)
			}
		}
		return nil
	}

	// Check if we're at the final key (leaf)
	if len(oldParts) == 1 {
		return migrateLeafKey(data, oldKey, newKey, fullOldPath, fullNewPath, logFn)
	}

	// Recurse into nested map
	nested, ok := data[oldKey]
	if !ok {
		return nil // Key doesn't exist, nothing to migrate
	}

	nestedMap, ok := nested.(map[string]any)
	if !ok {
		return nil // Not a map, skip
	}

	return migrateKeyPathRecursive(nestedMap, oldParts[1:], newParts[1:], fullOldPath, fullNewPath, logFn)
}

// migrateLeafKey handles the actual key migration at the leaf level.
func migrateLeafKey(data map[string]any, oldKey, newKey, fullOldPath, fullNewPath string, logFn Logf) error {
	oldValue, oldExists := data[oldKey]
	_, newExists := data[newKey]

	if !oldExists {
		return nil // Old key doesn't exist, nothing to migrate
	}

	if newExists {
		return fmt.Errorf("cannot specify both '%s' (deprecated) and '%s'; use only '%s'", fullOldPath, fullNewPath, fullNewPath)
	}

	// Log deprecation warning
	if logFn != nil {
		if logging.TopicEnabled(logging.DeprecatedFieldTopic) {
			logFn("[!] '%s' is deprecated; use '%s' instead\n", fullOldPath, fullNewPath)
		}
	}

	// Copy value to new key and remove old key
	data[newKey] = oldValue
	delete(data, oldKey)

	return nil
}
