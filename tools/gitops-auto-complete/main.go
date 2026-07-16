// Command gitops-auto-complete generates a JSON schema from Fleet's GitOps Go
// structs so editors (yamlls) can offer completion/validation for GitOps YAML.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"unicode"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/invopop/jsonschema"
)

// generatedOsqueryOptions is the Fleet-generated file listing every valid osquery
// option (config.options.*), relative to the repo root. It's an unexported struct,
// so we parse its AST rather than reflect it.
const generatedOsqueryOptions = "server/fleet/agent_options_generated.go"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println(`Usage: gitops-auto-complete [output-file]

Generates a JSON schema from Fleet's GitOps structs for yaml-language-server.
With an output-file, writes the schema there; otherwise prints to stdout.`)
		return
	}

	// Resolve Fleet source paths from this file's own location so the tool works
	// from any working directory, not just the module root.
	repoRoot := ""
	if _, thisFile, _, ok := runtime.Caller(0); ok {
		repoRoot = filepath.Join(filepath.Dir(thisFile), "..", "..")
	}

	renames := map[string]string{}
	collectRenames(reflect.TypeFor[GitOpsSpec](), map[reflect.Type]bool{}, renames)

	reflector := &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		Mapper:                     typeMapper,
		KeyNamer:                   toSnake,
		// Inline the root struct's properties instead of hiding them behind a
		// single top-level $ref, so yamlls offers root-level key completion.
		ExpandedStruct: true,
	}
	addFleetGoComments(reflector, repoRoot)

	raw, err := json.Marshal(reflector.Reflect(&GitOpsSpec{}))
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal schema:", err)
		os.Exit(1)
	}

	var schemaKeys map[string]any
	err = json.Unmarshal(raw, &schemaKeys)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unmarshal schema:", err)
		os.Exit(1)
	}

	// Merges run first so the injected keys get the same treatment as the rest. If
	// config.options can't be built, generation continues
	osqueryOptions, err := osqueryOptionsSchema(filepath.Join(repoRoot, generatedOsqueryOptions))
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not type config.options:", err)
	}
	mergeOsqueryOptions(schemaKeys, osqueryOptions)
	mergeMissingMDMKeys(schemaKeys, spec.GitOpsMDM{})

	// Order matters. annotate and addGitOpsKeyNotes read types that relaxNulls
	// later strips, so they run first. addPathReferences also runs before relaxNulls
	// so the path keys it adds get relaxed too. typeInstallerReferenceKeys runs after
	// relaxNulls to restore the string types it drops.
	nodes := collectNodes(schemaKeys)
	annotate(nodes, renames)
	addGitOpsKeyNotes(schemaKeys)
	addPathReferences(schemaKeys)
	addInstallerReferenceRequirement(schemaKeys)

	// Collect again so relaxNulls reaches the aliases and path keys added above.
	nodes = collectNodes(schemaKeys)
	relaxNulls(nodes)
	typeInstallerReferenceKeys(schemaKeys)

	out, err := json.MarshalIndent(schemaKeys, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal schema:", err)
		os.Exit(1)
	}

	if len(os.Args) <= 1 {
		fmt.Println(string(out))
		return
	}

	path := os.Args[1]
	err = os.WriteFile(path, append(out, '\n'), 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write file:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "wrote", path)
}

// --- building the base schema from Go types ---

// addFleetGoComments pulls Fleet's Go doc comments into field descriptions (shown
// on hover). AddGoComments derives package paths from the walk dir relative to cwd,
// so it runs from the repo root and restores cwd afterward.
func addFleetGoComments(reflector *jsonschema.Reflector, repoRoot string) {
	workingDir, err := os.Getwd()
	if err != nil {
		return
	}

	err = os.Chdir(repoRoot)
	if err != nil {
		return
	}
	defer func() { _ = os.Chdir(workingDir) }()

	const base = "github.com/fleetdm/fleet/v4"
	_ = reflector.AddGoComments(base, "server/fleet")
	_ = reflector.AddGoComments(base, "pkg/spec")
}

