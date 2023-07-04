package authz

import (
	"context"
	"encoding/json"
	"fmt"
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
	runNew    = fleet.ActionRunNew
	changePwd = fleet.ActionChangePassword
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
	action string
	allow  bool
}

func TestAuthorizeAppConfig(t *testing.T) {
	t.Parallel()

	config := &fleet.AppConfig{}
	runTestCases(t, []authTestCase{
		{user: nil, object: config, action: read, allow: false},
		{user: nil, object: config, action: write, allow: false},

		{user: test.UserNoRoles, object: config, action: read, allow: false},
		{user: test.UserNoRoles, object: config, action: write, allow: false},

		{user: test.UserAdmin, object: config, action: read, allow: true},
		{user: test.UserAdmin, object: config, action: write, allow: true},

		{user: test.UserMaintainer, object: config, action: read, allow: true},
		{user: test.UserMaintainer, object: config, action: write, allow: false},

		{user: test.UserObserver, object: config, action: read, allow: true},
		{user: test.UserObserver, object: config, action: write, allow: false},

		{user: test.UserObserverPlus, object: config, action: read, allow: true},
		{user: test.UserObserverPlus, object: config, action: write, allow: false},

		{user: test.UserGitOps, object: config, action: read, allow: false},
		{user: test.UserGitOps, object: config, action: write, allow: true},

		{user: test.UserTeamAdminTeam1, object: config, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: config, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: config, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: config, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: config, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: config, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: config, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: config, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: config, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: config, action: write, allow: false},
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

		{user: test.UserObserverPlus, object: session, action: read, allow: false},
		{user: test.UserObserverPlus, object: session, action: write, allow: false},
		{user: test.UserObserverPlus, object: &fleet.Session{UserID: test.UserObserverPlus.ID}, action: read, allow: true},
		{user: test.UserObserverPlus, object: &fleet.Session{UserID: test.UserObserverPlus.ID}, action: write, allow: true},
	})
}

func TestAuthorizeActivity(t *testing.T) {
	t.Parallel()

	activity := &fleet.Activity{}

	runTestCases(t, []authTestCase{
		// All global roles except GitOps can read activities.
		{user: nil, object: activity, action: read, allow: false},
		{user: test.UserAdmin, object: activity, action: read, allow: true},
		{user: test.UserMaintainer, object: activity, action: read, allow: true},
		{user: test.UserObserver, object: activity, action: read, allow: true},
		{user: test.UserObserverPlus, object: activity, action: read, allow: true},
		{user: test.UserGitOps, object: activity, action: read, allow: false},

		// Team roles cannot read activites.
		{user: test.UserTeamAdminTeam1, object: activity, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: activity, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: activity, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: activity, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: activity, action: read, allow: false},
	})
}

func TestAuthorizeUser(t *testing.T) {
	t.Parallel()

	newUser := &fleet.User{}
	user := &fleet.User{ID: 42}
	newTeamUser := &fleet.User{
		Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}},
	}
	teamAdmin := &fleet.User{
		ID:    101,
		Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}},
	}
	teamObserver := &fleet.User{
		ID:    102,
		Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}},
	}

	runTestCases(t, []authTestCase{
		{user: nil, object: user, action: read, allow: false},
		{user: nil, object: user, action: write, allow: false},
		{user: nil, object: user, action: writeRole, allow: false},
		{user: nil, object: user, action: changePwd, allow: false},
		{user: nil, object: newUser, action: write, allow: false},

		// Global admin can read/write all and create new.
		{user: test.UserAdmin, object: user, action: read, allow: true},
		{user: test.UserAdmin, object: user, action: write, allow: true},
		{user: test.UserAdmin, object: user, action: writeRole, allow: true},
		{user: test.UserAdmin, object: user, action: changePwd, allow: true},
		{user: test.UserAdmin, object: newUser, action: write, allow: true},
		{user: test.UserAdmin, object: test.UserAdmin, action: read, allow: true},
		{user: test.UserAdmin, object: test.UserAdmin, action: write, allow: true},
		{user: test.UserAdmin, object: test.UserAdmin, action: writeRole, allow: true},
		{user: test.UserAdmin, object: test.UserAdmin, action: changePwd, allow: true},

		// Global maintainers cannot read/write users.
		{user: test.UserMaintainer, object: user, action: read, allow: false},
		{user: test.UserMaintainer, object: user, action: write, allow: false},
		{user: test.UserMaintainer, object: user, action: writeRole, allow: false},
		{user: test.UserMaintainer, object: user, action: changePwd, allow: false},
		// Global maintainers cannot create users.
		{user: test.UserMaintainer, object: newUser, action: write, allow: false},
		// Global maintainers can read/write itself (besides roles).
		{user: test.UserMaintainer, object: test.UserMaintainer, action: read, allow: true},
		{user: test.UserMaintainer, object: test.UserMaintainer, action: write, allow: true},
		{user: test.UserMaintainer, object: test.UserMaintainer, action: writeRole, allow: false},
		{user: test.UserMaintainer, object: test.UserMaintainer, action: changePwd, allow: true},

		// Users without roles cannot read/write users.
		{user: test.UserNoRoles, object: user, action: read, allow: false},
		{user: test.UserNoRoles, object: user, action: write, allow: false},
		{user: test.UserNoRoles, object: user, action: writeRole, allow: false},
		{user: test.UserNoRoles, object: user, action: changePwd, allow: false},
		// User without roles cannot add new users.
		{user: test.UserNoRoles, object: newUser, action: write, allow: false},
		// User without roles can read/write itself (besides roles).
		{user: test.UserNoRoles, object: test.UserNoRoles, action: read, allow: true},
		{user: test.UserNoRoles, object: test.UserNoRoles, action: write, allow: true},
		{user: test.UserNoRoles, object: test.UserNoRoles, action: writeRole, allow: false},
		{user: test.UserNoRoles, object: test.UserNoRoles, action: changePwd, allow: true},

		// Global observers cannot read/write users.
		{user: test.UserObserver, object: user, action: read, allow: false},
		{user: test.UserObserver, object: user, action: write, allow: false},
		{user: test.UserObserver, object: user, action: writeRole, allow: false},
		{user: test.UserObserver, object: user, action: changePwd, allow: false},
		// Global observers cannot create users.
		{user: test.UserObserver, object: newUser, action: write, allow: false},
		// Global observers can read/write itself (besides roles).
		{user: test.UserObserver, object: test.UserObserver, action: read, allow: true},
		{user: test.UserObserver, object: test.UserObserver, action: write, allow: true},
		{user: test.UserObserver, object: test.UserObserver, action: writeRole, allow: false},
		{user: test.UserObserver, object: test.UserObserver, action: changePwd, allow: true},

		// Global observers+ cannot read/write users.
		{user: test.UserObserverPlus, object: user, action: read, allow: false},
		{user: test.UserObserverPlus, object: user, action: write, allow: false},
		{user: test.UserObserverPlus, object: user, action: writeRole, allow: false},
		{user: test.UserObserverPlus, object: user, action: changePwd, allow: false},
		// Global observers+ cannot create users.
		{user: test.UserObserverPlus, object: newUser, action: write, allow: false},
		// Global observers+ can read/write itself (besides roles).
		{user: test.UserObserverPlus, object: test.UserObserverPlus, action: read, allow: true},
		{user: test.UserObserverPlus, object: test.UserObserverPlus, action: write, allow: true},
		{user: test.UserObserverPlus, object: test.UserObserverPlus, action: writeRole, allow: false},
		{user: test.UserObserverPlus, object: test.UserObserverPlus, action: changePwd, allow: true},

		// Team admins cannot read/write global users.
		{user: teamAdmin, object: user, action: read, allow: false},
		{user: teamAdmin, object: user, action: write, allow: false},
		{user: teamAdmin, object: user, action: writeRole, allow: false},
		{user: teamAdmin, object: user, action: changePwd, allow: false},
		// Team admins cannot create new global users.
		{user: teamAdmin, object: newUser, action: write, allow: false},
		// Team admins can read/write team users (except change their password).
		{user: teamAdmin, object: teamObserver, action: read, allow: true},
		{user: teamAdmin, object: teamObserver, action: write, allow: true},
		{user: teamAdmin, object: teamObserver, action: writeRole, allow: true},
		{user: teamAdmin, object: teamObserver, action: changePwd, allow: false},
		// Team admins can add new users to the team.
		{user: teamAdmin, object: newTeamUser, action: write, allow: true},
		// Team admins can read/write itself.
		{user: teamAdmin, object: teamAdmin, action: read, allow: true},
		{user: teamAdmin, object: teamAdmin, action: write, allow: true},
		{user: teamAdmin, object: teamAdmin, action: writeRole, allow: true},
		{user: teamAdmin, object: teamAdmin, action: changePwd, allow: true},
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

		{user: test.UserObserverPlus, object: invite, action: read, allow: false},
		{user: test.UserObserverPlus, object: invite, action: write, allow: false},
	})
}

