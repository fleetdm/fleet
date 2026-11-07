package nvdsync

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
)

// cveOverridesJSON corrects CPE matches that NVD (or VulnCheck) got wrong. Each entry rewrites the
// version bounds of the matches it selects; see cveOverride for the format.
//
//go:embed cve_overrides.json
var cveOverridesJSON []byte

var cveIDRe = regexp.MustCompile(`^CVE-\d{4}-\d{4,}$`)

// cveOverride selects the CPE matches of one CVE and rewrites their version bounds.
type cveOverride struct {
	CVE string `json:"cve"`
	// Reason says why the published data is wrong and where the corrected values come from.
	Reason string        `json:"reason"`
	Match  overrideMatch `json:"match"`
	Set    versionBounds `json:"set"`
}

// overrideMatch selects CPE matches whose criteria contain CriteriaContains and whose bounds
// satisfy every bound given. Matching on the wrong value lets an entry stop applying on its own once
// the upstream record is corrected.
type overrideMatch struct {
	CriteriaContains string `json:"criteria_contains"`
	versionBounds
}

// versionBounds holds CPE match version bounds. In a match a bound left out means any value and
// null means not set; in a set a bound left out is kept and null removes it.
type versionBounds struct {
	VersionStartIncluding optionalVersion `json:"versionStartIncluding"`
	VersionStartExcluding optionalVersion `json:"versionStartExcluding"`
	VersionEndIncluding   optionalVersion `json:"versionEndIncluding"`
	VersionEndExcluding   optionalVersion `json:"versionEndExcluding"`
}

func (b versionBounds) fields() []optionalVersion {
	return []optionalVersion{b.VersionStartIncluding, b.VersionStartExcluding, b.VersionEndIncluding, b.VersionEndExcluding}
}

// optionalVersion tells a bound left out of the file apart from one set to null.
type optionalVersion struct {
	present bool
	value   *string
}

func (o *optionalVersion) UnmarshalJSON(b []byte) error {
	o.present = true
	if string(b) == "null" {
		o.value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		return errors.New("version must not be empty; use null for a bound that is not set")
	}
	o.value = &s
	return nil
}

// matches reports whether a legacy feed bound, where "" means not set, satisfies o.
func (o optionalVersion) matches(actual string) bool {
	switch {
	case !o.present:
		return true
	case o.value == nil:
		return actual == ""
	default:
		return actual == *o.value
	}
}

func (o optionalVersion) apply(field *string) {
	switch {
	case !o.present:
	case o.value == nil:
		*field = ""
	default:
		*field = *o.value
	}
}

func (m overrideMatch) matches(cm *schema.NVDCVEFeedJSON10DefCPEMatch) bool {
	return strings.Contains(cm.Cpe23Uri, m.CriteriaContains) &&
		m.VersionStartIncluding.matches(cm.VersionStartIncluding) &&
		m.VersionStartExcluding.matches(cm.VersionStartExcluding) &&
		m.VersionEndIncluding.matches(cm.VersionEndIncluding) &&
		m.VersionEndExcluding.matches(cm.VersionEndExcluding)
}

func (b versionBounds) apply(cm *schema.NVDCVEFeedJSON10DefCPEMatch) {
	b.VersionStartIncluding.apply(&cm.VersionStartIncluding)
	b.VersionStartExcluding.apply(&cm.VersionStartExcluding)
	b.VersionEndIncluding.apply(&cm.VersionEndIncluding)
	b.VersionEndExcluding.apply(&cm.VersionEndExcluding)
}