// osqueryOptionsSchema parses the generated osqueryOptions struct into a strict
// object schema, so config.options.* gets completion, types, and unknown-key
// validation matching what Fleet enforces.
func osqueryOptionsSchema(path string) (map[string]any, error) {
	parsedFile, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}

	properties := map[string]any{}
	ast.Inspect(parsedFile, func(astNode ast.Node) bool {
		typeSpec, ok := astNode.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "osqueryOptions" {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}

		for _, field := range structType.Fields.List {
			fieldType, ok := field.Type.(*ast.Ident)
			if !ok || field.Tag == nil {
				continue
			}

			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			name, _, _ := strings.Cut(tag.Get("json"), ",")
			jsonType := goTypeToJSON(fieldType.Name)
			if name == "" || name == "-" || jsonType == "" {
				continue
			}

			properties[name] = map[string]any{"type": jsonType}
		}

		return false
	})

	if len(properties) == 0 {
		return nil, fmt.Errorf("osqueryOptions struct not found in %s", path)
	}

	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties}, nil
}

// reflectProperties reflects a struct and returns its top-level property schemas.
func reflectProperties(v any) map[string]any {
	reflector := &jsonschema.Reflector{RequiredFromJSONSchemaTags: true, Mapper: typeMapper, KeyNamer: toSnake, ExpandedStruct: true}
	raw, err := json.Marshal(reflector.Reflect(v))
	if err != nil {
		return nil
	}

	var reflected map[string]any
	err = json.Unmarshal(raw, &reflected)
	if err != nil {
		return nil
	}

	properties, _ := reflected["properties"].(map[string]any)
	return properties
}

// toSnake converts a Go field name to snake_case. invopop applies KeyNamer to the
// json-tag name when a tag is present, so already-snake tags pass through unchanged.
// Untagged Fleet fields like GitOpsSoftware.Packages get fixed.
func toSnake(name string) string {
	runes := []rune(name)
	result := make([]rune, 0, len(runes)+4)

	for i, char := range runes {
		if !unicode.IsUpper(char) {
			result = append(result, char)
			continue
		}

		if i > 0 {
			previous := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			atBoundary := unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextIsLower)
			if atBoundary {
				result = append(result, '_')
			}
		}

		result = append(result, unicode.ToLower(char))
	}

	return string(result)
}

// collectRenames walks the type tree recording json-tag -> renameto name. Fleet
// aliases many config keys with a `renameto` tag for the new fleets/reports
// terminology, and GitOps YAML uses the renamed key, but invopop only reads json.
func collectRenames(goType reflect.Type, visited map[reflect.Type]bool, renames map[string]string) {
	for goType.Kind() == reflect.Pointer || goType.Kind() == reflect.Slice || goType.Kind() == reflect.Array || goType.Kind() == reflect.Map {
		goType = goType.Elem()
	}

	if goType.Kind() != reflect.Struct || visited[goType] {
		return
	}
	visited[goType] = true

	for field := range goType.Fields() {
		renameTo := field.Tag.Get("renameto")
		if renameTo != "" {
			jsonName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			renameName, _, _ := strings.Cut(renameTo, ",")
			if jsonName != "" && renameName != "" {
				renames[jsonName] = renameName
			}
		}

		collectRenames(field.Type, visited, renames)
	}
}

// mergeOsqueryOptions types agent_options.config.options with the generated osquery
// option list. config keeps its other keys (schedule, decorators, ...) open.
func mergeOsqueryOptions(schemaKeys map[string]any, osqueryOptions map[string]any) {
	// If the options couldn't be built, leave config open rather than pinning its
	// options to an empty or null schema.
	if len(osqueryOptions) == 0 {
		return
	}

	agentOptions, ok := definitionProperties(schemaKeys, "AgentOptions")
	if !ok {
		return
	}

	agentOptions["config"] = map[string]any{
		"type":       "object",
		"properties": map[string]any{"options": osqueryOptions},
	}
}

// mergeMissingMDMKeys copies gitops-only MDM keys into the "MDM" def, whose base
// fleet.MDM (org_settings.mdm) omits them. spec.GitOpsMDM embeds fleet.MDM and adds
// them (e.g. end_user_license_agreement), so add whichever the def is missing.
func mergeMissingMDMKeys(schemaKeys map[string]any, gitOpsMDM spec.GitOpsMDM) {
	extraProperties := reflectProperties(&gitOpsMDM)
	if extraProperties == nil {
		return
	}

	properties, ok := definitionProperties(schemaKeys, "MDM")
	if !ok {
		return
	}

	for key, value := range extraProperties {
		_, exists := properties[key]
		if !exists {
			properties[key] = value
		}
	}
}

