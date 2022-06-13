package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectStateSimpleValue(t *testing.T) {
	t.Run("NewObjectStateSimpleValue", func(t *testing.T) {
		sut := NewObjectStateSimpleValue("binary", "not equal", "0101")
		require.Equal(t, string(sut), "binary|not equal|0101")
	})

	t.Run("#unpack", func(t *testing.T) {
		sut := NewObjectStateSimpleValue("binary", "not equal", "0101")
		dType, op, val := sut.unpack()
		require.Equal(t, dType, Binary)
		require.Equal(t, op, NotEqual)
		require.Equal(t, val, "0101")
	})

	t.Run("#Eval", func(t *testing.T) {
		t.Run("it errors out if complex type used", func(t *testing.T) {
			invalidTypes := []string{
				"binary",
				"fileset_revision",
				"ios_version",
				"ipv4_address",
				"ipv4_address",
				"version",
			}

			for _, invalidT := range invalidTypes {
				sut := NewObjectStateSimpleValue(invalidT, "equals", "1")
				_, err := sut.Eval("2")
				require.Error(t, err)
			}
		})

		t.Run("compares simple data types", func(t *testing.T) {
			t.Run("booleans", func(t *testing.T) {
				trueValues := []string{"true", "1"}
				falseValues := []string{"false", "0"}
				validOps := []string{"equals", "not equal"}
				for _, v1 := range trueValues {
					for _, v2 := range falseValues {
						for _, op := range validOps {
							sut := NewObjectStateSimpleValue("boolean", op, v1)
							r, err := sut.Eval(v2)
							require.NoError(t, err)
							if op == "equals" {
								require.Equal(t, v1 == v2, r)
							}
						}
					}
				}

				invalidOps := []string{
					"case insensitive equals",
					"case insensitive not equal",
					"greater than",
					"less than",
					"greater than or equal",
					"less than or equal",
					"bitwise and",
					"bitwise or",
					"pattern match",
					"subset of",
					"superset of",
				}
				for _, op := range invalidOps {
					sut := NewObjectStateSimpleValue("boolean", op, "1")
					_, err := sut.Eval("2")
					require.Error(t, err)
				}

				testCases := []struct {
					val1        string
					val2        string
					shouldError bool
				}{
					{val1: "true", val2: "true", shouldError: false},
					{val1: "true", val2: "1", shouldError: false},
					{val1: "5", val2: "1", shouldError: true},
					{val1: "1", val2: "5", shouldError: true},
				}
				for _, tCase := range testCases {
					sut := NewObjectStateSimpleValue("boolean", "equals", tCase.val1)
					_, err := sut.Eval(tCase.val2)
					if tCase.shouldError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}

				}
			})

			t.Run("floats", func(t *testing.T) {
				invalidOps := []string{
					"case insensitive equals",
					"case insensitive not equal",
					"bitwise and",
					"bitwise or",
					"pattern match",
					"subset of",
					"superset of",
				}
				for _, op := range invalidOps {
					sut := NewObjectStateSimpleValue("float", op, "1")
					_, err := sut.Eval("2")
					require.Error(t, err)
				}

				invalidTypesTstCases := []struct {
					val1        string
					val2        string
					shouldError bool
				}{
					{val1: "1.2", val2: "1.2", shouldError: false},
					{val1: "sdfa", val2: "1", shouldError: true},
					{val1: "1", val2: "asdf", shouldError: true},
					{val1: "asdf", val2: "asdf", shouldError: true},
				}
				for _, tCase := range invalidTypesTstCases {
					sut := NewObjectStateSimpleValue("float", "equals", tCase.val1)
					_, err := sut.Eval(tCase.val2)
					if tCase.shouldError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				}

				validTstCases := []struct {
					val1   string
					val2   string
					op     OperationType
					result bool
				}{
					{val1: "1.2", val2: "1.2", op: Equals, result: true},
					{val1: "1.2", val2: "1.3", op: Equals, result: false},
					{val1: "1.2", val2: "1.2", op: NotEqual, result: false},
					{val1: "1.2", val2: "1.3", op: NotEqual, result: true},
					{val1: "1.3", val2: "1.2", op: GreaterThan, result: true},
					{val1: "1.2", val2: "1.3", op: GreaterThan, result: false},
					{val1: "1.3", val2: "1.2", op: GreaterThanOrEqual, result: true},
					{val1: "1.2", val2: "1.2", op: GreaterThanOrEqual, result: true},
					{val1: "1.2", val2: "1.3", op: GreaterThanOrEqual, result: false},
					{val1: "1.3", val2: "1.2", op: LessThan, result: false},
					{val1: "1.2", val2: "1.3", op: LessThan, result: true},
					{val1: "1.3", val2: "1.2", op: LessThanOrEqual, result: false},
					{val1: "1.2", val2: "1.2", op: LessThanOrEqual, result: true},
					{val1: "1.2", val2: "1.3", op: LessThanOrEqual, result: true},
				}
				for _, tCase := range validTstCases {
					sut := NewObjectStateSimpleValue("float", tCase.op.String(), tCase.val1)
					r, err := sut.Eval(tCase.val2)
					require.NoError(t, err)
					require.Equal(t, tCase.result, r)
				}
			})

			t.Run("ints", func(t *testing.T) {
				invalidOps := []string{
					"case insensitive equals",
					"case insensitive not equal",
					"bitwise and",
					"bitwise or",
					"pattern match",
					"subset of",
					"superset of",
				}
				for _, op := range invalidOps {
					sut := NewObjectStateSimpleValue("int", op, "1")
					_, err := sut.Eval("2")
					require.Error(t, err)
				}

				invalidTypesTstCases := []struct {
					val1        string
					val2        string
					shouldError bool
				}{
					{val1: "1", val2: "1", shouldError: false},
					{val1: "sdfa", val2: "1", shouldError: true},
					{val1: "1", val2: "asdf", shouldError: true},
					{val1: "asdf", val2: "asdf", shouldError: true},
				}
				for _, tCase := range invalidTypesTstCases {
					sut := NewObjectStateSimpleValue("int", "equals", tCase.val1)
					_, err := sut.Eval(tCase.val2)
					if tCase.shouldError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				}

				validTstCases := []struct {
					val1   string
					val2   string
					op     OperationType
					result bool
				}{
					{val1: "2", val2: "2", op: Equals, result: true},
					{val1: "2", val2: "3", op: Equals, result: false},
					{val1: "2", val2: "2", op: NotEqual, result: false},
					{val1: "2", val2: "3", op: NotEqual, result: true},
					{val1: "3", val2: "2", op: GreaterThan, result: true},
					{val1: "2", val2: "3", op: GreaterThan, result: false},
					{val1: "3", val2: "2", op: GreaterThanOrEqual, result: true},
					{val1: "2", val2: "2", op: GreaterThanOrEqual, result: true},
					{val1: "2", val2: "3", op: GreaterThanOrEqual, result: false},
					{val1: "3", val2: "2", op: LessThan, result: false},
					{val1: "2", val2: "3", op: LessThan, result: true},
					{val1: "3", val2: "2", op: LessThanOrEqual, result: false},
					{val1: "2", val2: "2", op: LessThanOrEqual, result: true},
					{val1: "2", val2: "3", op: LessThanOrEqual, result: true},
				}
				for _, tCase := range validTstCases {
					sut := NewObjectStateSimpleValue("int", tCase.op.String(), tCase.val1)
					r, err := sut.Eval(tCase.val2)
					require.NoError(t, err)
					require.Equal(t, tCase.result, r)
				}
			})

			t.Run("strings", func(t *testing.T) {
				invalidOps := []string{
					"greater than",
					"less than",
					"greater than or equal",
					"less than or equal",
					"bitwise and",
					"bitwise or",
					"subset of",
					"superset of",
				}
				for _, op := range invalidOps {
					sut := NewObjectStateSimpleValue("string", op, "1")
					_, err := sut.Eval("2")
					require.Error(t, err)
				}

				tstCases := []struct {
					val1   string
					val2   string
					op     OperationType
					result bool
				}{
					{val1: "a", val2: "a", op: Equals, result: true},
					{val1: "a", val2: "b", op: Equals, result: false},
					{val1: "a", val2: "a", op: NotEqual, result: false},
					{val1: "a", val2: "b", op: NotEqual, result: true},
					{val1: "a", val2: "A", op: CaseInsensitiveEquals, result: true},
					{val1: "a", val2: "B", op: CaseInsensitiveEquals, result: false},
					{val1: "a", val2: "A", op: CaseInsensitiveNotEqual, result: false},
					{val1: "a", val2: "B", op: CaseInsensitiveNotEqual, result: true},
					{val1: "a|b|c", val2: "a", op: PatternMatch, result: true},
					{val1: "a|b|c", val2: "z", op: PatternMatch, result: false},
				}
				for i, tCase := range tstCases {
					sut := NewObjectStateSimpleValue("string", tCase.op.String(), tCase.val1)
					r, err := sut.Eval(tCase.val2)
					require.NoError(t, err)
					require.Equal(t, tCase.result, r, i)
				}
			})
		})
	})
}
