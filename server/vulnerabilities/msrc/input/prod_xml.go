package msrc_input

import "strings"

// XML elements related to the 'prod' namespace used to describe Microsoft products

type ProductBranchXML struct {
	Type     string             `xml:"Type,attr"`
	Name     string             `xml:"Name,attr"`
	Branches []ProductBranchXML `xml:"Branch"`
	Products []ProductXML       `xml:"FullProductName"`
}

type ProductXML struct {
	ProductID string `xml:"ProductID,attr"`
	Name      string `xml:",chardata"`
}

// WindowsProducts traverses the ProductBranchXML tree returning only 'Windows' products.
func (b *ProductBranchXML) WindowsProducts() []ProductXML {
	var r []ProductXML
	queue := []ProductBranchXML{*b}

	for len(queue) > 0 {
		next := queue[0]

		// We want only products from the 'Windows' and the 'Extended Security Update (ESU)' branches
		if next.Type == "Product Family" && (next.Name == "Windows" || next.Name == "ESU") {
			for _, p := range next.Products {
				// Even if the product branch is for 'Windows/ESU', there could be a non-OS
				// product like 'Remote Desktop client for Windows Desktop' inside the branch.
				if strings.HasPrefix(p.Name, "Windows") {
					r = append(r, p)
				}
			}
		}

		queue = queue[1:]
		queue = append(queue, next.Branches...)
	}

	return r
}
