package spec

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/agnivade/levenshtein"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/hashicorp/go-multierror"
)

// fieldInfo holds metadata about a struct field extracted from its JSON tag.
type fieldInfo struct {
	jsonName string
	typ      reflect.Type
}

// ValidKeysProvider is implemented by types with custom JSON marshaling
// that want to declare valid keys for gitops unknown-key validation.
type ValidKeysProvider interface {
	ValidKeys() []string
}

var validKeysProviderType = reflect.TypeFor[ValidKeysProvider]()

var (
	knownKeysCache   = make(map[reflect.Type]map[string]fieldInfo)
	knownKeysCacheMu sync.Mutex
)

// knownJSONKeys extracts the set of valid JSON field names from a struct type,
// including fields from embedded structs. Results are cached per type.
// For types implementing ValidKeysProvider, the declared keys are used instead.
func knownJSONKeys(t reflect.Type) map[string]fieldInfo {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	knownKeysCacheMu.Lock()
	defer knownKeysCacheMu.Unlock()

	if cached, ok := knownKeysCache[t]; ok {
		return cached
	}

	keys := make(map[string]fieldInfo)

	// If the type (or pointer to it) implements ValidKeysProvider, use those
	// keys instead of reflecting on struct fields. This handles types with
	// custom JSON marshaling (e.g. GoogleCalendarApiKey).
	if reflect.PointerTo(t).Implements(validKeysProviderType) || t.Implements(validKeysProviderType) {
		provider := reflect.New(t).Interface().(ValidKeysProvider)
		for _, name := range provider.ValidKeys() {
			keys[name] = fieldInfo{
				jsonName: name,
				typ:      reflect.TypeFor[any](),
			}
		}
	} else {
		collectFields(t, keys)
	}

	knownKeysCache[t] = keys
	return keys
}

// collectFields recursively extracts JSON field names from a struct type,
// handling embedded structs by inlining their fields.
func collectFields(t reflect.Type, keys map[string]fieldInfo) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if strings.Contains(t.Name(), "BoolOr") && field.Name == "Other" {
			collectFields(field.Type, keys)
		}

		// Handle embedded structs: inline their fields.
		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				collectFields(ft, keys)
			}
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}

		keys[name] = fieldInfo{
			jsonName: name,
			typ:      field.Type,
		}

		// Also register the "renameto" alias (deprecated field name mappings)
		// so that both old and new names are accepted.
		if alias := field.Tag.Get("renameto"); alias != "" {
			keys[alias] = fieldInfo{
				jsonName: alias,
				typ:      field.Type,
			}
		}
	}
}

// anyFieldTypes maps parent struct types to overrides for fields typed as `any`/`interface{}`.
// When the walker encounters an `any`-typed field, it checks this registry to determine
// the concrete type to use for recursive validation.
var anyFieldTypes = map[reflect.Type]map[string]reflect.Type{
	reflect.TypeFor[GitOpsControls](): {
		"macos_updates":    reflect.TypeFor[fleet.AppleOSUpdateSettings](),
		"ios_updates":      reflect.TypeFor[fleet.AppleOSUpdateSettings](),
		"ipados_updates":   reflect.TypeFor[fleet.AppleOSUpdateSettings](),
		"macos_migration":  reflect.TypeFor[fleet.MacOSMigration](),
		"windows_updates":  reflect.TypeFor[fleet.WindowsUpdates](),
		"macos_settings":   reflect.TypeFor[fleet.MacOSSettings](),
		"windows_settings": reflect.TypeFor[fleet.WindowsSettings](),
		"android_settings": reflect.TypeFor[fleet.AndroidSettings](),
	},
	reflect.TypeFor[GitOpsOrgSettings](): {
		"certificate_authorities": reflect.TypeFor[fleet.GroupedCertificateAuthorities](),
		"mdm":                     reflect.TypeFor[GitOpsMDM](),
		"org_info":                reflect.TypeFor[GitOpsOrgInfo](),
	},
}

// suggestKey returns the closest known key name if one is within a reasonable
// edit distance, or empty string if no good match exists.
func suggestKey(unknown string, known map[string]fieldInfo) string {
	bestKey := ""
	bestDist := len(unknown) // worst case: replace every character
	for candidate := range known {
		d := levenshtein.ComputeDistance(unknown, candidate)
		if d < bestDist {
			bestDist = d
			bestKey = candidate
		}
	}
	// Suggest only if the distance is at most ~40% of the longer string's length,
	// with a minimum threshold of 1 (exact single-char typos always suggest).
	maxDist := max(1, max(len(unknown), len(bestKey))*2/5)
	if bestDist <= maxDist {
		return bestKey
	}
	return ""
}

