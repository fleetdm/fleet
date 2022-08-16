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

				for _, p := range branch.WindowsProducts() {
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
