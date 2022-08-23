package msrc_parsed

import (
	"time"
)

type SecurityBulletin struct {
	// The name portion of the full product name. We will have one bulletin per 'product' name (e.g. Windows 10)
	ProductName string
	// When was the bulletin last updated.
	LastUpdated time.Time
	// All products contained in this bulletin (Product ID => Product full name).
	// We can have many different 'products' under a single name, for example, for 'Windows 10':
	// - Windows 10 Version 1809 for 32-bit Systems
	// - Windows 10 Version 1909 for x64-based Systems
	Products map[string]string
	// All vulnerabilities contained in this bulletin, by CVE
	Vulnerabities map[string]Vulnerability
	// All vendor fixes for remediating the vulnerabilities contained in this bulletin, by KBID
	VendorFixes map[string]VendorFix
}

func NewSecurityBulletin(pName string) *SecurityBulletin {
	return &SecurityBulletin{
		ProductName:   pName,
		LastUpdated:   time.Now().UTC(),
		Products:      make(map[string]string),
		Vulnerabities: make(map[string]Vulnerability),
		VendorFixes:   make(map[string]VendorFix),
	}
}

type VendorFixNodeRef struct {
	RefName  string
	RefValue int
}

type Vulnerability struct {
	PublishedEpoch *int64
	// Set of products that are susceptible to this vuln.
	ProductIDsSet map[string]bool
	// Set of Vendor fixes that remediate this vuln.
	RemediatedBySet map[int]bool
}

func NewVendorFixNodeRef(val int) VendorFixNodeRef {
	return VendorFixNodeRef{
		RefName:  "vendor_fixes",
		RefValue: val,
	}
}

type VendorFix struct {
	FixedBuild        string
	TargetProductsIDs []string
	// Reference to what vendor fix this particular vendor fix 'replaces'.
	Supersedes VendorFixNodeRef
}
