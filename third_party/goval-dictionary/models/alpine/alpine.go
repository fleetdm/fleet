package alpine

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/vulsio/goval-dictionary/models"
)

// ConvertToModel Convert OVAL to models
func ConvertToModel(data *SecDB) (defs []models.Definition) {
	cveIDPacks := map[string][]models.Package{}
	for _, pack := range data.Packages {
		for ver, vulnIDs := range pack.Pkg.Secfixes {
			for _, s := range vulnIDs {
				cveID := strings.Split(s, " ")[0]
				if !strings.HasPrefix(cveID, "CVE") {
					continue
				}

				if packs, ok := cveIDPacks[cveID]; ok {
					packs = append(packs, models.Package{
						Name:    pack.Pkg.Name,
						Version: ver,
					})
					cveIDPacks[cveID] = packs
				} else {
					cveIDPacks[cveID] = []models.Package{{
						Name:    pack.Pkg.Name,
						Version: ver,
					}}
				}
			}
		}
	}

	for cveID, packs := range cveIDPacks {
		def := models.Definition{
			DefinitionID: fmt.Sprintf("def-%s-%s-%s", data.Reponame, data.Distroversion, cveID),
			Title:        cveID,
			Description:  "",
			Advisory: models.Advisory{
				Severity:        "",
				Cves:            []models.Cve{{CveID: cveID, Href: fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", cveID)}},
				Bugzillas:       []models.Bugzilla{},
				AffectedCPEList: []models.Cpe{},
				Issued:          time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
				Updated:         time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			Debian:        nil,
			AffectedPacks: packs,
			References: []models.Reference{
				{
					Source: "CVE",
					RefID:  cveID,
					RefURL: fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", cveID),
				},
			},
		}

		if viper.GetBool("no-details") {
			def.Title = ""
			def.Description = ""
			def.Advisory.Severity = ""
			def.Advisory.Bugzillas = []models.Bugzilla{}
			def.Advisory.AffectedCPEList = []models.Cpe{}
			def.Advisory.Issued = time.Time{}
			def.Advisory.Updated = time.Time{}
			def.References = []models.Reference{}
		}

		defs = append(defs, def)
	}
	return
}
