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

type BaseMetrics struct {
	AccessVector
	AccessComplexity
	Authentication
	ConfidentialityImpact
	IntegrityImpact
	AvailabilityImpact
}

type AccessVector int

const (
	AccessVectorLocal AccessVector = iota + 1
	AccessVectorAdjecentNetwork
	AccessVectorNetwork
)

var (
	weightsAccessVector = []float64{0, 0.395, 0.646, 1}
	codeAccessVector    = []string{"", "L", "A", "N"}
)

func (av AccessVector) defined() bool {
	return int(av) != 0
}

func (av AccessVector) weight() float64 {
	return weightsAccessVector[av]
}

func (av AccessVector) String() string {
	return codeAccessVector[av]
}

func (av *AccessVector) parse(str string) error {
	idx, found := findIndex(str, codeAccessVector)
	if found {
		*av = AccessVector(idx)
		return nil
	}
	return fmt.Errorf("illegal access vector code %s", str)
}

type AccessComplexity int

const (
	AccessComplexityHigh AccessComplexity = iota + 1
	AccessComplexityMedium
	AccessComplexityLow
)

var (
	weightsAccessComplexity = []float64{0, 0.35, 0.61, 0.71}
	codeAccessComplexity    = []string{"", "H", "M", "L"}
)

func (ac AccessComplexity) defined() bool {
	return int(ac) != 0
}

func (ac AccessComplexity) weight() float64 {
	return weightsAccessComplexity[ac]
}

func (ac AccessComplexity) String() string {
	return codeAccessComplexity[ac]
}

func (ac *AccessComplexity) parse(str string) error {
	idx, found := findIndex(str, codeAccessComplexity)
	if found {
		*ac = AccessComplexity(idx)
		return nil
	}
	return fmt.Errorf("illegal access complexity code %s", str)
}

type Authentication int

const (
	AuthenticationMultiple Authentication = iota + 1
	AuthenticationSingle
	AuthenticationNone
)

var (
	weightsAuthentication = []float64{0, 0.45, 0.56, 0.704}
	codeAuthentication    = []string{"", "M", "S", "N"}
)

func (au Authentication) defined() bool {
	return int(au) != 0
}

func (au Authentication) weight() float64 {
	return weightsAuthentication[au]
}

func (au Authentication) String() string {
	return codeAuthentication[au]
}

func (au *Authentication) parse(str string) error {
	idx, found := findIndex(str, codeAuthentication)
	if found {
		*au = Authentication(idx)
		return nil
	}
	return fmt.Errorf("illegal authentication code %s", str)
}

type ConfidentialityImpact int

const (
	ConfidentialityImpactNone ConfidentialityImpact = iota + 1
	ConfidentialityImpactPartial
	ConfidentialityImpactComplete
)

var (
	weightsConfidentialityImpact = []float64{0, 0, 0.275, 0.66}
	codeConfidentialityImpact    = []string{"", "N", "P", "C"}
)

func (ci ConfidentialityImpact) defined() bool {
	return int(ci) != 0
}

func (ci ConfidentialityImpact) weight() float64 {
	return weightsConfidentialityImpact[ci]
}

func (ci ConfidentialityImpact) String() string {
	return codeConfidentialityImpact[ci]
}

func (ci *ConfidentialityImpact) parse(str string) error {
	idx, found := findIndex(str, codeConfidentialityImpact)
	if found {
		*ci = ConfidentialityImpact(idx)
		return nil
	}
	return fmt.Errorf("illegal confidentiality impact %s", str)
}

type IntegrityImpact int

const (
	IntegerityImpactNone IntegrityImpact = iota + 1
	IntegrityImpactPartial
	IntegrityImpactComplete
)

var (
	weightsIntegrityImpact = []float64{0, 0, 0.275, 0.66}
	codeIntegrityImpact    = []string{"", "N", "P", "C"}
)

func (ii IntegrityImpact) defined() bool {
	return int(ii) != 0
}

func (ii IntegrityImpact) weight() float64 {
	return weightsIntegrityImpact[ii]
}

func (ii IntegrityImpact) String() string {
	return codeIntegrityImpact[ii]
}

func (ii *IntegrityImpact) parse(str string) error {
	idx, found := findIndex(str, codeIntegrityImpact)
	if found {
		*ii = IntegrityImpact(idx)
		return nil
	}
	return fmt.Errorf("illegal integrity impact code %s", str)
}

type AvailabilityImpact int

const (
	AvailabilityImpactNone AvailabilityImpact = iota + 1
	AvailabilityImpactPartial
	AvailabilityImpactComplete
)

var (
	weightsAvailabilityImpact = []float64{0, 0, 0.275, 0.66}
	codeAvailabilityImpact    = []string{"", "N", "P", "C"}
)

func (ai AvailabilityImpact) defined() bool {
	return int(ai) != 0
}

func (ai AvailabilityImpact) weight() float64 {
	return weightsAvailabilityImpact[ai]
}

func (ai AvailabilityImpact) String() string {
	return codeAvailabilityImpact[ai]
}

func (ai *AvailabilityImpact) parse(str string) error {
	idx, found := findIndex(str, codeAvailabilityImpact)
	if found {
		*ai = AvailabilityImpact(idx)
		return nil
	}
	return fmt.Errorf("illegal availability impact code %s", str)
}
