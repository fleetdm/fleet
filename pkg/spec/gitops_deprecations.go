package spec

import (
	"fmt"
	"strings"
)

// DeprecatedKeyMapping defines a mapping from an old YAML key path to a new one.
// Paths use dot notation for nested keys, with [] to indicate "all array elements".
// Examples:
//   - "team_settings" -> "settings"
//   - "queries" -> "reports"
//   - "org_settings.mdm.apple_business_manager[].macos_team" -> "org_settings.mdm.apple_business_manager[].macos_fleet"
type DeprecatedKeyMapping struct {
	OldPath string
	NewPath string
}

// DeprecatedGitOpsKeyMappings defines all deprecated GitOps YAML key mappings.
// When adding new deprecations, add them here.
var DeprecatedGitOpsKeyMappings = []DeprecatedKeyMapping{
	// Top-level keys
	{"team_settings", "settings"},
	{"queries", "reports"},

	// Nested keys in org_settings.mdm.apple_business_manager[]
	{"org_settings.mdm.apple_business_manager[].macos_team", "org_settings.mdm.apple_business_manager[].macos_fleet"},
	{"org_settings.mdm.apple_business_manager[].ios_team", "org_settings.mdm.apple_business_manager[].ios_fleet"},
	{"org_settings.mdm.apple_business_manager[].ipados_team", "org_settings.mdm.apple_business_manager[].ipados_fleet"},

	// Nested keys in org_settings.mdm.volume_purchasing_program[]
	{"org_settings.mdm.volume_purchasing_program[].teams", "org_settings.mdm.volume_purchasing_program[].fleets"},
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
		logFn("[!] '%s' is deprecated; use '%s' instead\n", fullOldPath, fullNewPath)
	}

	// Copy value to new key and remove old key
	data[newKey] = oldValue
	delete(data, oldKey)

	return nil
}
