// Command gitops-auto-complete generates a JSON schema from Fleet's GitOps Go
// structs so editors (yamlls) can offer completion/validation for GitOps YAML.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"unicode"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/invopop/jsonschema"
)

// generatedOsqueryOptions is the Fleet-generated file listing every valid osquery
// option (config.options.*). It's an unexported struct, so we parse its AST rather
// than reflect it.
const generatedOsqueryOptions = "../../server/fleet/agent_options_generated.go"

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

// osqueryOptionsSchema parses the generated osqueryOptions struct into a strict
// object schema, so config.options.* gets completion, types, and unknown-key
// validation matching what Fleet enforces.
func osqueryOptionsSchema(path string) (map[string]any, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	props := map[string]any{}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "osqueryOptions" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, fld := range st.Fields.List {
			ident, ok := fld.Type.(*ast.Ident)
			if !ok || fld.Tag == nil {
				continue
			}
			name := strings.Split(reflect.StructTag(strings.Trim(fld.Tag.Value, "`")).Get("json"), ",")[0]
			if jt := goTypeToJSON(ident.Name); name != "" && name != "-" && jt != "" {
				props[name] = map[string]any{"type": jt}
			}
		}
		return false
	})
	if len(props) == 0 {
		return nil, fmt.Errorf("osqueryOptions struct not found in %s", path)
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": props}, nil
}

// GitOpsSpec is our own top-level struct with the real GitOps YAML keys (the
// fields on spec.GitOps have no json tags, so reflecting it directly gives
// PascalCase keys). We reuse Fleet's existing typed structs for each section.
type GitOpsSpec struct {
	Name         string                    `json:"name,omitempty"`
	OrgSettings  *spec.GitOpsOrgSettings   `json:"org_settings,omitempty"`
	Settings     *spec.GitOpsFleetSettings `json:"settings,omitempty"`
	AgentOptions *fleet.AgentOptions       `json:"agent_options,omitempty"`
	Controls     Controls                  `json:"controls,omitempty"`
	Policies     []*spec.GitOpsPolicySpec  `json:"policies,omitempty"`
	Reports      []*spec.Query             `json:"reports,omitempty"`
	Software     spec.GitOpsSoftware       `json:"software,omitempty"`
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

	// Less-common controls keys left loose for now (accept anything).
	MacOSSettings                   any `json:"macos_settings" renameto:"apple_settings"`
	WindowsSettings                 any `json:"windows_settings"`
	MacOSMigration                  any `json:"macos_migration"`
	WindowsMigrationEnabled         any `json:"windows_migration_enabled"`
	EnableTurnOnWindowsMDMManually  any `json:"enable_turn_on_windows_mdm_manually"`
	WindowsEntraTenantIDs           any `json:"windows_entra_tenant_ids"`
	WindowsEntraClientIDs           any `json:"windows_entra_client_ids"`
	AppleRequireHardwareAttestation any `json:"apple_require_hardware_attestation"`
}

// typeMapper overrides schemas for Fleet/optjson wrapper types that don't
// reflect to the JSON they actually marshal to.
func typeMapper(t reflect.Type) *jsonschema.Schema {
	pkg := t.PkgPath()
	// json.RawMessage reflects to a bare `true` schema, which yamlls won't offer
	// in completion. In GitOps these blobs (agent_options.config, command_line_flags,
	// extensions, update_channels) are objects, so type them as such.
	if pkg == "encoding/json" && t.Name() == "RawMessage" {
		return &jsonschema.Schema{Type: "object"}
	}
	// fleet.Duration embeds time.Duration (also named "Duration"), so invopop
	// emits a self-referential $def that overflows yamlls' resolver. It marshals
	// to a string like "24h".
	if strings.HasSuffix(pkg, "server/fleet") && t.Name() == "Duration" {
		return &jsonschema.Schema{Type: "string"}
	}
	// optjson.Bool/String/Int/Slice[T]/Any[T] marshal to their Value, not the
	// internal {Set, Valid, Value} struct.
	if strings.Contains(pkg, "pkg/optjson") {
		if vf, ok := t.FieldByName("Value"); ok {
			return schemaForType(vf.Type)
		}
		// BoolOr[T]/StringOr[T] marshal to a scalar OR T (e.g. install_software is
		// `true` or an object). Their generic def names contain the qualified type
		// param (slashes) which yamlls can't resolve as a $ref, and a bare {} isn't
		// offered in completion, so express both arms as an anyOf.
		scalar := "boolean"
		if _, ok := t.FieldByName("String"); ok {
			scalar = "string"
		}
		return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: scalar}, {Type: "object"}}}
	}
	return nil
}

