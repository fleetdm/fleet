package profilevalidation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
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
	// Decode stops after the first value; anything but whitespace after it is invalid JSON.
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("decode declaration: unexpected data after the top-level object")
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

var knownAppleValueTypes = []AppleValueType{
	AppleValueTypeDomain, AppleValueTypeDomainPrefix, AppleValueTypeEmail, AppleValueTypeHostname,
	AppleValueTypeLocalTime, AppleValueTypeRegex, AppleValueTypeTimestamp, AppleValueTypeURL, AppleValueTypeUUID,
}

// appleValueType is key's valuetype without the angle brackets the data carries.
func appleValueType(key *AppleKey) AppleValueType {
	return AppleValueType(strings.Trim(string(key.ValueType), "<>"))
}

// acceptsAnything reports whether key takes any value whole: no types, any, or a type or valuetype
// this validator doesn't know.
func acceptsAnything(key *AppleKey) bool {
	if len(key.Type) == 0 || (key.ValueType != "" && !slices.Contains(knownAppleValueTypes, appleValueType(key))) {
		return true
	}
	for _, t := range key.Type {
		if !slices.Contains(knownAppleTypes, t) {
			return true
		}
	}
	return false
}

// typeAccepts reports whether v fits one of key's types. declaration is set for JSON values, which
// have no data or date types.
func typeAccepts(key *AppleKey, v any, declaration bool) bool {
	// A secret can expand to anything.
	if s, ok := v.(string); ok && variables.SecretRegex.MatchString(s) {
		return true
	}
	if acceptsAnything(key) {
		return true
	}
	kind := AppleType(kindOf(v))
	for _, t := range key.Type {
		switch t {
		case kind:
			return true
		case AppleTypeInteger, AppleTypeReal:
			if kind == AppleTypeInteger || kind == AppleTypeReal {
				return true
			}
		case AppleTypeString:
			// Apple is moving <date> keys to string with a time valuetype.
			vt := appleValueType(key)
			if kind == AppleTypeDate && (vt == AppleValueTypeTimestamp || vt == AppleValueTypeLocalTime) {
				return true
			}
		case AppleTypeData, AppleTypeDate:
			if declaration && kind == AppleTypeString {
				return true
			}
		}
	}
	return false
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
	if acceptsAnything(key) {
		return
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

// strictUnknownAppleProfiles are payload types Apple documents but strict mode treats as unknown
// unless an override vouches for them: they accept any preference domain, so the generator could
// put anything there.
var strictUnknownAppleProfiles = []string{"com.apple.ManagedClient.preferences"}

var (
	appleUUIDRe       = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)
	appleReverseDNSRe = regexp.MustCompile(`^[^.\s]+(\.[^.\s]+)+$`)
)

// TopLevelPayloadType is the Finding.PayloadType for problems in a profile's outer dictionary.
const TopLevelPayloadType = "TopLevel"

// ValidateAppleProfile checks a .mobileconfig against the schema. It returns an error only when
// the profile can't be read: invalid or signed, or not a Configuration profile.
func (s *ProfileValidationSchema) ValidateAppleProfile(data []byte, mode Mode) ([]Finding, error) {
	root, err := decodeAppleProfile(data)
	if err != nil {
		return nil, err
	}
	if pt := root["PayloadType"]; pt != "Configuration" {
		return nil, fmt.Errorf("invalid PayloadType: %v", pt)
	}

	top := s.AppleEnvelope.TopLevel
	w := &appleWalker{root: top.Keys, mode: mode, payloadType: TopLevelPayloadType}
	w.walkKeys(top.Keys, top.Open, top.Values, root, "")
	if mode == ModeStrict {
		w.checkEnvelope(root)
	}

	contents, _ := root["PayloadContent"].([]any)
	for _, item := range contents {
		// A non-dictionary item is already reported against TopLevel.
		payload, ok := item.(map[string]any)
		if !ok {
			continue
		}
		payloadType, _ := payload["PayloadType"].(string)
		w.payloadType = payloadType
		p := s.AppleProfiles[payloadType]
		if p == nil || (mode == ModeStrict && p.Override == "" && slices.Contains(strictUnknownAppleProfiles, payloadType)) {
			if mode == ModeStrict {
				w.findings = append(w.findings, Finding{Code: CodeUnknownType, PayloadType: payloadType})
			}
			continue
		}
		// Every payload carries the common keys; a payload's own definition of one wins.
		keys := maps.Clone(s.AppleEnvelope.CommonPayloadKeys.Keys)
		maps.Copy(keys, p.Keys)
		w.root = p.Keys
		w.walkKeys(keys, p.Open, p.Values, payload, "")
		if mode == ModeStrict {
			w.checkEnvelope(payload)
		}
	}
	return w.findings, nil
}

// checkEnvelope reports a missing PayloadVersion and a malformed PayloadUUID or PayloadIdentifier.
// Wrong types are left to the walk, and values with Fleet variables aren't checked.
func (w *appleWalker) checkEnvelope(dict map[string]any) {
	if _, ok := dict["PayloadVersion"]; !ok {
		w.findings = append(w.findings, Finding{Code: CodeMissingPayloadVersion, PayloadType: w.payloadType, KeyPath: "PayloadVersion"})
	}
	formats := []struct {
		key  string
		re   *regexp.Regexp
		code Code
	}{
		{"PayloadUUID", appleUUIDRe, CodeMalformedUUID},
		{"PayloadIdentifier", appleReverseDNSRe, CodeMalformedIdentifier},
	}
	for _, f := range formats {
		v, ok := dict[f.key].(string)
		if ok && !variables.Contains(v) && !variables.SecretRegex.MatchString(v) && !f.re.MatchString(v) {
			w.findings = append(w.findings, Finding{Code: f.code, PayloadType: w.payloadType, KeyPath: f.key})
		}
	}
}

// ValidateAppleDeclaration checks a DDM declaration against the schema: the envelope keys every
// declaration shares, then Payload against its Type. It returns an error only when the
// declaration isn't a JSON object.
func (s *ProfileValidationSchema) ValidateAppleDeclaration(data []byte, mode Mode) ([]Finding, error) {
	root, err := decodeAppleDeclaration(data)
	if err != nil {
		return nil, err
	}
	declarationType, _ := root["Type"].(string)
	env := s.AppleDeclarationEnvelope
	w := &appleWalker{root: env.Keys, mode: mode, declaration: true, payloadType: declarationType}
	// The envelope's Payload is an open dictionary, so this stops at it.
	w.walkKeys(env.Keys, env.Open, env.Values, root, "")

	d := s.AppleDeclarations[declarationType]
	if d == nil {
		if mode == ModeStrict {
			w.findings = append(w.findings, Finding{Code: CodeUnknownType, PayloadType: declarationType})
		}
		return w.findings, nil
	}
	// A Payload of the wrong type is already reported against the envelope.
	if payload, ok := root["Payload"].(map[string]any); ok {
		w.root = d.Keys
		w.walkKeys(d.Keys, d.Open, d.Values, payload, "")
	}
	return w.findings, nil
}