func TestAuthorizeEnrollSecret(t *testing.T) {
	t.Parallel()

	teamAdmin := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin},
		},
	}
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
	teamObserverPlus := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus},
		},
	}
	teamGitOps := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps},
		},
	}
	globalSecret := &fleet.EnrollSecret{TeamID: nil}
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
		{user: test.UserObserverPlus, object: globalSecret, action: read, allow: false},
		{user: test.UserObserverPlus, object: globalSecret, action: write, allow: false},
		{user: test.UserObserverPlus, object: teamSecret, action: read, allow: false},
		{user: test.UserObserverPlus, object: teamSecret, action: write, allow: false},
		{user: teamObserver, object: globalSecret, action: read, allow: false},
		{user: teamObserver, object: globalSecret, action: write, allow: false},
		{user: teamObserver, object: teamSecret, action: read, allow: false},
		{user: teamObserver, object: teamSecret, action: write, allow: false},
		{user: teamObserverPlus, object: globalSecret, action: read, allow: false},
		{user: teamObserverPlus, object: globalSecret, action: write, allow: false},
		{user: teamObserverPlus, object: teamSecret, action: read, allow: false},
		{user: teamObserverPlus, object: teamSecret, action: write, allow: false},
		{user: teamGitOps, object: globalSecret, action: read, allow: false},
		{user: teamGitOps, object: globalSecret, action: write, allow: false},
		{user: teamGitOps, object: teamSecret, action: read, allow: false},
		{user: teamGitOps, object: teamSecret, action: write, allow: false},

		// Global admin can read/write all.
		{user: test.UserAdmin, object: globalSecret, action: read, allow: true},
		{user: test.UserAdmin, object: globalSecret, action: write, allow: true},
		{user: test.UserAdmin, object: teamSecret, action: read, allow: true},
		{user: test.UserAdmin, object: teamSecret, action: write, allow: true},

		// Global maintainer can read/write all.
		{user: test.UserMaintainer, object: globalSecret, action: read, allow: true},
		{user: test.UserMaintainer, object: globalSecret, action: write, allow: true},
		{user: test.UserMaintainer, object: teamSecret, action: read, allow: true},
		{user: test.UserMaintainer, object: teamSecret, action: write, allow: true},

		// Global GitOps can write global secret but not read it.
		{user: test.UserGitOps, object: globalSecret, action: read, allow: false},
		{user: test.UserGitOps, object: globalSecret, action: write, allow: true},
		// Global GitOps cannot read/write team secrets.
		{user: test.UserGitOps, object: teamSecret, action: read, allow: false},
		{user: test.UserGitOps, object: teamSecret, action: write, allow: false},

		// Team admin cannot read/write global secret.
		{user: teamAdmin, object: globalSecret, action: read, allow: false},
		{user: teamAdmin, object: globalSecret, action: write, allow: false},
		// Team admin can read/write team secret.
		{user: teamAdmin, object: teamSecret, action: read, allow: true},
		{user: teamAdmin, object: teamSecret, action: write, allow: true},

		// Team maintainer cannot read/write global secret.
		{user: teamMaintainer, object: globalSecret, action: read, allow: false},
		{user: teamMaintainer, object: globalSecret, action: write, allow: false},
		// Team maintainer can read/write team secret.
		{user: teamMaintainer, object: teamSecret, action: read, allow: true},
		{user: teamMaintainer, object: teamSecret, action: write, allow: true},
	})
}

func TestAuthorizeTeam(t *testing.T) {
	t.Parallel()

	team := &fleet.Team{} // Empty team is used to "list teams"
	team1 := &fleet.Team{ID: 1}
	team2 := &fleet.Team{ID: 2}
	runTestCases(t, []authTestCase{
		{user: nil, object: team, action: read, allow: false},
		{user: nil, object: team, action: write, allow: false},

		{user: test.UserNoRoles, object: team, action: read, allow: true},
		{user: test.UserNoRoles, object: team, action: write, allow: false},
		{user: test.UserNoRoles, object: team1, action: read, allow: false},
		{user: test.UserNoRoles, object: team1, action: write, allow: false},

		{user: test.UserAdmin, object: team, action: read, allow: true},
		{user: test.UserAdmin, object: team, action: write, allow: true},
		{user: test.UserAdmin, object: team1, action: read, allow: true},
		{user: test.UserAdmin, object: team1, action: write, allow: true},

		{user: test.UserMaintainer, object: team, action: read, allow: true},
		{user: test.UserMaintainer, object: team, action: write, allow: false},
		{user: test.UserMaintainer, object: team1, action: read, allow: true},
		{user: test.UserMaintainer, object: team1, action: write, allow: false},

		{user: test.UserObserver, object: team, action: read, allow: true},
		{user: test.UserObserver, object: team, action: write, allow: false},
		{user: test.UserObserver, object: team1, action: read, allow: true},
		{user: test.UserObserver, object: team1, action: write, allow: false},

		{user: test.UserObserverPlus, object: team, action: read, allow: true},
		{user: test.UserObserverPlus, object: team, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1, action: read, allow: true},
		{user: test.UserObserverPlus, object: team1, action: write, allow: false},

		{user: test.UserGitOps, object: team, action: read, allow: true},
		{user: test.UserGitOps, object: team, action: write, allow: true},
		{user: test.UserGitOps, object: team1, action: read, allow: false},
		{user: test.UserGitOps, object: team1, action: write, allow: true},

		{user: test.UserTeamAdminTeam1, object: team, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: team, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team2, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team2, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: team, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team2, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team2, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: team, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: team, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: team1, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: team, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: team, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: team1, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: team, action: read, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team, action: write, allow: false},
		// Team GitOps cannot read its team but can write it.
		{user: test.UserTeamGitOpsTeam1, object: team1, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team2, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team2, action: write, allow: false},
	})
}

