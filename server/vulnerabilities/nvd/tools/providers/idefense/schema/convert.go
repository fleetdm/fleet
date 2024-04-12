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
	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"

	"github.com/pkg/errors"
)

const (
	cveDataVersion = "4.0"
)

// Convert implements runner.Convertible interface
func (item *Vulnerability) Convert() (*nvd.NVDCVEFeedJSON10DefCVEItem, error) {
	lastModifiedDate, err := convertTime(item.LastModified)
	if err != nil {
		return nil, errors.Wrap(err, "can't convert last modified date")
	}
	publishedDate, err := convertTime(item.LastPublished)
	if err != nil {
		return nil, errors.Wrap(err, "can't convert published date")
	}

	configurations, err := item.makeConfigurations()
	if err != nil {
		return nil, errors.Wrap(err, "can't create configurations")
	}

	return &nvd.NVDCVEFeedJSON10DefCVEItem{
		CVE: &nvd.CVEJSON40{
			CVEDataMeta: &nvd.CVEJSON40CVEDataMeta{
				ID:       item.ID(),
				ASSIGNER: "idefense",
			},
			DataFormat:  "MITRE",
			DataType:    "CVE",
			DataVersion: cveDataVersion,
			Description: &nvd.CVEJSON40Description{
				DescriptionData: []*nvd.CVEJSON40LangString{
					{Lang: "en", Value: item.Description},
				},
			},
			Problemtype: &nvd.CVEJSON40Problemtype{
				ProblemtypeData: []*nvd.CVEJSON40ProblemtypeProblemtypeData{
					{
						Description: []*nvd.CVEJSON40LangString{
							{Lang: "en", Value: item.Cwe},
						},
					},
				},
			},
			References: item.makeReferences(),
		},
		Configurations: configurations,
		Impact: &nvd.NVDCVEFeedJSON10DefImpact{
			BaseMetricV2: &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV2{
				CVSSV2: &nvd.CVSSV20{
					BaseScore:     item.Cvss2BaseScore,
					TemporalScore: item.Cvss2TemporalScore,
					VectorString:  item.Cvss2,
				},
			},
			BaseMetricV3: &nvd.NVDCVEFeedJSON10DefImpactBaseMetricV3{
				CVSSV3: &nvd.CVSSV30{
					BaseScore:     item.Cvss3BaseScore,
					TemporalScore: item.Cvss3TemporalScore,
					VectorString:  item.Cvss3,
				},
			},
		},
		LastModifiedDate: lastModifiedDate,
		PublishedDate:    publishedDate,
	}, nil
}

func (item *Vulnerability) ID() string {
	return "idefense-" + item.Key
}

func (item *Vulnerability) makeReferences() *nvd.CVEJSON40References {
	if len(item.SourcesExternal) == 0 {
		return nil
	}

	var refsData []*nvd.CVEJSON40Reference
	addRef := func(name, url string) {
		refsData = append(refsData, &nvd.CVEJSON40Reference{
			Name: name,
			URL:  url,
		})
	}

	for _, source := range item.SourcesExternal {
		addRef(source.Name, source.URL)
	}
	if item.AlsoIdentifies != nil {
		for _, vuln := range item.AlsoIdentifies.Vulnerability {
			addRef(vuln.Key, "")
		}
	}
	for _, poc := range item.Pocs {
		addRef(poc.PocName, poc.URL)
	}
	for _, fix := range item.VendorFixExternal {
		addRef(fix.ID, fix.URL)
	}

	return &nvd.CVEJSON40References{
		ReferenceData: refsData,
	}
}

func (item *Vulnerability) makeConfigurations() (*nvd.NVDCVEFeedJSON10DefConfigurations, error) {
	configs := item.findConfigurations()
	if len(configs) == 0 {
		return nil, errors.New("unable to find any configurations in data")
	}

	var matches []*nvd.NVDCVEFeedJSON10DefCPEMatch
	for _, cfg := range configs {
		for _, affected := range cfg.Affected {
			match := &nvd.NVDCVEFeedJSON10DefCPEMatch{
				Cpe23Uri:   cfg.Cpe23Uri,
				Vulnerable: true,
			}

			// determine version ranges
			if cfg.HasFixedBy {
				if affected.Prior {
					match.VersionEndExcluding = cfg.FixedByVersion
				} else {
					match.VersionStartIncluding = affected.Version
					match.VersionEndExcluding = cfg.FixedByVersion
				}
			} else {
				if !affected.Prior {
					match.VersionStartIncluding = affected.Version
				}
			}
			matches = append(matches, match)
		}
	}

	v := nvd.NVDCVEFeedJSON10DefConfigurations{
		CVEDataVersion: cveDataVersion,
		Nodes: []*nvd.NVDCVEFeedJSON10DefNode{
			{
				CPEMatch: matches,
				Operator: "OR",
			},
		},
	}

	return &v, nil
}