// parseCVEOverrides decodes and validates the overrides file. Unknown keys are rejected so a typo
// fails instead of leaving an entry that silently never applies.
func parseCVEOverrides(raw []byte) ([]cveOverride, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var overrides []cveOverride
	if err := dec.Decode(&overrides); err != nil {
		return nil, fmt.Errorf("decode cve overrides: %w", err)
	}

	seen := make(map[string]struct{}, len(overrides))
	for i, o := range overrides {
		switch {
		case !cveIDRe.MatchString(o.CVE):
			return nil, fmt.Errorf("cve override %d: invalid CVE ID %q", i, o.CVE)
		case strings.TrimSpace(o.Reason) == "":
			return nil, fmt.Errorf("cve override %d (%s): reason is required", i, o.CVE)
		case o.Match.CriteriaContains == "":
			return nil, fmt.Errorf("cve override %d (%s): match.criteria_contains is required", i, o.CVE)
		}
		hasSet := false
		for _, f := range o.Set.fields() {
			hasSet = hasSet || f.present
		}
		if !hasSet {
			return nil, fmt.Errorf("cve override %d (%s): set has nothing to change", i, o.CVE)
		}
		key := o.CVE + "|" + o.Match.CriteriaContains
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("cve override %d (%s): another entry already matches %q", i, o.CVE, o.Match.CriteriaContains)
		}
		seen[key] = struct{}{}
	}
	return overrides, nil
}

// applyCVEOverrides rewrites the matching CPE matches in items and returns how many it changed.
func applyCVEOverrides(items []*schema.NVDCVEFeedJSON10DefCVEItem, overrides []cveOverride) int {
	byCVE := make(map[string][]cveOverride, len(overrides))
	for _, o := range overrides {
		byCVE[o.CVE] = append(byCVE[o.CVE], o)
	}

	changed := 0
	var walk func(nodes []*schema.NVDCVEFeedJSON10DefNode, ovs []cveOverride)
	walk = func(nodes []*schema.NVDCVEFeedJSON10DefNode, ovs []cveOverride) {
		for _, node := range nodes {
			if node == nil {
				continue
			}
			for _, cm := range node.CPEMatch {
				if cm == nil {
					continue
				}
				for _, o := range ovs {
					if o.Match.matches(cm) {
						o.Set.apply(cm)
						changed++
					}
				}
			}
			walk(node.Children, ovs)
		}
	}
	for _, item := range items {
		if item == nil || item.CVE == nil || item.CVE.CVEDataMeta == nil || item.Configurations == nil {
			continue
		}
		if ovs, ok := byCVE[item.CVE.CVEDataMeta.ID]; ok {
			walk(item.Configurations.Nodes, ovs)
		}
	}
	return changed
}

// ApplyOverrides applies cve_overrides.json to the stored legacy feeds. It runs on every feed
// generation after the NVD and VulnCheck merges, so a new entry reaches the next published feed
// instead of waiting for NVD to modify the CVE, and it also covers configurations that came from
// VulnCheck.
func (s *CVE) ApplyOverrides(ctx context.Context) error {
	overrides, err := parseCVEOverrides(cveOverridesJSON)
	if err != nil {
		return err
	}

	byYear := make(map[int][]cveOverride)
	for _, o := range overrides {
		year, err := strconv.Atoi(o.CVE[4:8])
		if err != nil {
			return fmt.Errorf("cve override %s: %w", o.CVE, err)
		}
		// Feeds start at 2002; older CVEs are stored in that year's file, as in updateYearFile.
		year = max(year, 2002)
		byYear[year] = append(byYear[year], o)
	}

	for year, ovs := range byYear {
		ok, err := fileExists(filepath.Join(s.dbDir, fmt.Sprintf("nvdcve-1.1-%d.json", year)))
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		feed, err := readCVEsLegacyFormat(s.dbDir, year)
		if err != nil {
			return fmt.Errorf("read %d feed for overrides: %w", year, err)
		}
		changed := applyCVEOverrides(feed.CVEItems, ovs)
		s.logger.DebugContext(ctx, "applied cve overrides", "year", year, "entries", len(ovs), "matches_changed", changed)
		if changed == 0 {
			continue
		}
		if err := storeCVEsInLegacyFormat(s.dbDir, year, feed); err != nil {
			return fmt.Errorf("store %d feed after overrides: %w", year, err)
		}
	}
	return nil
}
