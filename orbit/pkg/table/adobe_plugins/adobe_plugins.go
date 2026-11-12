// Package adobe_plugins implements an osquery extension table that detects
// Adobe plugins (CEP extensions, UXP extensions, and native plug-ins) on
// macOS and Windows endpoints by scanning well-known directories and parsing
// plugin manifests.
//
// The table supports a scan_level constraint in the WHERE clause:
//
//	SELECT * FROM adobe_plugins;                              -- standard (default)
//	SELECT * FROM adobe_plugins WHERE scan_level = 'deep';    -- includes native plug-ins
//
// Standard: scans CEP and UXP extension directories only.
// Deep: additionally scans application-specific native plug-in directories
// (Photoshop, Premiere Pro, After Effects, Illustrator).
package adobe_plugins

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

// maxManifestSize is the maximum size of a manifest file we'll read.
// Prevents memory exhaustion from unexpectedly large files (orbit runs as root).
const maxManifestSize = 1 << 20 // 1 MB

const tableName = "adobe_plugins"

const (
	colPath            = "path"
	colName            = "name"
	colVersion         = "version"
	colVendor          = "vendor"
	colBundleID        = "bundle_id"
	colHostApplication = "host_application"
	colExtensionType   = "extension_type"
	colUser            = "user"
	colPlatform        = "platform"
	colScanLevel       = "scan_level"
)

// scanPath describes a directory to scan for Adobe plugins.
type scanPath struct {
	basePath      string // directory path, may contain glob wildcards
	extensionType string // "CEP", "UXP", or "native"
	hostApp       string // known host application from path context
	user          string // username for user-scoped installs, empty for system
}

// hostAppCodes maps Adobe host application codes found in manifests to
// human-readable application names.
var hostAppCodes = map[string]string{
	"PHXS": "Photoshop",
	"PHSP": "Photoshop",
	"PS":   "Photoshop",
	"ILST": "Illustrator",
	"AI":   "Illustrator",
	"PPRO": "Premiere Pro",
	"AEFT": "After Effects",
	"AE":   "After Effects",
	"IDSN": "InDesign",
	"ID":   "InDesign",
	"FLPR": "Animate",
	"DRWV": "Dreamweaver",
	"AUDT": "Audition",
	"AU":   "Audition",
	"KBRG": "Bridge",
	"LTRM": "Lightroom",
	"LRCC": "Lightroom Classic",
	"XD":   "XD",
	"AICY": "InCopy",
	"PRLD": "Prelude",
}

type adobePluginsTable struct {
	logger zerolog.Logger
}

// TablePlugin returns the osquery plugin for the adobe_plugins table.
func TablePlugin(logger zerolog.Logger) *table.Plugin {
	t := &adobePluginsTable{
		logger: logger.With().Str("table", tableName).Logger(),
	}
	return table.NewPlugin(tableName, Columns(), t.generate)
}

// Columns defines the table schema.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn(colPath),
		table.TextColumn(colName),
		table.TextColumn(colVersion),
		table.TextColumn(colVendor),
		table.TextColumn(colBundleID),
		table.TextColumn(colHostApplication),
		table.TextColumn(colExtensionType),
		table.TextColumn(colUser),
		table.TextColumn(colPlatform),
		// scan_level controls scan depth. Populated in results so osquery's
		// post-generate WHERE filter doesn't discard rows.
		table.TextColumn(colScanLevel),
	}
}

func (t *adobePluginsTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	scanLevels := tablehelpers.GetConstraints(queryContext, colScanLevel,
		tablehelpers.WithDefaults("standard"),
		tablehelpers.WithAllowedValues([]string{"standard", "deep"}),
		tablehelpers.WithLogger(t.logger),
	)

	level := "standard"
	if slices.Contains(scanLevels, "deep") {
		level = "deep"
	}

	paths, err := getScanPaths(level, t.logger)
	if err != nil {
		t.logger.Warn().Err(err).Msg("failed to build scan paths")
		return nil, nil
	}

	var results []map[string]string
	seen := make(map[string]struct{})

	for _, sp := range paths {
		if ctx.Err() != nil {
			return results, nil
		}

		matches, err := filepath.Glob(sp.basePath)
		if err != nil {
			t.logger.Debug().Err(err).Str("path", sp.basePath).Msg("glob error")
			continue
		}

		for _, dir := range matches {
			if ctx.Err() != nil {
				return results, nil
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.logger.Debug().Err(err).Str("dir", dir).Msg("cannot read directory")
				continue
			}

			for _, entry := range entries {
				// Skip symlinks to avoid traversing outside intended scan dirs.
				if entry.Type()&fs.ModeSymlink != 0 {
					continue
				}

				pluginPath := filepath.Join(dir, entry.Name())

				if _, ok := seen[pluginPath]; ok {
					continue
				}
				seen[pluginPath] = struct{}{}

				row := t.scanEntry(pluginPath, entry, sp)
				if row != nil {
					row[colScanLevel] = level
					results = append(results, row)
				}
			}
		}
	}

	return results, nil
}

