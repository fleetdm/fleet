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
				{"equals|1.1", "1.1", true},
				{"equals|1.1", "1.0", false},
				{"not equal|1.1", "2.1", true},
				{"not equal|1.1", "1.1", false},
				{"greater than|b", "a", true},
				{"greater than|a", "b", false},
				{"greater than|a", "a", false},
				{"greater than or equal|a", "a", true},
				{"greater than or equal|b", "a", true},
				{"greater than or equal|a", "b", false},
				{"less than|a", "b", true},
				{"less than|b", "a", false},
				{"less than|a", "a", false},
				{"less than or equal|a", "b", true},
				{"less than or equal|b", "a", false},
				{"less than or equal|a", "a", true},
				{"less than or equal|a", "a", true},
				{"pattern match|aarch64|ppc64le|s390x|x86_64", "abc", false},
				{"pattern match|aarch64|ppc64le|s390x|x86_64", "aarch64", true},
				{"pattern match|aarch64|ppc64le|s390x|x86_64", "x86_64", true},
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
				BitwiseAnd, BitwiseOr, SupersetOf, SubsetOf,
			}

			for _, op := range invalidOps {
				sut := ObjectStateString(fmt.Sprintf("%s|%s", op, "something"))
				_, err := sut.Eval("the thing")
				require.Errorf(t, err, "can not compute")
			}
		})
	})
}
