package macoffice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

type ProductType int

const (
	WholeSuite ProductType = iota
	Outlook
	Excel
	PowerPoint
	Word
	OneNote
)

type SecurityUpdate struct {
	Product       ProductType
	Vulnerability string
}

// ReleaseNote contains information about an Office release including security patches.
type ReleaseNote struct {
	Date            time.Time
	Version         string // Ths includes the Build ex: 16.69 (Build 23010700)
	SecurityUpdates []SecurityUpdate
}

func (or *ReleaseNote) AddSecurityUpdate(pt ProductType, vuln string) {
	or.SecurityUpdates = append(or.SecurityUpdates, SecurityUpdate{
		Product:       pt,
		Vulnerability: vuln,
	})
}

// Valid returns true if this release note can be used for vulnerability processing. Some release
// notes don't have a release version nor security updates
func (or *ReleaseNote) Valid() bool {
	return len(or.Version) != 0 && len(or.SecurityUpdates) != 0
}

// CmpVersion compares the release note version against 'otherVer' returning:
// -1 if rel. note version < other version
// 0 if rel. note version == other version
// 1 if rel. note version > other version
func (or *ReleaseNote) CmpVersion(otherVer string) int {
	relVersion := or.Version

	matches := VersionPattern.FindStringSubmatch(or.Version)
	if len(matches) >= 2 {
		relVersion = matches[1]
	}

	return utils.Rpmvercmp(relVersion, otherVer)
}

// CollectVulnerabilities collect all unique vulnerabilities that were patched in this release by matching
// their product type.
func (or *ReleaseNote) CollectVulnerabilities(product ProductType) []string {
	var vulns []string
	collected := make(map[string]struct{})

	for _, su := range or.SecurityUpdates {
		if su.Product == WholeSuite || su.Product == product {
			collected[su.Vulnerability] = struct{}{}
		}
	}

	for k := range collected {
		vulns = append(vulns, k)
	}
	return vulns
}

// OfficeProductFromBundleId looks at the provided 'bundleId' and tries to match the Office Product.
// If no match is found, false is returned as the second return value.
func OfficeProductFromBundleId(bundleId string) (ProductType, bool) {
	b := strings.ToLower(bundleId)
	switch {
	case strings.HasPrefix(b, "com.microsoft.powerpoint"):
		return PowerPoint, true
	case strings.HasPrefix(b, "com.microsoft.word"):
		return Word, true
	case strings.HasPrefix(b, "com.microsoft.excel"):
		return Excel, true
	case strings.HasPrefix(b, "com.microsoft.onenote"):
		return OneNote, true
	case strings.HasPrefix(b, "com.microsoft.outlook"):
		return Outlook, true
	}
	return WholeSuite, false
}

// BuildNumber returns the build number from the release note version.
// "16.69 (Build 23010700)" would return "23010700"
func (or *ReleaseNote) BuildNumber() string {
	matches := BuildNumberPattern.FindStringSubmatch(or.Version)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// ShortVersionFormat returns the version without the build number.
// "16.69 (Build 23010700)" would return "16.69".
// If the version cannot be extracted, an empty string is returned.
func (or *ReleaseNote) ShortVersionFormat() string {
	matches := VersionPattern.FindStringSubmatch(or.Version)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

type ReleaseNotes []ReleaseNote

func (rn ReleaseNotes) Serialize(d time.Time, dir string) error {
	payload, err := json.Marshal(rn)
	if err != nil {
		return err
	}

	fileName := io.MacOfficeRelNotesFileName(d)
	filePath := filepath.Join(dir, fileName)

	return os.WriteFile(filePath, payload, 0o644)
}
