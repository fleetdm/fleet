package msrc_input

// XML elements related to the 'prod' namespace used to describe Microsoft products

type ProductBranchXML struct {
	Type     string       `xml:"Type,attr"`
	Name     string       `xml:"Name,attr"`
	Products []ProductXML `xml:"FullProductName"`
}

type ProductXML struct {
	ProductID uint   `xml:"ProductID,attr"`
	Name      string `xml:",chardata"`
}

// ContainsWinProducts returns true if the ProductBranchXML is for Windows products
func (b *ProductBranchXML) ContainsWinProducts() bool {
	return b.Name == "Windows" && b.Type == "Product Family"
}

// IsWindows checks whether the FullProductNameXML targets a Windows product
func (pn *ProductXML) IsWindows() bool {
	panic("not implemented")
}
