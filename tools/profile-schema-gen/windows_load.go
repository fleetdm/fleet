package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ddfNode is one <Node> of a DDF file. Tags have no namespace, so they match the DDF's default and
// MSFT namespaces alike.
type ddfNode struct {
	NodeName string     `xml:"NodeName"`
	Path     *string    `xml:"Path"`
	Props    ddfProps   `xml:"DFProperties"`
	Nodes    []*ddfNode `xml:"Node"`
}

type ddfProps struct {
	Format ddfChoice `xml:"DFFormat"`
}

// ddfChoice is an element whose meaning is the name of its single child, e.g. <DFFormat><int/></DFFormat>.
type ddfChoice struct {
	Children []struct {
		XMLName xml.Name
	} `xml:",any"`
}

func (c ddfChoice) value() string {
	if len(c.Children) == 0 {
		return ""
	}
	return c.Children[0].XMLName.Local
}

// ddfLeaf is a node without children: the settings a profile can target.
type ddfLeaf struct {
	LocURI string
	File   string // base name, for logs and errors
	Props  ddfProps
}

// loadWindows reads every DDF file in dir and returns its leaves sorted by LocURI.
func loadWindows(ctx context.Context, logger *slog.Logger, dir string) ([]ddfLeaf, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return nil, err
	}
	// Glob is case-sensitive; the zip has .XML files too.
	upper, err := filepath.Glob(filepath.Join(dir, "*.XML"))
	if err != nil {
		return nil, err
	}
	paths = append(paths, upper...)
	slices.Sort(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no DDF files in %s", dir)
	}

	byLocURI := map[string]ddfLeaf{}
	var nodes, dynamic int
	for _, p := range paths {
		file := filepath.Base(p)
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var tree struct {
			Nodes []*ddfNode `xml:"Node"`
		}
		// encoding/xml rejects the UTF-8 byte order mark Microsoft writes.
		if err := xml.Unmarshal(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), &tree); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		var walk func(n *ddfNode, parent string) error
		walk = func(n *ddfNode, parent string) error {
			nodes++
			if n.Path != nil {
				parent = strings.TrimSpace(*n.Path)
			}
			name := strings.TrimSpace(n.NodeName)
			if name == "" {
				// Named by the server or client at runtime (DynamicNodeNaming).
				name = "*"
				dynamic++
			}
			locURI := name
			if parent != "" {
				locURI = parent + "/" + name
			}
			for _, c := range n.Nodes {
				if err := walk(c, locURI); err != nil {
					return err
				}
			}
			if len(n.Nodes) > 0 || n.Props.Format.value() == "node" {
				return nil
			}
			if prev, ok := byLocURI[locURI]; ok {
				return fmt.Errorf("%s is defined in both %s and %s", locURI, prev.File, file)
			}
			byLocURI[locURI] = ddfLeaf{LocURI: locURI, File: file, Props: n.Props}
			logger.DebugContext(ctx, "loaded leaf", "file", file, "loc_uri", locURI)
			return nil
		}
		for _, n := range tree.Nodes {
			if err := walk(n, ""); err != nil {
				return nil, err
			}
		}
	}

	leaves := make([]ddfLeaf, 0, len(byLocURI))
	for _, locURI := range slices.Sorted(maps.Keys(byLocURI)) {
		leaves = append(leaves, byLocURI[locURI])
	}
	logger.InfoContext(ctx, "loaded DDF files", "files", len(paths), "nodes", nodes, "leaves", len(leaves), "dynamic_segments", dynamic)
	return leaves, nil
}
