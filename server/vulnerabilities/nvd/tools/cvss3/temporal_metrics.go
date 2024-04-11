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

package cvss3

import "fmt"

type TemporalMetrics struct {
	ExploitCodeMaturity
	RemediationLevel
	ReportConfidence
}

type ExploitCodeMaturity int

const (
	ExploitCodeMaturityNotdefined ExploitCodeMaturity = iota
	ExploitCodeMaturityHigh
	ExploitCodeMaturityFunctional
	ExploitCodeMaturityProofOfConcept
	ExploitCodeMaturityUnproven
)

var (
	weightsExploitCodeMaturity = []float64{1.0, 1.0, 0.97, 0.94, 0.91}
	codeExploitCodeMaturity    = []string{"X", "H", "F", "P", "U"}
)

func (ecm ExploitCodeMaturity) defined() bool {
	return ecm != ExploitCodeMaturityNotdefined
}

func (ecm ExploitCodeMaturity) weight() float64 {
	return weightsExploitCodeMaturity[ecm]
}

func (ecm ExploitCodeMaturity) String() string {
	return codeExploitCodeMaturity[ecm]
}

func (ecm *ExploitCodeMaturity) parse(str string) error {
	idx, found := findIndex(str, codeExploitCodeMaturity)
	if found {
		*ecm = ExploitCodeMaturity(idx)
		return nil
	}
	return fmt.Errorf("illegal exploit code maturity code %s", str)
}

type RemediationLevel int

const (
	RemediationLevelNotdefined RemediationLevel = iota
	RemediationLevelUnavailable
	RemediationLevelWorkaround
	RemediationLevelTemporaryFix
	RemediationLevelOfficialFix
)

var (
	weightsRemediationLevel = []float64{1.0, 1.0, 0.97, 0.96, 0.95}
	codeRemediationLevel    = []string{"X", "U", "W", "T", "O"}
)

func (rl RemediationLevel) defined() bool {
	return rl != RemediationLevelNotdefined
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
	ReportConfidenceNotdefined ReportConfidence = iota
	ReportConfidenceConfirmed
	ReportConfidenceReasonable
	ReportConfidenceUnknown
)

var (
	weightsReportConfidence = []float64{1.0, 1.0, 0.96, 0.92}
	codeReportConfidence    = []string{"X", "C", "R", "U"}
)

func (rc ReportConfidence) defined() bool {
	return rc != ReportConfidenceNotdefined
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
