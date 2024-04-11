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

type EnvironmentalMetrics struct {
	CollateralDamagePotential
	TargetDistribution
	ConfidentialityRequirement
	IntegrityRequirement
	AvailabilityRequirement
	ModifiedExploitablity
	ModifiedRemediationLevel
	ModifiedReportConfidence
}

type CollateralDamagePotential int

const (
	CollateralDamagePotentialNotDefined CollateralDamagePotential = iota
	CollateralDamagePotentialNone
	CollateralDamagePotentialLow
	CollateralDamagePotentialLowMedium
	CollateralDamagePotentialMediumHigh
	CollateralDamagePotentialHigh
)

var (
	weightsCollateralDamagePotential = []float64{0, 0, 0.1, 0.3, 0.4, 0.5}
	codeCollateralDamagePotential    = []string{"ND", "N", "L", "LM", "MH", "H"}
)

func (cdp CollateralDamagePotential) defined() bool {
	return cdp != CollateralDamagePotentialNotDefined
}

func (cdp CollateralDamagePotential) weight() float64 {
	return weightsCollateralDamagePotential[cdp]
}

func (cdp CollateralDamagePotential) String() string {
	return codeCollateralDamagePotential[cdp]
}

func (cdp *CollateralDamagePotential) parse(str string) error {
	idx, found := findIndex(str, codeCollateralDamagePotential)
	if found {
		*cdp = CollateralDamagePotential(idx)
		return nil
	}
	return fmt.Errorf("illegal collateral damage potential code %s", str)
}

type TargetDistribution int

const (
	TargetDistributionNotDefined TargetDistribution = iota
	TargetDistributionNone
	TargetDistributionLow
	TargetDistributionMedium
	TargetDistributionHigh
)

var (
	weightsTargetDistribution = []float64{1, 0, 0.25, 0.75, 1.0}
	codeTargetDistribution    = []string{"ND", "N", "L", "M", "H"}
)

func (td TargetDistribution) defined() bool {
	return td != TargetDistributionNotDefined
}

func (td TargetDistribution) weight() float64 {
	return weightsTargetDistribution[td]
}

func (td TargetDistribution) String() string {
	return codeTargetDistribution[td]
}

func (td *TargetDistribution) parse(str string) error {
	idx, found := findIndex(str, codeTargetDistribution)
	if found {
		*td = TargetDistribution(idx)
		return nil
	}
	return fmt.Errorf("illegal target distribution code %s", str)
}

type ConfidentialityRequirement int

const (
	ConfidentialityRequirementNotdefined ConfidentialityRequirement = iota
	ConfidentialityRequirementLow
	ConfidentialityRequirementMedium
	ConfidentialityRequirementHigh
)

var (
	weightsConfidentialityRequirement = []float64{1.0, 0.5, 1.0, 1.51}
	codeConfidentialityRequirement    = []string{"ND", "L", "M", "H"}
)

func (cr ConfidentialityRequirement) defined() bool {
	return cr != ConfidentialityRequirementNotdefined
}

func (cr ConfidentialityRequirement) weight() float64 {
	return weightsConfidentialityRequirement[cr]
}

func (cr ConfidentialityRequirement) String() string {
	return codeConfidentialityRequirement[cr]
}

func (cr *ConfidentialityRequirement) parse(str string) error {
	idx, found := findIndex(str, codeConfidentialityRequirement)
	if found {
		*cr = ConfidentialityRequirement(idx)
		return nil
	}
	return fmt.Errorf("illegal confidentiality requirement code %s", str)
}

type IntegrityRequirement int

const (
	IntegrityRequirementNotdefined IntegrityRequirement = iota
	IntegrityRequirementLow
	IntegrityRequirementMedium
	IntegrityRequirementHigh
)

var (
	weightsIntegrityRequirement = []float64{1.0, 0.5, 1.0, 1.51}
	codeIntegrityRequirement    = []string{"ND", "L", "M", "H"}
)