func TestAuthorizeLabel(t *testing.T) {
	t.Parallel()

	label := &fleet.Label{}
	runTestCases(t, []authTestCase{
		{user: nil, object: label, action: read, allow: false},
		{user: nil, object: label, action: write, allow: false},

		{user: test.UserNoRoles, object: label, action: read, allow: false},
		{user: test.UserNoRoles, object: label, action: write, allow: false},

		{user: test.UserAdmin, object: label, action: read, allow: true},
		{user: test.UserAdmin, object: label, action: write, allow: true},

		{user: test.UserMaintainer, object: label, action: read, allow: true},
		{user: test.UserMaintainer, object: label, action: write, allow: true},

		{user: test.UserObserver, object: label, action: read, allow: true},
		{user: test.UserObserver, object: label, action: write, allow: false},

		{user: test.UserObserverPlus, object: label, action: read, allow: true},
		{user: test.UserObserverPlus, object: label, action: write, allow: false},

		// Global GitOps can write, but not read labels.
		{user: test.UserGitOps, object: label, action: read, allow: false},
		{user: test.UserGitOps, object: label, action: write, allow: true},

		// Team GitOps cannot read or write labels.
		{user: test.UserTeamGitOpsTeam1, object: label, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: label, action: write, allow: false},
	})
}

func TestAuthorizeSoftwareInventory(t *testing.T) {
	t.Parallel()

	softwareInventory := &fleet.AuthzSoftwareInventory{}
	runTestCases(t, []authTestCase{
		{user: nil, object: softwareInventory, action: read, allow: false},
		{user: test.UserNoRoles, object: softwareInventory, action: read, allow: false},
		{user: test.UserAdmin, object: softwareInventory, action: read, allow: true},
		{user: test.UserMaintainer, object: softwareInventory, action: read, allow: true},
		{user: test.UserObserver, object: softwareInventory, action: read, allow: true},
		{user: test.UserObserverPlus, object: softwareInventory, action: read, allow: true},
		{user: test.UserGitOps, object: softwareInventory, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: softwareInventory, action: read, allow: false},
	})
}

func TestAuthorizeHost(t *testing.T) {
	t.Parallel()

	teamAdmin := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin},
		},
	}
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
	teamObserverPlus := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus},
		},
	}
	teamGitOps := &fleet.User{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps},
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

		// No host access if the user has no roles.
		{user: test.UserNoRoles, object: host, action: read, allow: false},
		{user: test.UserNoRoles, object: host, action: write, allow: false},
		{user: test.UserNoRoles, object: host, action: list, allow: false},
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

		// Global observer+ can read all
		{user: test.UserObserverPlus, object: host, action: read, allow: true},
		{user: test.UserObserverPlus, object: host, action: write, allow: false},
		{user: test.UserObserverPlus, object: host, action: list, allow: true},
		{user: test.UserObserverPlus, object: hostTeam1, action: read, allow: true},
		{user: test.UserObserverPlus, object: hostTeam1, action: write, allow: false},
		{user: test.UserObserverPlus, object: hostTeam2, action: read, allow: true},
		{user: test.UserObserverPlus, object: hostTeam2, action: write, allow: false},

		// Global admin can read/write all
		{user: test.UserAdmin, object: host, action: read, allow: true},
		{user: test.UserAdmin, object: host, action: write, allow: true},
		{user: test.UserAdmin, object: host, action: list, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: read, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: write, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: read, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: write, allow: true},

		// Global maintainer can read/write all
		{user: test.UserMaintainer, object: host, action: read, allow: true},
		{user: test.UserMaintainer, object: host, action: write, allow: true},
		{user: test.UserMaintainer, object: host, action: list, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: read, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: write, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: read, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: write, allow: true},

		// Global GitOps can write (not read) all.
		{user: test.UserGitOps, object: host, action: read, allow: false},
		{user: test.UserGitOps, object: host, action: write, allow: true},
		{user: test.UserGitOps, object: host, action: list, allow: false},
		{user: test.UserGitOps, object: hostTeam1, action: read, allow: false},
		{user: test.UserGitOps, object: hostTeam1, action: write, allow: true},
		{user: test.UserGitOps, object: hostTeam2, action: read, allow: false},
		{user: test.UserGitOps, object: hostTeam2, action: write, allow: true},

		// Team observer can read only on appropriate team
		{user: teamObserver, object: host, action: read, allow: false},
		{user: teamObserver, object: host, action: write, allow: false},
		{user: teamObserver, object: host, action: list, allow: true},
		{user: teamObserver, object: hostTeam1, action: read, allow: true},
		{user: teamObserver, object: hostTeam1, action: write, allow: false},
		{user: teamObserver, object: hostTeam2, action: read, allow: false},
		{user: teamObserver, object: hostTeam2, action: write, allow: false},

		// Team observer+ can read only on appropriate team
		{user: teamObserverPlus, object: host, action: read, allow: false},
		{user: teamObserverPlus, object: host, action: write, allow: false},
		{user: teamObserverPlus, object: host, action: list, allow: true},
		{user: teamObserverPlus, object: hostTeam1, action: read, allow: true},
		{user: teamObserverPlus, object: hostTeam1, action: write, allow: false},
		{user: teamObserverPlus, object: hostTeam2, action: read, allow: false},
		{user: teamObserverPlus, object: hostTeam2, action: write, allow: false},

		// Team maintainer can read/write only on appropriate team
		{user: teamMaintainer, object: host, action: read, allow: false},
		{user: teamMaintainer, object: host, action: write, allow: false},
		{user: teamMaintainer, object: host, action: list, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: read, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: write, allow: true},
		{user: teamMaintainer, object: hostTeam2, action: read, allow: false},
		{user: teamMaintainer, object: hostTeam2, action: write, allow: false},

		// Team admin can read/write only on appropriate team
		{user: teamAdmin, object: host, action: read, allow: false},
		{user: teamAdmin, object: host, action: write, allow: false},
		{user: teamAdmin, object: host, action: list, allow: true},
		{user: teamAdmin, object: hostTeam1, action: read, allow: true},
		{user: teamAdmin, object: hostTeam1, action: write, allow: true},
		{user: teamAdmin, object: hostTeam2, action: read, allow: false},
		{user: teamAdmin, object: hostTeam2, action: write, allow: false},

		// Team GitOps can cannot read/write hosts.
		{user: teamGitOps, object: host, action: read, allow: false},
		{user: teamGitOps, object: host, action: write, allow: false},
		{user: teamGitOps, object: host, action: list, allow: false},
		{user: teamGitOps, object: hostTeam1, action: read, allow: false},
		{user: teamGitOps, object: hostTeam1, action: write, allow: false},
		{user: teamGitOps, object: hostTeam2, action: read, allow: false},
		{user: teamGitOps, object: hostTeam2, action: write, allow: false},
	})
}

