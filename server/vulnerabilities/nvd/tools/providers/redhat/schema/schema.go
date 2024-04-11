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
	"encoding/json"
	"fmt"
)

// based on this specification
// https://access.redhat.com/documentation/en-us/red_hat_security_data_api/1.0/html-single/red_hat_security_data_api/index#cve_format

type CVEList []struct {
	CVE string `json:"cve"`
	// don't need the rest
}

type CVE struct {
	Name           string `json:"name,omitempty"`
	ThreatSeverity string `json:"threat_severity,omitempty"`
	PublicDate     string `json:"public_date,omitempty"`
	Bugzilla       *struct {
		Description string `json:"description,omitempty"`
		ID          string `json:"id,omitempty"`
		URL         string `json:"url,omitempty"`
	} `json:"bugzilla,omitempty"`
	CVSS *struct {
		BaseScore string `json:"cvss_base_score,omitempty"`
		Vector    string `json:"cvss_scoring_vector,omitempty"`
		Status    string `json:"status,omitempty"`
	} `json:"CVSS,omitempty"`
	CVSS3 *struct {
		BaseScore string `json:"cvss3_base_score,omitempty"`
		Vector    string `json:"cvss3_scoring_vector,omitempty"`
		Status    string `json:"status,omitempty"`
	} `json:"CVSS3,omitempty"`
	CWE             string   `json:"cwe,omitempty"`
	Details         []string `json:"details,omitempty"`
	Statement       string   `json:"statement,omitempty"`
	References      []string `json:"references,omitempty"`
	Acknowledgement string   `json:"acknowledgement,omitempty"`
	Mitigation      *struct {
		Value string `json:"value"`
		Lang  string `json:"lang"`
	} `json:"mitigation,omitempty"`
	UpstreamFix string `json:"upstream_fix,omitempty"`

	// redhat uses a single object instead of an array when there's a single instance of that entity
	// that's why we need to do it manually
	// the types of these are just helper types

	AffectedRelease AffectedReleases `json:"affected_release,omitempty"`
	PackageState    PackageStates    `json:"package_state,omitempty"`
}

type AffectedRelease struct {
	ProductName string `json:"product_name,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"`
	Advisory    string `json:"advisory,omitempty"`
	Package     string `json:"package,omitempty"`
	CPE         string `json:"cpe,omitempty"`
}

type PackageState struct {
	ProductName string `json:"product_name,omitempty"`
	FixState    string `json:"fix_state,omitempty"`
	PackageName string `json:"package_name,omitempty"`
	CPE         string `json:"cpe,omitempty"`
}

// // functions bellow are used to "fix" redhat feed
// // for some parts of the struct, they can either send an object X, or an array of X's when there's multiple of those
// // I try to first decode it as an array. If that fails, try to decode it as an single entity.

// these implement the UnmarshalJSON function, which gets called when we do unmarshal or decode

type AffectedReleases []*AffectedRelease

func (ars *AffectedReleases) UnmarshalJSON(b []byte) error {
	// try to parse it as an array
	var array []*AffectedRelease
	if err := json.Unmarshal(b, &array); err == nil {
		*ars = array
		return nil
	}

	// try to parse it as a single object
	var object AffectedRelease
	if err := json.Unmarshal(b, &object); err == nil {
		*ars = []*AffectedRelease{&object}
		return nil
	}

	return fmt.Errorf("unable to decode affected release as an array nor as a single object")
}

type PackageStates []*PackageState

func (pss *PackageStates) UnmarshalJSON(b []byte) error {
	// try to parse it as an array
	var array []*PackageState
	if err := json.Unmarshal(b, &array); err == nil {
		*pss = array
		return nil
	}

	// try to parse it as a single object
	var object PackageState
	if err := json.Unmarshal(b, &object); err == nil {
		*pss = []*PackageState{&object}
		return nil
	}

	return fmt.Errorf("unable to decode package state as an array nor as a single object")
}
