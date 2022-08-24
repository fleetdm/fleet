package msrc_parsed

type SecurityBulletin struct {
	// The 'product' name this bulletin targets (e.g. Windows 10)
	ProductName string
	// All products contained in this bulletin (Product ID => Product full name).
	// We can have many different 'products' under a single name, for example, for 'Windows 10':
	// - Windows 10 Version 1809 for 32-bit Systems
	// - Windows 10 Version 1909 for x64-based Systems
	Products map[string]string
	// All vulnerabilities contained in this bulletin, by CVE
	Vulnerabities map[string]Vulnerability
	// All vendor fixes for remediating the vulnerabilities contained in this bulletin, by KBID
	VendorFixes map[int]VendorFix
}

func NewSecurityBulletin(pName string) *SecurityBulletin {
	return &SecurityBulletin{
		ProductName:   pName,
		Products:      make(map[string]string),
		Vulnerabities: make(map[string]Vulnerability),
		VendorFixes:   make(map[int]VendorFix),
	}
}

type Vulnerability struct {
	PublishedEpoch *int64
	// Set of products that are susceptible to this vuln.
	ProductIDs map[string]bool
	// Set of Vendor fixes that remediate this vuln.
	RemediatedBy map[int]bool
}

func NewVulnerability(publishedDateEpoch *int64) Vulnerability {
	return Vulnerability{
		PublishedEpoch: publishedDateEpoch,
		ProductIDs:     make(map[string]bool),
		RemediatedBy:   make(map[int]bool),
	}
}

type VendorFix struct {
	// TODO (juan): Do we need this?
	FixedBuild string
	ProductIDs map[string]bool
	// A Reference to what vendor fix this particular vendor fix 'replaces'.
	Supersedes *int `json:",omitempty"`
}

func NewVendorFix(fixedBuild string) VendorFix {
	return VendorFix{
		FixedBuild: fixedBuild,
		ProductIDs: make(map[string]bool),
	}
}