func TestAuthorizeQuery(t *testing.T) {
	t.Parallel()

	teamMaintainer := &fleet.User{
		ID: 100,
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer},
		},
	}
	teamAdmin := &fleet.User{
		ID: 101,
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin},
		},
	}
	teamObserver := &fleet.User{
		ID: 102,
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver},
		},
	}
	twoTeamsAdminObs := &fleet.User{
		ID: 103,
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin},
			{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver},
		},
	}
	teamObserverPlus := &fleet.User{
		ID: 104,
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus},
		},
	}
	teamGitOps := &fleet.User{
		ID: 105,
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps},
		},
	}

	query := &fleet.Query{ObserverCanRun: false}
	emptyTquery := &fleet.TargetedQuery{Query: query}
	team1Query := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{1}}, Query: query}
	team12Query := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2}}, Query: query}
	team2Query := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{2}}, Query: query}
	team123Query := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2, 3}}, Query: query}

	observerQuery := &fleet.Query{ObserverCanRun: true}
	emptyTobsQuery := &fleet.TargetedQuery{Query: observerQuery}
	team1ObsQuery := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{1}}, Query: observerQuery}
	team12ObsQuery := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2}}, Query: observerQuery}
	team2ObsQuery := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{2}}, Query: observerQuery}
	team123ObsQuery := &fleet.TargetedQuery{HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2, 3}}, Query: observerQuery}

	teamAdminQuery := &fleet.Query{ID: 1, AuthorID: ptr.Uint(teamAdmin.ID), ObserverCanRun: false}
	teamMaintQuery := &fleet.Query{ID: 2, AuthorID: ptr.Uint(teamMaintainer.ID), ObserverCanRun: false}
	globalAdminQuery := &fleet.Query{ID: 3, AuthorID: ptr.Uint(test.UserAdmin.ID), ObserverCanRun: false}
	globalGitOpsQuery := &fleet.Query{ID: 4, AuthorID: ptr.Uint(test.UserGitOps.ID), ObserverCanRun: false}
	teamGitOpsQuery := &fleet.Query{ID: 5, AuthorID: ptr.Uint(teamGitOps.ID), ObserverCanRun: false}

	runTestCases(t, []authTestCase{
		// No access
		{user: nil, object: query, action: read, allow: false},
		{user: nil, object: query, action: write, allow: false},
		{user: nil, object: teamAdminQuery, action: write, allow: false},
		{user: nil, object: emptyTquery, action: run, allow: false},
		{user: nil, object: team1Query, action: run, allow: false},
		{user: nil, object: query, action: runNew, allow: false},
		{user: nil, object: observerQuery, action: read, allow: false},
		{user: nil, object: observerQuery, action: write, allow: false},
		{user: nil, object: emptyTobsQuery, action: run, allow: false},
		{user: nil, object: team1ObsQuery, action: run, allow: false},
		{user: nil, object: observerQuery, action: runNew, allow: false},

		// User with no roles cannot access queries.
		{user: test.UserNoRoles, object: query, action: read, allow: false},
		{user: test.UserNoRoles, object: query, action: write, allow: false},
		{user: test.UserNoRoles, object: teamAdminQuery, action: write, allow: false},
		{user: test.UserNoRoles, object: emptyTquery, action: run, allow: false},
		{user: test.UserNoRoles, object: team1Query, action: run, allow: false},
		{user: test.UserNoRoles, object: query, action: runNew, allow: false},
		{user: test.UserNoRoles, object: observerQuery, action: read, allow: false},
		{user: test.UserNoRoles, object: observerQuery, action: write, allow: false},
		{user: test.UserNoRoles, object: emptyTobsQuery, action: run, allow: false},
		{user: test.UserNoRoles, object: team1ObsQuery, action: run, allow: false},
		{user: test.UserNoRoles, object: observerQuery, action: runNew, allow: false},

		// Global observer can read
		{user: test.UserObserver, object: query, action: read, allow: true},
		{user: test.UserObserver, object: query, action: write, allow: false},
		{user: test.UserObserver, object: teamAdminQuery, action: write, allow: false},
		{user: test.UserObserver, object: emptyTquery, action: run, allow: false},
		{user: test.UserObserver, object: team1Query, action: run, allow: false},
		{user: test.UserObserver, object: query, action: runNew, allow: false},
		{user: test.UserObserver, object: observerQuery, action: read, allow: true},
		{user: test.UserObserver, object: observerQuery, action: write, allow: false},
		{user: test.UserObserver, object: emptyTobsQuery, action: run, allow: true}, // can run observer query
		{user: test.UserObserver, object: team1ObsQuery, action: run, allow: true},  // can run observer query
		{user: test.UserObserver, object: team12ObsQuery, action: run, allow: true}, // can run observer query
		{user: test.UserObserver, object: observerQuery, action: runNew, allow: false},

		// Global observer+ can read all queries, not write them, and can run any query.
		{user: test.UserObserverPlus, object: query, action: read, allow: true},
		{user: test.UserObserverPlus, object: query, action: write, allow: false},
		{user: test.UserObserverPlus, object: teamAdminQuery, action: write, allow: false},
		{user: test.UserObserverPlus, object: emptyTquery, action: run, allow: true},
		{user: test.UserObserverPlus, object: team1Query, action: run, allow: true},
		{user: test.UserObserverPlus, object: query, action: runNew, allow: true},
		{user: test.UserObserverPlus, object: observerQuery, action: read, allow: true},
		{user: test.UserObserverPlus, object: observerQuery, action: write, allow: false},
		{user: test.UserObserverPlus, object: emptyTobsQuery, action: run, allow: true}, // can run observer query
		{user: test.UserObserverPlus, object: team1ObsQuery, action: run, allow: true},  // can run observer query
		{user: test.UserObserverPlus, object: team12ObsQuery, action: run, allow: true}, // can run observer query
		{user: test.UserObserverPlus, object: observerQuery, action: runNew, allow: true},

		// Global maintainer can read/write (even not authored by them)/run any.
		{user: test.UserMaintainer, object: query, action: read, allow: true},
		{user: test.UserMaintainer, object: query, action: write, allow: true},
		{user: test.UserMaintainer, object: teamMaintQuery, action: write, allow: true},
		{user: test.UserMaintainer, object: globalAdminQuery, action: write, allow: true},
		{user: test.UserMaintainer, object: emptyTquery, action: run, allow: true},
		{user: test.UserMaintainer, object: team1Query, action: run, allow: true},
		{user: test.UserMaintainer, object: query, action: runNew, allow: true},
		{user: test.UserMaintainer, object: observerQuery, action: read, allow: true},
		{user: test.UserMaintainer, object: observerQuery, action: write, allow: true},
		{user: test.UserMaintainer, object: emptyTobsQuery, action: run, allow: true},
		{user: test.UserMaintainer, object: team1ObsQuery, action: run, allow: true},
		{user: test.UserMaintainer, object: observerQuery, action: runNew, allow: true},

		// Global admin can read/write (even not authored by them)/run any
		{user: test.UserAdmin, object: query, action: read, allow: true},
		{user: test.UserAdmin, object: query, action: write, allow: true},
		{user: test.UserAdmin, object: teamMaintQuery, action: write, allow: true},
		{user: test.UserAdmin, object: globalAdminQuery, action: write, allow: true},
		{user: test.UserAdmin, object: emptyTquery, action: run, allow: true},
		{user: test.UserAdmin, object: team1Query, action: run, allow: true},
		{user: test.UserAdmin, object: query, action: runNew, allow: true},
		{user: test.UserAdmin, object: observerQuery, action: read, allow: true},
		{user: test.UserAdmin, object: observerQuery, action: write, allow: true},
		{user: test.UserAdmin, object: emptyTobsQuery, action: run, allow: true},
		{user: test.UserAdmin, object: team1ObsQuery, action: run, allow: true},
		{user: test.UserAdmin, object: observerQuery, action: runNew, allow: true},

		// Global GitOps cannot read, or run any query, but can write.
		{user: test.UserGitOps, object: query, action: read, allow: false},
		{user: test.UserGitOps, object: query, action: write, allow: true},
		{user: test.UserGitOps, object: teamAdminQuery, action: write, allow: true},
		{user: test.UserGitOps, object: emptyTquery, action: run, allow: false},
		{user: test.UserGitOps, object: team1Query, action: run, allow: false},
		{user: test.UserGitOps, object: query, action: runNew, allow: false},
		{user: test.UserGitOps, object: observerQuery, action: read, allow: false},
		{user: test.UserGitOps, object: observerQuery, action: write, allow: true},
		{user: test.UserGitOps, object: emptyTobsQuery, action: run, allow: false},
		{user: test.UserGitOps, object: team1ObsQuery, action: run, allow: false},
		{user: test.UserGitOps, object: team12ObsQuery, action: run, allow: false},
		{user: test.UserGitOps, object: observerQuery, action: runNew, allow: false},

		// Team observer can read and run observer_can_run only
		{user: teamObserver, object: query, action: read, allow: true},
		{user: teamObserver, object: query, action: write, allow: false},
		{user: teamObserver, object: teamAdminQuery, action: write, allow: false},
		{user: teamObserver, object: emptyTquery, action: run, allow: false},
		{user: teamObserver, object: team1Query, action: run, allow: false},
		{user: teamObserver, object: query, action: runNew, allow: false},
		{user: teamObserver, object: observerQuery, action: read, allow: true},
		{user: teamObserver, object: observerQuery, action: write, allow: false},
		{user: teamObserver, object: emptyTobsQuery, action: run, allow: true},  // can run observer query with no targeted team
		{user: teamObserver, object: team1ObsQuery, action: run, allow: true},   // can run observer query filtered to observed team
		{user: teamObserver, object: team12ObsQuery, action: run, allow: false}, // not filtered only to observed teams
		{user: teamObserver, object: team2ObsQuery, action: run, allow: false},  // not filtered only to observed teams
		{user: teamObserver, object: observerQuery, action: runNew, allow: false},

		// Team observer+ can read all queries, not write them, and can run any query.
		{user: teamObserverPlus, object: query, action: read, allow: true},
		{user: teamObserverPlus, object: query, action: write, allow: false},
		{user: teamObserverPlus, object: teamAdminQuery, action: write, allow: false},
		{user: teamObserverPlus, object: emptyTquery, action: run, allow: true},
		{user: teamObserverPlus, object: team1Query, action: run, allow: true},
		{user: teamObserverPlus, object: query, action: runNew, allow: true},
		{user: teamObserverPlus, object: observerQuery, action: read, allow: true},
		{user: teamObserverPlus, object: observerQuery, action: write, allow: false},
		{user: teamObserverPlus, object: emptyTobsQuery, action: run, allow: true},  // can run observer query with no targeted team
		{user: teamObserverPlus, object: team1ObsQuery, action: run, allow: true},   // can run observer query filtered to observed team
		{user: teamObserverPlus, object: team12ObsQuery, action: run, allow: false}, // not filtered only to observed teams
		{user: teamObserverPlus, object: team2ObsQuery, action: run, allow: false},  // not filtered only to observed teams
		{user: teamObserverPlus, object: observerQuery, action: runNew, allow: true},

		// Team maintainer can read/write their own queries/run queries filtered on their team(s)
		{user: teamMaintainer, object: query, action: read, allow: true},
		{user: teamMaintainer, object: query, action: write, allow: true},
		{user: teamMaintainer, object: teamMaintQuery, action: write, allow: true},
		{user: teamMaintainer, object: teamAdminQuery, action: write, allow: false},
		{user: teamMaintainer, object: emptyTquery, action: run, allow: true},
		{user: teamMaintainer, object: team1Query, action: run, allow: true},
		{user: teamMaintainer, object: team12Query, action: run, allow: false},
		{user: teamMaintainer, object: team2Query, action: run, allow: false},
		{user: teamMaintainer, object: query, action: runNew, allow: true},
		{user: teamMaintainer, object: observerQuery, action: read, allow: true},
		{user: teamMaintainer, object: observerQuery, action: write, allow: true},
		{user: teamMaintainer, object: emptyTobsQuery, action: run, allow: true},
		{user: teamMaintainer, object: team1ObsQuery, action: run, allow: true},
		{user: teamMaintainer, object: team12ObsQuery, action: run, allow: false},
		{user: teamMaintainer, object: team2ObsQuery, action: run, allow: false},
		{user: teamMaintainer, object: observerQuery, action: runNew, allow: true},

		// Team admin can read/write their own queries/run queries filtered on their team(s)
		{user: teamAdmin, object: query, action: read, allow: true},
		{user: teamAdmin, object: query, action: write, allow: true},
		{user: teamAdmin, object: teamAdminQuery, action: write, allow: true},
		{user: teamAdmin, object: teamMaintQuery, action: write, allow: false},
		{user: teamAdmin, object: globalAdminQuery, action: write, allow: false},
		{user: teamAdmin, object: emptyTquery, action: run, allow: true},
		{user: teamAdmin, object: team1Query, action: run, allow: true},
		{user: teamAdmin, object: team12Query, action: run, allow: false},
		{user: teamAdmin, object: team2Query, action: run, allow: false},
		{user: teamAdmin, object: query, action: runNew, allow: true},
		{user: teamAdmin, object: observerQuery, action: read, allow: true},
		{user: teamAdmin, object: observerQuery, action: write, allow: true},
		{user: teamAdmin, object: emptyTobsQuery, action: run, allow: true},
		{user: teamAdmin, object: team1ObsQuery, action: run, allow: true},
		{user: teamAdmin, object: team12ObsQuery, action: run, allow: false},
		{user: teamAdmin, object: team2ObsQuery, action: run, allow: false},
		{user: teamAdmin, object: observerQuery, action: runNew, allow: true},

		// Team GitOps cannot read or run any query, but can create new or edit (write) queries authored by it.
		{user: teamGitOps, object: query, action: read, allow: false},
		{user: teamGitOps, object: query, action: write, allow: true},              // create new
		{user: teamGitOps, object: teamAdminQuery, action: write, allow: false},    // not the author
		{user: teamGitOps, object: teamGitOpsQuery, action: write, allow: true},    // author
		{user: teamGitOps, object: globalGitOpsQuery, action: write, allow: false}, // not the author
		{user: teamGitOps, object: emptyTquery, action: run, allow: false},
		{user: teamGitOps, object: team1Query, action: run, allow: false},
		{user: teamGitOps, object: query, action: runNew, allow: false},
		{user: teamGitOps, object: emptyTobsQuery, action: run, allow: false},
		{user: teamGitOps, object: team1ObsQuery, action: run, allow: false},
		{user: teamGitOps, object: team12ObsQuery, action: run, allow: false},
		{user: teamGitOps, object: team2ObsQuery, action: run, allow: false},
		{user: teamGitOps, object: observerQuery, action: runNew, allow: false},

		// User admin on team 1, observer on team 2
		{user: twoTeamsAdminObs, object: query, action: read, allow: true},
		{user: twoTeamsAdminObs, object: query, action: write, allow: true},
		{user: twoTeamsAdminObs, object: teamAdminQuery, action: write, allow: false},
		{user: twoTeamsAdminObs, object: teamMaintQuery, action: write, allow: false},
		{user: twoTeamsAdminObs, object: globalAdminQuery, action: write, allow: false},
		{user: twoTeamsAdminObs, object: emptyTquery, action: run, allow: true},
		{user: twoTeamsAdminObs, object: team1Query, action: run, allow: true},
		{user: twoTeamsAdminObs, object: team12Query, action: run, allow: false}, // user is only observer on team 2
		{user: twoTeamsAdminObs, object: team2Query, action: run, allow: false},
		{user: twoTeamsAdminObs, object: team123Query, action: run, allow: false},
		{user: twoTeamsAdminObs, object: query, action: runNew, allow: true},
		{user: twoTeamsAdminObs, object: observerQuery, action: read, allow: true},
		{user: twoTeamsAdminObs, object: observerQuery, action: write, allow: true},
		{user: twoTeamsAdminObs, object: emptyTobsQuery, action: run, allow: true},
		{user: twoTeamsAdminObs, object: team1ObsQuery, action: run, allow: true},
		{user: twoTeamsAdminObs, object: team12ObsQuery, action: run, allow: true}, // user is at least observer on both teams
		{user: twoTeamsAdminObs, object: team2ObsQuery, action: run, allow: true},
		{user: twoTeamsAdminObs, object: team123ObsQuery, action: run, allow: false}, // not member of team 3
		{user: twoTeamsAdminObs, object: observerQuery, action: runNew, allow: true},
	})
}

