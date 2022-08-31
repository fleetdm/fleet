package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStateMatchType(t *testing.T) {
	cases := []struct {
		input    string
		expected StateMatchType
	}{
		{"all", All},
		{"at least one", AtLeastOne},
		{"none satisfy", NoneSatisfy},
		{"none exist", NoneSatisfy},
		{"only one", OnlyOne},
		{"", All},
		{"asdfa", All},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, NewStateMatchType(c.input))
	}
}

func TestStateMatchTypeEval(t *testing.T) {
	cases := []struct {
		nObjects int
		nStates  int
		op       StateMatchType
		expected bool
	}{
		{1, 1, All, true},
		{2, 1, All, false},
		{3, 1, AtLeastOne, true},
		{3, 0, AtLeastOne, false},
		{3, 3, NoneSatisfy, false},
		{3, 0, NoneSatisfy, true},
		{3, 1, OnlyOne, true},
		{3, 3, OnlyOne, false},
		{3, 0, OnlyOne, false},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, c.op.Eval(c.nObjects, c.nStates))
	}
}
