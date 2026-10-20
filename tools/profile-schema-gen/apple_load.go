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

// appleFile is one upstream YAML file. PayloadKeys point into the parsed document rather than
// being decoded: yaml.v3 copies nodes on decode, and node identity is how a definition that
// contains itself (through an alias) is recognized.
type appleFile struct {
	Path    string // relative to the download dir, for logs and errors
	Payload struct {
		PayloadType     string            `yaml:"payloadtype"`
		DeclarationType string            `yaml:"declarationtype"`
		SupportedOS     appleRawPlatforms `yaml:"supportedOS"`
	}
	PayloadKeys []*yaml.Node
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
// type (loginwindow.yaml).
func loadApple(ctx context.Context, logger *slog.Logger, dir string) (*appleSources, error) {
	src := &appleSources{Profiles: map[string]*appleGroup{}, Declarations: map[string]*appleGroup{}}

	profiles, err := loadAppleFiles(ctx, logger, dir, appleProfilesDir)
	if err != nil {
		return nil, err
	}
	for _, f := range profiles {
		name := f.Payload.PayloadType
		if name == "" {
			return nil, fmt.Errorf("%s has no payload.payloadtype", f.Path)
		}
		switch name {
		case appleTopLevel:
			src.TopLevel = addToGroup(src.TopLevel, name, f)
		case appleCommonPayloadKeys:
			src.CommonPayloadKeys = addToGroup(src.CommonPayloadKeys, name, f)
		default:
			src.Profiles[name] = addToGroup(src.Profiles[name], name, f)
		}
		logger.DebugContext(ctx, "loaded profile file", "file", f.Path, "payload_type", name)
	}
	if src.TopLevel == nil || src.CommonPayloadKeys == nil {
		return nil, fmt.Errorf("%s is missing %s.yaml or %s.yaml", appleProfilesDir, appleTopLevel, appleCommonPayloadKeys)
	}

	declarations, err := loadAppleFiles(ctx, logger, dir, appleDeclarationsDir)
	if err != nil {
		return nil, err
	}
	for _, f := range declarations {
		name := f.Payload.DeclarationType
		switch name {
		case "":
			// e.g. assets/credentials/*.yaml describe the data inside a credential asset (credentialtype).
			logger.InfoContext(ctx, "skipping declaration file without declarationtype", "file", f.Path)
			continue
		case appleDeclarationBase:
			src.DeclarationBase = addToGroup(src.DeclarationBase, name, f)
			logger.DebugContext(ctx, "loaded declaration envelope", "file", f.Path)
			continue
		}
		src.Declarations[name] = addToGroup(src.Declarations[name], name, f)
		logger.DebugContext(ctx, "loaded declaration file", "file", f.Path, "declaration_type", name)
	}

	if src.DeclarationBase == nil {
		return nil, fmt.Errorf("%s has no declaration envelope (declarationtype %q)", appleDeclarationsDir, appleDeclarationBase)
	}

	for _, groups := range []map[string]*appleGroup{src.Profiles, src.Declarations} {
		for _, name := range slices.Sorted(maps.Keys(groups)) {
			logSharedType(ctx, logger, groups[name])
		}
	}
	logger.InfoContext(ctx, "loaded Apple sources", "profile_files", len(profiles), "payload_types", len(src.Profiles),
		"declaration_files", len(declarations), "declaration_types", len(src.Declarations))
	return src, nil
}

// loadAppleFiles parses every .yaml file under dir/sub, in path order so output is deterministic.
func loadAppleFiles(ctx context.Context, logger *slog.Logger, dir, sub string) ([]*appleFile, error) {
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
	logger.DebugContext(ctx, "found YAML files", "dir", sub, "count", len(paths))

	files := make([]*appleFile, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return nil, err
		}
		f := &appleFile{Path: filepath.ToSlash(rel)}
		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Path, err)
		}
		if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
			return nil, fmt.Errorf("parse %s: not a YAML mapping", f.Path)
		}
		if payload := mappingValue(doc.Content[0], "payload"); payload != nil {
			if err := payload.Decode(&f.Payload); err != nil {
				return nil, fmt.Errorf("parse %s: payload: %w", f.Path, err)
			}
		}
		if keys := mappingValue(doc.Content[0], "payloadkeys"); keys != nil {
			f.PayloadKeys = keys.Content
		}
		files = append(files, f)
	}
	return files, nil
}

func addToGroup(g *appleGroup, name string, f *appleFile) *appleGroup {
	if g == nil {
		g = &appleGroup{Name: name}
	}
	g.Files = append(g.Files, f)
	return g
}

// logSharedType logs types built from several files and every key more than one of them defines.
// Those keys are merged to their loosest definition after conversion.
func logSharedType(ctx context.Context, logger *slog.Logger, g *appleGroup) {
	if len(g.Files) < 2 {
		return
	}
	paths := make([]string, 0, len(g.Files))
	definedIn := map[string][]string{}
	var order []string
	for _, f := range g.Files {
		paths = append(paths, f.Path)
		for _, k := range f.PayloadKeys {
			name := keyName(k)
			if _, ok := definedIn[name]; !ok {
				order = append(order, name)
			}
			definedIn[name] = append(definedIn[name], f.Path)
		}
	}
	logger.InfoContext(ctx, "combining files into one type", "type", g.Name, "files", strings.Join(paths, ", "))
	for _, name := range order {
		if len(definedIn[name]) > 1 {
			logger.InfoContext(ctx, "key defined in several files; will merge to loosest", "type", g.Name, "key", name,
				"files", strings.Join(definedIn[name], ", "))
		}
	}
}

// keyName returns the "key" field of a payload key mapping node.
func keyName(n *yaml.Node) string {
	if v := mappingValue(n, "key"); v != nil {
		return v.Value
	}
	return ""
}

// mappingValue returns the value node for key in a mapping node, or nil. Aliases are followed on
// both sides: Apple anchors whole subkeys lists ("subkeys: *id001"), not just single keys.
func mappingValue(n *yaml.Node, key string) *yaml.Node {
	if n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			v := n.Content[i+1]
			if v.Kind == yaml.AliasNode {
				v = v.Alias
			}
			return v
		}
	}
	return nil
}
