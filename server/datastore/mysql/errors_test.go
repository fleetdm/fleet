package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlreadyExistsError(t *testing.T) {
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{{
		name: "WithTeamID",
		fn: func(t *testing.T) {
			err := alreadyExists("User", "alice").WithTeamID(42)
			expectedMsg := `User "alice" already exists with TeamID 42.`
			require.Equal(t, expectedMsg, err.Error())
		},
	}, {
		name: "WithTeamName",
		fn: func(t *testing.T) {
			err := alreadyExists("User", "alice").WithTeamName("Falcon Team")
			expectedMsg := `User "alice" already exists with team "Falcon Team".`
			require.Equal(t, expectedMsg, err.Error())
		},
	}, {
		name: "IsError",
		fn: func(t *testing.T) {
			err := alreadyExists("User", "alice")

			require.True(t, err.IsExists())
			expectedMsg := `User "alice" already exists`
			require.Equal(t, expectedMsg, err.Error())
			require.Equal(t, err.Resource(), "User")
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t)
		})
	}
}
