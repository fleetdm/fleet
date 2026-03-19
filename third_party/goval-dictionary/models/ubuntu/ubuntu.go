package ubuntu

import (
	"regexp"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/util"
)

// ConvertToModel Convert OVAL to models
func ConvertToModel(root *Root) ([]models.Definition, error) {
	tests, err := parseTests(*root)
	if err != nil {
		return nil, xerrors.Errorf("Failed to parse oval.Tests. err: %w", err)
	}
	return parseDefinitions(root.Definitions.Definitions, tests), nil
}

var rePkgComment = regexp.MustCompile(`The '(.*)' package binar.+`)

func parseObjects(ovalObjs Objects) map[string]string {
	objs := map[string]string{}
	for _, obj := range ovalObjs.Textfilecontent54Object {
		matched := rePkgComment.FindAllStringSubmatch(obj.Comment, 1)
		if len(matched[0]) != 2 {
			continue
		}
		objs[obj.ID] = matched[0][1]
	}
	return objs
}

func parseStates(objStates States) map[string]Textfilecontent54State {
	states := map[string]Textfilecontent54State{}
	for _, state := range objStates.Textfilecontent54State {
		states[state.ID] = state
	}
	return states
}

func parseTests(root Root) (map[string]dpkgInfoTest, error) {
	objs := parseObjects(root.Objects)
	states := parseStates(root.States)
	tests := map[string]dpkgInfoTest{}
	for _, test := range root.Tests.Textfilecontent54Test {
		t, err := followTestRefs(test, objs, states)
		if err != nil {
			return nil, xerrors.Errorf("Failed to follow test refs. err: %w", err)
		}
		tests[test.ID] = t
	}
	return tests, nil
}

func followTestRefs(test Textfilecontent54Test, objects map[string]string, states map[string]Textfilecontent54State) (dpkgInfoTest, error) {
	var t dpkgInfoTest

	// Follow object ref
	if test.Object.ObjectRef == "" {
		return t, nil
	}

	pkgName, ok := objects[test.Object.ObjectRef]
	if !ok {
		return t, xerrors.Errorf("Failed to find object ref. object ref: %s, test ref: %s, err: invalid tests data", test.Object.ObjectRef, test.ID)
	}
	t.Name = pkgName

	// Follow state ref
	if test.State.StateRef == "" {
		return t, nil
	}

	state, ok := states[test.State.StateRef]
	if !ok {
		return t, xerrors.Errorf("Failed to find state ref. state ref: %s, test ref: %s, err: invalid tests data", test.State.StateRef, test.ID)
	}

	if state.Subexpression.Datatype == "debian_evr_string" && state.Subexpression.Operation == "less than" {
		t.FixedVersion = state.Subexpression.Text
	}

	return t, nil
}

func parseDefinitions(ovalDefs []Definition, tests map[string]dpkgInfoTest) []models.Definition {
	defs := []models.Definition{}

	for _, d := range ovalDefs {
		if strings.Contains(d.Description, "** REJECT **") {
			continue
		}

		cves := []models.Cve{}
		rs := make([]models.Reference, 0, len(d.References))
		for _, r := range d.References {
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

		for _, r := range d.Advisory.Refs {
			rs = append(rs, models.Reference{
				Source: "Ref",
				RefURL: r.URL,
			})
		}

		for _, r := range d.Advisory.Bugs {
			rs = append(rs, models.Reference{
				Source: "Bug",
				RefURL: r.URL,
			})
		}

		date := util.ParsedOrDefaultTime([]string{"2006-01-02", "2006-01-02 15:04:05", "2006-01-02 15:04:05 +0000", "2006-01-02 15:04:05 MST"}, d.Advisory.PublicDate)

		def := models.Definition{
			DefinitionID: d.ID,
			Title:        d.Title,
			Description:  d.Description,
			Advisory: models.Advisory{
				Severity:        d.Advisory.Severity,
				Cves:            cves,
				Bugzillas:       []models.Bugzilla{},
				AffectedCPEList: []models.Cpe{},
				Issued:          date,
				Updated:         date,
			},
			Debian:        nil,
			AffectedPacks: collectUbuntuPacks(d.Criteria, tests),
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

		defs = append(defs, def)
	}

	return defs
}

func collectUbuntuPacks(cri Criteria, tests map[string]dpkgInfoTest) []models.Package {
	return walkCriterion(cri, tests)
}

func walkCriterion(cri Criteria, tests map[string]dpkgInfoTest) []models.Package {
	pkgs := []models.Package{}
	for _, c := range cri.Criterions {
		t, ok := tests[c.TestRef]
		if !ok {
			continue
		}

		if strings.Contains(c.Comment, "is related to the CVE in some way and has been fixed") || // status: not vulnerable(= not affected)
			strings.Contains(c.Comment, "is affected and may need fixing") { // status: needs-triage
			continue
		}

		if strings.Contains(c.Comment, "is affected and needs fixing") || // status: needed
			strings.Contains(c.Comment, "is affected, but a decision has been made to defer addressing it") || // status: deferred
			strings.Contains(c.Comment, "is affected. An update containing the fix has been completed and is pending publication") || // status: pending
			strings.Contains(c.Comment, "while related to the CVE in some way, a decision has been made to ignore this issue") { // status: ignored
			pkgs = append(pkgs, models.Package{
				Name:        t.Name,
				NotFixedYet: true,
			})
		} else if strings.Contains(c.Comment, "was vulnerable but has been fixed") || // status: released
			strings.Contains(c.Comment, "was vulnerable and has been fixed") { // status: released, only this comment: "firefox package in $RELEASE_NAME was vulnerable and has been fixed, but no release version available for it."
			pkgs = append(pkgs, models.Package{
				Name:        t.Name,
				Version:     t.FixedVersion,
				NotFixedYet: false,
			})
		} else {
			log15.Warn("Failed to detect patch status.", "comment", c.Comment)
		}
	}

	for _, c := range cri.Criterias {
		if ps := walkCriterion(c, tests); len(ps) > 0 {
			pkgs = append(pkgs, ps...)
		}
	}
	return pkgs
}
