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

import (
	"fmt"
	"math"
)

func roundUp(x float64) float64 {
	// round up to one decimal
	return math.Ceil(x*10) / 10
}

// Validate should be called before calculating any scores on vector
// If there's an error, there's no guarantee that a call to *Score() won't panic
func (v Vector) Validate() error {
	switch {
	case !v.BaseMetrics.AttackVector.defined():
		return fmt.Errorf("base metric attack vector not defined")
	case !v.BaseMetrics.AttackComplexity.defined():
		return fmt.Errorf("base metric attack complexity not defined")
	case !v.BaseMetrics.PrivilegesRequired.defined():
		return fmt.Errorf("base metric privileges required not defined")
	case !v.BaseMetrics.UserInteraction.defined():
		return fmt.Errorf("base metric user interaction not defined")
	case !v.BaseMetrics.Scope.defined():
		return fmt.Errorf("base metric scope not defined")
	case !v.BaseMetrics.Confidentiality.defined():
		return fmt.Errorf("base metric confidentiality not defined")
	case !v.BaseMetrics.Integrity.defined():
		return fmt.Errorf("base metric integrity not defined")
	case !v.BaseMetrics.Availability.defined():
		return fmt.Errorf("base metric availability not defined")
	default:
		return nil
	}
}

// Score = combined score for the whole Vector
func (v Vector) Score() float64 {
	// combines all of them
	return v.EnvironmentalScore()
}

// BaseScore returns base score of the vector
func (v Vector) BaseScore() float64 {
	i, e := v.impactScore(), v.exploitabilityScore()
	if i <= 0 {
		return 0
	}
	c := 1.0
	if v.baseScopeChanged() {
		c = 1.08
	}
	return roundUp(math.Min(c*(e+i), 10.0))
}

func (v Vector) impactScore() float64 {
	iscBase := 1 -
		(1-v.BaseMetrics.Confidentiality.weight())*
			(1-v.BaseMetrics.Integrity.weight())*
			(1-v.BaseMetrics.Availability.weight())
	if v.baseScopeChanged() {
		return 7.52*(iscBase-0.029) - 3.25*math.Pow((iscBase-0.02), 15)
	}
	return 6.42 * iscBase
}

func (v Vector) exploitabilityScore() float64 {
	return 8.22 *
		v.BaseMetrics.AttackVector.weight() *
		v.BaseMetrics.AttackComplexity.weight() *
		v.BaseMetrics.PrivilegesRequired.weight(v.baseScopeChanged()) *
		v.BaseMetrics.UserInteraction.weight()
}

// TemporalScore returns temporal score of the vector
func (v Vector) TemporalScore() float64 {
	return roundUp(v.BaseScore() *
		v.TemporalMetrics.ExploitCodeMaturity.weight() *
		v.TemporalMetrics.RemediationLevel.weight() *
		v.TemporalMetrics.ReportConfidence.weight())
}

// EnvironmentalScore returns environmental score of the vector
func (v Vector) EnvironmentalScore() float64 {
	i, e := v.modifiedImpactScore(), v.modifiedExploitabilityScore()
	if i < 0 {
		return 0
	}
	c := 1.0
	if v.modifiedScopeChanged() {
		c = 1.08
	}

	modifiedTemporalMetricsMult := v.modifiedTemporalMetricsMult()

	return roundUp(roundUp(math.Min(c*(e+i), 10.0)) *
		modifiedTemporalMetricsMult)
}

func (v Vector) modifiedTemporalMetricsMult() float64 {
	var me, mrl, mrc float64

	if v.EnvironmentalMetrics.ModifiedExploitCodeMaturity.defined() {
		me = v.EnvironmentalMetrics.ModifiedExploitCodeMaturity.weight()
	} else {
		me = v.TemporalMetrics.ExploitCodeMaturity.weight()
	}

	if v.EnvironmentalMetrics.ModifiedRemediationLevel.defined() {
		mrl = v.EnvironmentalMetrics.ModifiedRemediationLevel.weight()
	} else {
		mrl = v.TemporalMetrics.RemediationLevel.weight()
	}

	if v.EnvironmentalMetrics.ModifiedReportConfidence.defined() {
		mrc = v.EnvironmentalMetrics.ModifiedReportConfidence.weight()
	} else {
		mrc = v.TemporalMetrics.ReportConfidence.weight()
	}

	return me * mrl * mrc
}

func (v Vector) modifiedImpactScore() float64 {
	var mc, mi, ma float64

	if v.EnvironmentalMetrics.ModifiedConfidentiality.defined() {
		mc = v.EnvironmentalMetrics.ModifiedConfidentiality.weight()
	} else {
		mc = v.BaseMetrics.Confidentiality.weight()
	}

	if v.EnvironmentalMetrics.ModifiedIntegrity.defined() {
		mi = v.EnvironmentalMetrics.ModifiedIntegrity.weight()
	} else {
		mi = v.BaseMetrics.Integrity.weight()
	}

	if v.EnvironmentalMetrics.ModifiedAvailability.defined() {
		ma = v.EnvironmentalMetrics.ModifiedAvailability.weight()
	} else {
		ma = v.BaseMetrics.Availability.weight()
	}

	iscModified := math.Min(
		1-(1-mc*v.EnvironmentalMetrics.ConfidentialityRequirement.weight())*
			(1-mi*v.EnvironmentalMetrics.IntegrityRequirement.weight())*
			(1-ma*v.EnvironmentalMetrics.AvailabilityRequirement.weight()),
		0.915,
	)
	if v.modifiedScopeChanged() {
		switch v.version {
		case version(1):
			return 7.52*(iscModified-0.029) - 3.25*math.Pow((iscModified*0.9731-0.02), 13)
		case version(0):
			fallthrough
		default:
			return 7.52*(iscModified-0.029) - 3.25*math.Pow((iscModified-0.02), 15)
		}
	} else {
		return 6.42 * iscModified
	}
}

func (v Vector) modifiedExploitabilityScore() float64 {
	var mav, mac, mpr, mui float64

	if v.EnvironmentalMetrics.ModifiedAttackVector.defined() {
		mav = v.EnvironmentalMetrics.ModifiedAttackVector.weight()
	} else {
		mav = v.BaseMetrics.AttackVector.weight()
	}

	if v.EnvironmentalMetrics.ModifiedAttackComplexity.defined() {
		mac = v.EnvironmentalMetrics.ModifiedAttackComplexity.weight()
	} else {
		mac = v.BaseMetrics.AttackComplexity.weight()
	}

	if v.EnvironmentalMetrics.ModifiedPrivilegesRequired.defined() {
		mpr = v.EnvironmentalMetrics.ModifiedPrivilegesRequired.weight(v.modifiedScopeChanged())
	} else {
		mpr = v.BaseMetrics.PrivilegesRequired.weight(v.modifiedScopeChanged())
	}

	if v.EnvironmentalMetrics.ModifiedUserInteraction.defined() {
		mui = v.EnvironmentalMetrics.ModifiedUserInteraction.weight()
	} else {
		mui = v.BaseMetrics.UserInteraction.weight()
	}

	return 8.22 * mav * mac * mpr * mui
}

// scope functions

func (v Vector) baseScopeChanged() bool {
	return v.BaseMetrics.Scope == ScopeChanged
}

func (v Vector) modifiedScopeChanged() bool {
	if v.EnvironmentalMetrics.ModifiedScope.defined() {
		return v.EnvironmentalMetrics.ModifiedScope == ModifiedScope(ScopeChanged)
	}
	return v.baseScopeChanged()
}
