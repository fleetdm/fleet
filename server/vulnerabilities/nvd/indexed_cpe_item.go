package nvd

import (
	"regexp"
	"strings"

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
	cpe.TargetSW = targetSW(s)

	// Some version strings (e.g. Python pre-releases) contain a part that should be placed in the
	// CPE's update field. Parse that out (if it exists).
	// See https://github.com/fleetdm/fleet/issues/25882.
	version, update := parseUpdateFromVersion(sanitizeVersion(s.Version))
	cpe.Version = version
	cpe.Update = update

	if i.Part != "" {
		cpe.Part = i.Part
	}

	// Make sure we don't return a 'match all' CPE
	if cpe.Vendor == wfn.Any || cpe.Product == wfn.Any {
		return ""
	}

	return cpe.BindToFmtString()
}

var cpeUpdate = regexp.MustCompile(`(\d+\.\d+\.\d+)((?:a|b|rc)\d+)$`)

func parseUpdateFromVersion(originalVersion string) (version, update string) {
	// Return the unchanged original version by default
	version = originalVersion

	if cpeUpdate.MatchString(originalVersion) {
		versionBytes := []byte{}
		updateBytes := []byte{}
		for _, submatches := range cpeUpdate.FindAllStringSubmatchIndex(originalVersion, -1) {
			versionBytes = cpeUpdate.ExpandString(versionBytes, "${1}", originalVersion, submatches)
			updateBytes = cpeUpdate.ExpandString(updateBytes, "${2}", originalVersion, submatches)
			version = string(versionBytes)
			switch updateBytes[0] {
			case 'a':
				update = strings.ReplaceAll(string(updateBytes), "a", "alpha")
			case 'b':
				update = strings.ReplaceAll(string(updateBytes), "b", "beta")
			case 'r':
				update = string(updateBytes)
			}
		}
	}

	return version, update
}
