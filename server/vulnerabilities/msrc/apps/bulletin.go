package msrcapps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	// ProductID is the MSRC product identifier (e.g., "11655" for Edge).
	ProductID string `json:"product_id"`
	// Product is the MSRC product name (e.g., "Microsoft Edge (Chromium-based)").
	Product string `json:"product"`
	// SecurityUpdates is the list of CVEs with their fixed versions.
	SecurityUpdates []SecurityUpdate `json:"security_updates"`
}

// MatchCriteria defines patterns for matching software.
// Multiple values in an array are OR-matched (any can match).
// Multiple fields are AND-matched (all must match).
type MatchCriteria struct {
	// Name patterns to match against software name (OR-matched).
	// Supports regex patterns enclosed in "/" delimiters (e.g., "/^Microsoft Edge.*/").
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
	// If empty, default mappings are used.
	Mappings []ProductMapping `json:"mappings,omitempty"`
	Products []AppBulletin    `json:"products"`
}

// Serialize writes the bulletins to a JSON file in the specified directory.
func (b *AppBulletinFile) Serialize(d time.Time, dir string) error {
	payload, err := json.Marshal(b)
	if err != nil {
		return err
	}

	fileName := io.MSRCAppFileName(d)
	filePath := filepath.Join(dir, fileName)

	return os.WriteFile(filePath, payload, 0o644)
}

// LoadAppBulletinFile reads and parses an app bulletin file from the given path.
func LoadAppBulletinFile(filePath string) (*AppBulletinFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var bulletinFile AppBulletinFile
	if err := json.Unmarshal(data, &bulletinFile); err != nil {
		return nil, err
	}

	return &bulletinFile, nil
}

// ToMap converts the bulletin file to a map keyed by product ID.
func (b *AppBulletinFile) ToMap() map[string]AppBulletin {
	result := make(map[string]AppBulletin, len(b.Products))
	for _, p := range b.Products {
		result[p.ProductID] = p
	}
	return result
}

// FromMap creates an AppBulletinFile from a map of bulletins.
func FromMap(bulletins map[string]*AppBulletin) *AppBulletinFile {
	var products []AppBulletin
	for _, b := range bulletins {
		if b != nil && len(b.SecurityUpdates) > 0 {
			products = append(products, *b)
		}
	}

	// Sort for deterministic output
	sort.Slice(products, func(i, j int) bool {
		return products[i].Product < products[j].Product
	})

	return &AppBulletinFile{Products: products}
}

// WithMappings returns a copy of the bulletin file with the given mappings.
func (b *AppBulletinFile) WithMappings(mappings []ProductMapping) *AppBulletinFile {
	return &AppBulletinFile{
		Mappings: mappings,
		Products: b.Products,
	}
}

// NormalizeProductName normalizes a product name for consistent matching.
// This handles variations in how products appear in the MSRC feed.
func NormalizeProductName(name string) string {
	// Common normalizations
	name = strings.TrimSpace(name)
	return name
}

// MergeBulletins merges security updates from src into dst, avoiding duplicates.
func MergeBulletins(dst, src *AppBulletin) {
	if dst == nil || src == nil {
		return
	}

	existing := make(map[string]bool)
	for _, su := range dst.SecurityUpdates {
		key := fmt.Sprintf("%s:%s", su.CVE, su.FixedVersion)
		existing[key] = true
	}

	for _, su := range src.SecurityUpdates {
		key := fmt.Sprintf("%s:%s", su.CVE, su.FixedVersion)
		if !existing[key] {
			dst.SecurityUpdates = append(dst.SecurityUpdates, su)
			existing[key] = true
		}
	}
}
