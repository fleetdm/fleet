package parsed

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/fleetdm/fleet/v4/server/ptr"
)

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

func UnmarshalBulletin(fPath string) (*SecurityBulletin, error) {
	payload, err := os.ReadFile(fPath)
	if err != nil {
		return nil, err
	}

	bulletin := SecurityBulletin{}
	err = json.Unmarshal(payload, &bulletin)
	if err != nil {
		return nil, err
	}
	return &bulletin, nil
}

// Merge merges in-place the contents of 'other' into the current bulletin.
func (b *SecurityBulletin) Merge(other *SecurityBulletin) error {
	if b.ProductName != other.ProductName {
		return errors.New("bulletins are for different products")
	}

	// Products
	for pID, pName := range other.Products {
		if _, ok := b.Products[pID]; !ok {
			b.Products[pID] = pName
		}
	}

	// Vulnerabilities
	for cve, vuln := range other.Vulnerabities {
		if _, ok := b.Vulnerabities[cve]; !ok {
			newVuln := NewVulnerability(vuln.PublishedEpoch)
			for pID, v := range vuln.ProductIDs {
				newVuln.ProductIDs[pID] = v
			}
			for rID, v := range vuln.RemediatedBy {
				newVuln.RemediatedBy[rID] = v
			}
			b.Vulnerabities[cve] = newVuln
		}
	}

	// Vendor fixes
	for kbID, r := range other.VendorFixes {
		if _, ok := b.VendorFixes[kbID]; !ok {
			newVF := NewVendorFix(r.FixedBuild)
			for pID, v := range r.ProductIDs {
				newVF.ProductIDs[pID] = v
			}
			if r.Supersedes != nil {
				newVF.Supersedes = ptr.Int(*r.Supersedes)
			}
			b.VendorFixes[kbID] = newVF
		}
	}

	return nil
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
