package oval_parsed

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectStateEvrString(t *testing.T) {
	t.Run("#Eval", func(t *testing.T) {
		t.Run("evaluates an evr string", func(t *testing.T) {
			cases := []struct {
				val      string
				other    string
				cmp      func(string, string) int
				expected bool
			}{
				{"equals|1.1", "1.1", func(s1, s2 string) int { return 0 }, true},
				{"equals|1.1", "1.0", func(s1, s2 string) int { return 1 }, false},
				{"not equal|1.1", "2.1", func(s1, s2 string) int { return -1 }, true},
				{"not equal|1.1", "1.1", func(s1, s2 string) int { return 0 }, false},
				{"greater than|1.1", "2.1", func(s1, s2 string) int { return -1 }, false},
				{"greater than|1.1", "1.1", func(s1, s2 string) int { return 0 }, false},
				{"greater than|1.2", "1.1", func(s1, s2 string) int { return 1 }, true},
				{"greater than or equal|1.1", "2.1", func(s1, s2 string) int { return -1 }, false},
				{"greater than or equal|1.1", "1.1", func(s1, s2 string) int { return 0 }, true},
				{"greater than or equal|1.2", "1.1", func(s1, s2 string) int { return 1 }, true},
				{"less than|1.1", "2.1", func(s1, s2 string) int { return -1 }, true},
				{"less than|1.1", "1.1", func(s1, s2 string) int { return 0 }, false},
				{"less than|1.2", "1.1", func(s1, s2 string) int { return 1 }, false},
				{"less than or equal|1.1", "2.1", func(s1, s2 string) int { return -1 }, true},
				{"less than or equal|1.1", "1.1", func(s1, s2 string) int { return 0 }, true},
				{"less than or equal|1.2", "1.1", func(s1, s2 string) int { return 1 }, false},
			}

			for _, c := range cases {
				r, err := ObjectStateEvrString(c.val).Eval(c.other, c.cmp, false)
				require.NoError(t, err)
				require.Equal(t, c.expected, r)
			}
		})

		t.Run("it errors out if operation can not be computed", func(t *testing.T) {
			invalidOps := []OperationType{
				BitwiseAnd,
				BitwiseOr,
				SupersetOf,
				SubsetOf,
				CaseInsensitiveEquals,
				CaseInsensitiveNotEqual,
				PatternMatch,
			}

			for _, op := range invalidOps {
				sut := ObjectStateEvrString(fmt.Sprintf("%s|%s", op, "something"))
				_, err := sut.Eval("the thing", func(s1, s2 string) int { return 0 }, false)
				require.Errorf(t, err, "can not compute")
			}
		})
	})
}
