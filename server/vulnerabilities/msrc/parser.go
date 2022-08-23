package msrc

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"

	msrc_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/input"
	msrc_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
)

func parseMSRC(inputFile string, outputFile string) error {
	r, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("oval parser: %w", err)
	}
	defer r.Close()

	parseMSRCXMLFeed(r)

	// if err != nil {
	// 	return fmt.Errorf("msrc parser: %w", err)
	// }

	// err = ioutil.WriteFile(outputFile, payload, 0o644)
	// if err != nil {
	// 	return fmt.Errorf("msrc parser: %w", err)
	// }

	return nil
}

func mapToSecurityBulletins(rXML *msrc_input.ResultXML) (map[string]*msrc_parsed.SecurityBulletin, error) {
	// We will have one bulletin for each product.
	bulletins := make(map[string]*msrc_parsed.SecurityBulletin)

	for pID, p := range rXML.WinProducts {
		name := NameFromFullProdName(p.FullName)
		if bulletins[name] == nil {
			bulletins[name] = msrc_parsed.NewSecurityBulletin(name)
		}
		bulletins[name].Products[pID] = p.FullName
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
				name := NameFromFullProdName(rXML.WinProducts[pID].FullName)
				b, ok := bulletins[name]
				if !ok {
					continue
				}

				// Check if the vulnerability referenced by this remediation exists, if not
				// initialize it.
				var vuln msrc_parsed.Vulnerability
				if vuln, ok = b.Vulnerabities[v.CVE]; !ok {
					vuln = msrc_parsed.NewVulnerability(v.PublishedDateEpoch())
				}
				vuln.ProductIDsSet[pID] = true
				vuln.RemediatedBySet[remediatedKBID] = true

				// Check if the vendor fix referenced by this remediation exists, if not
				// initialize it.
				var vFix msrc_parsed.VendorFix
				if vFix, ok = b.VendorFixes[remediatedKBID]; !ok {
					vFix = msrc_parsed.VendorFix{
						FixedBuild:        rem.FixedBuild,
						TargetProductsIDs: make(map[string]bool),
					}
				}
				vFix.Supersedes = supersedes
				vFix.TargetProductsIDs[pID] = true

				// Update the bulletin
				b.Vulnerabities[v.CVE] = vuln
				b.VendorFixes[remediatedKBID] = vFix
			}
		}
	}

	return bulletins, nil
}

func parseMSRCXMLFeed(reader io.Reader) (*msrc_input.ResultXML, error) {
	r := &msrc_input.ResultXML{
		WinProducts: map[string]msrc_input.ProductXML{},
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
				branch := msrc_input.ProductBranchXML{}
				if err = d.DecodeElement(&branch, &t); err != nil {
					return nil, err
				}

				for _, p := range branch.WinProducts() {
					r.WinProducts[p.ProductID] = p
				}
			}

			if t.Name.Local == "Vulnerability" {
				vuln := msrc_input.VulnerabilityXML{}
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
