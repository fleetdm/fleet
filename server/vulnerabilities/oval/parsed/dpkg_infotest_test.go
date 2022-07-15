package oval_parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestDpkgInfoTestEvalNoHostList(t *testing.T) {
	t.Run("#Eval", func(t *testing.T) {
		t.Run("with no packages", func(t *testing.T) {
			sut := DpkgInfoTest{}

			r, err := sut.Eval(nil)
			require.NoError(t, err)
			require.Nil(t, r)

			r, err = sut.Eval(make([]fleet.Software, 0))
			require.NoError(t, err)
			require.Nil(t, r)
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

			nObjects, _, _, _ := sut.matches(packages)
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

			_, nStates, _, _ := sut.matches(packages)
			require.Equal(t, 1, nStates)
		})
	})
}
