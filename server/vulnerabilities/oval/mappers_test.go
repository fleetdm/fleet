package oval

import (
	"testing"

	oval_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/input"
	"github.com/stretchr/testify/require"
)

func TestOvalMapper(t *testing.T) {
	t.Run("#extractId", func(t *testing.T) {
		testCases := []struct {
			input     string
			errorsOut bool
			output    int
		}{
			{input: "", errorsOut: true},
			{input: "asdfasdf", errorsOut: true},
			{input: "oval:com.ubuntu.eoan:obj:100", output: 100},
			{input: "oval:com.redhat.rhsa:ste:20070123005", output: 20070123005},
			{input: "oval:com.redhat.rhsa:", errorsOut: true},
		}

		for _, tCase := range testCases {
			r, err := extractId(tCase.input)
			if tCase.errorsOut {
				require.Error(t, err)
			} else {
				require.Equal(t, tCase.output, r)
			}
		}
	})

	t.Run("#mapDpkgInfoObject", func(t *testing.T) {
		t.Run("name defined inline", func(t *testing.T) {
			input := oval_input.DpkgInfoObjectXML{
				Name: oval_input.ObjectNameXML{
					Value: "some name",
				},
			}
			output, err := mapDpkgInfoObject(input, nil)
			require.NoError(t, err)
			require.Contains(t, output, "some name")
		})

		t.Run("name defined in var ref", func(t *testing.T) {
			input := oval_input.DpkgInfoObjectXML{
				Name: oval_input.ObjectNameXML{
					VarRef: "1",
				},
			}
			varRefs := map[string]oval_input.ConstantVariableXML{
				"1": {
					Values: []string{"donut"},
				},
			}
			output, err := mapDpkgInfoObject(input, varRefs)
			require.NoError(t, err)
			require.Contains(t, output, "donut")
		})

		t.Run("name not defined inline nor using a variable ref", func(t *testing.T) {
			input := oval_input.DpkgInfoObjectXML{}
			_, err := mapDpkgInfoObject(input, nil)
			require.Errorf(t, err, "variable not found")
		})
	})

	t.Run("#mapDpkgInfoState", func(t *testing.T) {
		t.Run("errors out if one of non-supported state information is provided", func(t *testing.T) {
			simpleStrType := func(s string) *oval_input.SimpleTypeXML {
				return &oval_input.SimpleTypeXML{
					Value: s,
				}
			}

			testCases := []struct {
				state     oval_input.DpkgInfoStateXML
				errorsOut bool
			}{
				{state: oval_input.DpkgInfoStateXML{Name: simpleStrType("abc")}, errorsOut: true},
				{state: oval_input.DpkgInfoStateXML{Name: simpleStrType("")}, errorsOut: true},
				{state: oval_input.DpkgInfoStateXML{Arch: simpleStrType("")}, errorsOut: true},
				{state: oval_input.DpkgInfoStateXML{Epoch: simpleStrType("")}, errorsOut: true},
				{state: oval_input.DpkgInfoStateXML{Version: simpleStrType("")}, errorsOut: true},
				{state: oval_input.DpkgInfoStateXML{}, errorsOut: true},
				{state: oval_input.DpkgInfoStateXML{Evr: simpleStrType("123.12")}},
			}

			for _, tCase := range testCases {
				r, err := mapDpkgInfoState(tCase.state)
				if tCase.errorsOut {
					require.Error(t, err)
				} else {
					require.NotEmpty(t, r)
				}
			}
		})
	})

	t.Run("#mapDpkgInfoTest", func(t *testing.T) {
		input := oval_input.DpkgInfoTestXML{
			Id:             "some:oval:namespace:123",
			CheckExistence: "at_least_one_exists",
			Check:          "all",
			StateOperator:  "AND",
		}

		id, result, err := mapDpkgInfoTest(input)

		require.NoError(t, err)
		require.Equal(t, id, 123)
		require.NotNil(t, result.StateOperator)
		require.NotNil(t, result.ObjectMatch)
		require.NotNil(t, result.StateMatch)
	})

	t.Run("#mapCriteria", func(t *testing.T) {
		t.Run("errors out if Id can not be parsed on any Criterion", func(t *testing.T) {
			input := oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{{}},
			}
			_, err := mapCriteria(input)
			require.Error(t, err)
		})

		t.Run("errors out if no Criteriums", func(t *testing.T) {
			input := oval_input.CriteriaXML{}
			_, err := mapCriteria(input)
			require.Errorf(t, err, "invalid Criteria, Criteriums missing")

			input = oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{{TestId: "oval:123"}},
				Criterias:  []oval_input.CriteriaXML{{}},
			}
			_, err = mapCriteria(input)
			require.Errorf(t, err, "invalid Criteria, Criteriums missing")
		})

		t.Run("maps Criteriums", func(t *testing.T) {
			input := oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{
					{TestId: "oval:123"},
					{TestId: "oval:456"},
				},
			}

			r, err := mapCriteria(input)
			require.NoError(t, err)
			require.ElementsMatch(t, []int{123, 456}, r.Criteriums)
		})

		t.Run("maps nested Criterias", func(t *testing.T) {
			input := oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{
					{TestId: "oval:123"},
				},
				Criterias: []oval_input.CriteriaXML{
					{
						Criteriums: []oval_input.CriterionXML{
							{TestId: "oval:456"},
							{TestId: "oval:789"},
						},
					},
				},
			}

			r, err := mapCriteria(input)
			require.NoError(t, err)

			require.ElementsMatch(t, []int{456, 789}, r.Criterias[0].Criteriums)
		})
	})

	t.Run("#mapDefinition", func(t *testing.T) {
		t.Run("errors out if no vulnerabilities", func(t *testing.T) {
			input := oval_input.DefinitionXML{Criteria: oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{
					{TestId: "oval:123"},
				},
			}}
			_, err := mapDefinition(input)
			require.Errorf(t, err, "definition contains no vulnerabilities")
		})
	})
}
