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

type VulnerabilityResult struct {
	TotalEntries    int              `json:"total_entries"`
	CurrentPage     int              `json:"current_page"`
	TotalPages      int              `json:"total_pages"`
	Vulnerabilities []*Vulnerability `json:"results"`
}

type Vulnerability struct {
	VulndbID               int                  `json:"vulndb_id"`
	Title                  string               `json:"title"`
	VulndbPublishedDate    string               `json:"vulndb_published_date"`
	VulndbLastModified     string               `json:"vulndb_last_modified"`
	DisclosureDate         string               `json:"disclosure_date"`
	DiscoveryDate          string               `json:"discovery_date"`
	ExploitPublishDate     string               `json:"exploit_publish_date"`
	Keywords               string               `json:"keywords"`
	Description            string               `json:"description"`
	Solution               string               `json:"solution"`
	ManualNotes            string               `json:"manual_notes"`
	TDescription           string               `json:"t_description"`
	SolutionDate           string               `json:"solution_date"`
	VendorInformedDate     string               `json:"vendor_informed_date"`
	VendorAckDate          string               `json:"vendor_ack_date"`
	ThirdPartySolutionDate string               `json:"third_party_solution_date"`
	Changelogs             []*Changelog         `json:"changelog"`
	Classifications        []*Classification    `json:"classifications"`
	Authors                []*Author            `json:"authors"`
	ExtReferences          []*ExtReference      `json:"ext_references"`
	CVSSMetrics            []*CVSSMetric        `json:"cvss_metrics"`
	CVSS3Metrics           []*CVSS3Metric       `json:"cvss_version_three_metrics"`
	Vendors                []*Vendor            `json:"vendors"`
	Packages               []*Package           `json:"packages"`
	NVDAdditionalInfo      []*NVDAdditionalInfo `json:"nvd_additional_information"`
}

type Changelog struct {
	Date        string `json:"date"`
	Description string `json:"description"`
}

type Classification struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Longname    string `json:"longname"`
	Description string `json:"description"`
}

type Author struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Company    string `json:"company"`
	Email      string `json:"email"`
	CompanyURL string `json:"company_url"`
	Country    string `json:"country"`
}

type ExtReference struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type CVSSMetric struct {
	ID                      int     `json:"id"`
	AccessVector            string  `json:"access_vector"`
	AccessComplexity        string  `json:"access_complexity"`
	Authentication          string  `json:"authentication"`
	ConfidentialityImpact   string  `json:"confidentiality_impact"`
	IntegrityImpact         string  `json:"integrity_impact"`
	AvailabilityImpact      string  `json:"availability_impact"`
	Source                  string  `json:"source"`
	GeneratedOn             string  `json:"generated_on"`
	CVEID                   string  `json:"cve_id"`
	Score                   float64 `json:"score"`
	CalculatedCVSSBaseScore float64 `json:"calculated_cvss_base_score"`
}

type CVSS3Metric struct {
	ID                      int     `json:"id"`
	AttackVector            string  `json:"attack_vector"`
	AttackComplexity        string  `json:"attack_complexity"`
	PrivilegesRequired      string  `json:"privileges_required"`
	UserInteraction         string  `json:"user_interaction"`
	Scope                   string  `json:"scope"`
	ConfidentialityImpact   string  `json:"confidentiality_impact"`
	IntegrityImpact         string  `json:"integrity_impact"`
	AvailabilityImpact      string  `json:"availability_impact"`
	Source                  string  `json:"source"`
	GeneratedOn             string  `json:"generated_on"`
	CVEID                   string  `json:"cve_id"`
	Score                   float64 `json:"score"`
	CalculatedCVSSBaseScore float64 `json:"calculated_cvss_base_score"`
}

type Vendor struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Products []*Product `json:"products"`
}

type Product struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Versions []*Version `json:"versions"`
}

type Version struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Affected string `json:"affected"`
	CPEs     []*CPE `json:"cpe"`
}

type CPE struct {
	CPE  string `json:"cpe"`
	Type string `json:"type"`
}

type Package struct {
	OS              string `json:"OS"`
	OSVersion       string `json:"OSVersion"`
	OSArchitecture  string `json:"OSArchitecture"`
	PackageName     string `json:"packageName"`
	PackageVersion  string `json:"packageVersion"`
	Operator        string `json:"operator"`
	PackageFileName string `json:"packageFileName"`
	Purl            string `json:"purl"`
}

type NVDAdditionalInfo struct {
	CVEID   string `json:"cve_id"`
	Summary string `json:"summary"`
	CWEID   string `json:"cwe_id"`
	// dropped cvss_score, it's already included in the vulndb
	References []struct {
		Source string `json:"source"`
		Name   string `json:"name"`
		URL    string `json:"url"`
	} `json:"references"`
	// droppped vulnerable configurations, as we construct it from vendors.products.versions.cpe
}
