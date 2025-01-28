package parsed

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"golang.org/x/exp/slices"
)

type SecurityBulletin struct {
	// The 'product' name this bulletin targets (e.g. Windows 10)
	ProductName string
	// All products contained in this bulletin (Product ID => Product full name).
	// We can have many different 'products' under a single name, for example, for 'Windows 10':
	// - Windows 10 Version 1809 for 32-bit Systems
	// - Windows 10 Version 1909 for x64-based Systems
	Products Products
	// All vulnerabilities contained in this bulletin, by CVE
	Vulnerabities map[string]Vulnerability
	// All vendor fixes for remediating the vulnerabilities contained in this bulletin, by KBID
	VendorFixes map[uint]VendorFix

	// Data struct used for telling if two KBID are 'connected'
	vfForest *weightedUnionFind
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
	for pID, name := range bulletin.Products {
		bulletin.Products[pID] = NewProductFromFullName(string(name))
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
			newVF := NewVendorFix(r.FixedBuilds...)
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

func (b *SecurityBulletin) initUnionFind() *weightedUnionFind {
	uf := &weightedUnionFind{}

	uf.ids = make(map[uint]uint, len(b.VendorFixes))
	uf.size = make(map[uint]uint16, len(b.VendorFixes))

	// Init forest
	for KBID := range b.VendorFixes {
		uf.ids[KBID] = KBID
		uf.size[KBID] = 1
	}

	// Create unions
	for KBID, vf := range b.VendorFixes {
		if vf.Supersedes != nil {
			uf.union(KBID, *vf.Supersedes)
		}
	}

	return uf
}

func (b *SecurityBulletin) getVFForest() *weightedUnionFind {
	if b.vfForest == nil {
		b.vfForest = b.initUnionFind()
	}
	return b.vfForest
}

// KBIDsConnected returns whether two updates are 'connected', used for dealing with cumulative
// updates. A cumulative update can replace another update (we determine this via the 'Supersedes'
// prop. in the VendorFix type), when determining whether a host is susceptible to a vulnerability we
// are interested in determining whether the host has a specific update installed or any of the
// superseded updates.
func (b *SecurityBulletin) KBIDsConnected(p, q uint) bool {
	return b.getVFForest().connected(p, q)
}

// ----
// UnionFind
// ----

// We will be using a weighted union-find data struct for determining whether two KBIDs are 'connected',
// this will be used for handling cumulative updates.
type weightedUnionFind struct {
	// Each 'value' points to the parent of 'key', each key is a KBID
	ids map[uint]uint
	// The size of each tree by 'KBID'
	size map[uint]uint16
}

// union connects two components (KBID)
func (uf *weightedUnionFind) union(p uint, q uint) {
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

// root returns the root of the 'p' tree
func (uf *weightedUnionFind) root(p uint) uint {
	if _, ok := uf.ids[p]; !ok {
		return p
	}

	for uf.ids[p] != p {
		uf.ids[p] = uf.ids[uf.ids[p]]
		p = uf.ids[p]
	}

	return p
}

// connected returns whether two components are connected, for example:
// A -> B -> C -> D; connected(A, C) -> true
func (uf *weightedUnionFind) connected(p uint, q uint) bool {
	return uf.root(p) == uf.root(q)
}

// ----------------------
// Vulnerability
// ----------------------

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

// ----------------------
// VendorFix
// ----------------------

type VendorFix struct {
	FixedBuilds []string
	// Set of products ids that target this vendor fix
	ProductIDs map[string]bool
	// A Reference to what vendor fix this particular vendor fix 'replaces'.
	Supersedes *uint `json:",omitempty"`
}

func (vf *VendorFix) AddFixedBuild(fixedBuild string) {
	if fixedBuild != "" && !slices.Contains(vf.FixedBuilds, fixedBuild) {
		vf.FixedBuilds = append(vf.FixedBuilds, fixedBuild)
	}
}

func NewVendorFix(fixedBuilds ...string) VendorFix {
	fixedBuildsCopy := make([]string, len(fixedBuilds))
	copy(fixedBuildsCopy, fixedBuilds)
	return VendorFix{
		FixedBuilds: fixedBuildsCopy,
		ProductIDs:  make(map[string]bool),
	}
}
