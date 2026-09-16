package browserext

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

type geckoProfile struct {
	name string
	path string
}

type geckoAddon struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Version        string `json:"version"`
	Location       string `json:"location"`
	SignedState    *int   `json:"signedState"`
	ForeignInstall bool   `json:"foreignInstall"`
	DefaultLocale  struct {
		Name string `json:"name"`
	} `json:"defaultLocale"`
	UserPermissions struct {
		Origins []string `json:"origins"`
	} `json:"userPermissions"`
}

// collectGeckoProfile enumerates AI extensions in one Gecko profile from its
// extensions.json registry, hashing the matching .xpi.
func collectGeckoProfile(profilePath, browser, profileName string, h homes.Home) []Extension {
	b, err := fsutil.ReadFileBounded(filepath.Join(profilePath, "extensions.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Addons []geckoAddon `json:"addons"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return nil
	}
	var out []Extension
	for _, a := range doc.Addons {
		if a.Type != "extension" {
			continue // skip themes, dictionaries, locales
		}
		if a.Location != "" && a.Location != "app-profile" {
			continue // skip system/builtin addons
		}
		isAI, cat := classify.BrowserExtension(a.ID, a.DefaultLocale.Name)
		if !isAI {
			continue
		}
		// a.ID is read from the user-writable extensions.json; a value containing
		// path separators or ".." would escape the profile's extensions dir when
		// joined below (and then be hashed as root). Reject those.
		if strings.ContainsAny(a.ID, `/\`) || strings.Contains(a.ID, "..") {
			continue
		}
		xpi := filepath.Join(profilePath, "extensions", a.ID+".xpi")
		signed := signedStateUnknown
		if a.SignedState != nil {
			signed = *a.SignedState
		}
		hostPerms := a.UserPermissions.Origins
		if len(hostPerms) == 0 {
			hostPerms = hostPermsFromXPI(xpi)
		}
		e := Extension{
			UID:          h.UID,
			Username:     h.Username,
			Browser:      browser,
			Engine:       "gecko",
			Profile:      profileName,
			ID:           a.ID,
			Name:         firstNonEmpty(a.DefaultLocale.Name, a.ID),
			Version:      a.Version,
			Path:         xpi,
			Category:     cat,
			Scope:        "user",
			HostPerms:    hostPerms,
			FromWebstore: -1,
			SignedState:  signed,
			Sideloaded:   geckoSideloaded(signed, a.ForeignInstall),
			SHA256:       fsutil.SHA256(xpi),
		}
		e.computeRisk()
		out = append(out, e)
	}
	return out
}

// hostPermsFromXPI reads the WebExtension manifest.json from inside an .xpi zip
// when the registry's userPermissions.origins is empty. Bounded 1 MiB read,
// mirroring ide/jetbrains.go's jar reader.
func hostPermsFromXPI(xpiPath string) []string {
	// Open through fsutil.OpenRegular so a .xpi planted as a symlink or special
	// file cannot steer the root scan at another target or block it.
	xf, err := fsutil.OpenRegular(xpiPath)
	if err != nil {
		return nil
	}
	defer func() { _ = xf.Close() }()
	fi, err := xf.Stat()
	if err != nil {
		return nil
	}
	zr, err := zip.NewReader(xf, fi.Size())
	if err != nil {
		return nil
	}
	for _, f := range zr.File {
		if f.Name != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		_ = rc.Close()
		if err != nil {
			return nil
		}
		var m chromiumManifest // WebExtension manifest, same shape
		if json.Unmarshal(data, &m) != nil {
			return nil
		}
		return m.hostPatterns()
	}
	return nil
}

// geckoProfiles returns the profiles for one Gecko root, parsing profiles.ini
// and falling back to globbing when it is absent or yields nothing.
func geckoProfiles(root, home string) []geckoProfile {
	b, err := fsutil.ReadFileBounded(filepath.Join(root, "profiles.ini"))
	if err != nil {
		return globGeckoProfiles(root)
	}
	var out []geckoProfile
	var curPath string
	var curRel, inProfile bool
	flush := func() {
		if inProfile && curPath != "" {
			p := curPath
			if curRel {
				p = filepath.Join(root, filepath.FromSlash(curPath))
			}
			// profiles.ini is user-writable; a relative Path containing ".." or an
			// absolute Path could point outside the user's home. Contain it so the
			// root scanner is not steered at an arbitrary location.
			if !underHome(home, p) {
				return
			}
			out = append(out, geckoProfile{name: filepath.Base(p), path: p})
		}
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			flush()
			curPath, curRel, inProfile = "", true, strings.HasPrefix(strings.ToLower(line), "[profile")
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "Path":
			curPath = strings.TrimSpace(v)
		case "IsRelative":
			curRel = strings.TrimSpace(v) != "0"
		}
	}
	flush()
	if len(out) == 0 {
		return globGeckoProfiles(root)
	}
	return out
}

func globGeckoProfiles(root string) []geckoProfile {
	parent := filepath.Join(root, "Profiles")
	entries, err := os.ReadDir(parent)
	if err != nil {
		parent = root // Linux layout: profiles directly under root
		if entries, err = os.ReadDir(parent); err != nil {
			return nil
		}
	}
	var out []geckoProfile
	for _, e := range entries {
		if e.IsDir() && fsutil.Exists(filepath.Join(parent, e.Name(), "extensions.json")) {
			out = append(out, geckoProfile{name: e.Name(), path: filepath.Join(parent, e.Name())})
		}
	}
	return out
}
