package apps

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
)

// This file holds everything about MSIX/Appx detection that is not a Windows
// path: the package-name parsing and the directory scan itself. It is
// platform-neutral (rather than living in appx_windows.go) so the scan can be
// driven against a temporary directory in tests on any platform; only the
// Windows build calls it, and only appx_windows.go knows where the real
// directories are.

// appxPackage is the identity encoded in an MSIX/Appx package full name.
type appxPackage struct {
	Name        string // identity name, e.g. "OpenAI.ChatGPT-Desktop"
	Version     string // e.g. "1.2024.30.0"
	ResourceID  string // empty for the app itself; "split.*" for satellite packages
	PublisherID string // 13-character publisher hash, e.g. "2p2nqsd0c76g0"
}

// FamilyName returns the package family name, "<Name>_<PublisherID>". That is
// how a package's per-user data directory under %LOCALAPPDATA%\Packages is
// named, which is what attributes an install to a specific user.
func (p appxPackage) FamilyName() string {
	if p.Name == "" || p.PublisherID == "" {
		return ""
	}
	return p.Name + "_" + p.PublisherID
}

// isResourcePackage reports whether p's name marks it as a satellite resource
// package (a scale or language split) rather than the app itself. Those repeat
// the app's identity but point at a different install folder, so reporting one
// would attribute the wrong path to the app.
//
// This is the name-only test, and it is not conclusive: the "split." prefix is
// the modern convention, but older resource packages use a bare resource id
// ("scale-100"). The manifest's Properties/ResourcePackage is authoritative and
// is checked in scanAppxDirs once a candidate's manifest has been read; this
// stays as the cheap pre-filter that avoids reading those manifests at all, and
// as the fallback when a manifest cannot be read.
func (p appxPackage) isResourcePackage() bool {
	return strings.HasPrefix(p.ResourceID, "split.")
}

// parsePackageFullName splits an MSIX package full name into its identity
// fields. The format is Name_Version_Architecture_ResourceId_PublisherId, and
// none of those fields may itself contain an underscore, so splitting on "_" is
// exact. Anything that does not parse is rejected rather than reported as a row.
//
// The package full name is the source of truth for identity and version because
// the registry's DisplayName is usually an unresolved MUI indirect string (see
// appxLiteral).
func parsePackageFullName(pfn string) (appxPackage, bool) {
	parts := strings.Split(pfn, "_")
	if len(parts) != 5 {
		return appxPackage{}, false
	}
	name, version, resourceID, publisherID := parts[0], parts[1], parts[3], parts[4]
	if name == "" || publisherID == "" || !isDottedNumber(version) {
		return appxPackage{}, false
	}
	return appxPackage{Name: name, Version: version, ResourceID: resourceID, PublisherID: publisherID}, true
}

// appxInstallDirName is the packaged-app install directory, under Program Files.
// There is exactly one of them per host: unlike Win32 installers, packaged apps
// are not subject to the "Program Files (x86)" split, and every architecture
// stages here with x86/x64/arm64/neutral distinguished inside the package full
// name instead.
const appxInstallDirName = "WindowsApps"

