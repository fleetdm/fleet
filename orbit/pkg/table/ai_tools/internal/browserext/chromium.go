package browserext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// chromiumManifest is the subset of an extension manifest.json we read. Also
// reused for Gecko WebExtension manifests (same shape).
type chromiumManifest struct {
	Name            string          `json:"name"`
	Version         string          `json:"version"`
	DefaultLocale   string          `json:"default_locale"`
	ManifestVer     int             `json:"manifest_version"`
	Permissions     json.RawMessage `json:"permissions"` // MV2 mixes API strings, host strings, and objects
	HostPermissions []string        `json:"host_permissions"`
	ContentScripts  []struct {
		Matches []string `json:"matches"`
	} `json:"content_scripts"`
}

// hostPatterns flattens every host-permission source the manifest exposes.
func (m chromiumManifest) hostPatterns() []string {
	out := append([]string{}, m.HostPermissions...)
	out = append(out, stringList(m.Permissions)...) // MV2 host perms live here
	for _, cs := range m.ContentScripts {
		out = append(out, cs.Matches...)
	}
	return out
}

// stringList extracts only the string elements of a JSON array, ignoring
// objects/numbers (MV2 `permissions` can contain optional-permission objects).
func stringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var arr []any
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	var out []string
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

type prefEntry struct {
	Manifest     chromiumManifest `json:"manifest"`
	FromWebstore *bool            `json:"from_webstore"`
	Location     int              `json:"location"`
	Path         string           `json:"path"`
}

// readChromiumPrefs returns the merged extension settings registry. Secure
// Preferences is authoritative and read first; plain Preferences fills gaps.
func readChromiumPrefs(profileDir string) map[string]prefEntry {
	out := map[string]prefEntry{}
	for _, fn := range []string{"Secure Preferences", "Preferences"} {
		b, err := fsutil.ReadFileBounded(filepath.Join(profileDir, fn))
		if err != nil {
			continue
		}
		var top struct {
			Extensions struct {
				Settings map[string]prefEntry `json:"settings"`
			} `json:"extensions"`
		}
		if json.Unmarshal(b, &top) != nil {
			continue
		}
		for id, e := range top.Extensions.Settings {
			if _, exists := out[id]; !exists {
				out[id] = e
			}
		}
	}
	return out
}

// latestVersionManifest returns the lexically-greatest version subdir and its
// manifest.json path under an extension id dir.
func latestVersionManifest(idDir string) (verDir, manifestPath string, ok bool) {
	entries, err := os.ReadDir(idDir)
	if err != nil {
		return "", "", false
	}
	best := ""
	for _, e := range entries {
		if e.IsDir() && e.Name() > best {
			best = e.Name()
		}
	}
	if best == "" {
		return "", "", false
	}
	mp := filepath.Join(idDir, best, "manifest.json")
	if !fsutil.Exists(mp) {
		return "", "", false
	}
	return best, mp, true
}

// resolveChromiumName resolves a `__MSG_key__` i18n placeholder name via
// _locales/<locale>/messages.json (default_locale, then en / en_US).
func resolveChromiumName(versionDir string, m chromiumManifest) string {
	key, ok := msgKey(m.Name)
	if !ok {
		return m.Name
	}
	for _, loc := range append([]string{m.DefaultLocale}, "en", "en_US") {
		if loc == "" {
			continue
		}
		b, err := fsutil.ReadFileBounded(filepath.Join(versionDir, "_locales", loc, "messages.json"))
		if err != nil {
			continue
		}
		var msgs map[string]struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(b, &msgs) != nil {
			continue
		}
		for k, v := range msgs {
			if strings.EqualFold(k, key) && v.Message != "" {
				return v.Message
			}
		}
	}
	return m.Name
}

