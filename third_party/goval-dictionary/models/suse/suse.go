package suse

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/models"
)

type distroPackage struct {
	osVer string
	pack  models.Package
}

// ConvertToModel Convert OVAL to models
func ConvertToModel(xmlName string, root *Root) (map[string][]models.Definition, error) {
	tests, err := parseTests(*root)
	if err != nil {
		return nil, xerrors.Errorf("Failed to parse oval.Tests. err: %w", err)
	}
	return parseDefinitions(xmlName, root.Definitions, tests), nil
}

type rpmInfoTest struct {
	Name           string
	SignatureKeyID SignatureKeyid
	FixedVersion   string
	Arch           []string
}

func parseObjects(ovalObjs Objects) map[string]string {
	objs := map[string]string{}
	for _, obj := range ovalObjs.RpminfoObject {
		objs[obj.ID] = obj.Name
	}
	return objs
}

func parseStates(objStates States) map[string]RpminfoState {
	states := map[string]RpminfoState{}
	for _, state := range objStates.RpminfoState {
		states[state.ID] = state
	}
	return states
}

func parseTests(root Root) (map[string]rpmInfoTest, error) {
	objs := parseObjects(root.Objects)
	states := parseStates(root.States)
	tests := map[string]rpmInfoTest{}
	for _, test := range root.Tests.RpminfoTest {
		t, err := followTestRefs(test, objs, states)
		if err != nil {
			return nil, xerrors.Errorf("Failed to follow test refs. err: %w", err)
		}
		tests[test.ID] = t
	}
	return tests, nil
}

func followTestRefs(test RpminfoTest, objects map[string]string, states map[string]RpminfoState) (rpmInfoTest, error) {
	var t rpmInfoTest

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

	t.SignatureKeyID = state.SignatureKeyid

	if state.Arch.Datatype == "string" && (state.Arch.Operation == "pattern match" || state.Arch.Operation == "equals") {
		// state.Arch.Text: (aarch64|ppc64le|s390x|x86_64)
		t.Arch = strings.Split(state.Arch.Text[1:len(state.Arch.Text)-1], "|")
	}

	if state.Evr.Datatype == "evr_string" && state.Evr.Operation == "less than" {
		t.FixedVersion = state.Evr.Text
	}

	return t, nil
}

