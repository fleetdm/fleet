package msrc

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"

	msrc_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	msrc_xml "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/xml"
)

func parseFeed(feedFilePath string) (map[string]*msrc_parsed.SecurityBulletin, error) {
	r, err := os.Open(feedFilePath)
	if err != nil {
		return nil, fmt.Errorf("msrc parser: %w", err)
	}
	defer r.Close()

	feedResultXML, err := parseXML(r)
	if err != nil {
		return nil, fmt.Errorf("msrc parser: %w", err)
	}

	bulletins, err := mapToSecurityBulletins(feedResultXML)
	if err != nil {
		return nil, fmt.Errorf("msrc parser: %w", err)
	}

	return bulletins, nil
}

func mapToSecurityBulletins(rXML *msrc_xml.FeedResult) (map[string]*msrc_parsed.SecurityBulletin, error) {
	// We will have one bulletin for each product.
	bulletins := make(map[string]*msrc_parsed.SecurityBulletin)
	pIDToPName := make(map[string]string)

	for pID, p := range rXML.WinProducts {
		name := msrc_parsed.NewFullProductName(p.FullName).Name()
		if bulletins[name] == nil {
			bulletins[name] = msrc_parsed.NewSecurityBulletin(name)
		}
		bulletins[name].Products[pID] = p.FullName
		pIDToPName[pID] = name
	}

	for _, v := range rXML.WinVulnerabities {
		for _, rem := range v.Remediations {
			// We will only be able to detect vulns for which they are vendor fixes.
			if !rem.IsVendorFix() {
				continue
			}

			// We assume that rem.Description will contain the ID portion of a KBID, which should
			// be always a numeric value.
			remediatedKBID, err := strconv.Atoi(rem.Description)
			if err != nil {
				return nil, fmt.Errorf("invalid remediation KBID %q for %s", rem.Description, v.CVE)
			}

			// rem.Supercedence should have the ID portion of a KBID which the current vendor fix replaces.
			var supersedes *int
			if rem.Supercedence != "" {
				r, err := strconv.Atoi(rem.Supercedence)
				if err != nil {
					return nil, fmt.Errorf("invalid supercedence KBID %q for %s", rem.Supercedence, v.CVE)
				}
				supersedes = &r
			}

			for _, pID := range rem.ProductIDs {
				// Get the bulletin for the current product ID, skip further processing if is a
				// non-windows product.
				b, ok := bulletins[pIDToPName[pID]]
				if !ok {
					continue
				}

				// Check if the vulnerability referenced by this remediation exists, if not
				// initialize it.
				var vuln msrc_parsed.Vulnerability
				if vuln, ok = b.Vulnerabities[v.CVE]; !ok {
					vuln = msrc_parsed.NewVulnerability(v.PublishedDateEpoch())
				}
				vuln.ProductIDs[pID] = true
				vuln.RemediatedBy[remediatedKBID] = true

				// Check if the vendor fix referenced by this remediation exists, if not
				// initialize it.
				var vFix msrc_parsed.VendorFix
				if vFix, ok = b.VendorFixes[remediatedKBID]; !ok {
					vFix = msrc_parsed.NewVendorFix(rem.FixedBuild)
				}
				vFix.Supersedes = supersedes
				vFix.ProductIDs[pID] = true

				// Update the bulletin
				b.Vulnerabities[v.CVE] = vuln
				b.VendorFixes[remediatedKBID] = vFix
			}
		}
	}

	return bulletins, nil
}

func parseXML(reader io.Reader) (*msrc_xml.FeedResult, error) {
	r := &msrc_xml.FeedResult{
		WinProducts: map[string]msrc_xml.Product{},
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

		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Local == "Branch" {
				branch := msrc_xml.ProductBranch{}
				if err = d.DecodeElement(&branch, &t); err != nil {
					return nil, err
				}

				for _, p := range branch.WinProducts() {
					r.WinProducts[p.ProductID] = p
				}
			}

			if t.Name.Local == "Vulnerability" {
				vuln := msrc_xml.Vulnerability{}
				if err = d.DecodeElement(&vuln, &t); err != nil {
					return nil, err
				}

				for pID := range r.WinProducts {
					// We only care about vulnerabilities that have a vendor fix targeting a Windows
					// product.
					if vuln.IncludesVendorFix(pID) {
						r.WinVulnerabities = append(r.WinVulnerabities, vuln)
						break
					}
				}
			}
		}
	}
}
