package debian

import (
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/util"
)

type distroPackage struct {
	osVer string
	pack  models.Package
}

// ConvertToModel Convert OVAL to models
func ConvertToModel(root *Root) (defs []models.Definition) {
	for _, ovaldef := range root.Definitions.Definitions {
		if strings.Contains(ovaldef.Description, "** REJECT **") {
			continue
		}

		cves := []models.Cve{}
		rs := make([]models.Reference, 0, len(ovaldef.References))
		for _, r := range ovaldef.References {
			if r.Source == "CVE" {
				cves = append(cves, models.Cve{
					CveID: r.RefID,
					Href:  r.RefURL,
				})
			}

			rs = append(rs, models.Reference{
				Source: r.Source,
				RefID:  r.RefID,
				RefURL: r.RefURL,
			})
		}

		def := models.Definition{
			DefinitionID: ovaldef.ID,
			Title:        ovaldef.Title,
			Description:  ovaldef.Description,
			Advisory: models.Advisory{
				Severity:        "",
				Cves:            cves,
				Bugzillas:       []models.Bugzilla{},
				AffectedCPEList: []models.Cpe{},
				Issued:          time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Updated:         time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			Debian: &models.Debian{
				DSA:      ovaldef.Debian.DSA,
				MoreInfo: ovaldef.Debian.MoreInfo,
				Date:     util.ParsedOrDefaultTime([]string{"2006-01-02"}, ovaldef.Debian.Date),
			},
			AffectedPacks: collectDebianPacks(ovaldef.Criteria),
			References:    rs,
		}

		if viper.GetBool("no-details") {
			def.Title = ""
			def.Description = ""
			def.Advisory.Severity = ""
			def.Advisory.Bugzillas = []models.Bugzilla{}
			def.Advisory.AffectedCPEList = []models.Cpe{}
			def.Advisory.Issued = time.Time{}
			def.Advisory.Updated = time.Time{}
			def.Debian = nil
			def.References = []models.Reference{}
		}

		defs = append(defs, def)
	}
	return
}

func collectDebianPacks(cri Criteria) []models.Package {
	distPacks := walkDebian(cri, "", []distroPackage{})
	packs := make([]models.Package, len(distPacks))
	for i, distPack := range distPacks {
		packs[i] = distPack.pack
	}
	return packs
}

func walkDebian(cri Criteria, osVer string, acc []distroPackage) []distroPackage {
	for _, c := range cri.Criterions {
		if strings.HasPrefix(c.Comment, "Debian ") &&
			strings.HasSuffix(c.Comment, " is installed") {
			osVer = strings.TrimSuffix(strings.TrimPrefix(c.Comment, "Debian "), " is installed")
		}
		ss := strings.Split(c.Comment, " DPKG is earlier than ")
		if len(ss) != 2 {
			continue
		}

		// "0" means notyetfixed or erroneous information.
		// Not available because "0" includes erroneous info...
		if ss[1] == "0" {
			continue
		}
		acc = append(acc, distroPackage{
			osVer: osVer,
			pack: models.Package{
				Name:    ss[0],
				Version: strings.Split(ss[1], " ")[0],
			},
		})
	}

	if len(cri.Criterias) == 0 {
		return acc
	}
	for _, c := range cri.Criterias {
		acc = walkDebian(c, osVer, acc)
	}
	return acc
}
