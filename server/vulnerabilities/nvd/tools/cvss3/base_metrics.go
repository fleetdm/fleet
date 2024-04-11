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

type BaseMetrics struct {
	AttackVector
	AttackComplexity
	PrivilegesRequired
	UserInteraction
	Scope
	Confidentiality
	Integrity
	Availability
}

type AttackVector int

const (
	AttackVectorNetwork AttackVector = iota + 1
	AttackVectorAdjecent
	AttackVectorLocal
	AttackVectorPhysical
)

var (
	weightsAttackVector = []float64{0, 0.85, 0.62, 0.55, 0.2}
	codeAttackVector    = []string{"", "N", "A", "L", "P"}
)

func (av AttackVector) defined() bool {
	return int(av) != 0
}

func (av AttackVector) weight() float64 {
	return weightsAttackVector[av]
}

func (av AttackVector) String() string {
	return codeAttackVector[av]
}

func (av *AttackVector) parse(str string) error {
	idx, found := findIndex(str, codeAttackVector)
	if found {
		*av = AttackVector(idx)
		return nil
	}
	return fmt.Errorf("illegal attack vector code %s", str)
}

type AttackComplexity int

const (
	AttackComplexityLow AttackComplexity = iota + 1
	AttackComplexityHigh
)

var (
	weightsAttackComplexity = []float64{0, 0.77, 0.44}
	codeAttackComplexity    = []string{"", "L", "H"}
)

func (ac AttackComplexity) defined() bool {
	return int(ac) != 0
}

func (ac AttackComplexity) weight() float64 {
	return weightsAttackComplexity[ac]
}

func (ac AttackComplexity) String() string {
	return codeAttackComplexity[ac]
}

func (ac *AttackComplexity) parse(str string) error {
	idx, found := findIndex(str, codeAttackComplexity)
	if found {
		*ac = AttackComplexity(idx)
		return nil
	}
	return fmt.Errorf("illegal attack complexity code %s", str)
}

type PrivilegesRequired int

const (
	PrivilegesRequiredNone PrivilegesRequired = iota + 1
	PrivilegesRequiredLow
	PrivilegesRequiredHigh
)

var (
	weightsPrivilegesRequired = map[bool][]float64{
		false: {0, 0.85, 0.62, 0.27},
		true:  {0, 0.85, 0.68, 0.5},
	}
	codePrivilegesRequired = []string{"", "N", "L", "H"}
)

func (pr PrivilegesRequired) defined() bool {
	return int(pr) != 0
}

func (pr PrivilegesRequired) weight(scopeChanged bool) float64 {
	return weightsPrivilegesRequired[scopeChanged][pr]
}

func (pr PrivilegesRequired) String() string {
	return codePrivilegesRequired[pr]
}

func (pr *PrivilegesRequired) parse(str string) error {
	idx, found := findIndex(str, codePrivilegesRequired)
	if found {
		*pr = PrivilegesRequired(idx)
		return nil
	}
	return fmt.Errorf("illegal privileges required code %s", str)
}

type UserInteraction int

const (
	UserInteractionNone UserInteraction = iota + 1
	UserInteractionRequired
)

var (
	weightsUserInteraction = []float64{0, 0.85, 0.62}
	codeUserInteraction    = []string{"", "N", "R"}
)

func (ui UserInteraction) defined() bool {
	return int(ui) != 0
}

func (ui UserInteraction) weight() float64 {
	return weightsUserInteraction[ui]
}

func (ui UserInteraction) String() string {
	return codeUserInteraction[ui]
}

func (ui *UserInteraction) parse(str string) error {
	idx, found := findIndex(str, codeUserInteraction)
	if found {
		*ui = UserInteraction(idx)
		return nil
	}
	return fmt.Errorf("illegal user interaction code %s", str)
}

type Scope int

const (
	ScopeUnchanged Scope = iota + 1
	ScopeChanged
)

var (
	codeScope = []string{"", "U", "C"}
)

func (s Scope) defined() bool {
	return int(s) != 0
}

func (s Scope) String() string {
	return codeScope[s]
}

func (s *Scope) parse(str string) error {
	idx, found := findIndex(str, codeScope)
	if found {
		*s = Scope(idx)
		return nil
	}
	return fmt.Errorf("illegal scope code %s", str)
}

type Confidentiality int

const (
	ConfidentialityHigh Confidentiality = iota + 1
	ConfidentialityLow
	ConfidentialityNone
)

var (
	weightsConfidentiality = []float64{0, 0.56, 0.22, 0.0}
	codeConfidentiality    = []string{"", "H", "L", "N"}
)

func (c Confidentiality) defined() bool {
	return int(c) != 0
}

func (c Confidentiality) weight() float64 {
	return weightsConfidentiality[c]
}

func (c Confidentiality) String() string {
	return codeConfidentiality[c]
}

func (c *Confidentiality) parse(str string) error {
	idx, found := findIndex(str, codeConfidentiality)
	if found {
		*c = Confidentiality(idx)
		return nil
	}
	return fmt.Errorf("illegal confidentiality code %s", str)
}

type Integrity int

const (
	IntegrityHigh Integrity = iota + 1
	IntegrityLow
	IntegrityNone
)

var (
	weightsIntegrity = []float64{0, 0.56, 0.22, 0.0}
	codeIntegrity    = []string{"", "H", "L", "N"}
)

func (i Integrity) defined() bool {
	return int(i) != 0
}

func (i Integrity) weight() float64 {
	return weightsIntegrity[i]
}

func (i Integrity) String() string {
	return codeIntegrity[i]
}

func (i *Integrity) parse(str string) error {
	idx, found := findIndex(str, codeIntegrity)
	if found {
		*i = Integrity(idx)
		return nil
	}
	return fmt.Errorf("illegal integrity code %s", str)
}

type Availability int

const (
	AvailabilityHigh Availability = iota + 1
	AvailabilityLow
	AvailabilityNone
)

var (
	weightsAvailability = []float64{0, 0.56, 0.22, 0.0}
	codeAvailability    = []string{"", "H", "L", "N"}
)

func (a Availability) defined() bool {
	return int(a) != 0
}

func (a Availability) weight() float64 {
	return weightsAvailability[a]
}

func (a Availability) String() string {
	return codeAvailability[a]
}

func (a *Availability) parse(str string) error {
	idx, found := findIndex(str, codeAvailability)
	if found {
		*a = Availability(idx)
		return nil
	}
	return fmt.Errorf("illegal availability code %s", str)
}
