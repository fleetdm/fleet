package authz

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	read      = fleet.ActionRead
	list      = fleet.ActionList
	write     = fleet.ActionWrite
	writeRole = fleet.ActionWriteRole
	run       = fleet.ActionRun
)

var auth *Authorizer

func init() {
	var err error
	auth, err = NewAuthorizer()
	if err != nil {
		panic(err)
	}
}

type authTestCase struct {
	user   *fleet.User
	object interface{}
	action interface{}
	allow  bool
}

func TestAuthorizeAppConfig(t *testing.T) {
	t.Parallel()

	config := &fleet.AppConfig{}
	runTestCases(t, []authTestCase{
		{user: nil, object: config, action: read, allow: false},
		{user: nil, object: config, action: write, allow: false},

		{user: test.UserNoRoles, object: config, action: read, allow: true},
		{user: test.UserNoRoles, object: config, action: write, allow: false},

		{user: test.UserAdmin, object: config, action: read, allow: true},
		{user: test.UserAdmin, object: config, action: write, allow: true},

		{user: test.UserMaintainer, object: config, action: read, allow: true},
		{user: test.UserMaintainer, object: config, action: write, allow: false},

		{user: test.UserObserver, object: config, action: read, allow: true},
		{user: test.UserObserver, object: config, action: write, allow: false},
	})
}

func TestAuthorizeSession(t *testing.T) {
	t.Parallel()

	session := &fleet.Session{UserID: 42}
	runTestCases(t, []authTestCase{
		{user: nil, object: session, action: read, allow: false},
		{user: nil, object: session, action: write, allow: false},

		// Admin can read/write all
		{user: test.UserAdmin, object: session, action: read, allow: true},
		{user: test.UserAdmin, object: session, action: write, allow: true},

		// Regular users can read self
		{user: test.UserMaintainer, object: session, action: read, allow: false},
		{user: test.UserMaintainer, object: session, action: write, allow: false},
		{user: test.UserMaintainer, object: &fleet.Session{UserID: test.UserMaintainer.ID}, action: read, allow: true},
		{user: test.UserMaintainer, object: &fleet.Session{UserID: test.UserMaintainer.ID}, action: write, allow: true},

		{user: test.UserNoRoles, object: session, action: read, allow: false},
		{user: test.UserNoRoles, object: session, action: write, allow: false},
		{user: test.UserNoRoles, object: &fleet.Session{UserID: test.UserNoRoles.ID}, action: read, allow: true},
		{user: test.UserNoRoles, object: &fleet.Session{UserID: test.UserNoRoles.ID}, action: write, allow: true},

		{user: test.UserObserver, object: session, action: read, allow: false},
		{user: test.UserObserver, object: session, action: write, allow: false},
		{user: test.UserObserver, object: &fleet.Session{UserID: test.UserObserver.ID}, action: read, allow: true},
		{user: test.UserObserver, object: &fleet.Session{UserID: test.UserObserver.ID}, action: write, allow: true},
	})
}

func TestAuthorizeUser(t *testing.T) {
	t.Parallel()

	user := &fleet.User{ID: 42}
	runTestCases(t, []authTestCase{
		{user: nil, object: user, action: read, allow: false},
		{user: nil, object: user, action: write, allow: false},
		{user: nil, object: user, action: writeRole, allow: false},

		// Admin can read/write all
		{user: test.UserAdmin, object: user, action: read, allow: true},
		{user: test.UserAdmin, object: user, action: write, allow: true},
		{user: test.UserAdmin, object: user, action: writeRole, allow: true},

		// Regular users can read all users and write self (besides roles)
		{user: test.UserMaintainer, object: user, action: read, allow: true},
		{user: test.UserMaintainer, object: user, action: write, allow: false},
		{user: test.UserMaintainer, object: user, action: writeRole, allow: false},
		{user: test.UserMaintainer, object: test.UserMaintainer, action: read, allow: true},
		{user: test.UserMaintainer, object: test.UserMaintainer, action: write, allow: true},
		{user: test.UserMaintainer, object: test.UserMaintainer, action: writeRole, allow: false},

		{user: test.UserNoRoles, object: user, action: read, allow: true},
		{user: test.UserNoRoles, object: user, action: write, allow: false},
		{user: test.UserNoRoles, object: user, action: writeRole, allow: false},
		{user: test.UserNoRoles, object: test.UserNoRoles, action: read, allow: true},
		{user: test.UserNoRoles, object: test.UserNoRoles, action: write, allow: true},
		{user: test.UserNoRoles, object: test.UserNoRoles, action: writeRole, allow: false},

		{user: test.UserObserver, object: user, action: read, allow: true},
		{user: test.UserObserver, object: user, action: write, allow: false},
		{user: test.UserObserver, object: user, action: writeRole, allow: false},
		{user: test.UserObserver, object: test.UserObserver, action: read, allow: true},
		{user: test.UserObserver, object: test.UserObserver, action: write, allow: true},
		{user: test.UserObserver, object: test.UserObserver, action: writeRole, allow: false},
	})
}