// --- tree helpers ---

// collectNodes walks the schema iteratively and returns every object node,
// parents always before their children. Collecting once lets the passes below be
// plain loops instead of repeated recursive tree walks.
func collectNodes(schemaKeys any) []map[string]any {
	var nodes []map[string]any
	stack := []any{schemaKeys}

	for len(stack) > 0 {
		// Pop the next value off the stack.
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch node := current.(type) {
		case map[string]any:
			// Add the object to nodes, then push its values to visit next.
			nodes = append(nodes, node)
			for _, child := range node {
				stack = append(stack, child)
			}
		case []any:
			// Walk through arrays without collecting them.
			stack = append(stack, node...)
		}
	}

	return nodes
}

// definitionByName returns a named $def object and whether it was found.
func definitionByName(schemaKeys map[string]any, name string) (map[string]any, bool) {
	definitions, _ := schemaKeys["$defs"].(map[string]any)
	definition, ok := definitions[name].(map[string]any)
	return definition, ok
}

// definitionProperties returns the properties of a named $def and whether it was found.
func definitionProperties(schemaKeys map[string]any, name string) (map[string]any, bool) {
	definition, ok := definitionByName(schemaKeys, name)
	if !ok {
		return nil, false
	}

	properties, ok := definition["properties"].(map[string]any)
	return properties, ok
}

// appendDescription puts text below node's existing description, if any. The blank
// line matters, since yamlls renders the two parts as separate paragraphs on hover.
func appendDescription(node map[string]any, text string) {
	existing, ok := node["description"].(string)
	if ok && existing != "" {
		node["description"] = existing + "\n\n" + text
		return
	}

	node["description"] = text
}

// typeLabel returns a short type name for a schema node, for hover text.
func typeLabel(node map[string]any) string {
	ref, ok := node["$ref"].(string)
	if ok {
		return strings.TrimPrefix(ref, "#/$defs/")
	}

	schemaType, ok := node["type"].(string)
	if !ok {
		_, isAnyOf := node["anyOf"]
		if isAnyOf {
			return "boolean or object"
		}
		return ""
	}

	if schemaType != "array" {
		return schemaType
	}

	items, ok := node["items"].(map[string]any)
	if !ok {
		return "array"
	}

	innerLabel := typeLabel(items)
	if innerLabel == "" {
		return "array"
	}
	return "array<" + innerLabel + ">"
}

// resolveReference follows a chain of $ref links through definitions to the concrete
// schema object. Each iteration replaces node with the definition its $ref points at,
// and returns when node has no $ref, the ref is unknown, or it was already seen (a
// cycle), so it visits each definition at most once.
func resolveReference(definitions map[string]any, node map[string]any) map[string]any {
	seen := map[string]bool{}
	for {
		ref, isRef := node["$ref"].(string)
		if !isRef {
			return node // reached a concrete node
		}

		name := strings.TrimPrefix(ref, "#/$defs/")
		if seen[name] {
			return node // cycle: stop where we are
		}
		seen[name] = true

		definition, isObject := definitions[name].(map[string]any)
		if !isObject {
			return node // dangling ref: nothing to follow
		}
		node = definition
	}
}

// --- post-processing passes ---

// annotate walks the collected nodes once and, per node, does two things: label
// each property with its type (shown on hover), then add an alias for any renamed
// key alongside the deprecated original. Labeling comes first so an alias, a shallow
// copy of the property, inherits the label.
func annotate(nodes []map[string]any, renames map[string]string) {
	for _, node := range nodes {
		properties, ok := node["properties"].(map[string]any)
		if !ok {
			continue
		}

		for _, value := range properties {
			property, ok := value.(map[string]any)
			if !ok {
				continue
			}

			label := typeLabel(property)
			if label == "" {
				continue
			}

			appendDescription(property, "type: `"+label+"`")
		}

		for jsonName, renameName := range renames {
			original, present := properties[jsonName]
			if !present {
				continue
			}

			property, isObject := original.(map[string]any)
			_, aliasExists := properties[renameName]

			switch {
			case aliasExists:
				// Keep an alias that's already present rather than overwriting it.
			case isObject:
				properties[renameName] = maps.Clone(property)
			default:
				properties[renameName] = original
			}

			if isObject {
				property["deprecated"] = true
				property["deprecationMessage"] = "'" + jsonName + "' is deprecated, use '" + renameName + "' instead"
			}
		}
	}
}