func schemaForType(t reflect.Type) *jsonschema.Schema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
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
		return &jsonschema.Schema{Type: "array", Items: schemaForType(t.Elem())}
	default:
		return &jsonschema.Schema{Type: "object"}
	}
}

// toSnake converts a Go field name to snake_case. invopop applies KeyNamer to
// the json-tag name when a tag is present, so already-snake tags pass through
// unchanged; untagged Fleet fields (e.g. GitOpsSoftware.Packages) get fixed.
func toSnake(s string) string {
	rs := []rune(s)
	out := make([]rune, 0, len(rs)+4)
	for i, r := range rs {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rs[i-1]
				nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
					out = append(out, '_')
				}
			}
			out = append(out, unicode.ToLower(r))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// pathRefDefs are the $defs where GitOps supports a file reference (`path:` /
// `paths:`) in place of inline content: the top-level section values and the
// list-item types. We inject path/paths only there instead of on every object,
// so completion isn't cluttered with path everywhere.
var pathRefDefs = []string{
	"GitOpsOrgSettings", "GitOpsFleetSettings", "AgentOptions", "Controls", "GitOpsSoftware",
	"GitOpsPolicySpec", "Query", "LabelSpec",
	"SoftwarePackageSpec", "TeamSpecAppStoreApp", "MaintainedAppSpec",
}

// addPathRefs adds path/paths to the specific defs in pathRefDefs. The Go structs
// don't model the file-reference pattern, so without this real GitOps files using
// e.g. `- path: ./lib/foo.yml` light up with "Property path is not allowed".
func addPathRefs(doc map[string]any) {
	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		return
	}
	for _, name := range pathRefDefs {
		def, ok := defs[name].(map[string]any)
		if !ok {
			continue
		}
		props, ok := def["properties"].(map[string]any)
		if !ok {
			continue
		}
		if _, exists := props["path"]; !exists {
			props["path"] = map[string]any{"type": "string"}
		}
		if _, exists := props["paths"]; !exists {
			props["paths"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
		}
	}
}

// typeLabel returns a short type name for a property schema, for hover text.
func typeLabel(s map[string]any) string {
	if ref, ok := s["$ref"].(string); ok {
		return strings.TrimPrefix(ref, "#/$defs/")
	}
	if t, ok := s["type"].(string); ok {
		if t == "array" {
			if items, ok := s["items"].(map[string]any); ok {
				if il := typeLabel(items); il != "" {
					return "array<" + il + ">"
				}
			}
			return "array"
		}
		return t
	}
	if _, ok := s["anyOf"]; ok {
		return "boolean or object"
	}
	return ""
}

// addTypeDescriptions sets each property's description to its type so yamlls shows
// it on hover (hover renders `description`, not the bare type). Must run before
// relaxNulls strips scalar `type` fields.
func addTypeDescriptions(node any) {
	switch n := node.(type) {
	case map[string]any:
		if props, ok := n["properties"].(map[string]any); ok {
			for _, v := range props {
				if ps, ok := v.(map[string]any); ok {
					if _, has := ps["description"]; !has {
						if lbl := typeLabel(ps); lbl != "" {
							ps["description"] = "type: `" + lbl + "`"
						}
					}
				}
			}
		}
		for _, v := range n {
			addTypeDescriptions(v)
		}
	case []any:
		for _, v := range n {
			addTypeDescriptions(v)
		}
	}
}

