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
	"testing"
)

func TestRoundTo1Decimal(t *testing.T) {
	tests := map[float64]float64{
		1.50:  1.5,
		1.51:  1.5,
		1.54:  1.5,
		1.55:  1.6,
		1.56:  1.6,
		1.59:  1.6,
		-1.50: -1.5,
		-1.51: -1.5,
		-1.54: -1.5,
		-1.55: -1.6,
		-1.56: -1.6,
		-1.59: -1.6,
	}

	for x, expected := range tests {
		t.Run(fmt.Sprintf("roundTo1Decimal(%.2f)=%.1f", x, expected), func(t *testing.T) {
			if actual := roundTo1Decimal(x); expected != actual {
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
		{"(AV:N/AC:M/Au:S/C:P/I:N/A:N/E:U/RL:OF/RC:C)", 3.5},
		{"(AV:A/AC:L/Au:S/C:P/I:N/A:P/E:U/RL:OF/RC:C)", 4.1},
		{"(AV:L/AC:L/Au:S/C:P/I:N/A:C/E:U/RL:OF/RC:C)", 5.2},
		{"(AV:A/AC:L/Au:S/C:N/I:P/A:P/E:U/RL:OF/RC:C)", 4.1},
		{"(AV:N/AC:L/Au:N/C:P/I:N/A:N/E:POC/RL:U/RC:C)", 5},
		{"(AV:A/AC:L/Au:N/C:C/I:C/A:C/E:U/RL:OF/RC:C)", 8.3},
		{"(AV:L/AC:L/Au:S/C:N/I:N/A:C/E:U/RL:OF/RC:C)", 4.6},
		{"(AV:A/AC:L/Au:N/C:N/I:N/A:C/E:U/RL:OF/RC:C)", 6.1},
		{"(AV:L/AC:L/Au:S/C:P/I:N/A:N/E:U/RL:OF/RC:C)", 1.7},
		{"(AV:L/AC:M/Au:S/C:P/I:N/A:N/E:U/RL:OF/RC:C)", 1.5},
		{"(AV:A/AC:L/Au:N/C:N/I:N/A:P/E:U/RL:TF/RC:C)", 3.3},
		{"(AV:L/AC:L/Au:S/C:N/I:P/A:N/E:U/RL:OF/RC:C)", 1.7},
		{"(AV:L/AC:H/Au:S/C:P/I:P/A:P/E:U/RL:OF/RC:C)", 3.5},
		{"(AV:N/AC:L/Au:N/C:C/I:C/A:C/E:H/RL:OF/RC:C)", 10},
		{"(AV:A/AC:L/Au:N/C:P/I:N/A:C/E:U/RL:TF/RC:C)", 6.8},
		{"(AV:N/AC:M/Au:N/C:P/I:P/A:P/E:U/RL:TF/RC:C)", 6.8},
		{"(AV:L/AC:M/Au:S/C:C/I:C/A:C/E:U/RL:OF/RC:C)", 6.6},
		{"(AV:N/AC:L/Au:N/C:C/I:C/A:C/E:U/RL:ND/RC:C)", 10},
		{"(AV:A/AC:L/Au:N/C:N/I:P/A:N/E:U/RL:OF/RC:C)", 3.3},
		{"(AV:N/AC:H/Au:N/C:N/I:P/A:N/E:U/RL:OF/RC:C)", 2.6},
		{"(AV:N/AC:L/Au:N/C:C/I:C/A:C/E:U/RL:TF/RC:C)", 10},
		{"(AV:A/AC:M/Au:N/C:P/I:P/A:N/E:U/RL:OF/RC:C)", 4.3},
		{"(AV:A/AC:L/Au:N/C:N/I:N/A:P/E:U/RL:U/RC:C)", 3.3},
		{"(AV:A/AC:L/Au:S/C:N/I:N/A:C/E:U/RL:OF/RC:C)", 5.5},
		{"(AV:L/AC:L/Au:S/C:N/I:N/A:P/E:U/RL:OF/RC:C)", 1.7},
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
	// random vector chosen and validated at:
	// https://nvd.nist.gov/vuln-metrics/cvss/v2-calculator?calculator&adv&version=2

	cases := []struct {
		str           string
		base          float64
		temporal      float64
		environmental float64
	}{
		{"(AV:A/AC:L/Au:S/C:C/I:P/A:C/E:F/RL:W/RC:UR/CDP:MH/TD:M/CR:M/IR:L/AR:H)", 7.4, 6.3, 6.0},

		// Extended functionality: defined Modified Temporal metrics should override temporal metrics
		{"(AV:A/AC:L/Au:S/C:C/I:P/A:C/E:F/RL:W/RC:UR/CDP:MH/TD:M/CR:M/IR:L/AR:H/MRC:UC)", 7.4, 6.3, 5.8},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			v, err := VectorFromString(c.str)
			if err != nil {
				t.Errorf("parse error: %v", err)
			}
			if vbs := v.BaseScore(); vbs != c.base {
				t.Errorf("base score expected to be %.1f, got %.1f", c.base, vbs)
			}
			if vts := v.TemporalScore(); vts != c.temporal {
				t.Errorf("temporal score expected to be %.1f, got %.1f", c.temporal, vts)
			}
			if ves := v.EnvironmentalScore(); ves != c.environmental {
				t.Errorf("environmental score expected to be %.1f, got %.1f", c.environmental, ves)
			}
		})
	}
}

func BenchmarkScore(b *testing.B) {
	v, err := VectorFromString("(AV:A/AC:L/Au:S/C:C/I:P/A:C/E:F/RL:W/RC:UR/CDP:MH/TD:M/CR:M/IR:L/AR:H)")
	if err != nil {
		b.Errorf("parse error: %v", err)
	}
	for i := 0; i < b.N; i++ {
		v.Score()
	}
}