// addGitOpsKeyNotes appends each declarativeExceptions note to the schema node at
// its dotted key path. It descends the path from the root, following $refs and
// stepping transparently through array items, and attaches the note to the property
// node itself (so shared $defs aren't affected). Missing paths are skipped.
func addGitOpsKeyNotes(schemaKeys map[string]any) {
	definitions, _ := schemaKeys["$defs"].(map[string]any)

	for path, note := range declarativeExceptions {
		node := schemaKeys
		found := true

		for segment := range strings.SplitSeq(path, ".") {
			container := resolveReference(definitions, node)
			items, isArray := container["items"].(map[string]any)
			if isArray {
				container = resolveReference(definitions, items)
			}

			properties, hasProperties := container["properties"].(map[string]any)
			if !hasProperties {
				found = false
				break
			}

			next, isObject := properties[segment].(map[string]any)
			if !isObject {
				found = false
				break
			}
			node = next
		}

		if found {
			appendDescription(node, note)
		}
	}
}

// addPathReferences adds path/paths to the definitions in pathReferenceDefinitions.
// The Go structs don't model the file-reference pattern, so without this real GitOps
// files using e.g. `- path: ./lib/foo.yml` light up with "Property path is not
// allowed".
func addPathReferences(schemaKeys map[string]any) {
	for _, name := range pathReferenceDefinitions {
		properties, ok := definitionProperties(schemaKeys, name)
		if !ok {
			continue
		}

		_, hasPath := properties["path"]
		if !hasPath {
			properties["path"] = map[string]any{"type": "string"}
		}

		_, hasPaths := properties["paths"]
		if !hasPaths {
			properties["paths"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
		}
	}
}

// addInstallerReferenceRequirement injects an anyOf of single-key required branches (each
// with the same errorMessage) so an item is valid when any one installer-reference
// key is present.
func addInstallerReferenceRequirement(schemaKeys map[string]any) {
	for _, reference := range installerReferences {
		definition, ok := definitionByName(schemaKeys, reference.definition)
		if !ok {
			continue
		}

		branches := make([]any, 0, len(reference.keys))
		for _, key := range reference.keys {
			branches = append(branches, map[string]any{
				"required":     []any{key},
				"errorMessage": reference.message,
			})
		}

		definition["anyOf"] = branches
	}
}

// relaxNulls makes empty placeholder keys valid. GitOps files routinely leave keys
// empty, like `minimum_version:` or `scripts:`, which YAML parses as null, so every
// leaf has to accept null. How it does that depends on the type.
func relaxNulls(nodes []map[string]any) {
	for _, node := range nodes {
		schemaType, ok := node["type"].(string)
		if !ok || node["enum"] != nil {
			continue
		}

		switch schemaType {
		case "integer", "number":
			// Left untyped. Some Fleet ints marshal as string enums, like
			// label_membership_type (a uint that marshals as "dynamic"), so a number
			// check would reject a value fleetctl accepts. Reflection can't tell those
			// apart from real ints, so numeric leaves stay unchecked.
			delete(node, "type")
		case "string", "boolean", "object", "array":
			// Keep the type as [type, null] so a wrong type is still caught while an
			// empty null placeholder validates. This makes yamlls offer null in value
			// completion, a limitation we accept so a real error like an unquoted
			// version: 13.0 (which fleetctl rejects too, since ghodss decodes it as a
			// number into a Go string) surfaces in the editor, not at apply time.
			node["type"] = []any{schemaType, "null"}
		}
	}
}

// typeInstallerReferenceKeys re-applies a strict string type to the keys in
// strictInstallerReferenceKeys, undoing the relaxNulls pass for them.
func typeInstallerReferenceKeys(schemaKeys map[string]any) {
	for definitionName, referenceKeys := range strictInstallerReferenceKeys {
		properties, ok := definitionProperties(schemaKeys, definitionName)
		if !ok {
			continue
		}

		for _, key := range referenceKeys {
			node, ok := properties[key].(map[string]any)
			if !ok {
				continue
			}
			node["type"] = "string"
		}
	}
}
