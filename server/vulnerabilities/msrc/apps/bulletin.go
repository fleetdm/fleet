package msrcapps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
)

// VersionSecurityUpdate represents a CVE with its fixed build for a specific version.
type VersionSecurityUpdate struct {
	CVE        string `json:"cve"`
	FixedBuild string `json:"fixed_build"`
}

// VersionBulletin contains security data for a specific version branch.
type VersionBulletin struct {
	// Supported indicates if this version is currently supported by Microsoft.
	Supported bool `json:"supported"`
	// SecurityUpdates lists CVEs affecting this version with their fixed builds.
	SecurityUpdates []VersionSecurityUpdate `json:"security_updates"`
}

// AppBulletinFile contains Windows Office vulnerability data indexed by version.
type AppBulletinFile struct {
	// Version is the schema version for this file format.
	Version int `json:"version"`
	// BuildPrefixes maps build prefix to version branch (e.g., "19725" -> "2602").
	BuildPrefixes map[string]string `json:"build_prefixes"`
	// Versions contains security data indexed by version branch.
	Versions map[string]*VersionBulletin `json:"versions"`
}

// SerializeAsWinOffice writes the bulletins to a JSON file using the WinOffice naming.
func (b *AppBulletinFile) SerializeAsWinOffice(d time.Time, dir string) error {
	payload, err := json.Marshal(b)
	if err != nil {
		return err
	}

	fileName := io.WinOfficeFileName(d)
	filePath := filepath.Join(dir, fileName)

	return os.WriteFile(filePath, payload, 0o644)
}
