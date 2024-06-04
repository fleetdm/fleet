package oval_parsed

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectStateString(t *testing.T) {
	t.Run("#Eval", func(t *testing.T) {
		t.Run("it evaluates string values", func(t *testing.T) {
			cases := []struct {
				val      string
				other    string
				expected bool
			}{
				{val: "equals|1.1", other: "1.1", expected: true},
				{val: "equals|1.1", other: "1.0", expected: false},
				{val: "not equal|1.1", other: "2.1", expected: true},
				{val: "not equal|1.1", other: "1.1", expected: false},
				{val: "case insensitive equals|a", other: "A", expected: true},
				{val: "case insensitive equals|a", other: "B", expected: false},
				{val: "case insensitive not equal|a", other: "A", expected: false},
				{val: "case insensitive not equal|a", other: "B", expected: true},
				{val: "pattern match|aarch64|ppc64le|s390x|x86_64", other: "abc", expected: false},
				{val: "pattern match|aarch64|ppc64le|s390x|x86_64", other: "aarch64", expected: true},
				{val: "pattern match|aarch64|ppc64le|s390x|x86_64", other: "x86_64", expected: true},
			}

			for _, c := range cases {
				r, err := ObjectStateString(c.val).Eval(c.other)
				require.NoError(t, err)
				require.Equal(t, c.expected, r)
			}
		})

		t.Run("it errors out if regexp can not be parsed", func(t *testing.T) {
			// Go regexp engine does not support look-arounds
			regExp := `^\/(?!\/)(.*?)`
			sut := ObjectStateString(fmt.Sprintf("%s|%s", "pattern match", regExp))
			_, err := sut.Eval("scrambled eggs")
			require.Error(t, err)
		})

		t.Run("it errors out if operation can not be computed", func(t *testing.T) {
			invalidOps := []OperationType{
				BitwiseAnd,
				BitwiseOr,
				SupersetOf,
				SubsetOf,
				LessThan,
				LessThanOrEqual,
				GreaterThan,
				GreaterThanOrEqual,
			}
			for _, op := range invalidOps {
				sut := ObjectStateString(fmt.Sprintf("%s|%s", op, "something"))
				_, err := sut.Eval("the thing")
				require.Errorf(t, err, "can not compute")
			}

			validOps := []OperationType{
				Equals,
				NotEqual,
				CaseInsensitiveEquals,
				CaseInsensitiveNotEqual,
				PatternMatch,
			}
			for _, op := range validOps {
				sut := ObjectStateString(fmt.Sprintf("%s|%s", op, "something"))
				_, err := sut.Eval("the thing")
				require.NoError(t, err)
			}
		})
	})
}

func TestParseKernelVariantsinObject(t *testing.T) {
	tests := []struct {
		input    ObjectStateString
		expected []string
	}{
		{
			input:    ObjectStateString(`pattern match|5.15.0-\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k)`),
			expected: []string{"generic", "generic-64k", "generic-lpae", "lowlatency", "lowlatency-64k"},
		},
		{
			input:    ObjectStateString(`pattern match|5.15.0-\d+(-generic|-lowlatency)`),
			expected: []string{"generic", "lowlatency"},
		},
		{
			input:    ObjectStateString(`pattern match|5.15.0-\d+(-custom)`),
			expected: []string{"custom"},
		},
		{
			input:    ObjectStateString(`pattern match|5.15.0-\d+()`),
			expected: []string{},
		},
		{
			input:    ObjectStateString(`pattern match|invalid-string`),
			expected: []string{},
		},
		{
			input:    ObjectStateString(`less than|5.15.0-\d+(-generic)`),
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("input: %s", tc.input), func(t *testing.T) {
			require.Equal(t, tc.expected, tc.input.ParseKernelVariants())
		})
	}
}
