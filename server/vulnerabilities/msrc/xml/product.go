package xml

import "strings"

// XML elements related to the 'prod' namespace used to describe Microsoft products

// Describes a product three see
// http://docs.oasis-open.org/csaf/csaf-cvrf/v1.2/cs01/csaf-cvrf-v1.2-cs01.html#_Toc493508797
// for more details
type ProductBranch struct {
	Type     string          `xml:"Type,attr"`
	Name     string          `xml:"Name,attr"`
	Branches []ProductBranch `xml:"Branch"`
	Products []Product       `xml:"FullProductName"`
}

// Describes a full product name
// http://docs.oasis-open.org/csaf/csaf-cvrf/v1.2/cs01/csaf-cvrf-v1.2-cs01.html#_Toc493508797
type Product struct {
	ProductID string `xml:"ProductID,attr"`
	FullName  string `xml:",chardata"`
}

// WinProducts traverses the ProductBranchXML tree returning only 'Windows' products.
func (b *ProductBranch) WinProducts() []Product {
	var r []Product
	queue := []ProductBranch{*b}

	for len(queue) > 0 {
		next := queue[0]

		// We want only products from the 'Windows' and the 'Extended Security Update (ESU)' branches
		if next.Type == "Product Family" && (next.Name == "Windows" || next.Name == "ESU") {
			for _, p := range next.Products {
				// Even if the product branch is for 'Windows/ESU', there could be a non-OS
				// product like 'Remote Desktop client for Windows Desktop' inside the branch.
				if strings.HasPrefix(p.FullName, "Windows") {
					r = append(r, p)
				}
			}
		}

		queue = queue[1:]
		queue = append(queue, next.Branches...)
	}

	return r
}
