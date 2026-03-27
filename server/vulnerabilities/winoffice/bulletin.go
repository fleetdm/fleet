package winoffice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
)

// SecurityUpdate represents a CVE with its resolved version for a specific version branch.
type SecurityUpdate struct {
	CVE string `json:"cve"`
	// ResolvedInVersion is the build version that fixes this CVE (e.g., "16.0.19725.20172").
	ResolvedInVersion string `json:"resolved_in_version"`
}

// VersionBulletin contains security data for a specific version branch.
type VersionBulletin struct {
	SecurityUpdates []SecurityUpdate `json:"security_updates"`
}

// BulletinFile contains Windows Office vulnerability data indexed by version.
type BulletinFile struct {
	// Version is the schema version for this file format.
	Version int `json:"version"`
	// BuildPrefixes maps build prefix to version branch (e.g., "19725" -> "2602").
	BuildPrefixes map[string]string `json:"build_prefixes"`
	// Versions contains security data indexed by version branch.
	Versions map[string]*VersionBulletin `json:"versions"`
}

// Serialize writes the bulletin to a JSON file.
func (b *BulletinFile) Serialize(d time.Time, dir string) error {
	payload, err := json.Marshal(b)
	if err != nil {
		return err
	}

	fileName := io.WinOfficeFileName(d)
	filePath := filepath.Join(dir, fileName)

	return os.WriteFile(filePath, payload, 0o644)
}
