package main

import (
	"reflect"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/invopop/jsonschema"
)

// GitOpsSpec is our own top-level struct spelling out the real GitOps YAML keys.
// spec.GitOps has no json tags, so reflecting it directly would give PascalCase
// keys, so this reuses Fleet's existing typed structs for each section instead.
type GitOpsSpec struct {
	Name         string                    `json:"name,omitempty"`
	OrgSettings  *spec.GitOpsOrgSettings   `json:"org_settings,omitempty"`
	TeamSettings *spec.GitOpsFleetSettings `json:"settings,omitempty"`
	AgentOptions *fleet.AgentOptions       `json:"agent_options,omitempty"`
	Controls     Controls                  `json:"controls"`
	Policies     []*spec.GitOpsPolicySpec  `json:"policies,omitempty"`
	Reports      []*spec.Query             `json:"reports,omitempty"`
	Software     spec.GitOpsSoftware       `json:"software"`
	Labels       []*fleet.LabelSpec        `json:"labels,omitempty"`
}

// Controls covers the `controls:` section with real types. spec.GitOpsControls
// types nearly everything as `any` (so yamlls offers no completion for it) and
// leaks an internal `Defined` field, so we spell it out here.
type Controls struct {
	AndroidEnabledAndConfigured bool `json:"android_enabled_and_configured"`
	WindowsEnabledAndConfigured bool `json:"windows_enabled_and_configured"`
	EnableDiskEncryption        bool `json:"enable_disk_encryption"`
	EnableRecoveryLockPassword  bool `json:"enable_recovery_lock_password"`
	WindowsRequireBitLockerPIN  bool `json:"windows_require_bitlocker_pin"`

	MacOSUpdates   *fleet.AppleOSUpdateSettings `json:"macos_updates"`
	IOSUpdates     *fleet.AppleOSUpdateSettings `json:"ios_updates"`
	IPadOSUpdates  *fleet.AppleOSUpdateSettings `json:"ipados_updates"`
	WindowsUpdates *fleet.WindowsUpdates        `json:"windows_updates"`

	MacOSSetup               *fleet.MacOSSetup               `json:"macos_setup" renameto:"setup_experience"`
	AppleAccountProvisioning *fleet.AppleAccountProvisioning `json:"apple_account_provisioning"`
	Scripts                  []fleet.BaseItem                `json:"scripts"`

	MacOSSettings   *fleet.MacOSSettings   `json:"macos_settings" renameto:"apple_settings"`
	WindowsSettings *fleet.WindowsSettings `json:"windows_settings"`
	AndroidSettings *fleet.AndroidSettings `json:"android_settings"`

	// Remaining keys left loose for now, accepting any value.
	MacOSMigration                  any `json:"macos_migration"`
	WindowsMigrationEnabled         any `json:"windows_migration_enabled"`
	EnableTurnOnWindowsMDMManually  any `json:"enable_turn_on_windows_mdm_manually"`
	WindowsEntraTenantIDs           any `json:"windows_entra_tenant_ids"`
	WindowsEntraClientIDs           any `json:"windows_entra_client_ids"`
	AppleRequireHardwareAttestation any `json:"apple_require_hardware_attestation"`
}

// goTypeToJSON maps a Go scalar type name to its JSON Schema type.
func goTypeToJSON(name string) string {
	switch name {
	case "bool":
		return "boolean"
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	}
	return ""
}

// typeMapper overrides schemas for Fleet/optjson wrapper types that don't
// reflect to the JSON they actually marshal to.
func typeMapper(goType reflect.Type) *jsonschema.Schema {
	packagePath := goType.PkgPath()

	// json.RawMessage reflects to a bare `true` schema, which yamlls won't offer
	// in completion. In GitOps these blobs (agent_options.config, command_line_flags,
	// extensions, update_channels) are objects, so type them as such.
	if packagePath == "encoding/json" && goType.Name() == "RawMessage" {
		return &jsonschema.Schema{Type: "object"}
	}

	// fleet.Duration embeds time.Duration (also named "Duration"), so invopop
	// emits a self-referential $def that overflows yamlls' resolver. It marshals
	// to a string like "24h".
	if strings.HasSuffix(packagePath, "server/fleet") && goType.Name() == "Duration" {
		return &jsonschema.Schema{Type: "string"}
	}

	// optjson.Bool/String/Int/Slice[T]/Any[T] marshal to their Value, not the
	// internal {Set, Valid, Value} struct.
	if strings.Contains(packagePath, "pkg/optjson") {
		valueField, ok := goType.FieldByName("Value")
		if ok {
			return schemaForType(valueField.Type)
		}

		// BoolOr[T]/StringOr[T] marshal to a scalar OR T (e.g. install_software is
		// `true` or an object). Their generic def names contain the qualified type
		// param (slashes) which yamlls can't resolve as a $ref, and a bare {} isn't
		// offered in completion, so express both arms as an anyOf.
		scalar := "boolean"
		_, hasString := goType.FieldByName("String")
		if hasString {
			scalar = "string"
		}
		return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: scalar}, {Type: "object"}}}
	}

	return nil
}

// schemaForType maps a reflect.Type to a JSON Schema type (used for optjson Value fields).
func schemaForType(goType reflect.Type) *jsonschema.Schema {
	for goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	switch goType.Kind() {
	case reflect.Bool:
		return &jsonschema.Schema{Type: "boolean"}
	case reflect.String:
		return &jsonschema.Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &jsonschema.Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &jsonschema.Schema{Type: "number"}
	case reflect.Slice, reflect.Array:
		return &jsonschema.Schema{Type: "array", Items: schemaForType(goType.Elem())}
	default:
		return &jsonschema.Schema{Type: "object"}
	}
}
