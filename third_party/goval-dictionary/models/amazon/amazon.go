package amazon

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
	for _, alas := range data.UpdateList {
		if strings.Contains(alas.Description, "** REJECT **") {
			continue
		}

		cves := []models.Cve{}
		for _, cveID := range alas.CVEIDs {
			cves = append(cves, models.Cve{
				CveID: cveID,
				Href:  fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", cveID),
			})
		}

		packs := []models.Package{}
		for _, pack := range alas.Packages {
			packs = append(packs, models.Package{
				Name:    pack.Name,
				Version: fmt.Sprintf("%s:%s-%s", pack.Epoch, pack.Version, pack.Release),
				Arch:    pack.Arch,
			})
		}

		refs := []models.Reference{}
		for _, ref := range alas.References {
			refs = append(refs, models.Reference{
				Source: ref.Type,
				RefID:  ref.ID,
				RefURL: ref.Href,
			})
		}

		issuedAt := util.ParsedOrDefaultTime([]string{"2006-01-02 15:04", "2006-01-02 15:04:05"}, alas.Issued.Date)
		updatedAt := util.ParsedOrDefaultTime([]string{"2006-01-02 15:04", "2006-01-02 15:04:05"}, alas.Updated.Date)

		def := models.Definition{
			DefinitionID: "def-" + alas.ID,
			Title:        alas.ID,
			Description:  alas.Description,
			Advisory: models.Advisory{
				Severity:           alas.Severity,
				Cves:               cves,
				Bugzillas:          []models.Bugzilla{},
				AffectedCPEList:    []models.Cpe{},
				AffectedRepository: alas.Repository,
				Issued:             issuedAt,
				Updated:            updatedAt,
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
