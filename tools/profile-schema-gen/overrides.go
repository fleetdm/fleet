package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"gopkg.in/yaml.v3"
)

// loosenTypes are the values loosen's types may add, per kind. "any" has its own action, and
// widening a Windows setting into an interior node makes no sense.
var loosenTypes = func() map[string][]string {
	var apple []string
	for _, t := range appleTypes {
		if t != profilevalidation.AppleTypeAny {
			apple = append(apple, string(t))
		}
	}
	return map[string][]string{
		kindAppleProfile:     apple,
		kindAppleDeclaration: apple,
		kindWindows:          {"int", "chr", "bool", "b64", "xml", "null", "bin", "time", "date", "float"},
	}
}()

// Override kinds name the schema section an entry targets.
const (
	kindAppleProfile     = "apple_profile"
	kindAppleDeclaration = "apple_declaration"
	kindWindows          = "windows"

	// declarationBaseName addresses apple_declaration_envelope; Apple's own name for it is "any".
	declarationBaseName = "DeclarationBase"
)

// overridesFile is the overrides YAML; see pkg/mdm/profilevalidation/README.md for the rules.
type overridesFile struct {
	Loosen []overrideEntry `json:"loosen"`
	Add    []overrideEntry `json:"add"`
	Allow  []overrideEntry `json:"allow"`
}

// overrideEntry holds the fields of all three actions; validate checks each action uses only its own.
type overrideEntry struct {
	Kind            string `json:"kind"`
	PayloadType     string `json:"payload_type"`
	DeclarationType string `json:"declaration_type"`
	LocURI          string `json:"loc_uri"`
	Reason          string `json:"reason"`

	KeyPath string `json:"key_path"` // Apple loosen: the key; add: its parent dictionary ("" for the top level)

	Action string `json:"action"` // loosen: "types" or "any"
	// Types are Apple types, or DDF formats for windows entries (see loosenTypes); loosen with
	// action "types". Plain strings, so they're checked against the entry's kind before use.
	Types []string `json:"types"`

	Name string                      `json:"name"` // add
	Key  *profilevalidation.AppleKey `json:"key"`  // add

	Keys map[string]*profilevalidation.AppleKey `json:"keys"` // allow, optional
}

func (e overrideEntry) target() string {
	switch e.Kind {
	case kindAppleProfile:
		return e.PayloadType
	case kindAppleDeclaration:
		return e.DeclarationType
	case kindWindows:
		return e.LocURI
	}
	return ""
}

// applyOverrides applies the overrides file at path to schema. A missing file means no overrides.
// Every entry is validated before anything is applied, and all problems are reported together.
func applyOverrides(ctx context.Context, logger *slog.Logger, path string, schema *profilevalidation.ProfileValidationSchema) error {
	if path == "" {
		logger.InfoContext(ctx, "no overrides file given; skipping overrides")
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		logger.InfoContext(ctx, "overrides file not found; skipping overrides", "path", path)
		return nil
	}
	if err != nil {
		return err
	}
	file, err := decodeOverrides(data)
	if err != nil {
		return fmt.Errorf("overrides %s: %w", path, err)
	}
	if err := file.validate(); err != nil {
		return fmt.Errorf("overrides %s:\n%w", path, err)
	}

	a := &overrideApplier{ctx: ctx, logger: logger, schema: schema, merger: &appleMerger{ctx: ctx, logger: logger}}
	// allow first so add and loosen could target an allowed type; loosen last so it sees added keys.
	for i, e := range file.Allow {
		a.do("allow", i, e, a.allow)
	}
	for i, e := range file.Add {
		a.do("add", i, e, a.add)
	}
	for i, e := range file.Loosen {
		a.do("loosen", i, e, a.loosen)
	}
	if err := errors.Join(a.errs...); err != nil {
		return fmt.Errorf("overrides %s:\n%w", path, err)
	}
	logger.InfoContext(ctx, "applied overrides", "path", path, "allow", len(file.Allow), "add", len(file.Add), "loosen", len(file.Loosen))
	return nil
}