// appxInstallRootFrom resolves the packaged-app install root from the Windows
// environment. It is split out from its Windows-only caller so the redirection
// rule below is testable.
//
// %ProgramFiles% is redirected by process bitness: a 32-bit process on 64-bit
// Windows sees "C:\Program Files (x86)", which holds no WindowsApps directory,
// so trusting it would silently scan the wrong place and report nothing.
// %ProgramW6432% is never redirected and always names the real 64-bit Program
// Files; it is unset only on 32-bit Windows, where %ProgramFiles% is already
// right.
func appxInstallRootFrom(programW6432, programFiles, systemDrive string) string {
	base := programW6432
	if base == "" {
		base = programFiles
	}
	if base == "" {
		if systemDrive == "" {
			systemDrive = "C:"
		}
		base = filepath.Join(systemDrive+`\`, "Program Files")
	}
	return filepath.Join(base, appxInstallDirName)
}

// appxManifest is the subset of AppxManifest.xml this collector reads. Elements
// are matched by local name, so the manifest's XML namespace does not matter.
type appxManifest struct {
	Identity struct {
		Publisher string `xml:"Publisher,attr"`
	} `xml:"Identity"`
	Properties struct {
		DisplayName          string `xml:"DisplayName"`
		PublisherDisplayName string `xml:"PublisherDisplayName"`
		// ResourcePackage is the authoritative marker for a satellite package,
		// covering the ones whose resource id lacks the "split." prefix.
		ResourcePackage bool `xml:"ResourcePackage"`
	} `xml:"Properties"`
}

// scanAppxDirs adds the AI apps installed as MSIX/Appx packages to c.
//
// installRoot is the only source of rows: it holds one directory per staged
// package, named by package full name and sitting alongside the package
// manifest, which together give identity, version, publisher, and install path.
//
// userPkgDirs are the per-user package-state directories, named by package
// family name. They are read only to decide whether an install belongs to a user
// or to the machine — never to report an app. That directory records that a
// package once ran for a user, not that it is installed now: Windows leaves it
// behind after an uninstall, so a row sourced from it would assert an install
// with no version and no path to corroborate it, and would never expire. On an
// inventory table a missing row is cheaper than a phantom one.
func scanAppxDirs(c *appCollector, installRoot string, userPkgDirs []string) {
	userFamilies := appxUserFamilies(userPkgDirs)

	for _, name := range readDirNames(installRoot) {
		pkg, ok := parsePackageFullName(name)
		if !ok || pkg.isResourcePackage() {
			continue
		}
		// Match on the identity name before touching the package directory. A host
		// carries several hundred packages and at most a couple are AI apps, so
		// reading manifests first would mean hundreds of file reads per query for
		// data that is then discarded.
		if !c.wants(pkg.Name) {
			continue
		}
		dir := filepath.Join(installRoot, name)
		// Best effort: the row must not depend on the manifest being readable.
		man := readAppxManifest(dir)
		// The authoritative satellite check, which catches the resource packages
		// whose id lacks the "split." prefix that isResourcePackage looks for.
		// Safe to skip here because wants only reads the collector.
		if man.Properties.ResourcePackage {
			continue
		}

		scope := "system"
		if _, isUser := userFamilies[pkg.FamilyName()]; isUser {
			scope = "user"
		}
		c.add(appCandidate{
			MatchTokens: []string{pkg.Name, appxLiteral(man.Properties.DisplayName)},
			DisplayName: firstNonEmpty(appxLiteral(man.Properties.DisplayName), pkg.Name),
			Vendor:      appxVendor(man.Properties.PublisherDisplayName, man.Identity.Publisher),
			Version:     pkg.Version,
			Path:        dir,
			Scope:       scope,
			Source:      "appx",
		})
	}
}

// appxUserFamilies returns the set of package family names that have per-user
// state in any of dirs.
func appxUserFamilies(dirs []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, dir := range dirs {
		for _, name := range readDirNames(dir) {
			out[name] = struct{}{}
		}
	}
	return out
}

// readDirNames returns the names of the subdirectories of dir, or nil when dir
// cannot be read. Every caller treats an unreadable directory as "nothing here".
func readDirNames(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// readAppxManifest reads a package's manifest for its display and publisher
// names. A missing or malformed manifest yields a zero value rather than an
// error: the package full name already carries identity and version, so the row
// is still correct without it.
func readAppxManifest(dir string) appxManifest {
	var man appxManifest
	b, err := fsutil.ReadFileBounded(filepath.Join(dir, "AppxManifest.xml"))
	if err != nil {
		return man
	}
	_ = xml.Unmarshal(b, &man)
	return man
}

// isDottedNumber reports whether s looks like an Appx version quad: digits and
// dots, with at least one digit.
func isDottedNumber(s string) bool {
	digit := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digit = true
		case s[i] == '.':
		default:
			return false
		}
	}
	return digit
}

// appxLiteral returns s unless it is a MUI indirect string of the form
// "@{PackageFullName?ms-resource://.../SomeName}". Those resolve only through
// SHLoadIndirectString; the raw form is worse than nothing for token matching
// (it embeds the package name, which would match against itself), so it is
// reported as absent.
func appxLiteral(s string) string {
	if strings.HasPrefix(s, "@") {
		return ""
	}
	return s
}

// appxVendor picks the best available publisher name: the package's
// PublisherDisplayName when it is a literal string, otherwise the common name of
// the signing certificate's distinguished name. Store packages very often carry
// an indirect PublisherDisplayName, so without the fallback most rows would have
// no vendor at all.
func appxVendor(publisherDisplayName, publisherDN string) string {
	if v := appxLiteral(publisherDisplayName); v != "" {
		return v
	}
	return distinguishedNameCN(publisherDN)
}

// distinguishedNameCN extracts the CN value from a certificate distinguished
// name such as "CN=Element Labs, Inc., O=Element Labs, C=US".
//
// Common names routinely contain commas ("..., Inc."), so a value ends only at a
// comma that starts the next "Type=" attribute — not at the first comma.
func distinguishedNameCN(dn string) string {
	for i := 0; i < len(dn); {
		j := i
		for j < len(dn) && isAlpha(dn[j]) {
			j++
		}
		if j == i || j >= len(dn) || dn[j] != '=' {
			return "" // not a well-formed attribute; give up rather than guess
		}
		valStart := j + 1
		end := dnAttributeEnd(dn, valStart)
		if strings.EqualFold(dn[i:j], "CN") {
			return strings.TrimSpace(dn[valStart:end])
		}
		i = end
		if i < len(dn) && dn[i] == ',' {
			i++
		}
		for i < len(dn) && dn[i] == ' ' {
			i++
		}
	}
	return ""
}

// dnAttributeEnd returns the index at which the attribute value starting at from
// ends: the first comma that is followed by another "Type=" pair, or the end of
// the string.
func dnAttributeEnd(dn string, from int) int {
	for i := from; i < len(dn); i++ {
		if dn[i] != ',' {
			continue
		}
		j := i + 1
		for j < len(dn) && dn[j] == ' ' {
			j++
		}
		k := j
		for k < len(dn) && isAlpha(dn[k]) {
			k++
		}
		if k > j && k < len(dn) && dn[k] == '=' {
			return i
		}
	}
	return len(dn)
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
