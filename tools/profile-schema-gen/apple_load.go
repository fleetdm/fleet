package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	appleProfilesDir     = "mdm/profiles"
	appleDeclarationsDir = "declarative/declarations"

	appleTopLevel          = "TopLevel"
	appleCommonPayloadKeys = "CommonPayloadKeys"
	// declarationbase.yaml describes the envelope every declaration shares, not a declaration type.
	appleDeclarationBase = "any"
)

// appleFile is one upstream YAML file.
type appleFile struct {
	Path    string `yaml:"-"` // relative to the download dir, for logs and errors
	Payload struct {
		PayloadType     string            `yaml:"payloadtype"`
		DeclarationType string            `yaml:"declarationtype"`
		SupportedOS     appleRawPlatforms `yaml:"supportedOS"`
	} `yaml:"payload"`
	// Kept as a node: its children still point into the parsed document, and pointer identity is
	// how a definition that contains itself is recognized.
	PayloadKeys yaml.Node `yaml:"payloadkeys"`
}

// appleGroup is every file that declares the same payload or declaration type, in path order.
type appleGroup struct {
	Name  string
	Files []*appleFile
}

type appleSources struct {
	TopLevel          *appleGroup
	CommonPayloadKeys *appleGroup
	DeclarationBase   *appleGroup
	Profiles          map[string]*appleGroup
	Declarations      map[string]*appleGroup
}

// loadApple reads the downloaded YAML and groups files by the type declared inside them, not by
// filename: several files can describe one type (com.apple.MCX), and a filename can differ from its
// type (loginwindow.yaml). The envelope schemas are then taken out of the groups by name.
func loadApple(ctx context.Context, logger *slog.Logger, dir string) (*appleSources, error) {
	profiles, err := loadAppleGroups(ctx, logger, dir, appleProfilesDir, func(f *appleFile) (string, error) {
		if f.Payload.PayloadType == "" {
			return "", fmt.Errorf("%s has no payload.payloadtype", f.Path)
		}
		return f.Payload.PayloadType, nil
	})
	if err != nil {
		return nil, err
	}
	declarations, err := loadAppleGroups(ctx, logger, dir, appleDeclarationsDir, func(f *appleFile) (string, error) {
		if f.Payload.DeclarationType == "" {
			// e.g. assets/credentials/*.yaml describe the data inside a credential asset (credentialtype).
			logger.InfoContext(ctx, "skipping declaration file without declarationtype", "file", f.Path)
		}
		return f.Payload.DeclarationType, nil
	})
	if err != nil {
		return nil, err
	}

	src := &appleSources{
		TopLevel:          takeGroup(profiles, appleTopLevel),
		CommonPayloadKeys: takeGroup(profiles, appleCommonPayloadKeys),
		DeclarationBase:   takeGroup(declarations, appleDeclarationBase),
		Profiles:          profiles,
		Declarations:      declarations,
	}
	if src.TopLevel == nil || src.CommonPayloadKeys == nil {
		return nil, fmt.Errorf("%s is missing %s.yaml or %s.yaml", appleProfilesDir, appleTopLevel, appleCommonPayloadKeys)
	}
	if src.DeclarationBase == nil {
		return nil, fmt.Errorf("%s has no declaration envelope (declarationtype %q)", appleDeclarationsDir, appleDeclarationBase)
	}

	for _, groups := range []map[string]*appleGroup{profiles, declarations} {
		for _, name := range slices.Sorted(maps.Keys(groups)) {
			logSharedType(ctx, logger, groups[name])
		}
	}
	logger.InfoContext(ctx, "loaded Apple sources", "payload_types", len(profiles), "declaration_types", len(declarations))
	return src, nil
}

// loadAppleGroups parses every .yaml file under dir/sub in path order, so output is deterministic,
// and groups them by typeOf. Files typeOf names "" are skipped.
func loadAppleGroups(ctx context.Context, logger *slog.Logger, dir, sub string, typeOf func(*appleFile) (string, error)) (map[string]*appleGroup, error) {
	var paths []string
	err := filepath.WalkDir(filepath.Join(dir, sub), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(p) == ".yaml" {
			paths = append(paths, p)
		}
		return err
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s not found in Apple download %s", sub, dir)
	}
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)

	groups := map[string]*appleGroup{}
	for _, p := range paths {
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return nil, err
		}
		f := &appleFile{Path: filepath.ToSlash(rel)}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, f); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Path, err)
		}
		name, err := typeOf(f)
		if err != nil {
			return nil, err
		}
		if name == "" {
			continue
		}
		if groups[name] == nil {
			groups[name] = &appleGroup{Name: name}
		}
		groups[name].Files = append(groups[name].Files, f)
		logger.DebugContext(ctx, "loaded file", "file", f.Path, "type", name)
	}
	return groups, nil
}

// takeGroup removes and returns groups[name], or nil.
func takeGroup(groups map[string]*appleGroup, name string) *appleGroup {
	g := groups[name]
	delete(groups, name)
	return g
}

// logSharedType logs types built from several files and every key more than one of them defines.
// Those keys are merged to their loosest definition after conversion.
func logSharedType(ctx context.Context, logger *slog.Logger, g *appleGroup) {
	if g == nil || len(g.Files) < 2 {
		return
	}
	paths := make([]string, 0, len(g.Files))
	definedIn := map[string][]string{}
	for _, f := range g.Files {
		paths = append(paths, f.Path)
		for _, k := range nodeItems(&f.PayloadKeys) {
			name := keyName(k)
			definedIn[name] = append(definedIn[name], f.Path)
		}
	}
	logger.InfoContext(ctx, "combining files into one type", "type", g.Name, "files", strings.Join(paths, ", "))
	for _, name := range slices.Sorted(maps.Keys(definedIn)) {
		if len(definedIn[name]) > 1 {
			logger.InfoContext(ctx, "key defined in several files; will merge to loosest", "type", g.Name, "key", name,
				"files", strings.Join(definedIn[name], ", "))
		}
	}
}

// nodeItems returns a sequence's items, following an alias first: Apple anchors whole subkeys
// lists ("subkeys: *id001"), not just single keys.
func nodeItems(n *yaml.Node) []*yaml.Node {
	if n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n.Content
}

func keyName(n *yaml.Node) string {
	var k struct {
		Key string `yaml:"key"`
	}
	_ = n.Decode(&k) // a malformed key fails properly during conversion
	return k.Key
}
