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

// Package schema implements parsing the vendor's data.
package schema

const (
	vendor     = "vfeed"
	timeLayout = "2006-01-02T15:04Z"

	// We assume below that this string has no whitespace.
	exclusionString = "(excluding)"

	// When CVSS2 or CVSS3 data is not available, all the respective fields
	// will have this value.
	cvssUndefined = "NOT_DEFINED"
)

// Item defines the vendor's vulnerability schema.
type Item struct {
	Information    *Information    `json:"information"`
	Classification *Classification `json:"classification"`
	Risk           *Risk           `json:"risk"`
}

// ID returns the identification of an Item.
func (item *Item) ID() string {
	if item.Information != nil &&
		item.Information.Descriptions != nil &&
		len(item.Information.Descriptions) > 0 {
		return item.Information.Descriptions[0].ID
	}

	return "unknown"
}

// Information holds CVE data.
type Information struct {
	Descriptions []*Description `json:"description"`
	References   []*Reference   `json:"references"`
}

// Description has the CVE ID and metadata.
type Description struct {
	ID         string         `json:"id"`
	Parameters *DescParameter `json:"parameters"`
}

// DescParameter holds the CVE metadata.
type DescParameter struct {
	Published string `json:"published"`
	Modified  string `json:"modified"`
	Summary   string `json:"summary"`
}

// Reference holds related pointers to the CVE.
type Reference struct {
	Vendor string `json:"vendor"`
	URL    string `json:"url"`
}

// Classification has CWE and CVSS data.
type Classification struct {
	Targets    []*Target   `json:"targets"`
	Weaknesses []*Weakness `json:"weaknesses"`
}

// Target holds NVD Configuration information.
type Target struct {
	ID         int32              `json:"id"`
	Parameters []*TargetParameter `json:"parameters"`
}

// TargetParameter holds Configuration Match data.
type TargetParameter struct {
	Title           string             `json:"title"`
	CPE22           string             `json:"cpe2.2"`
	CPE23           string             `json:"cpe2.3"`
	VersionAffected VersionAffected    `json:"version_affected"`
	RunningOn       []*TargetParameter `json:"running_on"`
}

// VersionAffected has the version data, as well as whether they are
// inclusive or exclusive.
type VersionAffected struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Weakness holds CWE data.
type Weakness struct {
	ID string
}

// Risk holds all the CVSS data.
type Risk struct {
	CVSS *CVSS `json:"cvss"`
}

// CVSS holds CVSS2 and CVSS3 data.
type CVSS struct {
	CVSS2 *CVSS2 `json:"cvss2"`
	CVSS3 *CVSS3 `json:"cvss3"`
}

// CVSS2 information.
type CVSS2 struct {
	Vector                string `json:"vector"`
	BaseScore             string `json:"base_score"`
	ImpactScore           string `json:"impact_score"`
	ExploitScore          string `json:"exploit_score"`
	AccessVector          string `json:"access_vector"`
	AccessComplexity      string `json:"access_complexity"`
	Authentication        string `json:"authentication"`
	ConfidentialityImpact string `json:"confidentiality_impack"`
	IntegrityImpact       string `json:"integrety_impact"`
	AvailabilityImpact    string `json:"availability_impact"`
}

// CVSS3 information.
type CVSS3 struct {
	Vector                string `json:"vector"`
	BaseScore             string `json:"base_score"`
	ImpactScore           string `json:"impact_score"`
	ExploitScore          string `json:"exploit_score"`
	AccessVector          string `json:"access_vector"`
	AccessComplexity      string `json:"access_complexity"`
	PrivilegesRequired    string `json:"privileges_required"`
	UserInteraction       string `json:"user_interaction"`
	Score                 string `json:"score"`
	ConfidentialityImpact string `json:"confidentiality_impack"`
	IntegrityImpact       string `json:"integrety_impact"`
	AvailabilityImpact    string `json:"availability_impact"`
}
