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
	SubKeys     yaml.Node         `yaml:"subkeys"` // a node for the same reason as appleFile.PayloadKeys

	// def is the definition node this key was decoded from, after following an alias.
	def *yaml.Node
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

// convertPayload converts a file's payloadkeys into a dictionary key standing for the payload type,
// so payload-level ANY, platforms, and merging work exactly as for nested dictionaries.
func (c *appleConverter) convertPayload(payloadKeys *yaml.Node, supportedOS appleRawPlatforms) (*profilevalidation.AppleKey, error) {
	root := &profilevalidation.AppleKey{
		Type:      profilevalidation.AppleTypes{profilevalidation.AppleTypeDictionary},
		Platforms: supportedOS.resolve(nil),
	}
	return root, c.convertKeys(nodeItems(payloadKeys), "", root)
}

// convertKeys converts a payloadkeys or dictionary subkeys list into parent's keys. An ANY entry
// makes parent open, with its definition as the values every other key must match.
func (c *appleConverter) convertKeys(nodes []*yaml.Node, parentPath string, parent *profilevalidation.AppleKey) error {
	for _, n := range nodes {
		raw, err := c.decode(n, parentPath)
		if err != nil {
			return err
		}
		path := joinKeyPath(parentPath, raw.Key)
		key, err := c.convertKey(raw, path, parent.Platforms)
		if err != nil {
			return err
		}
		if raw.Key == appleAnyKey {
			c.logger.DebugContext(c.ctx, "ANY subkey: container accepts any key name", "path", path)
			parent.Open, parent.Values = true, key
			continue
		}
		if parent.Keys == nil {
			parent.Keys = map[string]*profilevalidation.AppleKey{}
		}
		if _, dup := parent.Keys[raw.Key]; dup {
			return fmt.Errorf("%s: key %q is defined twice", c.file, path)
		}
		parent.Keys[raw.Key] = key
	}
	return nil
}

// convertKey converts one key definition; path is its dotted key path, with "[]" for array items.
// The key holds its fully resolved platforms; trimPlatforms reduces them to differences once files
// sharing a type have been merged.
func (c *appleConverter) convertKey(raw *appleRawKey, path string, parentPlatforms map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) (*profilevalidation.AppleKey, error) {
	typ := profilevalidation.AppleType(strings.TrimSuffix(strings.TrimPrefix(raw.Type, "<"), ">"))
	platforms := raw.SupportedOS.resolve(parentPlatforms)
	if enclosing, ok := c.visiting[raw.def]; ok {
		// Only the first occurrence of a reused definition on a path is written out; this inner one
		// points back at it. Reuse that isn't nested in itself is still expanded in full.
		c.logger.DebugContext(c.ctx, "definition contains itself; emitting ref", "path", path, "ref", enclosing)
		return &profilevalidation.AppleKey{Type: profilevalidation.AppleTypes{typ}, Ref: enclosing, Platforms: platforms}, nil
	}
	c.visiting[raw.def] = path
	defer delete(c.visiting, raw.def)

	key := &profilevalidation.AppleKey{
		Type:       profilevalidation.AppleTypes{typ},
		Required:   raw.Presence == "required",
		RangeList:  raw.RangeList,
		Range:      convertRange(raw.Range),
		Format:     raw.Format,
		Repetition: convertRange(raw.Repetition),
		ValueType:  profilevalidation.AppleValueType(raw.ValueType),
		Platforms:  platforms,
	}
	if key.ValueType == "" && raw.SubType != "" {
		c.logger.DebugContext(c.ctx, "using deprecated subtype as valuetype", "path", path, "subtype", raw.SubType)
		key.ValueType = profilevalidation.AppleValueType(raw.SubType)
	}
	if !knownAppleType(typ) {
		c.logger.WarnContext(c.ctx, "unrecognized type; carried as declared", "path", path, "type", raw.Type)
	}

	subKeys := nodeItems(&raw.SubKeys)
	switch typ {
	case profilevalidation.AppleTypeAny:
		key.Open = true
	case profilevalidation.AppleTypeArray:
		switch len(subKeys) {
		case 0:
			c.logger.DebugContext(c.ctx, "array without subkeys; item type unknown", "path", path)
		case 1:
			item, err := c.decode(subKeys[0], path)
			if err != nil {
				return nil, err
			}
			if key.Items, err = c.convertKey(item, path+"[]", platforms); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s: array %s has %d subkeys, want 1", c.file, path, len(subKeys))
		}
	case profilevalidation.AppleTypeDictionary:
		if len(subKeys) == 0 {
			// Nothing is known about the contents, so any key must pass.
			c.logger.DebugContext(c.ctx, "dictionary without subkeys; treating as open", "path", path)
			key.Open = true
			break
		}
		if err := c.convertKeys(subKeys, path, key); err != nil {
			return nil, err
		}
	default:
		if len(subKeys) > 0 {
			c.logger.WarnContext(c.ctx, "subkeys on a non-container type; ignored", "path", path, "type", raw.Type)
		}
	}
	return key, nil
}

// decode follows an alias to its definition and decodes it.
func (c *appleConverter) decode(n *yaml.Node, parentPath string) (*appleRawKey, error) {
	def := n
	if def.Kind == yaml.AliasNode {
		def = def.Alias
	}
	raw := &appleRawKey{def: def}
	if err := def.Decode(raw); err != nil {
		return nil, fmt.Errorf("%s: decode key under %q (line %d): %w", c.file, parentPath, n.Line, err)
	}
	if raw.Key == "" {
		return nil, fmt.Errorf("%s: key without a name under %q (line %d)", c.file, parentPath, n.Line)
	}
	return raw, nil
}

func joinKeyPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func convertRange(r *appleRange) *profilevalidation.Range {
	if r == nil || (r.Min == nil && r.Max == nil) {
		return nil
	}
	return &profilevalidation.Range{Min: r.Min, Max: r.Max}
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
