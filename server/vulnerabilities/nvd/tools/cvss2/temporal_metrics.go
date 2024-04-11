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

package cvss2

import "fmt"

type TemporalMetrics struct {
	Exploitablity
	RemediationLevel
	ReportConfidence
}

type Exploitablity int

const (
	ExploitablityNotDefined Exploitablity = iota
	ExploitablityUnproven
	ExploitablityProofOfConcept
	ExploitablityFunctional
	ExploitablityHigh
)

var (
	weightsExploitablity = []float64{1, 0.85, 0.9, 0.95, 1}
	codeExploitablity    = []string{"ND", "U", "POC", "F", "H"}
)

func (e Exploitablity) defined() bool {
	return e != ExploitablityNotDefined
}

func (e Exploitablity) weight() float64 {
	return weightsExploitablity[e]
}

func (e Exploitablity) String() string {
	return codeExploitablity[e]
}

func (e *Exploitablity) parse(str string) error {
	idx, found := findIndex(str, codeExploitablity)
	if found {
		*e = Exploitablity(idx)
		return nil
	}
	return fmt.Errorf("illegal exploitability code %s", str)
}

type RemediationLevel int

const (
	RemediationLevelNotDefined RemediationLevel = iota
	RemediationLevelOfficialFix
	RemediationLevelTemporaryFix
	RemediationLevelWorkaround
	RemediationLevelUnavailable
)

var (
	weightsRemediationLevel = []float64{1, 0.87, 0.9, 0.95, 1}
	codeRemediationLevel    = []string{"ND", "OF", "TF", "W", "U"}
)

func (rl RemediationLevel) defined() bool {
	return rl != RemediationLevelNotDefined
}

func (rl RemediationLevel) weight() float64 {
	return weightsRemediationLevel[rl]
}

func (rl RemediationLevel) String() string {
	return codeRemediationLevel[rl]
}

func (rl *RemediationLevel) parse(str string) error {
	idx, found := findIndex(str, codeRemediationLevel)
	if found {
		*rl = RemediationLevel(idx)
		return nil
	}
	return fmt.Errorf("illegal remediation level code %s", str)
}

type ReportConfidence int

const (
	ReportConfidenceNotDefined ReportConfidence = iota
	ReportConfidenceUnconfirmed
	ReportConfidenceUncorroborated
	ReportConfidenceConfirmed
)

var (
	weightsReportConfidence = []float64{1, 0.9, 0.95, 1}
	codeReportConfidence    = []string{"ND", "UC", "UR", "C"}
)

func (rc ReportConfidence) defined() bool {
	return rc != ReportConfidenceNotDefined
}

func (rc ReportConfidence) weight() float64 {
	return weightsReportConfidence[rc]
}

func (rc ReportConfidence) String() string {
	return codeReportConfidence[rc]
}

func (rc *ReportConfidence) parse(str string) error {
	idx, found := findIndex(str, codeReportConfidence)
	if found {
		*rc = ReportConfidence(idx)
		return nil
	}
	return fmt.Errorf("illegal report confidence code %s", str)
}