func TestAuthorizeTarget(t *testing.T) {
	t.Parallel()

	target := &fleet.Target{}
	runTestCases(t, []authTestCase{
		{user: nil, object: target, action: read, allow: false},

		{user: test.UserNoRoles, object: target, action: read, allow: false},
		{user: test.UserAdmin, object: target, action: read, allow: true},
		{user: test.UserMaintainer, object: target, action: read, allow: true},
		{user: test.UserObserver, object: target, action: read, allow: true},
		{user: test.UserObserverPlus, object: target, action: read, allow: true},
		{user: test.UserGitOps, object: target, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: target, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: target, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: target, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: target, action: read, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: target, action: read, allow: false},
	})
}

func TestAuthorizeUserCreatedPack(t *testing.T) {
	t.Parallel()

	userCreatedPack := &fleet.Pack{
		// Type nil is the type for user-created packs.
		Type: nil,
	}
	runTestCases(t, []authTestCase{
		{user: nil, object: userCreatedPack, action: read, allow: false},
		{user: nil, object: userCreatedPack, action: write, allow: false},

		{user: test.UserNoRoles, object: userCreatedPack, action: read, allow: false},
		{user: test.UserNoRoles, object: userCreatedPack, action: write, allow: false},

		{user: test.UserAdmin, object: userCreatedPack, action: read, allow: true},
		{user: test.UserAdmin, object: userCreatedPack, action: write, allow: true},

		{user: test.UserMaintainer, object: userCreatedPack, action: read, allow: true},
		{user: test.UserMaintainer, object: userCreatedPack, action: write, allow: true},

		{user: test.UserObserver, object: userCreatedPack, action: read, allow: false},
		{user: test.UserObserver, object: userCreatedPack, action: write, allow: false},

		{user: test.UserObserverPlus, object: userCreatedPack, action: read, allow: false},
		{user: test.UserObserverPlus, object: userCreatedPack, action: write, allow: false},

		// This is one exception to the "write only" nature of gitops. To be able to create
		// and edit packs currently it needs read access too.
		{user: test.UserGitOps, object: userCreatedPack, action: read, allow: true},
		{user: test.UserGitOps, object: userCreatedPack, action: write, allow: true},

		{user: test.UserTeamAdminTeam1, object: userCreatedPack, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: userCreatedPack, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: userCreatedPack, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: userCreatedPack, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: userCreatedPack, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: userCreatedPack, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: userCreatedPack, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: userCreatedPack, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: userCreatedPack, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: userCreatedPack, action: write, allow: false},
	})
}

