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
	"strings"
)

const (
	prefix          = "CVSS:"
	partSeparator   = "/"
	metricSeparator = ":"
)

// vector version, int part represents the minor version
// version(1) would mean version 3.1
type version int

func (v version) String() string {
	return fmt.Sprintf("3.%d", int(v))
}

func versionFromString(v string) (version, error) {
	switch v {
	case "3.0":
		return version(0), nil
	case "3.1":
		return version(1), nil
	default:
		return version(0), fmt.Errorf("cvss v3 version %s not supported", v)
	}
}

// Vector represents a CVSSv3 vector, holds all metrics inside (base, temporal and environmental)
type Vector struct {
	version version
	BaseMetrics
	TemporalMetrics
	EnvironmentalMetrics
}

// For some metrics, "undefined" is equivalent to specifying another value. eg.
// E:X is equivalent to E:H.
var undefinedEquivalent = map[string]string{
	// Temporal metrics
	"E":  "H",
	"RL": "U",
	"RC": "C",
	// Environmental metrics
	"CR": "M",
	"IR": "M",
	"AR": "M",
}

func equivalent(metric, value string) string {
	if value != "X" {
		return value
	}
	e, ok := undefinedEquivalent[metric]
	if !ok {
		return value
	}
	return e
}

// Equal returns true if o represents the same vector as v.
//
// Note that the definition of equal here means that two vectors with different
// string representations can still be equal. For instance RL:X is defined as
// the same as RL:U. From the spec:
//
//   Assigning this value indicates there is insufficient information to choose
//   one of the other values, and has no impact on the overall Temporal Score,
//   i.e., it has the same effect on scoring as assigning Unavailable.
//
// https://www.first.org/cvss/specification-document:
func (v Vector) Equal(o Vector) bool {
	vDefs := v.definables()
	oDefs := o.definables()

	for _, metric := range order {
		a := equivalent(metric, vDefs[metric].String())
		b := equivalent(metric, oDefs[metric].String())

		if a != b {
			return false
		}
	}
	return true
}

// String returns this vectors representation as a string
// it shouldn't depend on the order of metrics
func (v Vector) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s%s/", prefix, v.version)

	defineables := v.definables()

	first := true
	for _, metric := range order {
		def := defineables[metric]
		if !def.defined() {
			continue
		}
		if !first {
			fmt.Fprint(&sb, partSeparator)
		} else {
			first = false
		}
		fmt.Fprintf(&sb, "%s%s%s", metric, metricSeparator, def)
	}

	return sb.String()
}

// VectorFromString will parse a string into a Vector, or return an error if it can't be parsed
func VectorFromString(str string) (Vector, error) {
	var v Vector

	// check for prefix and trim it
	str = strings.ToUpper(str)
	if !strings.HasPrefix(str, prefix) {
		return v, fmt.Errorf("vector missing %q prefix: %q", prefix, str)
	}
	str = strings.TrimPrefix(str, prefix)

	// extract version
	slashIdx := strings.IndexByte(str, '/')
	if slashIdx == -1 {
		return v, fmt.Errorf("vector missing '/': %q", str)
	}
	var err error
	if v.version, err = versionFromString(str[:slashIdx]); err != nil {
		return v, err
	}
	str = str[slashIdx+1:]

	// parse all metrics
	parseables := v.parseables()

	for _, part := range strings.Split(str, partSeparator) {
		tmp := strings.Split(part, metricSeparator)
		if len(tmp) != 2 {
			return v, fmt.Errorf("need two values separated by %s, got %q", metricSeparator, part)
		}

		metric, value := tmp[0], tmp[1]
		if p, ok := parseables[metric]; !ok {
			return v, fmt.Errorf("undefined metric %s with value %s", metric, value)
		} else if err := p.parse(value); err != nil {
			return v, fmt.Errorf("error occurred while parsing metric %s: %v", metric, err)
		}
	}

	return v, nil
}

// Absorb will override only metrics in the current vector from the one given which are defined
// If the other vector specifies only a single metric with all others undefined, the resulting
// vector will contain all metrics it previously did, with only the new one overriden
func (v *Vector) Absorb(other Vector) {
	parseables := v.parseables()
	for metric, defineable := range other.definables() {
		if defineable.defined() {
			parseables[metric].parse(defineable.String())
		}
	}
}

// AbsorbIfDefined is like Absorb but will not override vector components that
// are not present in v.
func (v *Vector) AbsorbIfDefined(other Vector) {
	parseables := v.parseables()
	old := v.definables()
	for metric, defineable := range other.definables() {
		if old[metric].defined() && defineable.defined() {
			parseables[metric].parse(defineable.String())
		}
	}
}