func (t *adobePluginsTable) scanEntry(pluginPath string, entry os.DirEntry, sp scanPath) map[string]string {
	switch sp.extensionType {
	case "CEP":
		if !entry.IsDir() {
			return nil
		}
		return t.parseCEPPlugin(pluginPath, sp)
	case "UXP":
		if !entry.IsDir() {
			return nil
		}
		return t.parseUXPPlugin(pluginPath, sp)
	case "native":
		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		return parseNativePlugin(pluginPath, entry, sp)
	}
	return nil
}

// CEP manifest XML structures (CSXS/manifest.xml)

type cepManifest struct {
	XMLName  xml.Name `xml:"ExtensionManifest"`
	BundleID string   `xml:"ExtensionBundleId,attr"`
	Version  string   `xml:"ExtensionBundleVersion,attr"`
	Author   struct {
		Name string `xml:"Name,attr"`
	} `xml:"Author"`
	ExecutionEnvironment struct {
		HostList struct {
			Hosts []struct {
				Name string `xml:"Name,attr"`
			} `xml:"Host"`
		} `xml:"HostList"`
	} `xml:"ExecutionEnvironment"`
}

func (t *adobePluginsTable) parseCEPPlugin(pluginPath string, sp scanPath) map[string]string {
	row := map[string]string{
		colPath:          pluginPath,
		colName:          filepath.Base(pluginPath),
		colExtensionType: "CEP",
		colUser:          sp.user,
		colPlatform:      runtime.GOOS,
	}

	manifestPath := filepath.Join(pluginPath, "CSXS", "manifest.xml")
	data, err := readFileCapped(manifestPath, maxManifestSize)
	if err != nil {
		t.logger.Debug().Err(err).Str("path", manifestPath).Msg("no CEP manifest found")
		return row
	}

	var m cepManifest
	if err := xml.Unmarshal(data, &m); err != nil {
		t.logger.Debug().Err(err).Str("path", manifestPath).Msg("failed to parse CEP manifest")
		return row
	}

	row[colVersion] = m.Version
	row[colVendor] = m.Author.Name
	row[colBundleID] = m.BundleID

	var hostCodes []string
	for _, h := range m.ExecutionEnvironment.HostList.Hosts {
		hostCodes = append(hostCodes, h.Name)
	}
	hostApps := resolveHostApps(hostCodes)
	if hostApps == "" {
		hostApps = sp.hostApp
	}
	row[colHostApplication] = hostApps

	return row
}

// UXP manifest JSON structures (manifest.json)

type uxpManifest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Host    []struct {
		App string `json:"app"`
	} `json:"host"`
	Metadata struct {
		Publisher string `json:"publisher"`
	} `json:"metadata"`
}

func (t *adobePluginsTable) parseUXPPlugin(pluginPath string, sp scanPath) map[string]string {
	row := map[string]string{
		colPath:          pluginPath,
		colName:          filepath.Base(pluginPath),
		colExtensionType: "UXP",
		colUser:          sp.user,
		colPlatform:      runtime.GOOS,
	}

	manifestPath := filepath.Join(pluginPath, "manifest.json")
	data, err := readFileCapped(manifestPath, maxManifestSize)
	if err != nil {
		t.logger.Debug().Err(err).Str("path", manifestPath).Msg("no UXP manifest found")
		return row
	}

	var m uxpManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.logger.Debug().Err(err).Str("path", manifestPath).Msg("failed to parse UXP manifest")
		return row
	}

	name := m.Name
	if name == "" {
		name = m.ID
	}
	if name == "" {
		name = filepath.Base(pluginPath)
	}
	row[colName] = name
	row[colVersion] = m.Version
	row[colVendor] = m.Metadata.Publisher
	row[colBundleID] = m.ID

	var hostCodes []string
	for _, h := range m.Host {
		hostCodes = append(hostCodes, h.App)
	}
	hostApps := resolveHostApps(hostCodes)
	if hostApps == "" {
		hostApps = sp.hostApp
	}
	row[colHostApplication] = hostApps

	return row
}

func parseNativePlugin(pluginPath string, entry os.DirEntry, sp scanPath) map[string]string {
	name := entry.Name()
	for _, ext := range []string{".plugin", ".bundle", ".8bf", ".8bi", ".dll", ".aex"} {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			name = name[:len(name)-len(ext)]
			break
		}
	}

	return map[string]string{
		colPath:            pluginPath,
		colName:            name,
		colHostApplication: sp.hostApp,
		colExtensionType:   "native",
		colUser:            sp.user,
		colPlatform:        runtime.GOOS,
	}
}

// readFileCapped reads up to maxBytes from a file. This prevents memory
// exhaustion from unexpectedly large files since orbit runs as root.
func readFileCapped(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxBytes))
}

// resolveHostApps converts a list of Adobe host application codes to
// human-readable names, deduplicating entries.
func resolveHostApps(codes []string) string {
	seen := make(map[string]struct{})
	var apps []string
	for _, code := range codes {
		app := code
		if resolved, ok := hostAppCodes[strings.ToUpper(code)]; ok {
			app = resolved
		}
		if _, exists := seen[app]; !exists {
			seen[app] = struct{}{}
			apps = append(apps, app)
		}
	}
	return strings.Join(apps, ", ")
}