func TestAuthorizeGlobalPack(t *testing.T) {
	t.Parallel()

	globalPack := &fleet.Pack{
		// Type "global" is the type for the one global pack.
		Type: ptr.String("global"),
	}
	runTestCases(t, []authTestCase{
		{user: nil, object: globalPack, action: read, allow: false},
		{user: nil, object: globalPack, action: write, allow: false},

		{user: test.UserNoRoles, object: globalPack, action: read, allow: false},
		{user: test.UserNoRoles, object: globalPack, action: write, allow: false},

		{user: test.UserAdmin, object: globalPack, action: read, allow: true},
		{user: test.UserAdmin, object: globalPack, action: write, allow: true},

		{user: test.UserMaintainer, object: globalPack, action: read, allow: true},
		{user: test.UserMaintainer, object: globalPack, action: write, allow: true},

		{user: test.UserObserver, object: globalPack, action: read, allow: true},
		{user: test.UserObserver, object: globalPack, action: write, allow: false},

		{user: test.UserObserverPlus, object: globalPack, action: read, allow: true},
		{user: test.UserObserverPlus, object: globalPack, action: write, allow: false},

		// This is one exception to the "write only" nature of gitops. To be able to create
		// and edit packs currently it needs read access too.
		{user: test.UserGitOps, object: globalPack, action: read, allow: true},
		{user: test.UserGitOps, object: globalPack, action: write, allow: true},

		{user: test.UserTeamAdminTeam1, object: globalPack, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: globalPack, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalPack, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: globalPack, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalPack, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: globalPack, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalPack, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: globalPack, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalPack, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalPack, action: write, allow: false},
	})
}

func TestAuthorizeTeamPack(t *testing.T) {
	t.Parallel()

	team1Pack := &fleet.Pack{Type: ptr.String("team-1")}
	team2Pack := &fleet.Pack{Type: ptr.String("team-2")}
	runTestCases(t, []authTestCase{
		{user: test.UserAdmin, object: team1Pack, action: read, allow: true},
		{user: test.UserAdmin, object: team1Pack, action: write, allow: true},

		{user: test.UserMaintainer, object: team1Pack, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Pack, action: write, allow: true},

		{user: test.UserObserver, object: team1Pack, action: read, allow: false},
		{user: test.UserObserver, object: team1Pack, action: write, allow: false},

		{user: test.UserObserverPlus, object: team1Pack, action: read, allow: false},
		{user: test.UserObserverPlus, object: team1Pack, action: write, allow: false},

		// This is one exception to the "write only" nature of gitops. To be able to create
		// and edit packs currently it needs read access too.
		{user: test.UserGitOps, object: team1Pack, action: read, allow: true},
		{user: test.UserGitOps, object: team1Pack, action: write, allow: true},

		{user: test.UserTeamAdminTeam1, object: team1Pack, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Pack, action: write, allow: true},

		{user: test.UserTeamMaintainerTeam1, object: team1Pack, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Pack, action: read, allow: true},

		{user: test.UserTeamObserverTeam1, object: team1Pack, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: team1Pack, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: team1Pack, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: team1Pack, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: team1Pack, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Pack, action: write, allow: true},

		{user: test.UserTeamAdminTeam1, object: team2Pack, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team2Pack, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: team2Pack, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team2Pack, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: team2Pack, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2Pack, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: team2Pack, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2Pack, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: team2Pack, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team2Pack, action: write, allow: false},
	})
}