// decodeOverrides reads the YAML through JSON, so entries use the output contract's own field
// names and decoding (a key's "type" may be a string or a list), and unknown fields are rejected:
// a misspelled field would otherwise silently do nothing.
func decodeOverrides(data []byte) (*overridesFile, error) {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	asJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var file overridesFile
	if raw == nil {
		return &file, nil
	}
	dec := json.NewDecoder(bytes.NewReader(asJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return nil, err
	}
	return &file, nil
}

func (f *overridesFile) validate() error {
	var errs []error
	seen := map[string]string{}
	check := func(action string, i int, e overrideEntry, problems []string, identity string) {
		label := fmt.Sprintf("%s[%d] (%s %s %s)", action, i, e.Kind, e.target(), e.KeyPath)
		if prev, dup := seen[identity]; dup && identity != "" {
			problems = append(problems, "targets the same thing as "+prev)
		}
		seen[identity] = label
		for _, p := range problems {
			errs = append(errs, fmt.Errorf("%s: %s", label, p))
		}
	}

	for i, e := range f.Loosen {
		p := e.commonProblems("loosen", kindAppleProfile, kindAppleDeclaration, kindWindows)
		switch {
		case e.Kind == kindWindows && e.KeyPath != "":
			p = append(p, "windows entries take no key_path; loc_uri is the node")
		case e.Kind != kindWindows && e.KeyPath == "":
			p = append(p, "key_path is required")
		}
		switch e.Action {
		case "types":
			if len(e.Types) == 0 {
				p = append(p, `action "types" needs a non-empty types list`)
			}
			if allowed, ok := loosenTypes[e.Kind]; ok {
				for _, t := range e.Types {
					if !slices.Contains(allowed, t) {
						p = append(p, fmt.Sprintf("types: %q isn't valid for %s (%s)", t, e.Kind, strings.Join(allowed, ", ")))
					}
				}
			}
		case "any":
			if len(e.Types) > 0 {
				p = append(p, `action "any" takes no types`)
			}
		default:
			p = append(p, fmt.Sprintf(`action must be "types" or "any", got %q`, e.Action))
		}
		if e.Name != "" || e.Key != nil || e.Keys != nil {
			p = append(p, "loosen takes no name, key, or keys")
		}
		check("loosen", i, e, p, e.Kind+"|"+e.target()+"|"+e.KeyPath)
	}

	for i, e := range f.Add {
		p := e.commonProblems("add", kindAppleProfile, kindAppleDeclaration)
		if e.Name == "" {
			p = append(p, "name is required")
		}
		if e.Key == nil || len(e.Key.Type) == 0 {
			p = append(p, "key with a type is required")
		}
		if e.Action != "" || e.Types != nil || e.Keys != nil {
			p = append(p, "add takes no action, types, or keys")
		}
		check("add", i, e, p, e.Kind+"|"+e.target()+"|"+joinKeyPath(e.KeyPath, e.Name))
	}

	for i, e := range f.Allow {
		// Third-party preference domains exist only in profiles.
		p := e.commonProblems("allow", kindAppleProfile)
		if e.PayloadType == appleTopLevel || e.PayloadType == appleCommonPayloadKeys {
			p = append(p, "the profile envelope can't be allowed; use add for its keys")
		}
		if e.KeyPath != "" || e.Action != "" || e.Types != nil || e.Name != "" || e.Key != nil {
			p = append(p, "allow takes only payload_type, keys, and reason")
		}
		for name, k := range e.Keys {
			if k == nil || len(k.Type) == 0 {
				p = append(p, fmt.Sprintf("keys.%s needs a type", name))
			}
		}
		check("allow", i, e, p, "allow|"+e.target())
	}
	return errors.Join(errs...)
}

// commonProblems checks the fields every entry shares and that action applies to the entry's kind.
// Each kind is addressed by its own field.
func (e overrideEntry) commonProblems(action string, kinds ...string) []string {
	var p []string
	if strings.TrimSpace(e.Reason) == "" {
		p = append(p, "reason is required")
	}
	if slices.Contains([]string{kindAppleProfile, kindAppleDeclaration, kindWindows}, e.Kind) && !slices.Contains(kinds, e.Kind) {
		return append(p, fmt.Sprintf("%s doesn't apply to %s entries, only %s", action, e.Kind, strings.Join(kinds, ", ")))
	}
	set := 0
	for _, v := range []string{e.PayloadType, e.DeclarationType, e.LocURI} {
		if v != "" {
			set++
		}
	}
	switch e.Kind {
	case kindAppleProfile:
		if e.PayloadType == "" || set != 1 {
			p = append(p, "apple_profile entries need payload_type (and no declaration_type or loc_uri)")
		}
	case kindAppleDeclaration:
		if e.DeclarationType == "" || set != 1 {
			p = append(p, "apple_declaration entries need declaration_type (and no payload_type or loc_uri)")
		}
	case kindWindows:
		if e.LocURI == "" || set != 1 {
			p = append(p, "windows entries need loc_uri (and no payload_type or declaration_type)")
		}
	default:
		p = append(p, fmt.Sprintf("kind must be apple_profile, apple_declaration, or windows, got %q", e.Kind))
	}
	return p
}

type overrideApplier struct {
	ctx    context.Context
	logger *slog.Logger
	schema *profilevalidation.ProfileValidationSchema
	merger *appleMerger
	errs   []error
}

func (a *overrideApplier) do(action string, i int, e overrideEntry, apply func(overrideEntry) error) {
	logger := a.logger.With("action", action, "kind", e.Kind, "target", e.target())
	if err := apply(e); err != nil {
		a.errs = append(a.errs, fmt.Errorf("%s[%d] (%s %s %s): %w", action, i, e.Kind, e.target(), e.KeyPath, err))
		return
	}
	if e.KeyPath != "" {
		logger = logger.With("key_path", e.KeyPath)
	}
	if e.Name != "" {
		logger = logger.With("name", e.Name)
	}
	logger.InfoContext(a.ctx, "applied override", "reason", e.Reason)
}

// payloadType returns the Apple entry an override addresses, or nil if the schema has none.
func (a *overrideApplier) payloadType(e overrideEntry) *profilevalidation.ApplePayloadType {
	s := a.schema
	switch {
	case e.Kind == kindAppleProfile && e.PayloadType == appleTopLevel:
		return s.AppleEnvelope.TopLevel
	case e.Kind == kindAppleProfile && e.PayloadType == appleCommonPayloadKeys:
		return s.AppleEnvelope.CommonPayloadKeys
	case e.Kind == kindAppleProfile:
		return s.AppleProfiles[e.PayloadType]
	case e.Kind == kindAppleDeclaration && e.DeclarationType == declarationBaseName:
		return s.AppleDeclarationEnvelope
	default:
		return s.AppleDeclarations[e.DeclarationType]
	}
}

// withRoot runs fn on pt viewed as a dictionary key, the same view the generator merges with, so
// key paths resolve the same way at every level.
func withRoot(pt *profilevalidation.ApplePayloadType, fn func(root *profilevalidation.AppleKey) error) error {
	root := &profilevalidation.AppleKey{
		Type: profilevalidation.AppleTypes{profilevalidation.AppleTypeDictionary},
		Keys: pt.Keys, Open: pt.Open, Values: pt.Values,
	}
	err := fn(root)
	pt.Keys, pt.Open, pt.Values = root.Keys, root.Open, root.Values
	return err
}

func (a *overrideApplier) allow(e overrideEntry) error {
	pt := &profilevalidation.ApplePayloadType{
		Source:   profilevalidation.AppleSourceOverride,
		Override: e.Reason,
		Keys:     e.Keys,
		// Vendors add settings all the time; allow says the type is legitimate, not that its keys are complete.
		Open: true,
	}
	if existing := a.schema.AppleProfiles[e.PayloadType]; existing != nil {
		// Apple now ships this type. Merging would apply Apple's closed key set and typed keys to
		// profiles the allow entry accepted, so the allow definition replaces Apple's. It still
		// counts as Apple's toward the schema floor.
		pt.Source = existing.Source
		a.logger.WarnContext(a.ctx, "allow targets a type Apple now defines; keeping the allow definition so no profile gets stricter, "+
			"replace it with add/loosen entries to adopt Apple's schema",
			"payload_type", e.PayloadType, "replaced_apple_keys", slices.Sorted(maps.Keys(existing.Keys)), "apple_open", existing.Open)
	}
	a.schema.AppleProfiles[e.PayloadType] = pt
	return nil
}

func (a *overrideApplier) add(e overrideEntry) error {
	pt := a.payloadType(e)
	if pt == nil {
		return fmt.Errorf("%s isn't in the schema", e.target())
	}
	return withRoot(pt, func(root *profilevalidation.AppleKey) error {
		parent, rest := resolveKeyPath(root, e.KeyPath)
		if rest != "" {
			return fmt.Errorf("key_path %q not found (no key matches %q)", e.KeyPath, rest)
		}
		if !slices.Contains(parent.Type, profilevalidation.AppleTypeDictionary) {
			return fmt.Errorf("key_path %q is a %v, not a dictionary", e.KeyPath, parent.Type)
		}
		a.addKey(parent, e.Name, e.Key, e.Reason, joinKeyPath(e.KeyPath, e.Name))
		return nil
	})
}

// addKey adds k under parent. If upstream already defines the name, the two merge loosest.
func (a *overrideApplier) addKey(parent *profilevalidation.AppleKey, name string, k *profilevalidation.AppleKey, reason, path string) {
	if parent.Keys == nil {
		parent.Keys = map[string]*profilevalidation.AppleKey{}
	}
	if existing := parent.Keys[name]; existing != nil {
		a.logger.WarnContext(a.ctx, "add targets a key Apple now defines; merging loosest, consider removing the entry", "path", path)
		k = a.merger.mergeKey(path, existing, k)
	}
	k.Override = reason
	parent.Keys[name] = k
}

func (a *overrideApplier) loosen(e overrideEntry) error {
	if e.Kind == kindWindows {
		return a.loosenWindows(e)
	}
	pt := a.payloadType(e)
	if pt == nil {
		return fmt.Errorf("%s isn't in the schema", e.target())
	}
	return withRoot(pt, func(root *profilevalidation.AppleKey) error {
		k, rest := resolveKeyPath(root, e.KeyPath)
		if rest != "" || k == root {
			return fmt.Errorf("key_path %q not found (no key matches %q); Apple may have renamed or removed it", e.KeyPath, rest)
		}
		if e.Action == "any" {
			k.Type, k.Open = profilevalidation.AppleTypes{profilevalidation.AppleTypeAny}, true
			// "Any value" must not stay narrowed by value constraints. Required, Items, Keys, and
			// Values stay: any value isn't optional, and nested keys may come from an earlier add.
			k.RangeList, k.Range, k.Format, k.ValueType, k.Repetition = nil, nil, "", "", nil
		} else {
			// Unioned, never replaced: an override can only widen.
			types := make(profilevalidation.AppleTypes, 0, len(e.Types))
			for _, t := range e.Types {
				types = append(types, profilevalidation.AppleType(t))
			}
			k.Type = mergeTypes(k.Type, types)
		}
		k.Override = e.Reason
		return nil
	})
}

// loosenWindows widens the node at e.LocURI, matched exactly: a "*" path is targeted as written.
func (a *overrideApplier) loosenWindows(e overrideEntry) error {
	n := a.schema.Windows[e.LocURI]
	if n == nil {
		return errors.New("loc_uri not found; Microsoft may have renamed or removed it")
	}
	if e.Action == "any" {
		// "Any value" must not stay narrowed by value constraints, including ones added to
		// WindowsNode later, so keep only what describes the node rather than its value.
		*n = profilevalidation.WindowsNode{
			Format: profilevalidation.WindowsFormats{profilevalidation.WindowsFormatAny},
			Scope:  n.Scope, MinBuild: n.MinBuild, Deprecated: n.Deprecated, Source: n.Source,
		}
	} else {
		// Unioned, never replaced: an override can only widen.
		formats := make(profilevalidation.WindowsFormats, 0, len(e.Types))
		for _, t := range e.Types {
			formats = append(formats, profilevalidation.WindowsFormat(t))
		}
		n.Format = mergeTypes(n.Format, formats)
	}
	n.Override = e.Reason
	return nil
}

// resolveKeyPath walks path ("A.B[].C") from root and returns the deepest key reached and the part
// of path it couldn't match. Apple key names can contain dots ("com.apple.login.mcx.X") and
// spaces, so each step matches the longest key name that fits.
func resolveKeyPath(root *profilevalidation.AppleKey, path string) (*profilevalidation.AppleKey, string) {
	cur, rest := root, path
	for rest != "" {
		if after, ok := strings.CutPrefix(rest, "[]"); ok {
			if cur.Items == nil {
				return cur, rest
			}
			cur, rest = cur.Items, strings.TrimPrefix(after, ".")
			continue
		}
		best := ""
		for name := range cur.Keys {
			if len(name) > len(best) && strings.HasPrefix(rest, name) &&
				(len(rest) == len(name) || rest[len(name)] == '.' || rest[len(name)] == '[') {
				best = name
			}
		}
		next := cur.Keys[best]
		if best == "" || next == nil {
			// Apple's ANY subkey lives in Values, never in Keys, so it can't shadow a literal name.
			after, ok := strings.CutPrefix(rest, appleAnyKey)
			if !ok || cur.Values == nil || (after != "" && after[0] != '.' && after[0] != '[') {
				return cur, rest
			}
			best, next = appleAnyKey, cur.Values
		}
		cur, rest = next, strings.TrimPrefix(rest[len(best):], ".")
	}
	return cur, ""
}
