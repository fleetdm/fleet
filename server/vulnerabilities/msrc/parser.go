package msrc

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"

	msrc_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/input"
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

func parseMSRCXML(reader io.Reader) (*msrc_input.ResultXML, error) {
	r := &msrc_input.ResultXML{
		Products: map[uint]msrc_input.ProductXML{},
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
			if t.Name.Space == "prod" && t.Name.Local == "Branch" {
				branch := msrc_input.ProductBranchXML{}
				if err = d.DecodeElement(&branch, &t); err != nil {
					return nil, err
				}

				if branch.ContainsWinProducts() {
					for _, p := range branch.Products {
						// Even if the product branch is for 'Windows', there could be a non-OS
						// product like 'Remote Desktop client for Windows Desktop' in which case we
						// want to skip those.
						if p.IsWindows() {
							r.Products[p.ProductID] = p
						}
					}
				}
			}

			if t.Name.Space == "vuln" && t.Name.Local == "Vulnerability" {
				vuln := msrc_input.VulnerabilityXML{}
				if err = d.DecodeElement(&vuln, &t); err != nil {
					return nil, err
				}

				// We only care about vulnerabilities that have a vendor fix targeting a Windows product
				for _, rem := range vuln.VendorFixes() {
					for _, pID := range rem.ProductIDs {
						if _, ok := r.Products[pID]; ok {
							r.Vulnerabities = append(r.Vulnerabities, vuln)
						}
					}
				}
			}
		}
	}
}