func TestAuthorizeInvite(t *testing.T) {
	t.Parallel()

	invite := &fleet.Invite{}
	runTestCases(t, []authTestCase{
		{user: nil, object: invite, action: read, allow: false},
		{user: nil, object: invite, action: write, allow: false},

		{user: test.UserNoRoles, object: invite, action: read, allow: false},
		{user: test.UserNoRoles, object: invite, action: write, allow: false},

		{user: test.UserAdmin, object: invite, action: read, allow: true},
		{user: test.UserAdmin, object: invite, action: write, allow: true},

		{user: test.UserMaintainer, object: invite, action: read, allow: false},
		{user: test.UserMaintainer, object: invite, action: write, allow: false},

		{user: test.UserObserver, object: invite, action: read, allow: false},
		{user: test.UserObserver, object: invite, action: write, allow: false},
	})
}

func TestAuthorizeEnrollSecret(t *testing.T) {
	t.Parallel()

	teamMaintainer := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer},
		},
	}
	teamObserver := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver},
		},
	}
	globalSecret := &fleet.EnrollSecret{}
	teamSecret := &fleet.EnrollSecret{TeamID: ptr.Uint(1)}
	runTestCases(t, []authTestCase{
		// No access
		{user: nil, object: globalSecret, action: read, allow: false},
		{user: nil, object: globalSecret, action: write, allow: false},
		{user: nil, object: teamSecret, action: read, allow: false},
		{user: nil, object: teamSecret, action: write, allow: false},
		{user: test.UserNoRoles, object: globalSecret, action: read, allow: false},
		{user: test.UserNoRoles, object: globalSecret, action: write, allow: false},
		{user: test.UserNoRoles, object: teamSecret, action: read, allow: false},
		{user: test.UserNoRoles, object: teamSecret, action: write, allow: false},
		{user: test.UserObserver, object: globalSecret, action: read, allow: false},
		{user: test.UserObserver, object: globalSecret, action: write, allow: false},
		{user: test.UserObserver, object: teamSecret, action: read, allow: false},
		{user: test.UserObserver, object: teamSecret, action: write, allow: false},
		{user: teamObserver, object: globalSecret, action: read, allow: false},
		{user: teamObserver, object: globalSecret, action: write, allow: false},
		{user: teamObserver, object: teamSecret, action: read, allow: false},
		{user: teamObserver, object: teamSecret, action: write, allow: false},

		// Admin can read/write all
		{user: test.UserAdmin, object: globalSecret, action: read, allow: true},
		{user: test.UserAdmin, object: globalSecret, action: write, allow: true},
		{user: test.UserAdmin, object: teamSecret, action: read, allow: true},
		{user: test.UserAdmin, object: teamSecret, action: write, allow: true},

		// Maintainer can read all
		{user: test.UserMaintainer, object: globalSecret, action: read, allow: true},
		{user: test.UserMaintainer, object: globalSecret, action: write, allow: false},
		{user: test.UserMaintainer, object: teamSecret, action: read, allow: true},
		{user: test.UserMaintainer, object: teamSecret, action: write, allow: false},

		// Team maintainer can read team secret
		{user: teamMaintainer, object: globalSecret, action: read, allow: false},
		{user: teamMaintainer, object: globalSecret, action: write, allow: false},
		{user: teamMaintainer, object: teamSecret, action: read, allow: true},
		{user: teamMaintainer, object: teamSecret, action: write, allow: false},
	})
}

