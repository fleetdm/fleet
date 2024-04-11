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

	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

const (
	cveDataVersion = "4.0"
)

// Convert converts  advisories to NVD format
func (item *Advisory) Convert() (*nvd.NVDCVEFeedJSON10DefCVEItem, error) {
	if item.Products == nil {
		return nil, fmt.Errorf("no products associated with advisory")
	}

	var cpes []string
	for _, product := range item.Products {
		if productCPEs, err := findCPEs(product); err == nil {
			cpes = append(cpes, productCPEs...)
		}
	}
	if len(cpes) == 0 {
		return nil, fmt.Errorf("no cpes associated with advisory")
	}

	lastModifiedDate, err := convertTime(item.ModifiedDate)
	if err != nil {
		return nil, err
	}

	publishedDate, err := convertTime(item.Released)
	if err != nil {
		return nil, err
	}

	return &nvd.NVDCVEFeedJSON10DefCVEItem{
		CVE: &nvd.CVEJSON40{
			CVEDataMeta: &nvd.CVEJSON40CVEDataMeta{
				ID:       item.ID(),
				ASSIGNER: "flexera",
			},
			DataFormat:  "MITRE",
			DataType:    "CVE",
			DataVersion: cveDataVersion,
			Description: &nvd.CVEJSON40Description{
				DescriptionData: []*nvd.CVEJSON40LangString{
					{Lang: "en", Value: item.Description},
				},
			},
			References: item.makeReferences(),
		},
		Configurations:   makeConfigurations(cpes),
		Impact:           item.makeImpact(),
		LastModifiedDate: lastModifiedDate,
		PublishedDate:    publishedDate,
	}, nil
}

func (item *Advisory) ID() string {
	return "flexera-" + item.AdvisoryIdentifier
}

func (item *Advisory) makeReferences() *nvd.CVEJSON40References {
	var refsData []*nvd.CVEJSON40Reference
	addRef := func(name, url string) {
		refsData = append(refsData, &nvd.CVEJSON40Reference{
			Name: name,
			URL:  url,
		})
	}

	if item.References != nil {
		for _, ref := range item.References {
			addRef(ref.Description, ref.URL)
		}
	}

	if item.Vulnerabilities != nil {
		for _, vuln := range item.Vulnerabilities {
			addRef(vuln.Cve, "")
		}
	}

	return &nvd.CVEJSON40References{
		ReferenceData: refsData,
	}
}

func makeConfigurations(cpes []string) *nvd.NVDCVEFeedJSON10DefConfigurations {
	matches := make([]*nvd.NVDCVEFeedJSON10DefCPEMatch, len(cpes))
	for i, cpe := range cpes {
		matches[i] = &nvd.NVDCVEFeedJSON10DefCPEMatch{
			Cpe22Uri:   cpe,
			Vulnerable: true,
		}
	}

	return &nvd.NVDCVEFeedJSON10DefConfigurations{
		CVEDataVersion: cveDataVersion,
		Nodes: []*nvd.NVDCVEFeedJSON10DefNode{
			&nvd.NVDCVEFeedJSON10DefNode{
				CPEMatch: matches,
				Operator: "OR",
			},
		},
	}
}

func (item *Advisory) makeImpact() *nvd.NVDCVEFeedJSON10DefImpact {
	var cvssv2 nvd.CVSSV20
	if item.CvssInfo != nil {
		cvssv2.BaseScore = item.CvssInfo.BaseScore
		cvssv2.VectorString = item.CvssInfo.Vector
	}
	var cvssv3 nvd.CVSSV30
	if item.Cvss3Info != nil {
		cvssv3.BaseScore = item.Cvss3Info.BaseScore
		cvssv3.VectorString = item.Cvss3Info.Vector
	}

	return &nvd.NVDCVEFeedJSON10DefImpact{
		BaseMetricV2: &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV2{
			CVSSV2: &cvssv2,
		},
		BaseMetricV3: &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV3{
			CVSSV3: &cvssv3,
		},
	}
}