func TestAuthorizeCarve(t *testing.T) {
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
		{user: test.UserObserverPlus, object: carve, action: read, allow: false},
		{user: test.UserObserverPlus, object: carve, action: write, allow: false},
		{user: test.UserGitOps, object: carve, action: read, allow: false},
		{user: test.UserGitOps, object: carve, action: write, allow: false},

		// Only admins allowed
		{user: test.UserAdmin, object: carve, action: read, allow: true},
		{user: test.UserAdmin, object: carve, action: write, allow: true},
	})
}

func TestAuthorizeGlobalPolicy(t *testing.T) {
	t.Parallel()

	globalPolicy := &fleet.Policy{}
	runTestCases(t, []authTestCase{
		{user: nil, object: globalPolicy, action: write, allow: false},
		{user: nil, object: globalPolicy, action: read, allow: false},

		{user: test.UserNoRoles, object: globalPolicy, action: write, allow: false},
		{user: test.UserNoRoles, object: globalPolicy, action: read, allow: false},

		{user: test.UserAdmin, object: globalPolicy, action: write, allow: true},
		{user: test.UserAdmin, object: globalPolicy, action: read, allow: true},

		{user: test.UserMaintainer, object: globalPolicy, action: write, allow: true},
		{user: test.UserMaintainer, object: globalPolicy, action: read, allow: true},

		{user: test.UserObserver, object: globalPolicy, action: write, allow: false},
		{user: test.UserObserver, object: globalPolicy, action: read, allow: true},

		{user: test.UserObserverPlus, object: globalPolicy, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalPolicy, action: read, allow: true},

		{user: test.UserGitOps, object: globalPolicy, action: write, allow: true},
		{user: test.UserGitOps, object: globalPolicy, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalPolicy, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalPolicy, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam1, object: globalPolicy, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalPolicy, action: read, allow: true},

		{user: test.UserTeamObserverTeam1, object: globalPolicy, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalPolicy, action: read, allow: true},

		{user: test.UserTeamObserverPlusTeam1, object: globalPolicy, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalPolicy, action: read, allow: true},

		{user: test.UserTeamGitOpsTeam1, object: globalPolicy, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalPolicy, action: read, allow: false},
	})
}

func TestAuthorizeTeamPolicy(t *testing.T) {
	t.Parallel()

	team1Policy := &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(1),
		},
	}
	team2Policy := &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(2),
		},
	}
	runTestCases(t, []authTestCase{
		{user: nil, object: team1Policy, action: write, allow: false},
		{user: nil, object: team1Policy, action: read, allow: false},

		{user: test.UserNoRoles, object: team1Policy, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Policy, action: read, allow: false},

		{user: test.UserAdmin, object: team1Policy, action: write, allow: true},
		{user: test.UserAdmin, object: team1Policy, action: read, allow: true},

		{user: test.UserMaintainer, object: team1Policy, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Policy, action: read, allow: true},

		{user: test.UserObserver, object: team1Policy, action: write, allow: false},
		{user: test.UserObserver, object: team1Policy, action: read, allow: true},

		{user: test.UserObserverPlus, object: team1Policy, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Policy, action: read, allow: true},

		{user: test.UserGitOps, object: team1Policy, action: write, allow: true},
		{user: test.UserGitOps, object: team1Policy, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: team1Policy, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam1, object: team1Policy, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamObserverTeam1, object: team1Policy, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamObserverPlusTeam1, object: team1Policy, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamGitOpsTeam1, object: team1Policy, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Policy, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: team2Policy, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: team2Policy, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: team2Policy, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team2Policy, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: team2Policy, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2Policy, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: team2Policy, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2Policy, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: team2Policy, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team2Policy, action: read, allow: false},
	})
}

func TestAuthorizeMDMAppleConfigProfile(t *testing.T) {
	t.Parallel()

	globalProfile := &fleet.MDMAppleConfigProfile{}
	team1Profile := &fleet.MDMAppleConfigProfile{
		TeamID: ptr.Uint(1),
	}
	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalProfile, action: write, allow: false},
		{user: test.UserNoRoles, object: globalProfile, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Profile, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Profile, action: read, allow: false},

		{user: test.UserAdmin, object: globalProfile, action: write, allow: true},
		{user: test.UserAdmin, object: globalProfile, action: read, allow: true},
		{user: test.UserAdmin, object: team1Profile, action: write, allow: true},
		{user: test.UserAdmin, object: team1Profile, action: read, allow: true},

		{user: test.UserMaintainer, object: globalProfile, action: write, allow: true},
		{user: test.UserMaintainer, object: globalProfile, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Profile, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Profile, action: read, allow: true},

		{user: test.UserObserver, object: globalProfile, action: write, allow: false},
		{user: test.UserObserver, object: globalProfile, action: read, allow: false},
		{user: test.UserObserver, object: team1Profile, action: write, allow: false},
		{user: test.UserObserver, object: team1Profile, action: read, allow: false},

		{user: test.UserObserverPlus, object: globalProfile, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalProfile, action: read, allow: false},
		{user: test.UserObserverPlus, object: team1Profile, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Profile, action: read, allow: false},

		{user: test.UserGitOps, object: globalProfile, action: write, allow: true},
		{user: test.UserGitOps, object: globalProfile, action: read, allow: false},
		{user: test.UserGitOps, object: team1Profile, action: write, allow: true},
		{user: test.UserGitOps, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Profile, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Profile, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Profile, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Profile, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamObserverTeam2, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam2, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Profile, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Profile, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalProfile, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalProfile, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Profile, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Profile, action: read, allow: false},
	})
}

func TestAuthorizeMDMAppleSettings(t *testing.T) {
	t.Parallel()

	globalSettings := &fleet.MDMAppleSettingsPayload{}
	team1Settings := &fleet.MDMAppleSettingsPayload{
		TeamID: ptr.Uint(1),
	}
	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalSettings, action: write, allow: false},
		{user: test.UserNoRoles, object: globalSettings, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Settings, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Settings, action: read, allow: false},

		{user: test.UserAdmin, object: globalSettings, action: write, allow: true},
		{user: test.UserAdmin, object: globalSettings, action: read, allow: true},
		{user: test.UserAdmin, object: team1Settings, action: write, allow: true},
		{user: test.UserAdmin, object: team1Settings, action: read, allow: true},

		{user: test.UserMaintainer, object: globalSettings, action: write, allow: true},
		{user: test.UserMaintainer, object: globalSettings, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Settings, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Settings, action: read, allow: true},

		{user: test.UserObserver, object: globalSettings, action: write, allow: false},
		{user: test.UserObserver, object: globalSettings, action: read, allow: false},
		{user: test.UserObserver, object: team1Settings, action: write, allow: false},
		{user: test.UserObserver, object: team1Settings, action: read, allow: false},

		{user: test.UserObserverPlus, object: globalSettings, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalSettings, action: read, allow: false},
		{user: test.UserObserverPlus, object: team1Settings, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Settings, action: read, allow: false},

		{user: test.UserGitOps, object: globalSettings, action: write, allow: true},
		{user: test.UserGitOps, object: globalSettings, action: read, allow: false},
		{user: test.UserGitOps, object: team1Settings, action: write, allow: true},
		{user: test.UserGitOps, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Settings, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Settings, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Settings, action: read, allow: false},
	})
}

