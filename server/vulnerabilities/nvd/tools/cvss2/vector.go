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

import (
	"fmt"
	"strings"
)

const (
	partSeparator   = "/"
	metricSeparator = ":"
	prefix          = "("
	suffix          = ")"
)

// Vector represents a CVSSv3 vector, holds all metrics inside (base, temporal and environmental)
type Vector struct {
	BaseMetrics
	TemporalMetrics
	EnvironmentalMetrics
}

// For some metrics, "not defined" is equivalent to specifying another value. eg.
// E:X is equivalent to E:H.
var undefinedEquivalent = map[string]string{
	// Temporal metrics
	"E":  "H",
	"RL": "U",
	"RC": "C",
	// Environmental metrics
	"CDP": "N",
	"TD":  "H",
	"CR":  "M",
	"IR":  "M",
	"AR":  "M",
}

func equivalent(metric, value string) string {
	if value != "ND" {
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
// string representations can still be equal. For instance TD:ND is defined as
// the same as TD:H.
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
	fmt.Fprint(&sb, prefix)

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

	fmt.Fprint(&sb, suffix)

	return sb.String()
}

// VectorFromString will parse a string into a Vector, or return an error if it can't be parsed
func VectorFromString(str string) (Vector, error) {
	var v Vector
	str = strings.TrimPrefix(str, prefix)
	str = strings.TrimSuffix(str, suffix)
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

var order = []string{"AV", "AC", "Au", "C", "I", "A", "E", "RL", "RC", "CDP", "TD", "CR", "IR", "AR", "ME", "MRL", "MRC"}

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
		"AV": v.BaseMetrics.AccessVector,
		"AC": v.BaseMetrics.AccessComplexity,
		"Au": v.BaseMetrics.Authentication,
		"C":  v.BaseMetrics.ConfidentialityImpact,
		"I":  v.BaseMetrics.IntegrityImpact,
		"A":  v.BaseMetrics.AvailabilityImpact,
		// temporal metrics
		"E":  v.TemporalMetrics.Exploitablity,
		"RL": v.TemporalMetrics.RemediationLevel,
		"RC": v.TemporalMetrics.ReportConfidence,
		// environmental metrics
		"CDP": v.EnvironmentalMetrics.CollateralDamagePotential,
		"TD":  v.EnvironmentalMetrics.TargetDistribution,
		"CR":  v.EnvironmentalMetrics.ConfidentialityRequirement,
		"IR":  v.EnvironmentalMetrics.IntegrityRequirement,
		"AR":  v.EnvironmentalMetrics.AvailabilityRequirement,
		// extended environmental metrics
		"ME":  v.EnvironmentalMetrics.ModifiedExploitablity,
		"MRL": v.EnvironmentalMetrics.ModifiedRemediationLevel,
		"MRC": v.EnvironmentalMetrics.ModifiedReportConfidence,
	}
}

func (v *Vector) parseables() map[string]parseable {
	return map[string]parseable{
		// base metrics
		"AV": &v.BaseMetrics.AccessVector,
		"AC": &v.BaseMetrics.AccessComplexity,
		"Au": &v.BaseMetrics.Authentication,
		"C":  &v.BaseMetrics.ConfidentialityImpact,
		"I":  &v.BaseMetrics.IntegrityImpact,
		"A":  &v.BaseMetrics.AvailabilityImpact,
		// temporal metrics
		"E":  &v.TemporalMetrics.Exploitablity,
		"RL": &v.TemporalMetrics.RemediationLevel,
		"RC": &v.TemporalMetrics.ReportConfidence,
		// environmental metrics
		"CDP": &v.EnvironmentalMetrics.CollateralDamagePotential,
		"TD":  &v.EnvironmentalMetrics.TargetDistribution,
		"CR":  &v.EnvironmentalMetrics.ConfidentialityRequirement,
		"IR":  &v.EnvironmentalMetrics.IntegrityRequirement,
		"AR":  &v.EnvironmentalMetrics.AvailabilityRequirement,
		// extended environmental metrics
		"ME":  &v.EnvironmentalMetrics.ModifiedExploitablity,
		"MRL": &v.EnvironmentalMetrics.ModifiedRemediationLevel,
		"MRC": &v.EnvironmentalMetrics.ModifiedReportConfidence,
	}
}