// validateUnknownKeys walks parsed JSON data and compares keys against
// struct field tags at every nesting level. Returns all unknown key errors found.
func validateUnknownKeys(data any, targetType reflect.Type, path []string, filePath string) []error {
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	switch d := data.(type) {
	case map[string]any:
		return validateMapKeys(d, targetType, path, filePath)
	case []any:
		return validateSliceKeys(d, targetType, path, filePath)
	default:
		return nil
	}
}

// validateMapKeys validates keys in a JSON object against the known keys
// for the target struct type.
func validateMapKeys(data map[string]any, targetType reflect.Type, path []string, filePath string) []error {
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}
	if targetType.Kind() != reflect.Struct {
		return nil
	}

	known := knownJSONKeys(targetType)
	if len(known) == 0 {
		// No JSON-tagged fields: either not a struct or a struct with custom
		// serialization. Skip validation since we don't know the expected keys.
		return nil
	}

	var errs []error
	parentOverrides := anyFieldTypes[targetType]

	for key, val := range data {
		fi, ok := known[key]
		if !ok {
			errs = append(errs, &ParseUnknownKeyError{
				Filename:   filePath,
				Path:       strings.Join(path, "."),
				Field:      key,
				Suggestion: suggestKey(key, known),
			})
			continue
		}

		// Determine the type to recurse into.
		fieldType := fi.typ

		// Check the override registry for this field. This handles two cases:
		// 1. `any`/`interface{}` fields that need a concrete type for recursion
		// 2. Struct fields that need a gitops-extended type (e.g. fleet.MDM -> GitOpsMDM)
		if override, ok := parentOverrides[key]; ok {
			fieldType = override
		} else if fieldType.Kind() == reflect.Interface {
			continue // any-typed field with no override, skip
		}

		// Recurse into nested structs or slices.
		childPath := append(slices.Clone(path), key)
		childErrs := validateUnknownKeys(val, fieldType, childPath, filePath)
		errs = append(errs, childErrs...)
	}

	return errs
}

// validateSliceKeys validates each element in a JSON array.
func validateSliceKeys(data []any, targetType reflect.Type, path []string, filePath string) []error {
	// Determine the element type from the target slice type.
	var elemType reflect.Type
	switch targetType.Kind() {
	case reflect.Slice, reflect.Array:
		elemType = targetType.Elem()
	default:
		// If targetType isn't a slice, we can't determine element type.
		return nil
	}
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	var errs []error
	for i, elem := range data {
		elemPath := append(slices.Clone(path), fmt.Sprintf("[%d]", i))
		childErrs := validateUnknownKeys(elem, elemType, elemPath, filePath)
		errs = append(errs, childErrs...)
	}
	return errs
}

// validateRawKeys unmarshals raw JSON into a generic structure and validates
// all keys against the target type. This is a convenience wrapper for use at
// each integration point.
func validateRawKeys(raw json.RawMessage, targetType reflect.Type, filePath string, keysPath []string) []error {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return []error{err} // parse errors already caught by the struct unmarshal
	}
	return validateUnknownKeys(data, targetType, keysPath, filePath)
}

// validateYAMLKeys unmarshals raw YAML into a generic structure and validates
// all keys against the target type. Use this for path-referenced files that
// contain YAML rather than JSON.
func validateYAMLKeys(yamlBytes []byte, targetType reflect.Type, filePath string, keysPath []string) []error {
	var data any
	if err := YamlUnmarshal(yamlBytes, &data); err != nil {
		return []error{err}
	}
	return validateUnknownKeys(data, targetType, keysPath, filePath)
}

// filterWarnings removes errors matching the given types from a multierror,
// logging them as warnings instead. Returns the filtered error (nil if empty).
func filterWarnings(multiError *multierror.Error, logFn func(string, ...any), types ...reflect.Type) error {
	if multiError == nil {
		return nil
	}
	var filtered *multierror.Error
	for _, err := range multiError.Errors {
		if slices.Contains(types, reflect.TypeOf(err)) {
			logFn("[!] warning: %s\n", err.Error())
		} else {
			filtered = multierror.Append(filtered, err)
		}
	}
	return filtered.ErrorOrNil()
}