// helpers

var order = []string{"AV", "AC", "PR", "UI", "S", "C", "I", "A", "E", "RL", "RC", "CR", "IR", "AR", "MAV", "MAC", "MPR", "MUI", "MS", "MC", "MI", "MA", "ME", "MRL", "MRC"}

type defineable interface {
	defined() bool
	String() string
}

type parseable interface {
	parse(string) error
}

func (v *Vector) definables() map[string]defineable {
	return map[string]defineable{
		// base metrics
		"AV": v.BaseMetrics.AttackVector,
		"AC": v.BaseMetrics.AttackComplexity,
		"PR": v.BaseMetrics.PrivilegesRequired,
		"UI": v.BaseMetrics.UserInteraction,
		"S":  v.BaseMetrics.Scope,
		"C":  v.BaseMetrics.Confidentiality,
		"I":  v.BaseMetrics.Integrity,
		"A":  v.BaseMetrics.Availability,
		// temporal metrics
		"E":  v.TemporalMetrics.ExploitCodeMaturity,
		"RL": v.TemporalMetrics.RemediationLevel,
		"RC": v.TemporalMetrics.ReportConfidence,
		// environmental metrics
		"CR":  v.EnvironmentalMetrics.ConfidentialityRequirement,
		"IR":  v.EnvironmentalMetrics.IntegrityRequirement,
		"AR":  v.EnvironmentalMetrics.AvailabilityRequirement,
		"MAV": v.EnvironmentalMetrics.ModifiedAttackVector,
		"MAC": v.EnvironmentalMetrics.ModifiedAttackComplexity,
		"MPR": v.EnvironmentalMetrics.ModifiedPrivilegesRequired,
		"MUI": v.EnvironmentalMetrics.ModifiedUserInteraction,
		"MS":  v.EnvironmentalMetrics.ModifiedScope,
		"MC":  v.EnvironmentalMetrics.ModifiedConfidentiality,
		"MI":  v.EnvironmentalMetrics.ModifiedIntegrity,
		"MA":  v.EnvironmentalMetrics.ModifiedAvailability,
		"ME":  v.EnvironmentalMetrics.ModifiedExploitCodeMaturity,
		"MRL": v.EnvironmentalMetrics.ModifiedRemediationLevel,
		"MRC": v.EnvironmentalMetrics.ModifiedReportConfidence,
	}
}

func (v *Vector) parseables() map[string]parseable {
	return map[string]parseable{
		// base metrics
		"AV": &v.BaseMetrics.AttackVector,
		"AC": &v.BaseMetrics.AttackComplexity,
		"PR": &v.BaseMetrics.PrivilegesRequired,
		"UI": &v.BaseMetrics.UserInteraction,
		"S":  &v.BaseMetrics.Scope,
		"C":  &v.BaseMetrics.Confidentiality,
		"I":  &v.BaseMetrics.Integrity,
		"A":  &v.BaseMetrics.Availability,
		// temporal metrics
		"E":  &v.TemporalMetrics.ExploitCodeMaturity,
		"RL": &v.TemporalMetrics.RemediationLevel,
		"RC": &v.TemporalMetrics.ReportConfidence,
		// environmental metrics
		"CR":  &v.EnvironmentalMetrics.ConfidentialityRequirement,
		"IR":  &v.EnvironmentalMetrics.IntegrityRequirement,
		"AR":  &v.EnvironmentalMetrics.AvailabilityRequirement,
		"MAV": &v.EnvironmentalMetrics.ModifiedAttackVector,
		"MAC": &v.EnvironmentalMetrics.ModifiedAttackComplexity,
		"MPR": &v.EnvironmentalMetrics.ModifiedPrivilegesRequired,
		"MUI": &v.EnvironmentalMetrics.ModifiedUserInteraction,
		"MS":  &v.EnvironmentalMetrics.ModifiedScope,
		"MC":  &v.EnvironmentalMetrics.ModifiedConfidentiality,
		"MI":  &v.EnvironmentalMetrics.ModifiedIntegrity,
		"MA":  &v.EnvironmentalMetrics.ModifiedAvailability,
		"ME":  &v.EnvironmentalMetrics.ModifiedExploitCodeMaturity,
		"MRL": &v.EnvironmentalMetrics.ModifiedRemediationLevel,
		"MRC": &v.EnvironmentalMetrics.ModifiedReportConfidence,
	}
}
