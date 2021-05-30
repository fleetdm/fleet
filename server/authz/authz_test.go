package authz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorize(t *testing.T) {
	t.Parallel()

	auth, err := NewAuthorizer()
	require.NoError(t, err)

	require.NoError(t,
		auth.Authorize(
			context.Background(),
			&kolide.User{GlobalRole: ptr.String(kolide.RoleAdmin)},
			map[string]string{"type": "enroll_secret"},
			"read",
		),
	)
}

func TestJSONToInterfaceUser(t *testing.T) {
	t.Parallel()

	subject, err := jsonToInterface(&kolide.User{GlobalRole: ptr.String(kolide.RoleAdmin)})
	require.NoError(t, err)
	{
		subject := subject.(map[string]interface{})
		assert.Equal(t, kolide.RoleAdmin, subject["global_role"])
		assert.Nil(t, subject["teams"])
	}

	subject, err = jsonToInterface(&kolide.User{
		Teams: []kolide.UserTeam{
			{Team: kolide.Team{ID: 3}, Role: kolide.RoleObserver},
			{Team: kolide.Team{ID: 42}, Role: kolide.RoleMaintainer},
		},
	})
	require.NoError(t, err)
	{
		subject := subject.(map[string]interface{})
		assert.Equal(t, nil, subject["global_role"])
		assert.Len(t, subject["teams"], 2)
		assert.Equal(t, kolide.RoleObserver, subject["teams"].([]interface{})[0].(map[string]interface{})["role"])
		assert.Equal(t, json.Number("3"), subject["teams"].([]interface{})[0].(map[string]interface{})["id"])
		assert.Equal(t, kolide.RoleMaintainer, subject["teams"].([]interface{})[1].(map[string]interface{})["role"])
		assert.Equal(t, json.Number("42"), subject["teams"].([]interface{})[1].(map[string]interface{})["id"])
	}
}
