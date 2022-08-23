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
	// We will have one bulletin for each product name.
	bulletins := make(map[string]*msrc_parsed.SecurityBulletin)

	// Create each bulletin and populate its Products
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

			for _, remPID := range rem.ProductIDs {
				p := rXML.WinProducts[remPID]

				name := NameFromFullProdName(p.FullName)
				g, ok := bulletins[name]

				// Skip any non-windows products
				if !ok {
					continue
				}

				//------------
				// Create/update the vulnerability related to this vendor fix
				// for the current product bulletin
				var vuln msrc_parsed.Vulnerability
				if vuln, ok = g.Vulnerabities[v.CVE]; !ok {
					vuln = msrc_parsed.Vulnerability{
						PublishedEpoch:  v.PublishedDateEpoch(),
						ProductIDsSet:   make(map[string]bool),
						RemediatedBySet: make(map[int]bool),
					}
				}
				vuln.ProductIDsSet[remPID] = true
				remediatedKBID, err := strconv.Atoi(rem.Description)
				if err != nil {
					return nil, fmt.Errorf("invalid remediation KBID %s for %s in %s", rem.Description, name, v.CVE)
				}
				vuln.RemediatedBySet[remediatedKBID] = true
				g.Vulnerabities[v.CVE] = vuln

				//---------
				// Create/update the vendor fix
				var vFix msrc_parsed.VendorFix
				if vFix, ok = g.VendorFixes[rem.Description]; !ok {
					vFix = msrc_parsed.VendorFix{FixedBuild: rem.FixedBuild}
				}

				if rem.Supercedence != "" {
					supercedenceKBID, err := strconv.Atoi(rem.Supercedence)
					if err != nil {
						return nil, fmt.Errorf("invalid supercedence KBID %s for %s in %s", rem.Supercedence, name, v.CVE)
					}
					vFix.Supersedes = msrc_parsed.NewVendorFixNodeRef(supercedenceKBID)
				}

				vFix.TargetProductsIDs = append(vFix.TargetProductsIDs, remPID)
				g.VendorFixes[rem.Description] = vFix
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