func TestAuthorizeTeam(t *testing.T) {
	t.Parallel()

	team := &fleet.Team{}
	runTestCases(t, []authTestCase{
		{user: nil, object: team, action: read, allow: false},
		{user: nil, object: team, action: write, allow: false},

		{user: test.UserNoRoles, object: team, action: read, allow: true},
		{user: test.UserNoRoles, object: team, action: write, allow: false},

		{user: test.UserAdmin, object: team, action: read, allow: true},
		{user: test.UserAdmin, object: team, action: write, allow: true},

		{user: test.UserMaintainer, object: team, action: read, allow: true},
		{user: test.UserMaintainer, object: team, action: write, allow: false},

		{user: test.UserObserver, object: team, action: read, allow: true},
		{user: test.UserObserver, object: team, action: write, allow: false},
	})
}

func TestAuthorizeLabel(t *testing.T) {
	t.Parallel()

	label := &fleet.Label{}
	runTestCases(t, []authTestCase{
		{user: nil, object: label, action: read, allow: false},
		{user: nil, object: label, action: write, allow: false},

		{user: test.UserNoRoles, object: label, action: read, allow: true},
		{user: test.UserNoRoles, object: label, action: write, allow: false},

		{user: test.UserAdmin, object: label, action: read, allow: true},
		{user: test.UserAdmin, object: label, action: write, allow: true},

		{user: test.UserMaintainer, object: label, action: read, allow: true},
		{user: test.UserMaintainer, object: label, action: write, allow: true},

		{user: test.UserObserver, object: label, action: read, allow: true},
		{user: test.UserObserver, object: label, action: write, allow: false},
	})
}

func TestAuthorizeHost(t *testing.T) {
	t.Parallel()

	teamMaintainer := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer},
		},
	}
	teamObserver := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver},
		},
	}
	host := &fleet.Host{}
	hostTeam1 := &fleet.Host{TeamID: ptr.Uint(1)}
	hostTeam2 := &fleet.Host{TeamID: ptr.Uint(2)}
	runTestCases(t, []authTestCase{
		// No access
		{user: nil, object: host, action: read, allow: false},
		{user: nil, object: host, action: write, allow: false},
		{user: nil, object: host, action: list, allow: false},
		{user: nil, object: hostTeam1, action: read, allow: false},
		{user: nil, object: hostTeam1, action: write, allow: false},
		{user: nil, object: hostTeam2, action: read, allow: false},
		{user: nil, object: hostTeam2, action: write, allow: false},

		// List but no specific host access
		{user: test.UserNoRoles, object: host, action: read, allow: false},
		{user: test.UserNoRoles, object: host, action: write, allow: false},
		{user: test.UserNoRoles, object: host, action: list, allow: true},
		{user: test.UserNoRoles, object: hostTeam1, action: read, allow: false},
		{user: test.UserNoRoles, object: hostTeam1, action: write, allow: false},
		{user: test.UserNoRoles, object: hostTeam2, action: read, allow: false},
		{user: test.UserNoRoles, object: hostTeam2, action: write, allow: false},

		// Global observer can read all
		{user: test.UserObserver, object: host, action: read, allow: true},
		{user: test.UserObserver, object: host, action: write, allow: false},
		{user: test.UserObserver, object: host, action: list, allow: true},
		{user: test.UserObserver, object: hostTeam1, action: read, allow: true},
		{user: test.UserObserver, object: hostTeam1, action: write, allow: false},
		{user: test.UserObserver, object: hostTeam2, action: read, allow: true},
		{user: test.UserObserver, object: hostTeam2, action: write, allow: false},

		// Global admin/maintainer can read/write all
		{user: test.UserAdmin, object: host, action: read, allow: true},
		{user: test.UserAdmin, object: host, action: write, allow: true},
		{user: test.UserAdmin, object: host, action: list, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: read, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: write, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: read, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: write, allow: true},
		{user: test.UserMaintainer, object: host, action: read, allow: true},
		{user: test.UserMaintainer, object: host, action: write, allow: true},
		{user: test.UserMaintainer, object: host, action: list, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: read, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: write, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: read, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: write, allow: true},

		// Team observer/maintainer can read only on appropriate team
		{user: teamObserver, object: host, action: read, allow: false},
		{user: teamObserver, object: host, action: write, allow: false},
		{user: teamObserver, object: host, action: list, allow: true},
		{user: teamObserver, object: hostTeam1, action: read, allow: true},
		{user: teamObserver, object: hostTeam1, action: write, allow: false},
		{user: teamObserver, object: hostTeam2, action: read, allow: false},
		{user: teamObserver, object: hostTeam2, action: write, allow: false},
		{user: teamMaintainer, object: host, action: read, allow: false},
		{user: teamMaintainer, object: host, action: write, allow: false},
		{user: teamMaintainer, object: host, action: list, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: read, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: write, allow: false},
		{user: teamMaintainer, object: hostTeam2, action: read, allow: false},
		{user: teamMaintainer, object: hostTeam2, action: write, allow: false},
	})
}

