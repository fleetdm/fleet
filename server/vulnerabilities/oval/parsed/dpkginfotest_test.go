package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDpkgInfoTestEvalNoHostList(t *testing.T) {
	sut := DpkgInfoTest{}

	require.False(t, sut.Eval(nil))
	require.False(t, sut.Eval(make([]HostPackage, 0)))
}

func TestDpkgInfoTestMatchesNObjects(t *testing.T) {
	packages := []HostPackage{
		{"firefox", "1.2"},
		{"chrome", "1.2"},
		{"paint", "1.2"},
	}

	sut := DpkgInfoTest{
		Objects: []string{"firefox", "paint"},
	}

	nObjects, _ := sut.matches(packages)
	require.Equal(t, 2, nObjects)
}

func TestDpkgInfoTestMatchesNStates(t *testing.T) {
	packages := []HostPackage{
		{"firefox", "1.3"},
		{"firefox", "1.2"},
		{"chrome", "1.2"},
		{"paint", "1.2"},
	}

	sut := DpkgInfoTest{
		Objects:       []string{"firefox", "paint"},
		States:        []ObjectStateEvrString{"equals|1.3", "equals|1.0"},
		StateOperator: Or,
	}

	_, nStates := sut.matches(packages)
	require.Equal(t, 1, nStates)
}
