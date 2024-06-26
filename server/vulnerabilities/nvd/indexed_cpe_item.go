package nvd

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

type IndexedCPEItem struct {
	ID         int `json:"id" db:"rowid"`
	Part       string
	Product    string `json:"product" db:"product"`
	Vendor     string `json:"vendor" db:"vendor"`
	Deprecated bool   `json:"deprecated" db:"deprecated"`
	Weight     int    `db:"weight"`
}

func (i *IndexedCPEItem) FmtStr(s *fleet.Software) string {
	cpe := wfn.NewAttributesWithAny()
	cpe.Part = "a"
	cpe.Vendor = i.Vendor
	cpe.Product = i.Product
	cpe.Version = sanitizeVersion(s.Version)
	cpe.TargetSW = targetSW(s)

	if i.Part != "" {
		cpe.Part = i.Part
	}

	// Make sure we don't return a 'match all' CPE
	if cpe.Vendor == wfn.Any || cpe.Product == wfn.Any {
		return ""
	}

	return cpe.BindToFmtString()
}
