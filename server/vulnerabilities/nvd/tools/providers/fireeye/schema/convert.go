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
	"strings"

	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

const (
	cveDataVersion = "4.0"
)

func (item *Vulnerability) Convert() (*nvd.NVDCVEFeedJSON10DefCVEItem, error) {
	nvdItem := nvd.NVDCVEFeedJSON10DefCVEItem{
		CVE: &nvd.CVEJSON40{
			CVEDataMeta: &nvd.CVEJSON40CVEDataMeta{
				ID:       item.ID(),
				ASSIGNER: "fireeye",
			},
			DataFormat:  "MITRE",
			DataType:    "CVE",
			DataVersion: cveDataVersion,
			Description: &nvd.CVEJSON40Description{
				DescriptionData: []*nvd.CVEJSON40LangString{
					{Lang: "en", Value: item.Title},
				},
			},
			References: item.makeReferences(),
		},
		Configurations: item.makeConfigurations(),
		Impact: &nvd.NVDCVEFeedJSON10DefImpact{
			BaseMetricV2: &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV2{
				CVSSV2: &nvd.CVSSV20{
					BaseScore:     extractCVSSBaseScore(item),
					TemporalScore: extractCVSSTemporalScore(item),
					VectorString:  extractCVSSVectorString(item),
				},
			},
		},
		LastModifiedDate: convertTime(item.PublishDate),
		PublishedDate:    convertTime(item.Version1PublishDate),
	}

	return &nvdItem, nil
}

func (item *Vulnerability) ID() string {
	return "fireeye-" + item.ReportID
}

func (item *Vulnerability) makeReferences() *nvd.CVEJSON40References {
	var refsData []*nvd.CVEJSON40Reference
	addRef := func(name, url string) {
		refsData = append(refsData, &nvd.CVEJSON40Reference{
			Name: name,
			URL:  url,
		})
	}

	addRef("FireEye report API link", item.ReportLink)
	addRef("FireEye web link", item.WebLink)
	for _, cve := range item.CVEIds {
		for _, cveid := range strings.Split(cve, ",") {
			addRef(cveid, "")
		}
	}

	return &nvd.CVEJSON40References{
		ReferenceData: refsData,
	}
}

func (item *Vulnerability) makeConfigurations() *nvd.NVDCVEFeedJSON10DefConfigurations {
	var matches []*nvd.NVDCVEFeedJSON10DefCPEMatch
	for _, cpe := range extractCPEs(item) {
		matches = append(matches, &nvd.NVDCVEFeedJSON10DefCPEMatch{
			Cpe23Uri:   cpe,
			Vulnerable: true,
		})
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
