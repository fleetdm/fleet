package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"gopkg.in/yaml.v3"
)

// appleAnyKey is Apple's subkey name for "any key name"; its definition applies to every value.
const appleAnyKey = "ANY"

// appleRawKey is one entry of payloadkeys or subkeys. Fields Fleet doesn't use (title, content,
// default, combinetype, subkeytype, assettypes, ...) are dropped by not declaring them.
type appleRawKey struct {
	Key        string      `yaml:"key"`
	Type       string      `yaml:"type"`
	Presence   string      `yaml:"presence"`
	RangeList  []any       `yaml:"rangelist"`
	Range      *appleRange `yaml:"range"`
	Format     string      `yaml:"format"`
	Repetition *appleRange `yaml:"repetition"`
	ValueType  string      `yaml:"valuetype"`
	SubType    string      `yaml:"subtype"` // deprecated upstream in favor of valuetype
	// SupportedOS updates the parent's platforms field by field (Apple's docs/schema.md).
	SupportedOS appleRawPlatforms `yaml:"supportedOS"`
	// SubKeys points into the parsed document (see appleFile) and is filled by decode.
	SubKeys []*yaml.Node `yaml:"-"`
}

// appleRawPlatforms is a supportedOS object. Only lifecycle and supervised are kept; pointers tell
// "not set here, inherit" apart from a set value.
type appleRawPlatforms map[string]struct {
	Introduced *string `yaml:"introduced"`
	Deprecated *string `yaml:"deprecated"`
	Removed    *string `yaml:"removed"`
	Supervised *bool   `yaml:"supervised"`
}

// resolve applies p on top of parent and returns the result; parent isn't modified.
func (p appleRawPlatforms) resolve(parent map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) map[profilevalidation.AppleOS]profilevalidation.ApplePlatform {
	if len(p) == 0 {
		return parent
	}
	out := maps.Clone(parent)
	if out == nil {
		out = map[profilevalidation.AppleOS]profilevalidation.ApplePlatform{}
	}
	for os, raw := range p {
		plat := out[profilevalidation.AppleOS(os)]
		if raw.Introduced != nil {
			plat.Introduced = *raw.Introduced
		}
		if raw.Deprecated != nil {
			plat.Deprecated = *raw.Deprecated
		}
		if raw.Removed != nil {
			plat.Removed = *raw.Removed
		}
		if raw.Supervised != nil {
			plat.Supervised = *raw.Supervised
		}
		out[profilevalidation.AppleOS(os)] = plat
	}
	return out
}

// platformsDiff returns the entries of resolved that differ from parent, or nil if none do.
func platformsDiff(resolved, parent map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) map[profilevalidation.AppleOS]profilevalidation.ApplePlatform {
	var diff map[profilevalidation.AppleOS]profilevalidation.ApplePlatform
	for os, plat := range resolved {
		if prev, ok := parent[os]; ok && prev == plat {
			continue
		}
		if diff == nil {
			diff = map[profilevalidation.AppleOS]profilevalidation.ApplePlatform{}
		}
		diff[os] = plat
	}
	return diff
}

// trimPlatforms replaces each key's resolved platforms with only those that differ from its
// parent's, recursively. Run it after merging, so differences are relative to the merged parent.
func trimPlatforms(keys map[string]*profilevalidation.AppleKey, parent map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) {
	for _, k := range keys {
		trimKeyPlatforms(k, parent)
	}
}

func trimKeyPlatforms(k *profilevalidation.AppleKey, parent map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) {
	if k == nil {
		return
	}
	resolved := k.Platforms
	trimPlatforms(k.Keys, resolved)
	trimKeyPlatforms(k.Items, resolved)
	trimKeyPlatforms(k.Values, resolved)
	k.Platforms = platformsDiff(resolved, parent)
}

type appleRange struct {
	Min *float64 `yaml:"min"`
	Max *float64 `yaml:"max"`
}

// appleConverter turns one file's YAML keys into schema keys.
type appleConverter struct {
	ctx    context.Context
	logger *slog.Logger
	file   string
	// visiting maps each definition on the current path to the key path it was entered at, so a
	// definition that contains itself becomes a ref instead of an endless expansion.
	visiting map[*yaml.Node]string
}

func newAppleConverter(ctx context.Context, logger *slog.Logger, file string) *appleConverter {
	return &appleConverter{ctx: ctx, logger: logger.With("file", file), file: file, visiting: map[*yaml.Node]string{}}
}

// convertKeys converts a payloadkeys or dictionary subkeys list. An ANY entry makes the container
// open, with its definition as the values every other key must match. parentPlatforms are the
// container's resolved platforms, which each key inherits.
func (c *appleConverter) convertKeys(nodes []*yaml.Node, parentPath string, parentPlatforms map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) (keys map[string]*profilevalidation.AppleKey, open bool, values *profilevalidation.AppleKey, err error) {
	for _, n := range nodes {
		raw, def, err := c.decode(n, parentPath)
		if err != nil {
			return nil, false, nil, err
		}
		path := joinKeyPath(parentPath, raw.Key)
		key, err := c.convertKey(def, raw, path, parentPlatforms)
		if err != nil {
			return nil, false, nil, err
		}
		if raw.Key == appleAnyKey {
			c.logger.DebugContext(c.ctx, "ANY subkey: container accepts any key name", "path", path)
			open, values = true, key
			continue
		}
		if keys == nil {
			keys = map[string]*profilevalidation.AppleKey{}
		}
		if _, dup := keys[raw.Key]; dup {
			return nil, false, nil, fmt.Errorf("%s: key %q is defined twice", c.file, path)
		}
		keys[raw.Key] = key
	}
	return keys, open, values, nil
}