// relaxNulls makes empty placeholder keys valid without yamlls suggesting `null`
// in value completion. GitOps files routinely leave keys empty (e.g.
// `minimum_version:` or `scripts:`), which YAML parses as null. yamlls can't
// validate null without also offering it as a value, so for scalar leaves we drop
// the `type` entirely (empty stays valid, no null suggestion, but the field is no
// longer value-type-checked). Objects/arrays keep their type as `[type, null]` so
// their structure and key/item completion survive while empty sections validate.
func relaxNulls(node any) {
	switch n := node.(type) {
	case map[string]any:
		if t, ok := n["type"].(string); ok && n["enum"] == nil {
			switch t {
			case "string", "integer", "number":
				// Left untyped. Strings frequently have unquoted YAML values (a
				// version like 13.0 parses as a float) that would falsely fail a
				// string check. Several Fleet ints are really string enums (e.g.
				// label_membership_type is a Go uint marshaled as "dynamic") that
				// would falsely fail an integer check. Value completion would also
				// only ever offer null for these.
				delete(n, "type")
			case "boolean", "object", "array":
				// Keep the type (so wrong types are caught) but tolerate an empty
				// (null) placeholder value. yamlls will offer null in value
				// completion for these, unavoidable when null is valid.
				n["type"] = []any{t, "null"}
			}
		}
		for _, v := range n {
			relaxNulls(v)
		}
	case []any:
		for _, v := range n {
			relaxNulls(v)
		}
	}
}

// addRenameAliases adds each renamed key as an alias next to its json-tag name so
// both the old and new spellings validate (Fleet accepts both for backward compat).
func addRenameAliases(node any, renames map[string]string) {
	switch n := node.(type) {
	case map[string]any:
		if props, ok := n["properties"].(map[string]any); ok {
			for jsonName, renameName := range renames {
				if sub, present := props[jsonName]; present {
					if _, exists := props[renameName]; !exists {
						props[renameName] = sub
					}
				}
			}
		}
		for _, v := range n {
			addRenameAliases(v, renames)
		}
	case []any:
		for _, v := range n {
			addRenameAliases(v, renames)
		}
	}
}

// collectRenames walks the type tree recording json-tag -> renameto name. Fleet
// aliases many config keys with a `renameto` tag (new fleets/reports terminology)
// and GitOps YAML uses the renamed key, but invopop only reads the json tag.
func collectRenames(t reflect.Type, seen map[reflect.Type]bool, out map[string]string) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if rt := f.Tag.Get("renameto"); rt != "" {
			jsonName := strings.Split(f.Tag.Get("json"), ",")[0]
			renameName := strings.Split(rt, ",")[0]
			if jsonName != "" && renameName != "" {
				out[jsonName] = renameName
			}
		}
		collectRenames(f.Type, seen, out)
	}
}

func main() {
	renames := map[string]string{}
	collectRenames(reflect.TypeOf(GitOpsSpec{}), map[reflect.Type]bool{}, renames)

	r := &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		Mapper:                     typeMapper,
		KeyNamer:                   toSnake,
		// Inline the root struct's properties instead of hiding them behind a
		// single top-level $ref, so yamlls offers root-level key completion.
		ExpandedStruct: true,
	}

	schema := r.Reflect(&GitOpsSpec{})

	raw, err := json.Marshal(schema)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal schema:", err)
		os.Exit(1)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Fprintln(os.Stderr, "unmarshal schema:", err)
		os.Exit(1)
	}
	// Type agent_options.config.options with the generated osquery option list.
	// config keeps other keys (schedule, decorators, ...) open.
	if opts, err := osqueryOptionsSchema(generatedOsqueryOptions); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not type config.options:", err)
	} else if defs, ok := doc["$defs"].(map[string]any); ok {
		if ao, ok := defs["AgentOptions"].(map[string]any); ok {
			if props, ok := ao["properties"].(map[string]any); ok {
				props["config"] = map[string]any{
					"type":       "object",
					"properties": map[string]any{"options": opts},
				}
			}
		}
	}

	addTypeDescriptions(doc)
	addRenameAliases(doc, renames)
	addPathRefs(doc)
	relaxNulls(doc)

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal schema:", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		path := os.Args[1]
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write file:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "wrote", path)
		return
	}
	fmt.Println(string(out))
}
