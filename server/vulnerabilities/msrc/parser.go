package msrc

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"

	msrc_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/input"
	msrc_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
)

func parseMSRC(inputFile string, outputFile string) error {
	r, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("oval parser: %w", err)
	}
	defer r.Close()

	parseMSRCXML(r)

	// if err != nil {
	// 	return fmt.Errorf("msrc parser: %w", err)
	// }

	// err = ioutil.WriteFile(outputFile, payload, 0o644)
	// if err != nil {
	// 	return fmt.Errorf("msrc parser: %w", err)
	// }

	return nil
}

func mapToVulnGraphs(rXML *msrc_input.ResultXML) (map[string]*msrc_parsed.VulnGraph, error) {
	// We will have one graph for each product name.
	rGraphs := make(map[string]*msrc_parsed.VulnGraph)

	for pID, p := range rXML.WinProducts {
		name := NameFromFullProdName(p.FullName)
		if _, ok := rGraphs[name]; !ok {
			rGraphs[name] = msrc_parsed.NewVulnGraph(name)
		}
		rGraphs[name].Products[pID] = p.FullName
	}

	for _, v := range rXML.WinVulnerabities {
		for _, r := range v.Remediations {
			// We will only be able to detect vulns for which they are vendor fixes.
			if !r.IsVendorFix() {
				continue
			}

			for _, rPID := range r.ProductIDs {
				p := rXML.WinProducts[rPID]

				name := NameFromFullProdName(p.FullName)
				g, ok := rGraphs[name]
				if !ok {
					continue
				}

				// Create/update the vulnerability node related to this vendor fix
				var vNode msrc_parsed.VulnNode
				if vNode, ok = g.Vulnerabities[v.CVE]; !ok {
					vNode = msrc_parsed.VulnNode{
						Published: v.PublishedDate(),
					}
				}
				vNode.ProductsIDs = append(vNode.ProductsIDs, rPID)
				vNode.RemediatedBy = append(
					vNode.RemediatedBy,
					msrc_parsed.NewVendorFixNodeRef(r.Description),
				)
				g.Vulnerabities[v.CVE] = vNode

				// Create/update the vendor fix node
				var vfNode msrc_parsed.VendorFixNode
				if vfNode, ok = g.VendorFixes[r.Description]; !ok {
					vfNode = msrc_parsed.VendorFixNode{
						FixedBuild: r.FixedBuild,
						Supersedes: msrc_parsed.NewVendorFixNodeRef(r.Supercedence),
					}
				}
				vfNode.TargetProductsIDs = append(vfNode.TargetProductsIDs, rPID)
				g.VendorFixes[r.Description] = vfNode
			}
		}
	}

	return rGraphs, nil
}

func parseMSRCXML(reader io.Reader) (*msrc_input.ResultXML, error) {
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