// underHome reports whether cand, once cleaned, is contained within the home
// directory. Used to refuse attacker-controlled absolute paths (Chromium
// Preferences "path") that would otherwise escape the scanned home.
func underHome(home, cand string) bool {
	rel, err := filepath.Rel(filepath.Clean(home), filepath.Clean(cand))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func msgKey(name string) (string, bool) {
	if strings.HasPrefix(name, "__MSG_") && strings.HasSuffix(name, "__") {
		return strings.TrimSuffix(strings.TrimPrefix(name, "__MSG_"), "__"), true
	}
	return "", false
}

// collectChromiumProfile enumerates AI extensions in one Chromium profile by
// unioning the on-disk Extensions/ walk with the Preferences registry, then
// classifying (AI-only), hashing, and deriving risk flags.
func collectChromiumProfile(profileDir, browser, profileName string, h homes.Home) []Extension {
	byID := map[string]*Extension{}

	// 1. Disk-walk Extensions/<id>/<version>/manifest.json.
	extRoot := filepath.Join(profileDir, "Extensions")
	if idDirs, err := os.ReadDir(extRoot); err == nil {
		for _, idDir := range idDirs {
			if !idDir.IsDir() {
				continue
			}
			id := idDir.Name()
			verDir, manifestPath, ok := latestVersionManifest(filepath.Join(extRoot, id))
			if !ok {
				continue
			}
			b, err := fsutil.ReadFileBounded(manifestPath)
			if err != nil {
				continue
			}
			var m chromiumManifest
			if json.Unmarshal(b, &m) != nil {
				continue
			}
			byID[id] = &Extension{
				ID:           id,
				Name:         resolveChromiumName(filepath.Join(extRoot, id, verDir), m),
				Version:      firstNonEmpty(m.Version, verDir),
				Path:         manifestPath,
				ManifestVer:  m.ManifestVer,
				HostPerms:    m.hostPatterns(),
				FromWebstore: -1,
				SignedState:  signedStateUnknown,
			}
		}
	}

	// 2. Preferences cross-ref: provenance + recover unpacked/Preferences-only ids.
	for id, pe := range readChromiumPrefs(profileDir) {
		ext := byID[id]
		if ext == nil {
			if pe.Manifest.Name == "" && pe.Path == "" {
				continue
			}
			// pe.Path comes from the user-writable Preferences file and may be an
			// absolute path anywhere on disk. Contain it to the owning home so the
			// root scanner cannot be pointed at an arbitrary file to hash (below).
			mp := ""
			if pe.Path != "" {
				if cand := filepath.Join(pe.Path, "manifest.json"); underHome(h.Dir, cand) {
					mp = cand
				}
			}
			ext = &Extension{
				ID:           id,
				Name:         pe.Manifest.Name,
				Version:      pe.Manifest.Version,
				Path:         mp,
				ManifestVer:  pe.Manifest.ManifestVer,
				HostPerms:    pe.Manifest.hostPatterns(),
				FromWebstore: -1,
				SignedState:  signedStateUnknown,
			}
			byID[id] = ext
		}
		if pe.FromWebstore != nil {
			ext.FromWebstore = boolToInt(*pe.FromWebstore)
		}
		ext.Sideloaded = chromiumSideloaded(ext.FromWebstore, pe.Location)
		if len(ext.HostPerms) == 0 {
			ext.HostPerms = pe.Manifest.hostPatterns()
		}
		ext.Name = firstNonEmpty(ext.Name, pe.Manifest.Name)
		ext.Version = firstNonEmpty(ext.Version, pe.Manifest.Version)
	}

	// 3. Classify (AI-only), finalize, sort by id for deterministic output.
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []Extension
	for _, id := range ids {
		ext, ok := byID[id]
		if !ok || ext == nil {
			continue
		}
		isAI, cat := classify.BrowserExtension(id, ext.Name)
		if !isAI {
			continue
		}
		ext.Browser, ext.Engine, ext.Profile, ext.Scope = browser, "chromium", profileName, "user"
		ext.Category = cat
		ext.UID, ext.Username = h.UID, h.Username
		ext.SHA256 = fsutil.SHA256(ext.Path)
		ext.computeRisk()
		out = append(out, *ext)
	}
	return out
}
