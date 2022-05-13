package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefinitionEvalNoRootCriteria(t *testing.T) {
	sut := Definition{}
	require.False(t, sut.Eval(nil))
}

func TestDefinitionEvalWithEmptyTests(t *testing.T) {
	criteria := Criteria{
		And,
		[]int{1, 2, 3},
		nil,
	}
	sut := Definition{Criteria: &criteria}
	require.False(t, sut.Eval(nil))
	require.False(t, sut.Eval(make(map[int]bool)))
}

func TestDefinitionEvalWithSingleLevelCriteria(t *testing.T) {
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
		tests := map[int]bool{
			1: true,
			2: false,
			3: true,
		}
		sut := Definition{
			&criteria,
			nil,
		}

		require.Equal(t, c.expected, sut.Eval(tests))
	}
}

func TestDefinitionEvalWithLogicTreeCriteria(t *testing.T) {
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

	tests := map[int]bool{
		1: false,
		2: false,
		3: true,
		4: true,
	}

	sut := Definition{
		&root,
		nil,
	}

	require.True(t, sut.Eval(tests))
}