func (ir IntegrityRequirement) defined() bool {
	return ir != IntegrityRequirementNotdefined
}

func (ir IntegrityRequirement) weight() float64 {
	return weightsIntegrityRequirement[ir]
}

func (ir IntegrityRequirement) String() string {
	return codeIntegrityRequirement[ir]
}

func (ir *IntegrityRequirement) parse(str string) error {
	idx, found := findIndex(str, codeIntegrityRequirement)
	if found {
		*ir = IntegrityRequirement(idx)
		return nil
	}
	return fmt.Errorf("illegal integrity requirement code %s", str)
}

type AvailabilityRequirement int

const (
	AvailabilityRequirementNotdefined AvailabilityRequirement = iota
	AvailabilityRequirementLow
	AvailabilityRequirementMedium
	AvailabilityRequirementHigh
)

var (
	weightsAvailabilityRequirement = []float64{1.0, 0.5, 1.0, 1.51}
	codeAvailabilityRequirement    = []string{"ND", "L", "M", "H"}
)

func (ar AvailabilityRequirement) defined() bool {
	return ar != AvailabilityRequirementNotdefined
}

func (ar AvailabilityRequirement) weight() float64 {
	return weightsAvailabilityRequirement[ar]
}

func (ar AvailabilityRequirement) String() string {
	return codeAvailabilityRequirement[ar]
}

func (ar *AvailabilityRequirement) parse(str string) error {
	idx, found := findIndex(str, codeAvailabilityRequirement)
	if found {
		*ar = AvailabilityRequirement(idx)
		return nil
	}
	return fmt.Errorf("illegal availability requirement code %s", str)
}

/*
	EXTENDED FUNCTIONALITY
	The following metrics extend the CVSS Specification by allowing Temporal
	metrics to be modified in the Environmental Score.
	If not used they will not be serialized allowing backwards compatibility.
*/

/******** MODIFIED EXPLOITABILITY (ME) ********/

type ModifiedExploitablity Exploitablity

func (mecm ModifiedExploitablity) defined() bool {
	return Exploitablity(mecm).defined()
}

func (mecm ModifiedExploitablity) weight() float64 {
	if !mecm.defined() {
		return 1.00
	}
	return Exploitablity(mecm).weight()
}

func (mecm ModifiedExploitablity) String() string {
	return Exploitablity(mecm).String()
}

func (mecm *ModifiedExploitablity) parse(str string) error {
	a := Exploitablity(*mecm)
	err := a.parse(str)
	*mecm = ModifiedExploitablity(a)
	return err
}

/******** MODIFIED REMEDIATION LEVEL (MRL) ********/

type ModifiedRemediationLevel RemediationLevel

func (mrl ModifiedRemediationLevel) defined() bool {
	return RemediationLevel(mrl).defined()
}

func (mrl ModifiedRemediationLevel) weight() float64 {
	if !mrl.defined() {
		return 1.00
	}
	return RemediationLevel(mrl).weight()
}

func (mrl ModifiedRemediationLevel) String() string {
	return RemediationLevel(mrl).String()
}

func (mrl *ModifiedRemediationLevel) parse(str string) error {
	a := RemediationLevel(*mrl)
	err := a.parse(str)
	*mrl = ModifiedRemediationLevel(a)
	return err
}

/******** MODIFIED REPORT CONFIDENCE LEVEL (MRC) ********/

type ModifiedReportConfidence ReportConfidence

func (mrc ModifiedReportConfidence) defined() bool {
	return ReportConfidence(mrc).defined()
}

func (mrc ModifiedReportConfidence) weight() float64 {
	if !mrc.defined() {
		return 1.00
	}
	return ReportConfidence(mrc).weight()
}

func (mrc ModifiedReportConfidence) String() string {
	return ReportConfidence(mrc).String()
}

func (mrc *ModifiedReportConfidence) parse(str string) error {
	a := ReportConfidence(*mrc)
	err := a.parse(str)
	*mrc = ModifiedReportConfidence(a)
	return err
}