// convertKey converts one key definition; path is its dotted key path, with "[]" for array items.
// The key holds its fully resolved platforms; trimPlatforms reduces them to differences once files
// sharing a type have been merged.
func (c *appleConverter) convertKey(def *yaml.Node, raw *appleRawKey, path string, parentPlatforms map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) (*profilevalidation.AppleKey, error) {
	typ := profilevalidation.AppleType(strings.TrimSuffix(strings.TrimPrefix(raw.Type, "<"), ">"))
	if enclosing, ok := c.visiting[def]; ok {
		// Only the first occurrence of a reused definition on a path is written out; this inner one
		// points back at it. Reuse that isn't nested in itself is still expanded in full.
		c.logger.DebugContext(c.ctx, "definition contains itself; emitting ref", "path", path, "ref", enclosing)
		return &profilevalidation.AppleKey{
			Type:      profilevalidation.AppleTypes{typ},
			Ref:       enclosing,
			Platforms: raw.SupportedOS.resolve(parentPlatforms),
		}, nil
	}
	c.visiting[def] = path
	defer delete(c.visiting, def)

	key := &profilevalidation.AppleKey{
		Type:       profilevalidation.AppleTypes{typ},
		Required:   raw.Presence == "required",
		RangeList:  raw.RangeList,
		Range:      convertRange(raw.Range),
		Format:     raw.Format,
		Repetition: convertRange(raw.Repetition),
		ValueType:  profilevalidation.AppleValueType(raw.ValueType),
	}
	if key.ValueType == "" && raw.SubType != "" {
		c.logger.DebugContext(c.ctx, "using deprecated subtype as valuetype", "path", path, "subtype", raw.SubType)
		key.ValueType = profilevalidation.AppleValueType(raw.SubType)
	}
	platforms := raw.SupportedOS.resolve(parentPlatforms)
	key.Platforms = platforms
	if !knownAppleType(typ) {
		c.logger.WarnContext(c.ctx, "unrecognized type; carried as declared", "path", path, "type", raw.Type)
	}

	switch typ {
	case profilevalidation.AppleTypeAny:
		key.Open = true
	case profilevalidation.AppleTypeArray:
		switch len(raw.SubKeys) {
		case 0:
			c.logger.DebugContext(c.ctx, "array without subkeys; item type unknown", "path", path)
		case 1:
			itemRaw, itemDef, err := c.decode(raw.SubKeys[0], path)
			if err != nil {
				return nil, err
			}
			if key.Items, err = c.convertKey(itemDef, itemRaw, path+"[]", platforms); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s: array %s has %d subkeys, want 1", c.file, path, len(raw.SubKeys))
		}
	case profilevalidation.AppleTypeDictionary:
		if len(raw.SubKeys) == 0 {
			// Nothing is known about the contents, so any key must pass.
			c.logger.DebugContext(c.ctx, "dictionary without subkeys; treating as open", "path", path)
			key.Open = true
			break
		}
		var err error
		if key.Keys, key.Open, key.Values, err = c.convertKeys(raw.SubKeys, path, platforms); err != nil {
			return nil, err
		}
	default:
		if len(raw.SubKeys) > 0 {
			c.logger.WarnContext(c.ctx, "subkeys on a non-container type; ignored", "path", path, "type", raw.Type)
		}
	}
	return key, nil
}

// decode follows an alias to its definition and decodes it. The definition node identifies a
// reused definition for cycle detection.
func (c *appleConverter) decode(n *yaml.Node, parentPath string) (*appleRawKey, *yaml.Node, error) {
	def := n
	if def.Kind == yaml.AliasNode {
		def = def.Alias
	}
	var raw appleRawKey
	if err := def.Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("%s: decode key under %q (line %d): %w", c.file, parentPath, n.Line, err)
	}
	if raw.Key == "" {
		return nil, nil, fmt.Errorf("%s: key without a name under %q (line %d)", c.file, parentPath, n.Line)
	}
	if sub := mappingValue(def, "subkeys"); sub != nil {
		raw.SubKeys = sub.Content
	}
	return &raw, def, nil
}

func joinKeyPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func convertRange(r *appleRange) *profilevalidation.AppleRange {
	if r == nil || (r.Min == nil && r.Max == nil) {
		return nil
	}
	return &profilevalidation.AppleRange{Min: r.Min, Max: r.Max}
}

func knownAppleType(t profilevalidation.AppleType) bool {
	switch t {
	case profilevalidation.AppleTypeString, profilevalidation.AppleTypeInteger, profilevalidation.AppleTypeReal,
		profilevalidation.AppleTypeBoolean, profilevalidation.AppleTypeDate, profilevalidation.AppleTypeData,
		profilevalidation.AppleTypeArray, profilevalidation.AppleTypeDictionary, profilevalidation.AppleTypeAny:
		return true
	}
	return false
}
