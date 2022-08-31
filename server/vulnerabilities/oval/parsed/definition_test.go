package oval_parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestOvalParsedDefinition(t *testing.T) {
	t.Run("#CveVulnerabilities", func(t *testing.T) {
		t.Run("only returns cve vulnerabilities", func(t *testing.T) {
			sut := Definition{
				Vulnerabilities: []string{
					"CVE-2022-0001",
					"CVE-2022-0002",
					"USN-5469-1",
					"CVE-2022-0003",
					"RHSA-2022:5555",
				},
			}

			require.ElementsMatch(t, sut.CveVulnerabilities(), []string{
				"CVE-2022-0001",
				"CVE-2022-0002",
				"CVE-2022-0003",
			})
		})
	})

	t.Run("#Eval", func(t *testing.T) {
		t.Run("no root criteria", func(t *testing.T) {
			sut := Definition{}
			require.False(t, sut.Eval(nil, nil))
		})

		t.Run("with empty test results", func(t *testing.T) {
			criteria := Criteria{
				And,
				[]int{1, 2, 3},
				nil,
			}
			sut := Definition{Criteria: &criteria}
			require.False(t, sut.Eval(nil, nil))
			require.False(t, sut.Eval(make(map[int]bool), make(map[int][]fleet.Software)))
		})

		t.Run("with OS tests result only", func(t *testing.T) {
			criteria := Criteria{
				And,
				[]int{1, 2, 3},
				nil,
			}
			sut := Definition{Criteria: &criteria}
			OSTstResults := map[int]bool{
				1: true,
				2: true,
				3: true,
			}
			require.True(t, sut.Eval(OSTstResults, nil))
		})

		t.Run("with single level criteria", func(t *testing.T) {
			cases := []struct {
				op       OperatorType
				expected bool
			}{
				{And, false},
				{Or, true},
			}

			for _, c := range cases {
				criteria := Criteria{
					c.op,
					[]int{1, 2, 3},
					nil,
				}
				OSTsts := make(map[int]bool)
				pkgTsts := map[int][]fleet.Software{
					1: {{ID: 1}},
					2: nil,
					3: {{ID: 2}},
				}
				sut := Definition{
					&criteria,
					nil,
				}

				require.Equal(t, c.expected, sut.Eval(OSTsts, pkgTsts))
			}
		})

		t.Run("simple logic tree", func(t *testing.T) {
			//     OR
			//  /   |   \
			// 1:F 2:F AND
			//        /   \
			//      3:T   4:T

			leaf := Criteria{
				And,
				[]int{3, 4},
				nil,
			}
			root := Criteria{
				Or,
				[]int{1, 2},
				[]*Criteria{&leaf},
			}

			OSTsts := make(map[int]bool)
			pkgTsts := map[int][]fleet.Software{
				1: nil,
				2: nil,
				3: {{ID: 2}},
				4: {{ID: 3}},
			}

			sut := Definition{
				&root,
				nil,
			}

			require.True(t, sut.Eval(OSTsts, pkgTsts))
		})

		t.Run("tree reference non-existing test", func(t *testing.T) {
			//       OR
			//  /     |     \
			// 1:n/a  2:F   3:T

			root := Criteria{
				Or,
				[]int{1, 2, 3},
				nil,
			}

			OSTsts := make(map[int]bool)
			pkgTsts := map[int][]fleet.Software{
				2: nil,
				3: {{ID: 2}},
			}

			sut := Definition{
				&root,
				nil,
			}

			require.False(t, sut.Eval(OSTsts, pkgTsts))
		})

		t.Run("deep tree", func(t *testing.T) {
			// 		OR
			// 	   /  \
			//   1:F   AND                     (1)
			// 		 /   \
			//     2:T     OR                  (2)
			//        /    |    \
			//      AND   AND     AND          (3)
			//     /  \   /  \    /  \
			//   3:F 4:F 5:F 6:F 7:T 8:T

			thirdLeaf := Criteria{
				Operator:   And,
				Criteriums: []int{7, 8},
			}
			secondLeaf := Criteria{
				Operator:   And,
				Criteriums: []int{5, 6},
			}
			firstLeaf := Criteria{
				Operator:   And,
				Criteriums: []int{3, 4},
			}
			firstChildLeaf := Criteria{
				Operator:  Or,
				Criterias: []*Criteria{&firstLeaf, &secondLeaf, &thirdLeaf},
			}

			firstChild := Criteria{
				Operator:   And,
				Criteriums: []int{2},
				Criterias:  []*Criteria{&firstChildLeaf},
			}
			root := Criteria{
				Operator:   Or,
				Criteriums: []int{1},
				Criterias:  []*Criteria{&firstChild},
			}

			OSTsts := make(map[int]bool)
			pkgTsts := map[int][]fleet.Software{
				1: nil,
				2: {{ID: 1}},
				3: nil,
				4: nil,
				5: nil,
				6: nil,
				7: {{ID: 2}},
				8: {{ID: 3}},
			}

			sut := Definition{
				&root,
				nil,
			}

			require.True(t, sut.Eval(OSTsts, pkgTsts))
		})

		t.Run("tree with only criterias", func(t *testing.T) {
			// 		OR
			// 	   /  \
			//   1:F   AND                      (1)
			// 		 /     \
			//     OR        OR                 (2)
			//   /  |     /     \
			// 2:T 3:F  AND     AND             (3)
			//         /  \    /  \
			//       4:T  5:T 6:F 7:F

			secondLeaf := Criteria{
				Operator:   And,
				Criteriums: []int{6, 7},
			}
			firstLeaf := Criteria{
				Operator:   And,
				Criteriums: []int{4, 5},
			}

			levelTwoSecondChild := Criteria{
				Operator:  Or,
				Criterias: []*Criteria{&firstLeaf, &secondLeaf},
			}

			levelTwoFirstChild := Criteria{
				Operator:   Or,
				Criteriums: []int{2, 3},
			}

			firstChild := Criteria{
				Operator:  And,
				Criterias: []*Criteria{&levelTwoFirstChild, &levelTwoSecondChild},
			}
			root := Criteria{
				Operator:   Or,
				Criteriums: []int{1},
				Criterias:  []*Criteria{&firstChild},
			}

			OSTsts := make(map[int]bool)
			pkgTsts := map[int][]fleet.Software{
				1: nil,
				2: {{ID: 1}},
				3: nil,
				4: {{ID: 2}},
				5: {{ID: 3}},
				6: nil,
				7: nil,
			}

			sut := Definition{
				&root,
				nil,
			}

			require.True(t, sut.Eval(OSTsts, pkgTsts))
		})
	})

	t.Run("#CollectTestIds", func(t *testing.T) {
		t.Run("with logic tree", func(t *testing.T) {
			leaf := Criteria{
				And,
				[]int{30, 40},
				nil,
			}
			root := Criteria{
				Or,
				[]int{1, 2},
				[]*Criteria{&leaf},
			}
			sut := Definition{
				&root,
				nil,
			}

			require.ElementsMatch(t, sut.CollectTestIds(), []int{1, 2, 30, 40})
		})
	})
}
