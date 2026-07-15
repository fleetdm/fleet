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
	"github.com/invopop/jsonschema"
)

// generatedOsqueryOptions is the Fleet-generated file listing every valid osquery
// option (config.options.*). It's an unexported struct, so we parse its AST rather
// than reflect it.
const generatedOsqueryOptions = "../../server/fleet/agent_options_generated.go"

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

// reflectProps reflects a struct and returns its top-level property schemas.
func reflectProps(v any) map[string]any {
	r := &jsonschema.Reflector{RequiredFromJSONSchemaTags: true, Mapper: typeMapper, KeyNamer: toSnake, ExpandedStruct: true}
	raw, err := json.Marshal(r.Reflect(v))
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	props, _ := m["properties"].(map[string]any)
	return props
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

// requiredSources says each software item must specify at least one source key.
// Fleet enforces this in validation code, not via struct tags, so it's authored
// explicitly here. `path` is always accepted (the item is defined in another file).
var requiredSources = []struct {
	def     string
	keys    []string
	message string
}{
	{"SoftwarePackageSpec", []string{"url", "hash_sha256", "path"}, "A package must set one of: url, hash_sha256, or path."},
	{"TeamSpecAppStoreApp", []string{"app_store_id", "path"}, "An app_store_apps entry must set app_store_id (or path)."},
	{"MaintainedAppSpec", []string{"slug", "path"}, "A fleet_maintained_apps entry must set slug (or path)."},
}

// strictStringKeys are the source/identifier keys kept as strict `string` (no null,
// not untyped) so a wrong-typed value like `url: 12345` is caught. They pair with
// requiredSources and are always present-and-a-string when used. Applied after
// relaxNulls, which would otherwise drop their type.
var strictStringKeys = map[string][]string{
	"SoftwarePackageSpec": {"url", "hash_sha256"},
	"TeamSpecAppStoreApp": {"app_store_id"},
	"MaintainedAppSpec":   {"slug"},
}

func typeSourceKeys(doc map[string]any) {
	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		return
	}
	for def, keys := range strictStringKeys {
		d, ok := defs[def].(map[string]any)
		if !ok {
			continue
		}
		props, ok := d["properties"].(map[string]any)
		if !ok {
			continue
		}
		for _, k := range keys {
			if p, ok := props[k].(map[string]any); ok {
				p["type"] = "string"
			}
		}
	}
}

// addRequiredSources injects an anyOf of required branches (each with the same
// errorMessage) so the item is valid when any one source key is present.
func addRequiredSources(doc map[string]any) {
	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		return
	}
	for _, rs := range requiredSources {
		def, ok := defs[rs.def].(map[string]any)
		if !ok {
			continue
		}
		branches := make([]any, 0, len(rs.keys))
		for _, k := range rs.keys {
			branches = append(branches, map[string]any{
				"required":     []any{k},
				"errorMessage": rs.message,
			})
		}
		def["anyOf"] = branches
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

// addTypeDescriptions appends each property's type to its description so yamlls
// shows it on hover (hover renders `description`, not the bare type). It keeps any
// Go doc comment already set by AddGoComments and adds the type below it. Must run
// before relaxNulls strips scalar `type` fields.
func addTypeDescriptions(node any) {
	switch n := node.(type) {
	case map[string]any:
		if props, ok := n["properties"].(map[string]any); ok {
			for _, v := range props {
				if ps, ok := v.(map[string]any); ok {
					if lbl := typeLabel(ps); lbl != "" {
						tline := "type: `" + lbl + "`"
						if d, ok := ps["description"].(string); ok && d != "" {
							ps["description"] = d + "\n\n" + tline
						} else {
							ps["description"] = tline
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
// both the old and new spellings validate (Fleet accepts both for backward compat),
// and marks the legacy json-tag spelling deprecated (Fleet itself warns
// "'<old>' is deprecated, use '<new>' instead"). The new spelling is not deprecated.
func addRenameAliases(node any, renames map[string]string) {
	switch n := node.(type) {
	case map[string]any:
		if props, ok := n["properties"].(map[string]any); ok {
			for jsonName, renameName := range renames {
				sub, present := props[jsonName]
				if !present {
					continue
				}
				if _, exists := props[renameName]; !exists {
					// Shallow-copy map schemas; `any`-typed fields are a bare `true`,
					// so copy the value as-is.
					if m, ok := sub.(map[string]any); ok {
						alias := make(map[string]any, len(m))
						for k, v := range m {
							alias[k] = v
						}
						props[renameName] = alias
					} else {
						props[renameName] = sub
					}
				}
				// Deprecate the legacy spelling. Only possible on object schemas; a
				// bare `true` (from an `any` field) has nowhere to hang the marker.
				if m, ok := sub.(map[string]any); ok {
					m["deprecated"] = true
					m["deprecationMessage"] = "'" + jsonName + "' is deprecated, use '" + renameName + "' instead"
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

	// Pull Fleet's Go doc comments into field descriptions (shown on hover).
	// AddGoComments derives package paths from the walk dir relative to cwd, so
	// run it from the repo root (two levels up from this tool).
	if wd, err := os.Getwd(); err == nil {
		if chErr := os.Chdir("../.."); chErr == nil {
			const base = "github.com/fleetdm/fleet/v4"
			_ = r.AddGoComments(base, "server/fleet")
			_ = r.AddGoComments(base, "pkg/spec")
			_ = os.Chdir(wd)
		}
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

	// AppConfig/TeamConfig embed fleet.MDM, but GitOps allows extra mdm keys defined
	// on spec.GitOpsMDM (e.g. end_user_license_agreement). Merge those in.
	if gm := reflectProps(&spec.GitOpsMDM{}); gm != nil {
		if defs, ok := doc["$defs"].(map[string]any); ok {
			if mdm, ok := defs["MDM"].(map[string]any); ok {
				if props, ok := mdm["properties"].(map[string]any); ok {
					for k, v := range gm {
						if _, exists := props[k]; !exists {
							props[k] = v
						}
					}
				}
			}
		}
	}

	addTypeDescriptions(doc)
	addRenameAliases(doc, renames)
	addPathRefs(doc)
	addRequiredSources(doc)
	relaxNulls(doc)
	typeSourceKeys(doc)

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
