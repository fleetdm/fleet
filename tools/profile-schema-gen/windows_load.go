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
	// Applicability is inherited by child nodes; the loader copies the nearest ancestor's into a
	// node that has none.
	Applicability *ddfApplicability `xml:"Applicability"`
	Deprecated    *struct {
		Build string `xml:"OsBuildDeprecated,attr"`
	} `xml:"Deprecated"`
	AllowedValues *ddfAllowedValues `xml:"AllowedValues"`
	// Other collects every element not decoded above, so new upstream fields can be reported.
	Other []struct {
		XMLName xml.Name
	} `xml:",any"`
}

type ddfApplicability struct {
	OsBuildVersion string `xml:"OsBuildVersion"`
}

type ddfAllowedValues struct {
	ValueType string `xml:"ValueType,attr"`
	Enums     []struct {
		Value string `xml:"Value"`
	} `xml:"Enum"`
	Value string `xml:"Value"`
	List  *struct {
		Delimiter string `xml:"Delimiter,attr"`
	} `xml:"List"`
	AdmxBacked *struct{} `xml:"AdmxBacked"`
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

// ddfEntry is one node of the DDF tree. Leaves are settings; interior nodes are kept too, since
// profiles <Add> them to create dynamic instances (e.g. a SCEP profile or a firewall rule).
type ddfEntry struct {
	LocURI string
	File   string // base name, for logs and errors
	Props  ddfProps
}

// loadWindows reads every DDF file in dir and returns its nodes sorted by LocURI.
func loadWindows(ctx context.Context, logger *slog.Logger, dir string) ([]ddfEntry, error) {
	dirEntries, err := os.ReadDir(dir) // sorted by name
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, d := range dirEntries {
		// The same test the extractor uses: the zip has .XML files too.
		if !d.IsDir() && strings.EqualFold(filepath.Ext(d.Name()), ".xml") {
			paths = append(paths, filepath.Join(dir, d.Name()))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no DDF files in %s", dir)
	}

	byLocURI := map[string]ddfEntry{}
	var leaves, dynamic int
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
		var walk func(n *ddfNode, parent string, inherited *ddfApplicability) error
		walk = func(n *ddfNode, parent string, inherited *ddfApplicability) error {
			if n.Props.Applicability == nil {
				n.Props.Applicability = inherited
			}
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
				if err := walk(c, locURI, n.Props.Applicability); err != nil {
					return err
				}
			}
			if prev, ok := byLocURI[locURI]; ok {
				return fmt.Errorf("%s is defined in both %s and %s", locURI, prev.File, file)
			}
			byLocURI[locURI] = ddfEntry{LocURI: locURI, File: file, Props: n.Props}
			if len(n.Nodes) == 0 && n.Props.Format.value() != "node" {
				leaves++
			}
			logger.DebugContext(ctx, "loaded node", "file", file, "loc_uri", locURI)
			return nil
		}
		for _, n := range tree.Nodes {
			if err := walk(n, "", nil); err != nil {
				return nil, err
			}
		}
	}

	entries := make([]ddfEntry, 0, len(byLocURI))
	for _, locURI := range slices.Sorted(maps.Keys(byLocURI)) {
		entries = append(entries, byLocURI[locURI])
	}
	logger.InfoContext(ctx, "loaded DDF files", "files", len(paths), "nodes", len(entries), "leaves", leaves, "dynamic_segments", dynamic)
	return entries, nil
}
