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

type Advisory struct {
	Credits                  []string               `json:"credits"`
	CveIDs                   []string               `json:"cve_ids"`
	CVSSDetails              []*CVSSDetails         `json:"cvss_details"`
	CVSSV3BaseScore          float64                `json:"cvss_v3_base_score"`
	CVSSV3Vector             string                 `json:"cvss_v3_vector"`
	CweIDs                   []string               `json:"cwe_ids"`
	Description              string                 `json:"description"`
	DescriptionOverview      string                 `json:"description_overview"`
	DescriptionRemediation   string                 `json:"description_remediation"`
	Disclosed                string                 `json:"disclosed"`
	Ecosystem                string                 `json:"ecosystem"`
	ExploitCodeMaturity      string                 `json:"exploit_code_maturity"`
	InitiallyFixedInVersions []string               `json:"initially_fixed_in_versions"`
	IsFixable                bool                   `json:"is_fixable"`
	IsMalicious              bool                   `json:"is_malicious"`
	IsSocialMediaTrending    bool                   `json:"is_social_media_trending"`
	Modified                 string                 `json:"modified"`
	Package                  string                 `json:"package"`
	PackageRepositoryURL     string                 `json:"package_repository_url"`
	Published                string                 `json:"published"`
	References               []*Reference           `json:"references"`
	Severity                 string                 `json:"severity"`
	SnykAdvisoryURL          string                 `json:"snyk_advisory_url"`
	SnykID                   string                 `json:"snyk_id"`
	Title                    string                 `json:"title"`
	VulnerableFunctions      []*VulnerableFunctions `json:"vulnerable_functions"`
	VulnerableHashRanges     []string               `json:"vulnerable_hash_ranges,omitempty"`
	VulnerableHashes         []string               `json:"vulnerable_hashes,omitempty"`
	VulnerableVersions       []string               `json:"vulnerable_versions"`
}

type VulnerableFunctions struct {
	FunctionID struct {
		ClassName    string `json:"class_name"`
		FilePath     string `json:"file_path"`
		FunctionName string `json:"function_name"`
	} `json:"function_id"`
	Version []string `json:"version"`
}

type CVSSDetails struct {
	Assigner        string  `json:"assigner"`
	CVSSV3Vector    string  `json:"cvss_v3_vector"`
	CVSSV3BaseScore float64 `json:"cvss_v3_base_score"`
	Severity        string  `json:"severity"`
	Modified        string  `json:"modified"`
}

type Reference struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type RestAPI struct {
	JSONAPI struct {
		Version string `json:"version"`
	} `json:"jsonapi"`
	Data struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"data"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}
