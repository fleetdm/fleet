package oval_parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestDpkgInfoTestEvalNoHostList(t *testing.T) {
	t.Run("#Eval", func(t *testing.T) {
		t.Run("with no host list", func(t *testing.T) {
			sut := DpkgInfoTest{}

			require.False(t, sut.Eval(nil))
			require.False(t, sut.Eval(make([]fleet.Software, 0)))
		})

		t.Run("test matches NObjects", func(t *testing.T) {
			packages := []fleet.Software{
				{
					Name:    "firefox",
					Version: "1.2",
				},
				{
					Name:    "chrome",
					Version: "1.2",
				},
				{
					Name:    "paint",
					Version: "1.2",
				},
			}

			sut := DpkgInfoTest{
				Objects: []string{"firefox", "paint"},
			}

			nObjects, _ := sut.matches(packages)
			require.Equal(t, 2, nObjects)
		})

		t.Run("test matches NStates", func(t *testing.T) {
			packages := []fleet.Software{
				{
					Name:    "firefox",
					Version: "1.2",
				},
				{
					Name:    "firefox",
					Version: "1.3",
				},
				{
					Name:    "chrome",
					Version: "1.2",
				},
				{
					Name:    "paint",
					Version: "1.2",
				},
			}

			sut := DpkgInfoTest{
				Objects:       []string{"firefox", "paint"},
				States:        []ObjectStateEvrString{"equals|1.3", "equals|1.0"},
				StateOperator: Or,
			}

			_, nStates := sut.matches(packages)
			require.Equal(t, 1, nStates)
		})
	})
}
