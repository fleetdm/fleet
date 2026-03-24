package msrc

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	msrcapps "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/apps"
	msrcxml "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/xml"
)

// ParseAppFeed parses an MSRC XML feed and extracts vulnerabilities for Microsoft applications.
func ParseAppFeed(fPath string) (map[string]*msrcapps.AppBulletin, error) {
	r, err := os.Open(fPath)
	if err != nil {
		return nil, fmt.Errorf("msrc app parser: %w", err)
	}
	defer r.Close()

	feedResult, err := parseAppXML(r)
	if err != nil {
		return nil, fmt.Errorf("msrc app parser: %w", err)
	}

	bulletins, err := mapToAppBulletins(feedResult)
	if err != nil {
		return nil, fmt.Errorf("msrc app parser: %w", err)
	}

	return bulletins, nil
}

type appFeedResult struct {
	AppProducts        map[string]msrcxml.Product
	AppVulnerabilities []msrcxml.Vulnerability
}

// isAppVendorFix checks if a remediation is a vendor fix for an app product.
// Unlike OS vendor fixes which use KB-based URLs, app vendor fixes just need
// to have Type "Vendor Fix" and a FixedBuild version.
// We also exclude entries where FixedBuild is a URL (e.g., Office products
// that link to release notes instead of providing a version).
func isAppVendorFix(rem *msrcxml.VulnerabilityRemediation) bool {
	if rem.Type != "Vendor Fix" || rem.FixedBuild == "" {
		return false
	}
	// Skip URL-based "versions" (e.g., Office products)
	if strings.HasPrefix(rem.FixedBuild, "http") {
		return false
	}
	return true
}

// includesAppVendorFix checks if the vulnerability has an app vendor fix for the given product ID.
func includesAppVendorFix(v *msrcxml.Vulnerability, pID string) bool {
	for _, rem := range v.Remediations {
		if isAppVendorFix(&rem) {
			for _, vfPID := range rem.ProductIDs {
				if vfPID == pID {
					return true
				}
			}
		}
	}
	return false
}

func mapToAppBulletins(feed *appFeedResult) (map[string]*msrcapps.AppBulletin, error) {
	// Map product ID to product name for lookup
	pIDToName := make(map[string]string, len(feed.AppProducts))
	bulletins := make(map[string]*msrcapps.AppBulletin)

	for pID, p := range feed.AppProducts {
		// Use the full product name as the key
		name := p.FullName
		pIDToName[pID] = name

		if bulletins[name] == nil {
			bulletins[name] = &msrcapps.AppBulletin{
				ProductID:       pID,
				Product:         name,
				SecurityUpdates: []msrcapps.SecurityUpdate{},
			}
		}
	}

	// Process vulnerabilities
	for _, v := range feed.AppVulnerabilities {
		for _, rem := range v.Remediations {
			if !isAppVendorFix(&rem) {
				continue
			}

			for _, pID := range rem.ProductIDs {
				productName, ok := pIDToName[pID]
				if !ok {
					continue
				}

				bulletin := bulletins[productName]
				if bulletin == nil {
					continue
				}

				// Add the security update
				bulletin.SecurityUpdates = append(bulletin.SecurityUpdates, msrcapps.SecurityUpdate{
					CVE:          v.CVE,
					FixedVersion: rem.FixedBuild,
				})
			}
		}
	}

	return bulletins, nil
}

func parseAppXML(reader io.Reader) (*appFeedResult, error) {
	r := &appFeedResult{
		AppProducts: make(map[string]msrcxml.Product),
	}
	d := xml.NewDecoder(reader)

	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return r, nil
			}
			return nil, fmt.Errorf("decoding token: %v", err)
		}

		if t, ok := t.(xml.StartElement); ok {
			if t.Name.Local == "Branch" {
				branch := msrcxml.ProductBranch{}
				if err = d.DecodeElement(&branch, &t); err != nil {
					return nil, err
				}

				for _, p := range branch.AppProducts() {
					r.AppProducts[p.ProductID] = p
				}
			}

			if t.Name.Local == "Vulnerability" {
				vuln := msrcxml.Vulnerability{}
				if err = d.DecodeElement(&vuln, &t); err != nil {
					return nil, err
				}

				// Check if this vulnerability affects any of our app products
				for pID := range r.AppProducts {
					if includesAppVendorFix(&vuln, pID) {
						r.AppVulnerabilities = append(r.AppVulnerabilities, vuln)
						break
					}
				}
			}
		}
	}
}
