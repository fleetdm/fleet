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
	"time"

	"github.com/pkg/errors"

	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

type basicCVEData struct {
	id        string
	summary   string
	modified  *time.Time
	published *time.Time
}

// Convert reads a vendor item and outputs it in the NVD format.
func (item *Item) Convert() (*nvd.NVDCVEFeedJSON10DefCVEItem, error) {
	basicData, err := item.basicCVEData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract cve data")
	}

	references := ProvidersNewReferences()
	if item.Information != nil && item.Information.References != nil {
		for _, ref := range item.Information.References {
			references.Add(ref.Vendor, ref.URL)
		}
	}

	config, err := item.configurationData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract configuration data")
	}

	cvss2, cvss3, err := item.riskData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract risk data")
	}

	return ProvidersNewItem(
		&ProvidersItem{
			Vendor:           vendor,
			ID:               basicData.id,
			Description:      basicData.summary,
			References:       references,
			Configuration:    config,
			CWEs:             item.weaknessData(),
			CVSS2:            cvss2,
			CVSS3:            cvss3,
			LastModifiedDate: basicData.modified,
			PublishedDate:    basicData.published,
		},
	)
}

func (item *Item) riskData() (cvss2 *ProvidersCVSS, cvss3 *ProvidersCVSS, err error) {
	if item.Risk == nil || item.Risk.CVSS == nil {
		return
	}

	cvss := item.Risk.CVSS

	if cvss.CVSS2 != nil {
		cvss2, err = cvssData(cvss.CVSS2.Vector, cvss.CVSS2.BaseScore)
		if err != nil {
			err = errors.Wrap(err, "failed to extract cvss2 data")
			return
		}
	}

	if cvss.CVSS3 != nil {
		cvss3, err = cvssData(cvss.CVSS3.Vector, cvss.CVSS3.BaseScore)
		if err != nil {
			err = errors.Wrap(err, "failed to extract cvss3 data")
			return
		}
	}

	return
}

func cvssData(vector, strBaseScore string) (*ProvidersCVSS, error) {
	if vector == cvssUndefined {
		return nil, nil
	}

	baseScore, err := strconv.ParseFloat(strBaseScore, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base score: %v", err)
	}

	return &ProvidersCVSS{
		Vector:    vector,
		BaseScore: baseScore,
	}, nil
}

// extractVersion receives a version with a qualifier and returns the version
// number and whether it is exclusive. Example input version:
// "2.1 (excluding)"
func extractVersion(version string) (string, bool, error) {
	if version == "" {
		return "", false, nil
	}

	fields := strings.Fields(version)
	if n := len(fields); n != 2 {
		return "", false, fmt.Errorf("expected two fields in version %q, found %d", version, n)
	}

	return fields[0], fields[1] == exclusionString, nil
}

func (item *Item) weaknessData() []string {
	var cwes []string

	if item.Classification != nil && item.Classification.Weaknesses != nil {
		for _, w := range item.Classification.Weaknesses {
			cwes = append(cwes, w.ID)
		}
	}

	return cwes
}

func (item *Item) basicCVEData() (*basicCVEData, error) {
	if item.Information == nil {
		return nil, nil
	}

	// The schema has support for multiple CVEs, but in practice only one is used
	// per item, which is all we support.
	if n := len(item.Information.Descriptions); n != 1 {
		return nil, fmt.Errorf("we only support 1 CVE per item, found %d", n)
	}

	description := item.Information.Descriptions[0]

	descParams := description.Parameters
	if descParams == nil {
		return nil, nil
	}

	modified, err := ProvidersConvertStrTime(timeLayout, descParams.Modified)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert last modified date")
	}

	published, err := ProvidersConvertStrTime(timeLayout, descParams.Published)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert last modified date")
	}

	return &basicCVEData{
		id:        description.ID,
		summary:   descParams.Summary,
		modified:  modified,
		published: published,
	}, nil
}

func (item *Item) configurationData() (*ProvidersConfiguration, error) {
	if item.Classification == nil {
		return nil, nil
	}

	config := ProvidersNewConfiguration()

	if item.Classification.Targets != nil {
		for _, target := range item.Classification.Targets {
			node := config.NewNode()

			// Parameters will follow one of the cases:
			// 1 - A list of vulnerable CPEs with versions
			// 2 - One vulnerable CPE with versions followed by "running_on",
			//     conditional non-vulnerable CPEs without versions.

			for _, params := range target.Parameters {
				if params.RunningOn != nil {
					for _, running := range params.RunningOn {
						node.AddConditionalMatch(
							&ProvidersMatch{
								CPE22URI:   running.CPE22,
								CPE23URI:   running.CPE23,
								Vulnerable: false,
							},
						)
					}

					continue
				}

				match := &ProvidersMatch{
					CPE22URI:   params.CPE22,
					CPE23URI:   params.CPE23,
					Vulnerable: true,
				}

				version := params.VersionAffected

				from, excluding, err := extractVersion(version.From)
				if err != nil {
					return nil, err
				}
				match.AddVersionStart(from, excluding)

				to, excluding, err := extractVersion(version.To)
				if err != nil {
					return nil, err
				}
				match.AddVersionEnd(to, excluding)

				node.AddMatch(match)
			}
		}
	}

	return config, nil
}
