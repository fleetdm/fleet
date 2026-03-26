package msrcapps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
)

// SecurityUpdate represents a CVE that was fixed in a specific version.
type SecurityUpdate struct {
	CVE          string `json:"cve"`
	FixedVersion string `json:"fixed_version"`
}

// AppBulletin contains vulnerability information for a Microsoft application.
type AppBulletin struct {
	// ProductID is the product identifier (e.g., version branch like "2602").
	ProductID string `json:"product_id"`
	// Product is the product name (e.g., "Microsoft 365 Apps (Version 2602)").
	Product string `json:"product"`
	// SecurityUpdates is the list of CVEs with their fixed versions.
	SecurityUpdates []SecurityUpdate `json:"security_updates"`
}

// MatchCriteria defines patterns for matching software.
// Multiple values in an array are OR-matched (any can match).
// Multiple fields are AND-matched (all must match).
type MatchCriteria struct {
	// Name patterns to match against software name (OR-matched).
	Name []string `json:"name"`
	// Vendor patterns to match against software vendor (OR-matched).
	Vendor []string `json:"vendor"`
}

// ProductMapping represents a single software-to-product mapping rule.
type ProductMapping struct {
	Match     MatchCriteria `json:"match"`
	ProductID string        `json:"product_id"`
}

// AppBulletinFile contains all app bulletins in a single file.
type AppBulletinFile struct {
	// Mappings define rules for matching software to product IDs.
	Mappings []ProductMapping `json:"mappings,omitempty"`
	Products []AppBulletin    `json:"products"`
}

// WithMappings returns a copy of the bulletin file with the given mappings.
func (b *AppBulletinFile) WithMappings(mappings []ProductMapping) *AppBulletinFile {
	return &AppBulletinFile{
		Mappings: mappings,
		Products: b.Products,
	}
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

// DefaultMappings returns the default product mappings for Windows Office.
func DefaultMappings() []ProductMapping {
	return []ProductMapping{
		{
			Match: MatchCriteria{
				Name:   []string{"Microsoft 365", "Microsoft Office"},
				Vendor: []string{"Microsoft Corporation"},
			},
			ProductID: "winoffice",
		},
	}
}
