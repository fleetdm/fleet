package fedora

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/util"
)

// ConvertToModel Convert OVAL to models
func ConvertToModel(data *Updates) (defs []models.Definition) {
	for _, update := range data.UpdateList {
		if strings.Contains(update.Description, "** REJECT **") {
			continue
		}

		cves := make([]models.Cve, 0, len(update.CVEIDs))
		for _, cveID := range update.CVEIDs {
			cves = append(cves, models.Cve{
				CveID: cveID,
				Href:  fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", cveID),
			})
		}

		packs := make([]models.Package, 0, len(update.Packages))
		for _, pack := range update.Packages {
			packs = append(packs, models.Package{
				Name:            pack.Name,
				Version:         fmt.Sprintf("%s:%s-%s", pack.Epoch, pack.Version, pack.Release),
				Arch:            pack.Arch,
				ModularityLabel: update.ModularityLabel,
			})
		}

		refs := make([]models.Reference, 0, len(update.References))
		bs := []models.Bugzilla{}
		for _, ref := range update.References {
			refs = append(refs, models.Reference{
				Source: ref.Type,
				RefID:  ref.ID,
				RefURL: ref.Href,
			})
			if ref.Type == "bugzilla" {
				bs = append(bs, models.Bugzilla{
					BugzillaID: ref.ID,
					URL:        ref.Href,
					Title:      ref.Title,
				})
			}
		}

		issuedAt := util.ParsedOrDefaultTime([]string{"2006-01-02 15:04:05"}, update.Issued.Date)
		updatedAt := util.ParsedOrDefaultTime([]string{"2006-01-02 15:04:05"}, update.Updated.Date)
		def := models.Definition{
			DefinitionID: "def-" + update.ID,
			Title:        update.ID,
			Description:  update.Description,
			Advisory: models.Advisory{
				Severity:        update.Severity,
				Cves:            cves,
				Bugzillas:       bs,
				AffectedCPEList: []models.Cpe{},
				Issued:          issuedAt,
				Updated:         updatedAt,
			},
			Debian:        nil,
			AffectedPacks: packs,
			References:    refs,
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