func TestAuthorizeQuery(t *testing.T) {
	t.Parallel()

	teamMaintainer := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer},
		},
	}
	teamObserver := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver},
		},
	}
	query := &fleet.Query{}
	observerQuery := &fleet.Query{ObserverCanRun: true}
	runTestCases(t, []authTestCase{
		// No access
		{user: nil, object: query, action: read, allow: false},
		{user: nil, object: query, action: write, allow: false},
		{user: nil, object: query, action: run, allow: false},
		{user: nil, object: observerQuery, action: read, allow: false},
		{user: nil, object: observerQuery, action: write, allow: false},
		{user: nil, object: observerQuery, action: run, allow: false},

		// User can still read queries with no roles
		{user: test.UserNoRoles, object: query, action: read, allow: true},
		{user: test.UserNoRoles, object: query, action: write, allow: false},
		{user: test.UserNoRoles, object: query, action: run, allow: false},
		{user: test.UserNoRoles, object: observerQuery, action: read, allow: true},
		{user: test.UserNoRoles, object: observerQuery, action: write, allow: false},
		{user: test.UserNoRoles, object: query, action: run, allow: false},

		// Global observer can read
		{user: test.UserObserver, object: query, action: read, allow: true},
		{user: test.UserObserver, object: query, action: write, allow: false},
		{user: test.UserObserver, object: query, action: run, allow: false},
		{user: test.UserObserver, object: observerQuery, action: read, allow: true},
		{user: test.UserObserver, object: observerQuery, action: write, allow: false},
		// Can run observer query
		{user: test.UserObserver, object: observerQuery, action: run, allow: true},

		// Global maintainer can read/write/run
		{user: test.UserMaintainer, object: query, action: read, allow: true},
		{user: test.UserMaintainer, object: query, action: write, allow: true},
		{user: test.UserMaintainer, object: query, action: run, allow: true},
		{user: test.UserMaintainer, object: observerQuery, action: read, allow: true},
		{user: test.UserMaintainer, object: observerQuery, action: write, allow: true},
		{user: test.UserMaintainer, object: observerQuery, action: run, allow: true},

		// Global admin can read/write
		{user: test.UserAdmin, object: query, action: read, allow: true},
		{user: test.UserAdmin, object: query, action: write, allow: true},
		{user: test.UserAdmin, object: query, action: run, allow: true},
		{user: test.UserAdmin, object: observerQuery, action: read, allow: true},
		{user: test.UserAdmin, object: observerQuery, action: write, allow: true},
		{user: test.UserAdmin, object: observerQuery, action: run, allow: true},

		// Team observer read
		{user: teamObserver, object: query, action: read, allow: true},
		{user: teamObserver, object: query, action: write, allow: false},
		{user: teamObserver, object: query, action: run, allow: false},
		{user: teamObserver, object: observerQuery, action: read, allow: true},
		{user: teamObserver, object: observerQuery, action: write, allow: false},
		// Can run observer query
		{user: teamObserver, object: observerQuery, action: run, allow: true},

		// Team maintainer can read/write
		{user: teamMaintainer, object: query, action: read, allow: true},
		{user: teamMaintainer, object: query, action: write, allow: false},
		{user: teamMaintainer, object: query, action: run, allow: true},
		{user: teamMaintainer, object: observerQuery, action: read, allow: true},
		{user: teamMaintainer, object: observerQuery, action: write, allow: false},
		{user: teamMaintainer, object: observerQuery, action: run, allow: true},
	})
}

