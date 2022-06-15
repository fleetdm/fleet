package oval

import (
	"testing"

	oval_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/input"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
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

	t.Run("#mapPackageInfoTestObject", func(t *testing.T) {
		t.Run("name defined inline", func(t *testing.T) {
			input := oval_input.PackageInfoTestObjectXML{
				Name: oval_input.ObjectNameXML{
					Value: "some name",
				},
			}
			output, err := mapPackageInfoTestObject(input, nil)
			require.NoError(t, err)
			require.Contains(t, output, "some name")
		})

		t.Run("name defined in var ref", func(t *testing.T) {
			input := oval_input.PackageInfoTestObjectXML{
				Name: oval_input.ObjectNameXML{
					VarRef: "1",
				},
			}
			varRefs := map[string]oval_input.ConstantVariableXML{
				"1": {
					Values: []string{"donut"},
				},
			}
			output, err := mapPackageInfoTestObject(input, varRefs)
			require.NoError(t, err)
			require.Contains(t, output, "donut")
		})

		t.Run("name not defined inline nor using a variable ref", func(t *testing.T) {
			input := oval_input.PackageInfoTestObjectXML{}
			_, err := mapPackageInfoTestObject(input, nil)
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
		t.Run("maps a DpkgInfoTestXML", func(t *testing.T) {
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

		t.Run("errors out if id can not be parsed", func(t *testing.T) {
			input := oval_input.DpkgInfoTestXML{Id: "asdf"}
			_, _, err := mapDpkgInfoTest(input)
			require.Error(t, err)
		})
	})

	t.Run("#mapRpmInfoTest", func(t *testing.T) {
		t.Run("maps a RpmInfoTestXML", func(t *testing.T) {
			input := oval_input.RpmInfoTestXML{
				Id:             "some:oval:namespace:123",
				CheckExistence: "at_least_one_exists",
				Check:          "all",
				StateOperator:  "AND",
			}

			id, result, err := mapRpmInfoTest(input)

			require.NoError(t, err)
			require.Equal(t, id, 123)
			require.NotNil(t, result.StateOperator)
			require.NotNil(t, result.ObjectMatch)
			require.NotNil(t, result.StateMatch)
		})
		t.Run("errors out if id can not be parsed", func(t *testing.T) {
			input := oval_input.RpmInfoTestXML{Id: "asdf"}
			_, _, err := mapRpmInfoTest(input)
			require.Error(t, err)
		})
	})

	t.Run("#mapCriteria", func(t *testing.T) {
		t.Run("errors out if Id can not be parsed on any Criterion", func(t *testing.T) {
			input := oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{{}},
			}
			_, err := mapCriteria(input)
			require.Error(t, err)
		})

		t.Run("errors out if no Criteriums or nested criterias", func(t *testing.T) {
			input := oval_input.CriteriaXML{}
			_, err := mapCriteria(input)
			require.Errorf(t, err, "invalid Criteria, no Criteriums nor nested Criterias found")

			input = oval_input.CriteriaXML{
				Criteriums: []oval_input.CriterionXML{{TestId: "oval:123"}},
				Criterias:  []oval_input.CriteriaXML{{}},
			}
			_, err = mapCriteria(input)
			require.Errorf(t, err, "invalid Criteria, no Criteriums nor nested Criterias found")

			input = oval_input.CriteriaXML{
				Criterias: []oval_input.CriteriaXML{{
					Criterias: []oval_input.CriteriaXML{
						{Criteriums: []oval_input.CriterionXML{
							{TestId: "bc:1234"},
						}},
					},
				}},
			}
			_, err = mapCriteria(input)
			require.NoError(t, err)
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

	t.Run("#mapRpmInfoState", func(t *testing.T) {
		t.Run("maps the operator, if any", func(t *testing.T) {
			input := oval_input.RpmInfoStateXML{
				Name: &oval_input.SimpleTypeXML{
					Value: "name",
					Op:    "equals",
				},
			}
			result, err := mapRpmInfoState(input)
			require.NoError(t, err)
			require.Equal(t, result.Operator, oval_parsed.And)

			op := oval_parsed.Or.String()
			input.Operator = &op
			result, err = mapRpmInfoState(input)
			require.NoError(t, err)
			require.Equal(t, result.Operator, oval_parsed.Or)
		})

		t.Run("errors out if not supported state is provided", func(t *testing.T) {
			input := oval_input.RpmInfoStateXML{
				Filepath: &oval_input.SimpleTypeXML{},
			}
			_, err := mapRpmInfoState(input)
			require.Errorf(t, err, "object state based on filepath not supported")
		})

		t.Run("maps a RpmInfoStateXML", func(t *testing.T) {
			input := oval_input.RpmInfoStateXML{
				Name: &oval_input.SimpleTypeXML{
					Value: "name",
					Op:    "equals",
				},
				Arch: &oval_input.SimpleTypeXML{
					Value: "arch",
					Op:    "not equals",
				},
				Epoch: &oval_input.SimpleTypeXML{
					Datatype: "string",
					Value:    "epoch",
					Op:       "equals",
				},
				Release: &oval_input.SimpleTypeXML{
					Datatype: "boolean",
					Value:    "true",
					Op:       "less than",
				},
				Version: &oval_input.SimpleTypeXML{
					Datatype: "int",
					Value:    "123",
					Op:       "equals",
				},
				Evr: &oval_input.SimpleTypeXML{
					Value: "^12.12",
					Op:    "equals",
				},
				SignatureKeyId: &oval_input.SimpleTypeXML{
					Op:    "equals",
					Value: "12345",
				},
				ExtendedName: &oval_input.SimpleTypeXML{
					Op:    "equals",
					Value: "0:123:12",
				},
			}

			output, err := mapRpmInfoState(input)
			require.NoError(t, err)

			require.Equal(t, *output.Name, oval_parsed.NewObjectStateString("equals", "name"))
			require.Equal(t, *output.Arch, oval_parsed.NewObjectStateString("not equals", "arch"))
			require.Equal(t, *output.Epoch, oval_parsed.NewObjectStateSimpleValue("string", "equals", "epoch"))
			require.Equal(t, *output.Release, oval_parsed.NewObjectStateSimpleValue("boolean", "less than", "true"))
			require.Equal(t, *output.Version, oval_parsed.NewObjectStateSimpleValue("int", "equals", "123"))
			require.Equal(t, *output.Evr, oval_parsed.NewObjectStateEvrString("equals", "^12.12"))
			require.Equal(t, *output.SignatureKeyId, oval_parsed.NewObjectStateString("equals", "12345"))
			require.Equal(t, *output.ExtendedName, oval_parsed.NewObjectStateString("equals", "0:123:12"))
		})
	})

	t.Run("#mapRpmVerifyFileTest", func(t *testing.T) {
		input := oval_input.RpmVerifyFileTestXML{
			Id:             "some:oval:namespace:123",
			CheckExistence: "at_least_one_exists",
			Check:          "all",
			StateOperator:  "AND",
		}

		id, result, err := mapRpmVerifyFileTest(input)

		require.NoError(t, err)
		require.Equal(t, id, 123)
		require.NotNil(t, result.StateOperator)
		require.NotNil(t, result.ObjectMatch)
		require.NotNil(t, result.StateMatch)
	})

	t.Run("#mapRpmVerifyFileObject", func(t *testing.T) {
		t.Run("errors out if invalid children provided", func(t *testing.T) {
			testCases := []struct {
				input     oval_input.RpmVerifyFileObjectXML
				errorsOut bool
			}{
				{
					input:     oval_input.RpmVerifyFileObjectXML{Name: oval_input.SimpleTypeXML{Value: "123"}},
					errorsOut: true,
				},
				{
					input:     oval_input.RpmVerifyFileObjectXML{Epoch: oval_input.SimpleTypeXML{Value: "123"}},
					errorsOut: true,
				},
				{
					input:     oval_input.RpmVerifyFileObjectXML{Version: oval_input.SimpleTypeXML{Value: "123"}},
					errorsOut: true,
				},
				{
					input:     oval_input.RpmVerifyFileObjectXML{Release: oval_input.SimpleTypeXML{Value: "123"}},
					errorsOut: true,
				},
				{
					input:     oval_input.RpmVerifyFileObjectXML{Arch: oval_input.SimpleTypeXML{Value: "123"}},
					errorsOut: true,
				},
				{
					input:     oval_input.RpmVerifyFileObjectXML{FilePath: oval_input.SimpleTypeXML{Value: ""}},
					errorsOut: true,
				},
				{
					input:     oval_input.RpmVerifyFileObjectXML{FilePath: oval_input.SimpleTypeXML{Value: "/etc/red-hat"}},
					errorsOut: false,
				},
			}

			for _, tCase := range testCases {
				_, err := mapRpmVerifyFileObject(tCase.input)
				if tCase.errorsOut {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
		})

		t.Run("maps to a filepath", func(t *testing.T) {
			input := oval_input.RpmVerifyFileObjectXML{FilePath: oval_input.SimpleTypeXML{Value: "/etc/red-hat"}}
			r, err := mapRpmVerifyFileObject(input)
			require.NoError(t, err)
			require.Equal(t, *r, "/etc/red-hat")
		})
	})

	t.Run("#mapRpmVerifyFileState", func(t *testing.T) {
		t.Run("errors out if not supported state is provided", func(t *testing.T) {
			testCases := []struct {
				input       oval_input.RpmVerifyFileStateXML
				shouldError bool
			}{
				{
					input:       oval_input.RpmVerifyFileStateXML{SizeDiffers: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{ModeDiffers: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{Md5Differs: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{DeviceDiffers: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{LinkMismatch: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{OwnershipDiffers: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{GroupDiffers: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{MtimeDiffers: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{CapabilitiesDiffer: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{ConfigurationFile: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{LicenseFile: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{ReadmeFile: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{Name: &oval_input.SimpleTypeXML{}},
					shouldError: false,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{Arch: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{Epoch: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{Version: &oval_input.SimpleTypeXML{}},
					shouldError: false,
				},
				{
					input:       oval_input.RpmVerifyFileStateXML{ExtendedName: &oval_input.SimpleTypeXML{}},
					shouldError: true,
				},
			}

			for _, tCase := range testCases {
				_, err := mapRpmVerifyFileState(tCase.input)
				if tCase.shouldError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
		})

		t.Run("maps the operator, if any", func(t *testing.T) {
			input := oval_input.RpmVerifyFileStateXML{
				Name: &oval_input.SimpleTypeXML{
					Value: "name",
					Op:    "equals",
				},
			}
			result, err := mapRpmVerifyFileState(input)
			require.NoError(t, err)
			require.Equal(t, result.Operator, oval_parsed.And)

			op := oval_parsed.Or.String()
			input.Operator = &op
			result, err = mapRpmVerifyFileState(input)
			require.NoError(t, err)
			require.Equal(t, result.Operator, oval_parsed.Or)
		})

		t.Run("maps a RpmVerifyFileStateXML", func(t *testing.T) {
			input := oval_input.RpmVerifyFileStateXML{
				Name: &oval_input.SimpleTypeXML{
					Value: "name",
					Op:    "equals",
				},
				Version: &oval_input.SimpleTypeXML{
					Datatype: "int",
					Value:    "123",
					Op:       "equals",
				},
			}

			output, err := mapRpmVerifyFileState(input)
			require.NoError(t, err)

			require.Equal(t, *output.Name, oval_parsed.NewObjectStateString("equals", "name"))
			require.Equal(t, *output.Version, oval_parsed.NewObjectStateSimpleValue("int", "equals", "123"))
		})
	})
}