func TestAuthorizeMDMAppleSetupAssistant(t *testing.T) {
	t.Parallel()

	globalSettings := &fleet.MDMAppleSetupAssistant{}
	team1Settings := &fleet.MDMAppleSetupAssistant{
		TeamID: ptr.Uint(1),
	}
	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalSettings, action: write, allow: false},
		{user: test.UserNoRoles, object: globalSettings, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Settings, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Settings, action: read, allow: false},

		{user: test.UserAdmin, object: globalSettings, action: write, allow: true},
		{user: test.UserAdmin, object: globalSettings, action: read, allow: true},
		{user: test.UserAdmin, object: team1Settings, action: write, allow: true},
		{user: test.UserAdmin, object: team1Settings, action: read, allow: true},

		{user: test.UserMaintainer, object: globalSettings, action: write, allow: true},
		{user: test.UserMaintainer, object: globalSettings, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Settings, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Settings, action: read, allow: true},

		{user: test.UserObserver, object: globalSettings, action: write, allow: false},
		{user: test.UserObserver, object: globalSettings, action: read, allow: false},
		{user: test.UserObserver, object: team1Settings, action: write, allow: false},
		{user: test.UserObserver, object: team1Settings, action: read, allow: false},

		{user: test.UserObserverPlus, object: globalSettings, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalSettings, action: read, allow: false},
		{user: test.UserObserverPlus, object: team1Settings, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Settings, action: read, allow: false},

		{user: test.UserGitOps, object: globalSettings, action: write, allow: true},
		{user: test.UserGitOps, object: globalSettings, action: read, allow: false},
		{user: test.UserGitOps, object: team1Settings, action: write, allow: true},
		{user: test.UserGitOps, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Settings, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Settings, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Settings, action: read, allow: false},
	})
}

func TestAuthorizeMDMAppleBootstrapPackage(t *testing.T) {
	t.Parallel()

	globalSettings := &fleet.MDMAppleBootstrapPackage{}
	team1Settings := &fleet.MDMAppleBootstrapPackage{
		TeamID: 1,
	}
	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalSettings, action: write, allow: false},
		{user: test.UserNoRoles, object: globalSettings, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Settings, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Settings, action: read, allow: false},

		{user: test.UserAdmin, object: globalSettings, action: write, allow: true},
		{user: test.UserAdmin, object: globalSettings, action: read, allow: true},
		{user: test.UserAdmin, object: team1Settings, action: write, allow: true},
		{user: test.UserAdmin, object: team1Settings, action: read, allow: true},

		{user: test.UserMaintainer, object: globalSettings, action: write, allow: true},
		{user: test.UserMaintainer, object: globalSettings, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Settings, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Settings, action: read, allow: true},

		{user: test.UserObserver, object: globalSettings, action: write, allow: false},
		{user: test.UserObserver, object: globalSettings, action: read, allow: false},
		{user: test.UserObserver, object: team1Settings, action: write, allow: false},
		{user: test.UserObserver, object: team1Settings, action: read, allow: false},

		{user: test.UserObserverPlus, object: globalSettings, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalSettings, action: read, allow: false},
		{user: test.UserObserverPlus, object: team1Settings, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Settings, action: read, allow: false},

		{user: test.UserGitOps, object: globalSettings, action: write, allow: true},
		{user: test.UserGitOps, object: globalSettings, action: read, allow: false},
		{user: test.UserGitOps, object: team1Settings, action: write, allow: true},
		{user: test.UserGitOps, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Settings, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Settings, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Settings, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Settings, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalSettings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalSettings, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Settings, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Settings, action: read, allow: false},
	})
}

func assertAuthorized(t *testing.T, user *fleet.User, object, action interface{}) {
	t.Helper()

	b, _ := json.MarshalIndent(map[string]interface{}{"subject": user, "object": object, "action": action}, "", "  ")
	assert.NoError(t, auth.Authorize(test.UserContext(context.Background(), user), object, action), "should be authorized\n%s", string(b))
}

func assertUnauthorized(t *testing.T, user *fleet.User, object, action interface{}) {
	t.Helper()

	b, _ := json.MarshalIndent(map[string]interface{}{"subject": user, "object": object, "action": action}, "", "  ")
	assert.Error(t, auth.Authorize(test.UserContext(context.Background(), user), object, action), "should be unauthorized\n%s", string(b))
}

func runTestCases(t *testing.T, testCases []authTestCase) {
	t.Helper()

	for _, tt := range testCases {
		tt := tt

		// build a useful test name from user role, object, action and expected result
		action := tt.action
		role := "none"
		if tt.user != nil {
			if tt.user.GlobalRole != nil {
				role = "g:" + *tt.user.GlobalRole
			} else if len(tt.user.Teams) > 0 {
				role = ""
				for _, tm := range tt.user.Teams {
					if role != "" {
						role += ","
					}
					role += tm.Role
				}
			}
		}

		obj := fmt.Sprintf("%T", tt.object)
		if at, ok := tt.object.(AuthzTyper); ok {
			obj = at.AuthzType()
		}

		result := "allow"
		if !tt.allow {
			result = "deny"
		}

		t.Run(action+"_"+obj+"_"+role+"_"+result, func(t *testing.T) {
			t.Parallel()
			if tt.allow {
				assertAuthorized(t, tt.user, tt.object, tt.action)
			} else {
				assertUnauthorized(t, tt.user, tt.object, tt.action)
			}
		})
	}
}

func TestAuthorizeMDMAppleCommand(t *testing.T) {
	t.Parallel()

	globalCommand := &fleet.MDMAppleCommandAuthz{}
	team1Command := &fleet.MDMAppleCommandAuthz{
		TeamID: ptr.Uint(1),
	}
	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalCommand, action: write, allow: false},
		{user: test.UserNoRoles, object: globalCommand, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Command, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Command, action: read, allow: false},

		{user: test.UserAdmin, object: globalCommand, action: write, allow: true},
		{user: test.UserAdmin, object: globalCommand, action: read, allow: true},
		{user: test.UserAdmin, object: team1Command, action: write, allow: true},
		{user: test.UserAdmin, object: team1Command, action: read, allow: true},

		{user: test.UserMaintainer, object: globalCommand, action: write, allow: true},
		{user: test.UserMaintainer, object: globalCommand, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Command, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Command, action: read, allow: true},

		{user: test.UserObserver, object: globalCommand, action: write, allow: false},
		{user: test.UserObserver, object: globalCommand, action: read, allow: true},
		{user: test.UserObserver, object: team1Command, action: write, allow: false},
		{user: test.UserObserver, object: team1Command, action: read, allow: true},

		{user: test.UserObserverPlus, object: globalCommand, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalCommand, action: read, allow: true},
		{user: test.UserObserverPlus, object: team1Command, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Command, action: read, allow: true},

		{user: test.UserGitOps, object: globalCommand, action: write, allow: false},
		{user: test.UserGitOps, object: globalCommand, action: read, allow: false},
		{user: test.UserGitOps, object: team1Command, action: write, allow: false},
		{user: test.UserGitOps, object: team1Command, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Command, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Command, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Command, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Command, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Command, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Command, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Command, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Command, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Command, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Command, action: read, allow: true},

		{user: test.UserTeamObserverTeam2, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Command, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Command, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Command, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Command, action: read, allow: true},

		{user: test.UserTeamObserverPlusTeam2, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Command, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Command, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Command, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Command, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Command, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Command, action: read, allow: false},
	})
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
