package redhat

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	version "github.com/knqyf263/go-rpm-version"
	"github.com/spf13/viper"

	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/util"
)

// ConvertToModel Convert OVAL to models
func ConvertToModel(v string, roots []Root) []models.Definition {
	defs := map[string]models.Definition{}
	for _, root := range roots {

		ar := newArchResolver(root)

		for _, d := range root.Definitions.Definitions {
			if strings.HasPrefix(d.ID, "oval:com.redhat.unaffected:def:") || strings.Contains(d.Description, "** REJECT **") {
				continue
			}

			var cves = make([]models.Cve, 0, len(d.Advisory.Cves))
			for _, c := range d.Advisory.Cves {
				cves = append(cves, models.Cve{
					CveID:  c.CveID,
					Cvss2:  c.Cvss2,
					Cvss3:  c.Cvss3,
					Cwe:    c.Cwe,
					Impact: c.Impact,
					Href:   c.Href,
					Public: c.Public,
				})
			}

			var rs = make([]models.Reference, 0, len(d.References))
			for _, r := range d.References {
				rs = append(rs, models.Reference{
					Source: r.Source,
					RefID:  r.RefID,
					RefURL: r.RefURL,
				})
			}

			var cpes = make([]models.Cpe, 0, len(d.Advisory.AffectedCPEList))
			for _, cpe := range d.Advisory.AffectedCPEList {
				cpes = append(cpes, models.Cpe{
					Cpe: cpe,
				})
			}

			var bs = make([]models.Bugzilla, 0, len(d.Advisory.Bugzillas))
			for _, b := range d.Advisory.Bugzillas {
				bs = append(bs, models.Bugzilla{
					BugzillaID: b.ID,
					URL:        b.URL,
					Title:      b.Title,
				})
			}

			var ress = make([]models.Resolution, 0, len(d.Advisory.Affected.Resolution))
			for _, r := range d.Advisory.Affected.Resolution {
				ress = append(ress, models.Resolution{
					State: r.State,
					Components: func() []models.Component {
						var comps = make([]models.Component, 0, len(r.Component))
						for _, c := range r.Component {
							comps = append(comps, models.Component{
								Component: c,
							})
						}
						return comps
					}(),
				})
			}

			issued := util.ParsedOrDefaultTime([]string{"2006-01-02"}, d.Advisory.Issued.Date)
			updated := util.ParsedOrDefaultTime([]string{"2006-01-02"}, d.Advisory.Updated.Date)

			def := models.Definition{
				DefinitionID: d.ID,
				Title:        d.Title,
				Description:  d.Description,
				Advisory: models.Advisory{
					Severity:           d.Advisory.Severity,
					Cves:               cves,
					Bugzillas:          bs,
					AffectedResolution: ress,
					AffectedCPEList:    cpes,
					Issued:             issued,
					Updated:            updated,
				},
				AffectedPacks: collectRedHatPacks(v, d.Criteria, ar),
				References:    rs,
			}

			if viper.GetBool("no-details") {
				def.Title = ""
				def.Description = ""
				def.Advisory.Severity = ""
				def.Advisory.AffectedCPEList = []models.Cpe{}
				def.Advisory.Bugzillas = []models.Bugzilla{}
				def.Advisory.Issued = time.Time{}
				def.Advisory.Updated = time.Time{}
				def.References = []models.Reference{}
			}

			if _, ok := defs[def.DefinitionID]; !ok {
				defs[def.DefinitionID] = def
			}
		}
	}
	return slices.Collect(maps.Values(defs))
}

func collectRedHatPacks(v string, cri Criteria, ar archResolver) []models.Package {
	pkgs := map[string]models.Package{}
	for _, p := range walkRedHat(cri, []models.Package{}, "", ar) {
		n := p.Name
		if p.ModularityLabel != "" {
			n = fmt.Sprintf("%s::%s", p.ModularityLabel, p.Name)
		}

		key := n + "|" + p.Arch

		if p.NotFixedYet {
			pkgs[key] = p
			continue
		}

		// OVALv1 includes definitions other than the target RHEL version
		if !strings.Contains(p.Version, ".el"+v) && !strings.Contains(p.Version, ".module+el"+v) {
			continue
		}

		// since different versions are defined for the same package, the newer version is adopted
		// example: OVALv2: oval:com.redhat.rhsa:def:20111349, oval:com.redhat.rhsa:def:20120451
		if base, ok := pkgs[key]; ok {
			v1 := version.NewVersion(base.Version)
			v2 := version.NewVersion(p.Version)
			if v1.GreaterThan(v2) {
				p = base
			}
		}

		pkgs[key] = p
	}
	return slices.Collect(maps.Values(pkgs))
}

func walkRedHat(cri Criteria, acc []models.Package, label string, ar archResolver) []models.Package {
	for _, c := range cri.Criterions {
		switch {
		case strings.HasPrefix(c.Comment, "Module ") && strings.HasSuffix(c.Comment, " is enabled"):
			label = strings.TrimSuffix(strings.TrimPrefix(c.Comment, "Module "), " is enabled")
		case strings.Contains(c.Comment, " is earlier than "):
			ss := strings.Split(c.Comment, " is earlier than ")
			if len(ss) != 2 {
				continue
			}
			name := ss[0]
			ver := strings.Split(ss[1], " ")[0]

			arches := normalizeArches(ar.archForTestRef(c.TestRef))
			acc = emitPkgs(acc, name, ver, label, false, arches)

		case !strings.HasPrefix(c.Comment, "Red Hat Enterprise Linux") && !strings.HasPrefix(c.Comment, "Red Hat CoreOS") && strings.HasSuffix(c.Comment, " is installed"):

			name := strings.TrimSuffix(c.Comment, " is installed")
			arches := normalizeArches(ar.archForTestRef(c.TestRef))
			acc = emitPkgs(acc, name, "", label, true, arches)
		}
	}

	for _, c := range cri.Criterias {
		acc = walkRedHat(c, acc, label, ar)
	}
	return acc
}

type archResolver struct {
	testToState map[string]string // rpminfo_test id -> state id (state_ref)
	stateToArch map[string]string // state id -> arch text (may be "aarch64|ppc64le|...")
}

func newArchResolver(root Root) archResolver {
	ar := archResolver{
		testToState: map[string]string{},
		stateToArch: map[string]string{},
	}

	// Map test -> state_ref (NOTE: uses t.State.StateRef)
	for _, t := range root.Tests.RpminfoTests {
		if t.ID != "" && t.State.StateRef != "" {
			ar.testToState[t.ID] = t.State.StateRef
		}
	}

	// Map state id -> arch text
	for _, s := range root.States.RpminfoStates {
		if s.ID != "" && s.Arch.Text != "" {
			ar.stateToArch[s.ID] = s.Arch.Text
		}
	}
	return ar
}

func (ar archResolver) archForTestRef(testRef string) string {
	if testRef == "" {
		return ""
	}
	if st := ar.testToState[testRef]; st != "" {
		return ar.stateToArch[st]
	}
	return ""
}

// Split "aarch64|ppc64le|s390x|x86_64" -> []string{...}
func normalizeArches(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func emitPkgs(acc []models.Package, name, ver, label string, notFixed bool, arches []string) []models.Package {
	if len(arches) == 0 {
		arches = []string{""}
	}
	for _, a := range arches {
		acc = append(acc, models.Package{
			Name:            name,
			Version:         ver,
			ModularityLabel: label,
			Arch:            a,
			NotFixedYet:     notFixed,
		})
	}
	return acc
}
