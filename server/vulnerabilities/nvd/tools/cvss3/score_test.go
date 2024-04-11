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
	"testing"
)

func TestRoundUp(t *testing.T) {
	cases := map[float64]float64{
		1.50:  1.5,
		1.51:  1.6,
		1.54:  1.6,
		1.55:  1.6,
		1.56:  1.6,
		1.59:  1.6,
		-1.50: -1.5,
		-1.51: -1.5,
		-1.54: -1.5,
		-1.55: -1.5,
		-1.56: -1.5,
		-1.59: -1.5,
	}

	for x, expected := range cases {
		t.Run(fmt.Sprintf("roundUp(%.2f)=%.1f", x, expected), func(t *testing.T) {
			if actual := roundUp(x); expected != actual {
				t.Errorf("expected %.1f, actual %.1f", expected, actual)
			}
		})
	}
}

func TestBaseScores(t *testing.T) {
	cases := []struct {
		str       string
		baseScore float64
	}{
		{"CVSS:3.0/AV:A/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H/E:U/RL:U/RC:C", 6.8},
		{"CVSS:3.0/AV:L/AC:H/PR:L/UI:R/S:C/C:H/I:H/A:H/E:U/RL:O/RC:C", 7.5},
		{"CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H/E:H/RL:O/RC:C", 8.8},
		{"CVSS:3.0/AV:N/AC:L/PR:H/UI:N/S:U/C:N/I:N/A:H/E:U/RL:O/RC:C", 4.9},
		{"CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:C/C:N/I:H/A:N/E:U/RL:T/RC:C", 7.4},
		{"CVSS:3.0/AV:A/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N/E:U/RL:U/RC:C", 3.1},
		{"CVSS:3.0/AV:N/AC:L/PR:H/UI:N/S:U/C:L/I:N/A:N/E:U/RL:O/RC:C", 2.7},
		{"CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:C/C:N/I:L/A:N/E:U/RL:O/RC:C", 4.7},
		{"CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L/E:U/RL:O/RC:C", 5.6},
		{"CVSS:3.0/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N/E:U/RL:T/RC:C", 5.4},
		{"CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:N/E:U/RL:T/RC:C", 5.5},
		{"CVSS:3.0/AV:A/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N/E:U/RL:O/RC:C", 5.3},
		{"CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H/E:U/RL:O/RC:C", 5.5},
		{"CVSS:3.0/AV:L/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H/E:U/RL:O/RC:C", 6.7},
		{"CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H/E:U/RL:O/RC:C", 5.9},
		{"CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N/E:P/RL:U/RC:C", 4.3},
		{"CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H/E:U/RL:O/RC:C", 6.5},
		{"CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N/E:U/RL:O/RC:C", 4.3},
		{"CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N/E:U/RL:O/RC:C", 7.4},
		{"CVSS:3.0/AV:N/AC:H/PR:N/UI:R/S:U/C:N/I:L/A:N/E:U/RL:O/RC:C", 3.1},
		{"CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H/E:U/RL:T/RC:C", 7.8},
		{"CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L/E:U/RL:U/RC:C", 5.6},
		{"CVSS:3.0/AV:A/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N/E:U/RL:O/RC:C", 3.1},
		{"CVSS:3.0/AV:A/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N/E:U/RL:O/RC:C", 4.8},
		{"CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H/E:U/RL:T/RC:C", 6.5},
		{"CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N", 0.0},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			v, err := VectorFromString(c.str)
			if err != nil {
				t.Errorf("parse error: %v", err)
			}
			if vbs := v.BaseScore(); vbs != c.baseScore {
				t.Errorf("expected %.1f, got %.1f", c.baseScore, vbs)
			}
		})
	}
}

func TestScores(t *testing.T) {
	// here I took a random vector and just switched all possible values for scope and modified scope
	// since scope is the only metric that affects others. The scores were validated on https://www.first.org/cvss/calculator/3.0
	cases := []struct {
		str           string
		base          float64
		temporal      float64
		environmental float64
	}{
		// scope changed, all values of modified scope
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:C/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:C/MC:L/MA:N", 6.4, 5.7, 7.1},
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:C/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:U/MC:L/MA:N", 6.4, 5.7, 6.1},
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:C/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:X/MC:L/MA:N", 6.4, 5.7, 7.1},

		// scope unchanged, all values of modified scope
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:C/MC:L/MA:N", 5.1, 4.5, 7.1},
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:U/MC:L/MA:N", 5.1, 4.5, 6.1},
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:X/MC:L/MA:N", 5.1, 4.5, 6.1},

		// Extended functionality: defined Modified Temporal metrics should override temporal metrics
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:X/MC:L/MA:N/ME:F", 5.1, 4.5, 6.3},
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:X/MC:L/MA:N/MRL:T", 5.1, 4.5, 6.0},
		{"CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:X/MC:L/MA:N/MRC:U", 5.1, 4.5, 5.8},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			v, err := VectorFromString(c.str)
			if err != nil {
				t.Errorf("parse error: %v", err)
			}
			if vbs := v.BaseScore(); vbs != c.base {
				t.Errorf("expected %.1f, got %.1f", c.base, vbs)
			}
			if vts := v.TemporalScore(); vts != c.temporal {
				t.Errorf("expected %.1f, got %.1f", c.temporal, vts)
			}
			if ves := v.EnvironmentalScore(); ves != c.environmental {
				t.Errorf("expected %.1f, got %.1f", c.environmental, ves)
			}
		})
	}
}

func TestScoresV30V31(t *testing.T) {
	vec := "AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H/E:U/RL:T/RC:U/CR:L/IR:L/AR:H/MAV:P/MAC:H/MPR:H/MUI:R/MS:C/MC:H/MI:H/MA:H"

	for _, c := range []struct {
		ver                           version
		base, temporal, environmental float64
	}{
		{version(0), 10.0, 8.1, 5.5},
		{version(1), 10.0, 8.1, 5.6},
	} {
		fullVec := fmt.Sprintf("%s%s/%s", prefix, c.ver, vec)
		v, err := VectorFromString(fullVec)
		if err != nil {
			t.Fatal(err)
		}

		if base := v.BaseScore(); base != c.base {
			t.Fatalf("v %s: base score wrong: have %.1f, want %.1f", c.ver, base, c.base)
		}
		if temporal := v.TemporalScore(); temporal != c.temporal {
			t.Fatalf("v %s: temporal score wrong: have %.1f, want %.1f", c.ver, temporal, c.temporal)
		}
		if environmental := v.EnvironmentalScore(); environmental != c.environmental {
			t.Fatalf("v %s: environmental score wrong: have %.1f, want %.1f", c.ver, environmental, c.environmental)
		}
	}
}
