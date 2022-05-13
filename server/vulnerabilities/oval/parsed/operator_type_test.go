package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOperatorType(t *testing.T) {
	cases := []struct {
		input    string
		expected OperatorType
	}{
		{"AND", And},
		{"and", And},
		{"ONE", One},
		{"one", One},
		{"OR", Or},
		{"or", Or},
		{"XOR", Xor},
		{"xor", Xor},
		{"", And},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, NewOperatorType(c.input))
	}
}

func TestOperatorTypeNegate(t *testing.T) {
	cases := []struct {
		input    OperatorType
		op       string
		expected OperatorType
	}{
		{And, "true", NotAnd},
		{Or, "true", NotOr},
		{One, "true", NotOne},
		{Xor, "true", NotXor},
		{And, "false", And},
		{Or, "false", Or},
		{One, "false", One},
		{Xor, "false", Xor},
		{And, "", And},
		{Or, "", Or},
		{One, "", One},
		{Xor, "", Xor},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, c.input.Negate(c.op))
	}
}

func TestOperatorTypeEval(t *testing.T) {
	cases := []struct {
		op       OperatorType
		vals     []bool
		expected bool
	}{
		{And, []bool{true, true}, true},
		{And, []bool{true, false}, false},
		{One, []bool{true, true, true}, false},
		{One, []bool{false, false, false}, false},
		{One, []bool{true, false, false}, true},
		{Or, []bool{true, false}, true},
		{Or, []bool{false, true}, true},
		{Or, []bool{true, true}, true},
		{Or, []bool{false, false}, false},
		{Xor, []bool{false, false}, false},
		{Xor, []bool{true, true}, false},
		{Xor, []bool{false, true}, true},
		{Xor, []bool{true, false}, true},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, c.op.Eval(c.vals...))
		require.Equal(t, !c.expected, c.op.Negate("true").Eval(c.vals...))
	}
}
