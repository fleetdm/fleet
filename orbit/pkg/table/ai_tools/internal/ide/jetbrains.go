package ide

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// productDirRe matches a JetBrains per-product config directory, e.g.
// "IntelliJIdea2026.1", "PyCharm2025.3", "GoLand2026.1", "AndroidStudio2025.1".
var productDirRe = regexp.MustCompile(`^[A-Za-z][A-Za-z ]*\d{4}\.\d+$`)

// versionSuffixRe matches the trailing "YYYY.N" version on a product dir name.
var versionSuffixRe = regexp.MustCompile(`\d{4}\.\d+$`)

type ideaPlugin struct {
	ID      string `xml:"id"`
	Name    string `xml:"name"`
	Version string `xml:"version"`
	Vendor  string `xml:"vendor"`
}

func jetbrainsRoots(r paths.Roots) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(r.MacAppSupport, "JetBrains"), filepath.Join(r.MacAppSupport, "Google")}
	case "windows":
		return []string{filepath.Join(r.AppData, "JetBrains"), filepath.Join(r.AppData, "Google")}
	default:
		return []string{filepath.Join(r.XDGData, "JetBrains"), filepath.Join(r.XDGData, "Google")}
	}
}

func scanJetBrains(h homes.Home, r paths.Roots) []Plugin {
	var out []Plugin
	for _, root := range jetbrainsRoots(r) {
		products, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, prod := range products {
			if !prod.IsDir() || !productDirRe.MatchString(prod.Name()) {
				continue
			}
			editor := productEditorName(prod.Name())
			// Plugins may live directly under the product dir or under /plugins.
			for _, pluginsDir := range []string{
				filepath.Join(root, prod.Name(), "plugins"),
				filepath.Join(root, prod.Name()),
			} {
				out = append(out, scanJetBrainsPluginsDir(h, editor, pluginsDir)...)
			}
		}
	}
	return out
}

func scanJetBrainsPluginsDir(h homes.Home, editor, pluginsDir string) []Plugin {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil
	}
	var out []Plugin
	for _, e := range entries {
		full := filepath.Join(pluginsDir, e.Name())
		var meta ideaPlugin
		var manifest string
		switch {
		case e.IsDir():
			meta, manifest = readPluginFromDir(full)
		case strings.HasSuffix(strings.ToLower(e.Name()), ".jar"):
			if xmlData, ok := readPluginXMLFromJar(full); ok {
				meta = parsePluginXML(xmlData)
				manifest = full
			}
		default:
			continue
		}
		if meta.ID == "" && meta.Name == "" {
			continue
		}
		id := firstNonEmptyStr(meta.ID, meta.Name)
		isAI, cat := classify.JetBrainsPlugin(meta.ID, meta.Name)
		if !isAI {
			continue // AI tools only — skip non-AI plugins
		}
		p := Plugin{
			Editor:       editor,
			EditorFamily: "jetbrains",
			PluginID:     id,
			Name:         firstNonEmptyStr(meta.Name, meta.ID),
			Version:      meta.Version,
			Publisher:    strings.TrimSpace(meta.Vendor),
			InstallPath:  full,
			ManifestPath: manifest,
		}
		out = append(out, p.finish(h, cat))
	}
	return out
}

// readPluginFromDir reads META-INF/plugin.xml directly, or from the first jar
// under lib/ that contains it (the common exploded-plugin layout).
func readPluginFromDir(dir string) (ideaPlugin, string) {
	direct := filepath.Join(dir, "META-INF", "plugin.xml")
	if b, err := os.ReadFile(direct); err == nil { // #nosec G304 -- fixed path under enumerated plugin dir
		return parsePluginXML(b), direct
	}
	libEntries, err := os.ReadDir(filepath.Join(dir, "lib"))
	if err != nil {
		return ideaPlugin{}, ""
	}
	for _, le := range libEntries {
		if !strings.HasSuffix(strings.ToLower(le.Name()), ".jar") {
			continue
		}
		jar := filepath.Join(dir, "lib", le.Name())
		if xmlData, ok := readPluginXMLFromJar(jar); ok {
			return parsePluginXML(xmlData), jar
		}
	}
	return ideaPlugin{}, ""
}

func readPluginXMLFromJar(jarPath string) ([]byte, bool) {
	zr, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, false
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name == "META-INF/plugin.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, false
			}
			data, err := io.ReadAll(io.LimitReader(rc, 1<<20)) // cap at 1 MiB
			_ = rc.Close()                                     // read-only zip entry; close error is non-actionable
			if err != nil {
				return nil, false
			}
			return data, true
		}
	}
	return nil, false
}

func parsePluginXML(data []byte) ideaPlugin {
	var p ideaPlugin
	_ = xml.Unmarshal(data, &p)
	p.ID = strings.TrimSpace(p.ID)
	p.Name = strings.TrimSpace(p.Name)
	p.Version = strings.TrimSpace(p.Version)
	return p
}

// productEditorName turns a JetBrains product config dir like
// "IntelliJIdea2026.1" into a stable editor label ("intellijidea"). JetBrains
// product names are too irregular (IntelliJ, PhpStorm, CLion, GoLand) for clean
// kebab-casing, so we just strip the trailing version and lowercase.
func productEditorName(dir string) string {
	name := dir
	if i := versionSuffixRe.FindStringIndex(name); i != nil {
		name = name[:i[0]]
	}
	return strings.ToLower(strings.TrimSpace(name))
}
