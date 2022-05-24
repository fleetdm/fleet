package oval_parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestOvalParsedDefinition(t *testing.T) {
	t.Run("#Eval", func(t *testing.T) {
		t.Run("no root criteria", func(t *testing.T) {
			sut := Definition{}
			require.False(t, sut.Eval(nil))
		})

		t.Run("with empty test results", func(t *testing.T) {
			criteria := Criteria{
				And,
				[]int{1, 2, 3},
				nil,
			}
			sut := Definition{Criteria: &criteria}
			require.False(t, sut.Eval(nil))
			require.False(t, sut.Eval(make(map[int][]fleet.Software)))
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
				tests := map[int][]fleet.Software{
					1: {{ID: 1}},
					2: nil,
					3: {{ID: 2}},
				}
				sut := Definition{
					&criteria,
					nil,
				}

				require.Equal(t, c.expected, sut.Eval(tests))
			}
		})

		t.Run("evaluating logic tree", func(t *testing.T) {
			//   OR
			//  / | \
			// F  F AND
			//     /  \
			//    T    T

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

			tests := map[int][]fleet.Software{
				1: nil,
				2: nil,
				3: {{ID: 2}},
				4: {{ID: 3}},
			}

			sut := Definition{
				&root,
				nil,
			}

			require.True(t, sut.Eval(tests))
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
