// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"fmt"
	"strconv"
	"strings"

	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
)

const (
	cveVersion = "4.0"
)

func (cve *CVE) Convert() (*nvd.NVDCVEFeedJSON10DefCVEItem, error) {
	publishedDate, err := convertTime(cve.PublicDate)
	if err != nil {
		return nil, fmt.Errorf("unable to convert published date: %v", err)
	}
	configurations, err := cve.newConfigurations()
	if err != nil {
		return nil, fmt.Errorf("unable to construct configurations: %v", err)
	}
	impact, err := cve.newImpact()
	if err != nil {
		return nil, fmt.Errorf("unable to construct impact: %v", err)
	}

	item := nvd.NVDCVEFeedJSON10DefCVEItem{
		CVE: &nvd.CVEJSON40{
			CVEDataMeta: &nvd.CVEJSON40CVEDataMeta{
				ID:       cve.ID(),
				ASSIGNER: "redhat",
			},
			DataFormat:  "MITRE",
			DataType:    "CVE",
			DataVersion: cveVersion,
			Description: &nvd.CVEJSON40Description{
				DescriptionData: []*nvd.CVEJSON40LangString{
					{
						Lang:  "en",
						Value: strings.Join(cve.Details, "\n"),
					},
				},
			},
			Problemtype: cve.newProblemType(),
			References:  cve.newReferences(),
		},
		Configurations: configurations,
		Impact:         impact,
		PublishedDate:  publishedDate,
	}

	return &item, nil
}

func (cve *CVE) ID() string {
	return cve.Name
}

func (cve *CVE) newProblemType() *nvd.CVEJSON40Problemtype {
	cwes := findCWEs(cve.CWE)
	if len(cwes) == 0 {
		return nil
	}
	data := make([]*nvd.CVEJSON40ProblemtypeProblemtypeData, len(cwes))
	for i, cwe := range cwes {
		data[i] = &nvd.CVEJSON40ProblemtypeProblemtypeData{
			Description: []*nvd.CVEJSON40LangString{
				{Lang: "en", Value: cwe},
			},
		}
	}

	return &nvd.CVEJSON40Problemtype{ProblemtypeData: data}
}

func (cve *CVE) newReferences() *nvd.CVEJSON40References {
	if len(cve.References) == 0 {
		return nil
	}

	referenceData := make([]*nvd.CVEJSON40Reference, len(cve.References))
	for i, ref := range cve.References {
		referenceData[i] = &nvd.CVEJSON40Reference{URL: ref}
	}

	return &nvd.CVEJSON40References{ReferenceData: referenceData}
}

func (cve *CVE) newImpact() (*nvd.NVDCVEFeedJSON10DefImpact, error) {
	if cve.CVSS == nil && cve.CVSS3 == nil {
		return nil, fmt.Errorf("cvss v2 nor cvss v3 is set in the cve")
	}

	impact := nvd.NVDCVEFeedJSON10DefImpact{}

	if cve.CVSS != nil {
		score, err := strconv.ParseFloat(cve.CVSS.BaseScore, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse cvss v2 base score: %v", err)
		}
		impact.BaseMetricV2 = &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV2{
			CVSSV2: &nvd.CVSSV20{
				BaseScore:    score,
				VectorString: cve.CVSS.Vector,
			},
		}
	}

	if cve.CVSS3 != nil {
		score, err := strconv.ParseFloat(cve.CVSS3.BaseScore, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse cvss v3 base score: %v", err)
		}
		impact.BaseMetricV3 = &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV3{
			CVSSV3: &nvd.CVSSV30{
				BaseScore:    score,
				VectorString: cve.CVSS3.Vector,
			},
		}
	}

	return &impact, nil
}

// CPEs configuration, AKA the tricky part

func (cve *CVE) newConfigurations() (*nvd.NVDCVEFeedJSON10DefConfigurations, error) {
	nodes := make([]*nvd.NVDCVEFeedJSON10DefNode, len(cve.AffectedRelease)+len(cve.PackageState))

	var err error

	for i, ar := range cve.AffectedRelease {
		if nodes[i], err = ar.createNode(); err != nil {
			return nil, fmt.Errorf("can't create node for affected release %d: %v", i, err)
		}
	}

	offset := len(cve.AffectedRelease)
	for i, ps := range cve.PackageState {
		if nodes[i+offset], err = ps.createNode(); err != nil {
			return nil, fmt.Errorf("can't create node for package state %d: %v", i, err)
		}
	}

	conf := nvd.NVDCVEFeedJSON10DefConfigurations{
		CVEDataVersion: cveVersion,
		Nodes:          nodes,
	}

	return &conf, nil
}

func (ar *AffectedRelease) createNode() (*nvd.NVDCVEFeedJSON10DefNode, error) {
	node := nvd.NVDCVEFeedJSON10DefNode{
		Operator: "AND",
		CPEMatch: []*nvd.NVDCVEFeedJSON10DefCPEMatch{
			{
				Cpe22Uri:   ar.CPE,
				Vulnerable: false,
			},
		},
	}

	if ar.Package != "" {
		pkgAttrs, err := package2wfn(ar.Package)
		if err != nil {
			return nil, fmt.Errorf("can't create wfn from package: %v", err)
		}

		node.CPEMatch = append(node.CPEMatch, &nvd.NVDCVEFeedJSON10DefCPEMatch{
			Cpe22Uri:   pkgAttrs.BindToURI(),
			Cpe23Uri:   pkgAttrs.BindToFmtString(),
			Vulnerable: false,
		})
	}

	return &node, nil
}

func (ps *PackageState) createNode() (*nvd.NVDCVEFeedJSON10DefNode, error) {
	pkgAttrs, err := packageName2wfn(ps.PackageName)
	if err != nil {
		return nil, fmt.Errorf("can't create wfn from package name: %v", err)
	}

	node := nvd.NVDCVEFeedJSON10DefNode{
		Operator: "AND",
		CPEMatch: []*nvd.NVDCVEFeedJSON10DefCPEMatch{
			// package
			{
				Cpe22Uri:   pkgAttrs.BindToURI(),
				Cpe23Uri:   pkgAttrs.BindToFmtString(),
				Vulnerable: !IsFixed(ps.FixState),
			},
			// distribution
			{
				Cpe22Uri:   ps.CPE,
				Vulnerable: false,
			},
		},
	}

	return &node, nil
}
