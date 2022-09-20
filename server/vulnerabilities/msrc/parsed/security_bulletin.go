package parsed

import (
	"encoding/json"
	"errors"
	"fmt"
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
	Products map[string]Product
	// All vulnerabilities contained in this bulletin, by CVE
	Vulnerabities map[string]Vulnerability
	// All vendor fixes for remediating the vulnerabilities contained in this bulletin, by KBID
	VendorFixes map[uint]VendorFix
}

func NewSecurityBulletin(pName string) *SecurityBulletin {
	return &SecurityBulletin{
		ProductName:   pName,
		Products:      make(map[string]Product),
		Vulnerabities: make(map[string]Vulnerability),
		VendorFixes:   make(map[uint]VendorFix),
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
				newVF.Supersedes = ptr.Uint(*r.Supersedes)
			}
			b.VendorFixes[kbID] = newVF
		}
	}

	return nil
}

// We will be using a weighted union-find datastruct for determing whether two kbIDs are connected,
// this will be used for handling cumulative updates.
type wUF struct {
	// Each 'value' points to the parent of 'key', each key is a KBID
	ids map[uint]uint
	// The size of each tree by 'kbID'
	size map[uint]uint16
}

func (uf *wUF) union(p uint, q uint) {
	pRoot := uf.root(p)
	qRoot := uf.root(q)

	if uf.size[qRoot] < uf.size[pRoot] {
		uf.ids[qRoot] = uf.ids[pRoot]
		uf.size[pRoot] += uf.size[qRoot]
	} else {
		uf.ids[pRoot] = uf.ids[qRoot]
		uf.size[qRoot] += uf.size[pRoot]
	}
}

func (uf *wUF) root(p uint) uint {
	if _, ok := uf.ids[p]; !ok {
		return p
	}

	for uf.ids[p] != p {
		uf.ids[p] = uf.ids[uf.ids[p]]
		p = uf.ids[p]
	}

	return p
}

func (uf *wUF) connected(p uint, q uint) bool {
	rootP := uf.root(p)
	rootQ := uf.root(q)

	if rootP == 0 || rootQ == 0 {
		return false
	}

	return rootP == rootQ
}

func (b *SecurityBulletin) initUF() *wUF {
	uf := &wUF{}

	uf.ids = make(map[uint]uint, len(b.VendorFixes))
	uf.size = make(map[uint]uint16, len(b.VendorFixes))

	// Init forest
	for kbID := range b.VendorFixes {
		uf.ids[kbID] = kbID
		uf.size[kbID] = 1
	}

	// Create unions
	for kbID, vf := range b.VendorFixes {
		if vf.Supersedes != nil {
			uf.union(kbID, *vf.Supersedes)
		}
	}

	return uf
}

var vendorFixGraph *wUF

func (b *SecurityBulletin) getVendorFixGraph() *wUF {
	if vendorFixGraph == nil {
		vendorFixGraph = b.initUF()
	}
	return vendorFixGraph
}

func (b *SecurityBulletin) Connected(p, q uint) bool {
	uf := b.getVendorFixGraph()
	fmt.Println(uf)
	return uf.connected(p, q)
}

type Vulnerability struct {
	PublishedEpoch *int64
	// Set of products ids that are susceptible to this vuln.
	ProductIDs map[string]bool
	// Set of Vendor fixes that remediate this vuln.
	RemediatedBy map[uint]bool
}

func NewVulnerability(publishedDateEpoch *int64) Vulnerability {
	return Vulnerability{
		PublishedEpoch: publishedDateEpoch,
		ProductIDs:     make(map[string]bool),
		RemediatedBy:   make(map[uint]bool),
	}
}

type VendorFix struct {
	// TODO (juan): Do we need this?
	FixedBuild string
	// Set of products ids that target this vendor fix
	ProductIDs map[string]bool
	// A Reference to what vendor fix this particular vendor fix 'replaces'.
	Supersedes *uint `json:",omitempty"`
}

func NewVendorFix(fixedBuild string) VendorFix {
	return VendorFix{
		FixedBuild: fixedBuild,
		ProductIDs: make(map[string]bool),
	}
}