func TestAuthorizeTargets(t *testing.T) {
	t.Parallel()

	target := &fleet.Target{}
	runTestCases(t, []authTestCase{
		{user: nil, object: target, action: read, allow: false},

		// Everyone logged in can retrieve target (filter appropriately for their
		// access)
		{user: test.UserNoRoles, object: target, action: read, allow: true},
		{user: test.UserAdmin, object: target, action: read, allow: true},
		{user: test.UserMaintainer, object: target, action: read, allow: true},
		{user: test.UserObserver, object: target, action: read, allow: true},
	})
}

func TestAuthorizePacks(t *testing.T) {
	t.Parallel()

	pack := &fleet.Pack{}
	runTestCases(t, []authTestCase{
		{user: nil, object: pack, action: read, allow: false},
		{user: nil, object: pack, action: write, allow: false},

		{user: test.UserNoRoles, object: pack, action: read, allow: false},
		{user: test.UserNoRoles, object: pack, action: write, allow: false},

		{user: test.UserAdmin, object: pack, action: read, allow: true},
		{user: test.UserAdmin, object: pack, action: write, allow: true},

		{user: test.UserMaintainer, object: pack, action: read, allow: true},
		{user: test.UserMaintainer, object: pack, action: write, allow: true},

		{user: test.UserObserver, object: pack, action: read, allow: false},
		{user: test.UserObserver, object: pack, action: write, allow: false},
	})
}

func TestAuthorizeCarves(t *testing.T) {
	t.Parallel()

	carve := &fleet.CarveMetadata{}
	runTestCases(t, []authTestCase{
		{user: nil, object: carve, action: read, allow: false},
		{user: nil, object: carve, action: write, allow: false},
		{user: test.UserNoRoles, object: carve, action: read, allow: false},
		{user: test.UserNoRoles, object: carve, action: write, allow: false},
		{user: test.UserMaintainer, object: carve, action: read, allow: false},
		{user: test.UserMaintainer, object: carve, action: write, allow: false},
		{user: test.UserObserver, object: carve, action: read, allow: false},
		{user: test.UserObserver, object: carve, action: write, allow: false},

		// Only admins allowed
		{user: test.UserAdmin, object: carve, action: read, allow: true},
		{user: test.UserAdmin, object: carve, action: write, allow: true},
	})
}

func assertAuthorized(t *testing.T, user *fleet.User, object, action interface{}) {
	t.Helper()

	assert.NoError(t, auth.Authorize(test.UserContext(user), object, action), "should be authorized\n%v\n%v\n%v", user, object, action)
}

func assertUnauthorized(t *testing.T, user *fleet.User, object, action interface{}) {
	t.Helper()

	assert.Error(t, auth.Authorize(test.UserContext(user), object, action), "should be unauthorized\n%v\n%v\n%v", user, object, action)
}

func runTestCases(t *testing.T, testCases []authTestCase) {
	t.Helper()

	for _, tt := range testCases {
		tt := tt
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if tt.allow {
				assertAuthorized(t, tt.user, tt.object, tt.action)
			} else {
				assertUnauthorized(t, tt.user, tt.object, tt.action)
			}
		})
	}

}

func TestJSONToInterfaceUser(t *testing.T) {
	t.Parallel()

	subject, err := jsonToInterface(&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)})
	require.NoError(t, err)
	{
		subject := subject.(map[string]interface{})
		assert.Equal(t, fleet.RoleAdmin, subject["global_role"])
		assert.Nil(t, subject["teams"])
	}

	subject, err = jsonToInterface(&fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 3}, Role: fleet.RoleObserver},
			{Team: fleet.Team{ID: 42}, Role: fleet.RoleMaintainer},
		},
	})
	require.NoError(t, err)
	{
		subject := subject.(map[string]interface{})
		assert.Equal(t, nil, subject["global_role"])
		assert.Len(t, subject["teams"], 2)
		assert.Equal(t, fleet.RoleObserver, subject["teams"].([]interface{})[0].(map[string]interface{})["role"])
		assert.Equal(t, json.Number("3"), subject["teams"].([]interface{})[0].(map[string]interface{})["id"])
		assert.Equal(t, fleet.RoleMaintainer, subject["teams"].([]interface{})[1].(map[string]interface{})["role"])
		assert.Equal(t, json.Number("42"), subject["teams"].([]interface{})[1].(map[string]interface{})["id"])
	}
}