func parseDefinitions(xmlName string, ovalDefs Definitions, tests map[string]rpmInfoTest) map[string][]models.Definition {
	defs := map[string][]models.Definition{}

	for _, d := range ovalDefs.Definitions {
		if strings.Contains(d.Description, "** REJECT **") {
			continue
		}

		cves := []models.Cve{}
		if strings.Contains(xmlName, "opensuse.1") || strings.Contains(xmlName, "suse.linux.enterprise.desktop.10") || strings.Contains(xmlName, "suse.linux.enterprise.server.9") || strings.Contains(xmlName, "suse.linux.enterprise.server.10") {
			if strings.HasPrefix(d.Title, "CVE-") {
				cves = append(cves, models.Cve{
					CveID: d.Title,
					Href:  fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", d.Title),
				})
			}
		} else {
			for _, c := range d.Advisory.Cves {
				cves = append(cves, models.Cve{
					CveID:  strings.TrimSuffix(strings.TrimSuffix(c.CveID, " at NVD"), " at SUSE"),
					Cvss3:  c.Cvss3,
					Impact: c.Impact,
					Href:   c.Href,
				})
			}
		}

		references := []models.Reference{}
		for _, r := range d.References {
			references = append(references, models.Reference{
				Source: r.Source,
				RefID:  r.RefID,
				RefURL: r.RefURL,
			})
		}

		cpes := []models.Cpe{}
		for _, cpe := range d.Advisory.AffectedCPEList {
			cpes = append(cpes, models.Cpe{
				Cpe: cpe,
			})
		}

		bugzillas := []models.Bugzilla{}
		for _, b := range d.Advisory.Bugzillas {
			bugzillas = append(bugzillas, models.Bugzilla{
				URL:   b.URL,
				Title: b.Title,
			})
		}

		osVerPackages := map[string][]models.Package{}
		for _, distPack := range collectSUSEPacks(xmlName, d.Criteria, tests) {
			osVerPackages[distPack.osVer] = append(osVerPackages[distPack.osVer], distPack.pack)
		}

		for osVer, packs := range osVerPackages {
			def := models.Definition{
				DefinitionID: d.ID,
				Title:        d.Title,
				Description:  d.Description,
				Advisory: models.Advisory{
					Severity:        d.Advisory.Severity,
					Cves:            append([]models.Cve{}, cves...),           // If the same slice is used, it will only be stored once in the DB
					Bugzillas:       append([]models.Bugzilla{}, bugzillas...), // If the same slice is used, it will only be stored once in the DB
					AffectedCPEList: append([]models.Cpe{}, cpes...),           // If the same slice is used, it will only be stored once in the DB
					Issued:          time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
					Updated:         time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				Debian:        nil,
				AffectedPacks: packs,
				References:    append([]models.Reference{}, references...), // If the same slice is used, it will only be stored once in the DB
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

			defs[osVer] = append(defs[osVer], def)
		}
	}

	return defs
}

func collectSUSEPacks(xmlName string, cri Criteria, tests map[string]rpmInfoTest) []distroPackage {
	if strings.Contains(xmlName, "opensuse.12") {
		verPkgs := []distroPackage{}
		v := strings.TrimSuffix(strings.TrimPrefix(xmlName, "opensuse."), ".xml")
		_, pkgs := walkCriterion(cri, []string{}, []models.Package{}, tests)
		for _, pkg := range pkgs {
			verPkgs = append(verPkgs, distroPackage{
				osVer: v,
				pack:  pkg,
			})
		}
		return verPkgs
	}
	return walkCriteria(cri, []distroPackage{}, tests)
}

func walkCriteria(cri Criteria, acc []distroPackage, tests map[string]rpmInfoTest) []distroPackage {
	if cri.Operator == "AND" {
		vs, pkgs := walkCriterion(cri, []string{}, []models.Package{}, tests)
		for _, v := range vs {
			for _, pkg := range pkgs {
				acc = append(acc, distroPackage{
					osVer: v,
					pack:  pkg,
				})
			}
		}
		return acc
	}
	for _, criteria := range cri.Criterias {
		acc = walkCriteria(criteria, acc, tests)
	}
	return acc
}

func walkCriterion(cri Criteria, versions []string, packages []models.Package, tests map[string]rpmInfoTest) ([]string, []models.Package) {
	for _, c := range cri.Criterions {
		if isOSComment(c.Comment) {
			comment := strings.TrimSuffix(c.Comment, " is installed")
			v, err := getOSVersion(comment)
			if err != nil {
				log15.Warn("Failed to getOSVersion", "comment", comment, "err", err)
				continue
			}
			if v != "" {
				versions = append(versions, v)
			}
			continue
		}

		if strings.HasSuffix(c.Comment, "is not affected") {
			continue
		}

		t, ok := tests[c.TestRef]
		if !ok {
			continue
		}

		// Skip red-def:signature_keyid
		if t.SignatureKeyID.Text != "" {
			continue
		}

		packages = append(packages, models.Package{
			Name:    t.Name,
			Version: t.FixedVersion,
		})
	}

	if len(cri.Criterias) == 0 {
		return versions, packages
	}
	for _, c := range cri.Criterias {
		versions, packages = walkCriterion(c, versions, packages, tests)
	}
	return versions, packages
}

func isOSComment(comment string) bool {
	if !strings.HasSuffix(comment, "is installed") {
		return false
	}
	if strings.HasPrefix(comment, "suse1") || // os: suse102 is installed, pkg: suseRegister less than
		comment == "core9 is installed" ||
		(strings.HasPrefix(comment, "sles10") && !strings.Contains(comment, "-docker-image-")) || // os: sles10-sp1 is installed, pkg: sles12-docker-image-1.1.4-20171002 is installed
		strings.HasPrefix(comment, "sled10") || // os: sled10-sp1 is installed
		strings.HasPrefix(comment, "openSUSE") || strings.HasPrefix(comment, "SUSE Linux Enterprise") || strings.HasPrefix(comment, "SUSE Manager") {
		return true
	}
	return false
}

// base: https://github.com/aquasecurity/trivy-db/blob/main/pkg/vulnsrc/suse-cvrf/suse-cvrf.go
var versionReplacer = strings.NewReplacer("-SECURITY", "", "-LTSS", "", "-TERADATA", "", "-CLIENT-TOOLS", "", "-PUBCLOUD", "")

func getOSVersion(platformName string) (string, error) {
	if strings.HasPrefix(platformName, "suse") {
		s := strings.TrimPrefix(platformName, "suse")
		if len(s) < 3 {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: invalid version", platformName)
		}
		ss := strings.Split(s, "-")
		v := fmt.Sprintf("%s.%s", ss[0][:2], ss[0][2:])
		if _, err := version.NewVersion(v); err != nil {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
		}
		return v, nil
	}

	if strings.HasPrefix(platformName, "sled") {
		s := strings.TrimPrefix(platformName, "sled")
		ss := strings.Split(s, "-")
		var v string
		switch len(ss) {
		case 0:
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: invalid version string", platformName)
		case 1:
			v = ss[0]
		case 2:
			if strings.HasPrefix(ss[1], "sp") {
				v = fmt.Sprintf("%s.%s", ss[0], strings.TrimPrefix(ss[1], "sp"))
			} else {
				v = ss[0]
			}
		default:
			v = fmt.Sprintf("%s.%s", ss[0], strings.TrimPrefix(ss[1], "sp"))
		}
		if _, err := version.NewVersion(v); err != nil {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
		}
		return v, nil
	}

	if strings.HasPrefix(platformName, "sles") {
		s := strings.TrimPrefix(platformName, "sles")
		ss := strings.Split(s, "-")
		var v string
		switch len(ss) {
		case 0:
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: invalid version string", platformName)
		case 1:
			v = ss[0]
		case 2:
			if strings.HasPrefix(ss[1], "sp") {
				v = fmt.Sprintf("%s.%s", ss[0], strings.TrimPrefix(ss[1], "sp"))
			} else {
				v = ss[0]
			}
		default:
			v = fmt.Sprintf("%s.%s", ss[0], strings.TrimPrefix(ss[1], "sp"))
		}
		if _, err := version.NewVersion(v); err != nil {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
		}
		return v, nil
	}

	if strings.HasPrefix(platformName, "core9") {
		return "9", nil
	}

	if strings.HasPrefix(platformName, "openSUSE") {
		if strings.HasPrefix(platformName, "openSUSE Leap") {
			// openSUSE Leap 15.0
			ss := strings.Fields(platformName)
			if len(ss) < 3 {
				return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: invalid version", platformName)
			}
			if _, err := version.NewVersion(ss[2]); err != nil {
				return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
			}
			return ss[2], nil
		}
		// openSUSE 13.2, openSUSE Tumbleweed
		ss := strings.Fields(platformName)
		if len(ss) < 2 {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: invalid version", platformName)
		}
		if ss[1] == "Tumbleweed" {
			return "tumbleweed", nil
		}
		if _, err := version.NewVersion(ss[1]); err != nil {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
		}
		return ss[1], nil
	}

	if strings.HasPrefix(platformName, "SUSE Linux Enterprise") {
		// e.g. SUSE Linux Enterprise Storage 7, SUSE Linux Enterprise Micro 5.1
		if strings.HasPrefix(platformName, "SUSE Linux Enterprise Storage") || strings.HasPrefix(platformName, "SUSE Linux Enterprise Micro") {
			return "", nil
		}

		// e.g. SUSE Linux Enterprise Server 12 SP1-LTSS
		ss := strings.Fields(platformName)
		if strings.HasPrefix(ss[len(ss)-1], "SP") || isInt(ss[len(ss)-2]) {
			// Remove suffix such as -TERADATA, -LTSS
			sps := strings.Split(ss[len(ss)-1], "-")
			// Remove "SP" prefix
			sp := strings.TrimPrefix(sps[0], "SP")
			// Check if the version is integer
			spVersion, err := strconv.Atoi(sp)
			if err != nil {
				return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
			}
			return fmt.Sprintf("%s.%d", ss[len(ss)-2], spVersion), nil
		}
		// e.g. SUSE Linux Enterprise Server 11-SECURITY
		ver := versionReplacer.Replace(ss[len(ss)-1])
		if _, err := version.NewVersion(ver); err != nil {
			return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: %w", platformName, err)
		}
		return ver, nil
	}

	if strings.HasPrefix(platformName, "SUSE Manager") {
		// e.g. SUSE Manager Proxy 4.0, SUSE Manager Server 4.0
		return "", nil
	}

	return "", xerrors.Errorf("Failed to detect os version. platformName: %s, err: not support platform", platformName)
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
