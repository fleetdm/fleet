package profilevalidation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/micromdm/plist"
)

// decodeAppleProfile decodes a .mobileconfig into a value tree that keeps each plist element's type.
func decodeAppleProfile(data []byte) (map[string]any, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("<?xml")) {
		return nil, errors.New("signed profiles are not supported")
	}
	// DigiCert <data> placeholders aren't valid base64 until expansion; blank them as the upload
	// parser does.
	data = variables.ProfileDataVariableRegex.ReplaceAll(data, nil)
	var root any
	if err := plist.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	dict, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode profile: root is %s, not dictionary", kindOf(root))
	}
	return dict, nil
}

// decodeAppleDeclaration decodes a DDM declaration, keeping integers and reals apart.
func decodeAppleDeclaration(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode declaration: %w", err)
	}
	dict, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode declaration: root is %s, not dictionary", kindOf(root))
	}
	return dict, nil
}

// kindOf names a decoded value's type with the schema's type names.
func kindOf(v any) string {
	switch v := v.(type) {
	case string:
		return string(AppleTypeString)
	case bool:
		return string(AppleTypeBoolean)
	case int64, uint64:
		return string(AppleTypeInteger)
	case float32, float64:
		return string(AppleTypeReal)
	case json.Number:
		if strings.ContainsAny(v.String(), ".eE") {
			return string(AppleTypeReal)
		}
		return string(AppleTypeInteger)
	case time.Time:
		return string(AppleTypeDate)
	case []byte:
		return string(AppleTypeData)
	case []any:
		return string(AppleTypeArray)
	case map[string]any:
		return string(AppleTypeDictionary)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// knownAppleTypes are the types typeAccepts checks; any other type accepts every value, so newer
// schema data can only loosen an older validator.
var knownAppleTypes = []AppleType{
	AppleTypeString, AppleTypeInteger, AppleTypeReal, AppleTypeBoolean, AppleTypeDate,
	AppleTypeData, AppleTypeArray, AppleTypeDictionary,
}

// typeAccepts reports whether v fits one of key's types. declaration is set for JSON values, which
// have no data or date types.
func typeAccepts(key *AppleKey, v any, declaration bool) bool {
	// A secret can expand to anything.
	if s, ok := v.(string); ok && variables.SecretRegex.MatchString(s) {
		return true
	}
	kind := AppleType(kindOf(v))
	for _, t := range key.Type {
		if !slices.Contains(knownAppleTypes, t) {
			return true
		}
		switch t {
		case kind:
			return true
		case AppleTypeInteger, AppleTypeReal:
			if kind == AppleTypeInteger || kind == AppleTypeReal {
				return true
			}
		case AppleTypeString:
			// Apple is moving <date> keys to string with a time valuetype; the data carries
			// valuetypes in angle brackets.
			vt := AppleValueType(strings.Trim(string(key.ValueType), "<>"))
			if kind == AppleTypeDate && (vt == AppleValueTypeTimestamp || vt == AppleValueTypeLocalTime) {
				return true
			}
		case AppleTypeData, AppleTypeDate:
			if declaration && kind == AppleTypeString {
				return true
			}
		}
	}
	// No types at all is treated like an unknown type.
	return len(key.Type) == 0
}

// appleWalker checks one payload's or declaration's values against its schema keys.
type appleWalker struct {
	// root is the payload type's top-level keys, where refs resolve from.
	root        map[string]*AppleKey
	mode        Mode
	declaration bool
	payloadType string
	findings    []Finding
}

// walkKeys checks a dictionary. Listed keys are checked against their definition and other keys
// against values; with neither, an unknown key is reported in strict mode unless the dictionary
// is open.
func (w *appleWalker) walkKeys(keys map[string]*AppleKey, open bool, values *AppleKey, dict map[string]any, path string) {
	for _, name := range slices.Sorted(maps.Keys(dict)) {
		keyPath := name
		if path != "" {
			keyPath = path + "." + name
		}
		switch {
		case keys[name] != nil:
			w.walkValue(keys[name], dict[name], keyPath)
		case values != nil:
			w.walkValue(values, dict[name], keyPath)
		case !open && w.mode == ModeStrict:
			w.findings = append(w.findings, Finding{Code: CodeUnknownKey, PayloadType: w.payloadType, KeyPath: keyPath})
		}
	}
}

func (w *appleWalker) walkValue(key *AppleKey, v any, path string) {
	if key.Ref != "" {
		// A ref names an enclosing definition, so this can't loop; an unresolvable one passes.
		if key = resolveAppleRef(w.root, key.Ref); key == nil || key.Ref != "" {
			return
		}
	}
	if !typeAccepts(key, v, w.declaration) {
		expected := make([]string, 0, len(key.Type))
		for _, t := range key.Type {
			expected = append(expected, string(t))
		}
		w.findings = append(w.findings, Finding{
			Code: CodeTypeMismatch, PayloadType: w.payloadType, KeyPath: path, Expected: expected, Actual: kindOf(v),
		})
		return
	}
	// any, or a type this validator doesn't know, accepts the value whole.
	for _, t := range key.Type {
		if !slices.Contains(knownAppleTypes, t) {
			return
		}
	}
	switch v := v.(type) {
	case map[string]any:
		if len(key.Keys) > 0 || key.Values != nil {
			w.walkKeys(key.Keys, key.Open, key.Values, v, path)
		}
	case []any:
		if key.Items != nil {
			for i, item := range v {
				w.walkValue(key.Items, item, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
}

// resolveAppleRef finds the definition a ref names, e.g. "Dock[]" or
// "ManagedBookmarks[].Bookmarks[].Folder", starting from a payload type's keys. Key names can
// contain dots, so each step takes the longest key name that fits.
func resolveAppleRef(root map[string]*AppleKey, ref string) *AppleKey {
	var cur *AppleKey
	keys := root
	for rest := ref; rest != ""; {
		if after, ok := strings.CutPrefix(rest, "[]"); ok {
			if cur == nil || cur.Items == nil {
				return nil
			}
			cur, keys, rest = cur.Items, cur.Items.Keys, after
			continue
		}
		if cur != nil {
			var ok bool
			if rest, ok = strings.CutPrefix(rest, "."); !ok {
				return nil
			}
		}
		var next *AppleKey
		best := ""
		for name, k := range keys {
			if k != nil && len(name) > len(best) && strings.HasPrefix(rest, name) &&
				(len(rest) == len(name) || rest[len(name)] == '.' || rest[len(name)] == '[') {
				next, best = k, name
			}
		}
		if next == nil {
			return nil
		}
		cur, keys, rest = next, next.Keys, rest[len(best):]
	}
	return cur
}
