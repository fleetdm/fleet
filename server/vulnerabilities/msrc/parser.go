package msrc

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	msrcxml "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/xml"
)

func ParseFeed(fPath string) (map[string]*parsed.SecurityBulletin, error) {
	r, err := os.Open(fPath)
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

func mapToSecurityBulletins(rXML *msrcxml.FeedResult) (map[string]*parsed.SecurityBulletin, error) {
	// We will have one bulletin for each product name.
	bulletins := make(map[string]*parsed.SecurityBulletin)
	pIDToPName := make(map[string]string, len(rXML.WinProducts))

	for pID, p := range rXML.WinProducts {
		name := parsed.NewProductFromFullName(p.FullName).Name()
		// If the name could not be determined means that we have an un-supported Windows product
		if name == "" {
			continue
		}

		if bulletins[name] == nil {
			bulletins[name] = parsed.NewSecurityBulletin(name)
		}
		bulletins[name].Products[pID] = parsed.NewProductFromFullName(p.FullName)
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
			remediatedKBIDRaw, err := strconv.Atoi(rem.Description)
			if err != nil {
				return nil, fmt.Errorf("invalid remediation KBID %q for %s", rem.Description, v.CVE)
			}
			if remediatedKBIDRaw < 0 {
				return nil, fmt.Errorf("invalid remediation KBID %q for %s", rem.Description, v.CVE)
			}
			remediatedKBID := uint(remediatedKBIDRaw)

			// rem.Supercedence should have the ID portion of a KBID which the current vendor fix replaces.
			var supersedes *uint
			if rem.Supercedence != "" {
				r, err := strconv.Atoi(rem.Supercedence)
				if err != nil {
					return nil, fmt.Errorf("invalid supercedence KBID %q for %s", rem.Supercedence, v.CVE)
				}
				if r < 0 {
					return nil, fmt.Errorf("invalid supercedence KBID %q for %s", rem.Supercedence, v.CVE)
				}
				supersedes = ptr.Uint(uint(r))
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
				var vuln parsed.Vulnerability
				if vuln, ok = b.Vulnerabities[v.CVE]; !ok {
					vuln = parsed.NewVulnerability(v.PublishedDateEpoch())
				}
				// At this point we know that the remediation is a vendor fix that targets a windows
				// product, so we add the remediation's product ID to the vulnerability's targeted products.
				vuln.ProductIDs[pID] = true
				vuln.RemediatedBy[remediatedKBID] = true

				// Check if the vendor fix referenced by this remediation exists, if not
				// initialize it.
				var vFix parsed.VendorFix
				if vFix, ok = b.VendorFixes[remediatedKBID]; !ok {
					vFix = parsed.NewVendorFix(rem.FixedBuild)
				} else {
					vFix.AddFixedBuild(rem.FixedBuild)
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

func parseXML(reader io.Reader) (*msrcxml.FeedResult, error) {
	r := &msrcxml.FeedResult{
		WinProducts: map[string]msrcxml.Product{},
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

		switch t := t.(type) { //nolint:gocritic // ignore singleCaseSwitch
		case xml.StartElement:
			if t.Name.Local == "Branch" {
				branch := msrcxml.ProductBranch{}
				if err = d.DecodeElement(&branch, &t); err != nil {
					return nil, err
				}

				for _, p := range branch.WinProducts() {
					r.WinProducts[p.ProductID] = p
				}
			}

			if t.Name.Local == "Vulnerability" {
				vuln := msrcxml.Vulnerability{}
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
