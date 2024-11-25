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
	read          = fleet.ActionRead
	list          = fleet.ActionList
	write         = fleet.ActionWrite
	writeRole     = fleet.ActionWriteRole
	run           = fleet.ActionRun
	runNew        = fleet.ActionRunNew
	changePwd     = fleet.ActionChangePassword
	selectiveRead = fleet.ActionSelectiveRead
	selectiveList = fleet.ActionSelectiveList
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

		{user: test.UserGitOps, object: config, action: read, allow: true},
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

func TestAuthorizeSoftwareInstaller(t *testing.T) {
	t.Parallel()

	noTeamInstaller := &fleet.SoftwareInstaller{}
	team1Installer := &fleet.SoftwareInstaller{TeamID: ptr.Uint(1)}
	team2Installer := &fleet.SoftwareInstaller{TeamID: ptr.Uint(2)}
	runTestCases(t, []authTestCase{
		{user: nil, object: noTeamInstaller, action: read, allow: false},
		{user: nil, object: noTeamInstaller, action: write, allow: false},
		{user: nil, object: team1Installer, action: read, allow: false},
		{user: nil, object: team1Installer, action: write, allow: false},
		{user: nil, object: team2Installer, action: read, allow: false},
		{user: nil, object: team2Installer, action: write, allow: false},

		{user: test.UserNoRoles, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserNoRoles, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Installer, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Installer, action: write, allow: false},
		{user: test.UserNoRoles, object: team2Installer, action: read, allow: false},
		{user: test.UserNoRoles, object: team2Installer, action: write, allow: false},

		{user: test.UserAdmin, object: noTeamInstaller, action: read, allow: true},
		{user: test.UserAdmin, object: noTeamInstaller, action: write, allow: true},
		{user: test.UserAdmin, object: team1Installer, action: read, allow: true},
		{user: test.UserAdmin, object: team1Installer, action: write, allow: true},
		{user: test.UserAdmin, object: team2Installer, action: read, allow: true},
		{user: test.UserAdmin, object: team2Installer, action: write, allow: true},

		{user: test.UserMaintainer, object: noTeamInstaller, action: read, allow: true},
		{user: test.UserMaintainer, object: noTeamInstaller, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Installer, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Installer, action: write, allow: true},
		{user: test.UserMaintainer, object: team2Installer, action: read, allow: true},
		{user: test.UserMaintainer, object: team2Installer, action: write, allow: true},

		{user: test.UserObserver, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserObserver, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserObserver, object: team1Installer, action: read, allow: false},
		{user: test.UserObserver, object: team1Installer, action: write, allow: false},
		{user: test.UserObserver, object: team2Installer, action: read, allow: false},
		{user: test.UserObserver, object: team2Installer, action: write, allow: false},

		{user: test.UserObserverPlus, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserObserverPlus, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Installer, action: read, allow: false},
		{user: test.UserObserverPlus, object: team1Installer, action: write, allow: false},
		{user: test.UserObserverPlus, object: team2Installer, action: read, allow: false},
		{user: test.UserObserverPlus, object: team2Installer, action: write, allow: false},

		// TODO: confirm gitops permissions
		{user: test.UserGitOps, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserGitOps, object: noTeamInstaller, action: write, allow: true},
		{user: test.UserGitOps, object: team1Installer, action: read, allow: false},
		{user: test.UserGitOps, object: team1Installer, action: write, allow: true},
		{user: test.UserGitOps, object: team2Installer, action: read, allow: false},
		{user: test.UserGitOps, object: team2Installer, action: write, allow: true},

		// TODO: confirm gitops permissions
		{user: test.UserTeamGitOpsTeam1, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Installer, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Installer, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team2Installer, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team2Installer, action: write, allow: false},

		{user: test.UserTeamAdminTeam1, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Installer, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Installer, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team2Installer, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team2Installer, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Installer, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Installer, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team2Installer, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team2Installer, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Installer, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Installer, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2Installer, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2Installer, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: noTeamInstaller, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: noTeamInstaller, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Installer, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Installer, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2Installer, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2Installer, action: write, allow: false},
	})
}

func TestAuthorizeHostSoftwareInstallerResult(t *testing.T) {
	t.Parallel()

	noTeamInstallResult := &fleet.HostSoftwareInstallerResultAuthz{}
	team1InstallResult := &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: ptr.Uint(1)}
	team2InstallResult := &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: ptr.Uint(2)}
	runTestCases(t, []authTestCase{
		// Write permissions
		{user: nil, object: noTeamInstallResult, action: write, allow: false},
		{user: nil, object: team1InstallResult, action: write, allow: false},
		{user: nil, object: team2InstallResult, action: write, allow: false},

		{user: test.UserNoRoles, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserNoRoles, object: team1InstallResult, action: write, allow: false},
		{user: test.UserNoRoles, object: team2InstallResult, action: write, allow: false},

		{user: test.UserAdmin, object: noTeamInstallResult, action: write, allow: true},
		{user: test.UserAdmin, object: team1InstallResult, action: write, allow: true},
		{user: test.UserAdmin, object: team2InstallResult, action: write, allow: true},

		{user: test.UserMaintainer, object: noTeamInstallResult, action: write, allow: true},
		{user: test.UserMaintainer, object: team1InstallResult, action: write, allow: true},
		{user: test.UserMaintainer, object: team2InstallResult, action: write, allow: true},

		{user: test.UserObserver, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserObserver, object: team1InstallResult, action: write, allow: false},
		{user: test.UserObserver, object: team2InstallResult, action: write, allow: false},

		{user: test.UserObserverPlus, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1InstallResult, action: write, allow: false},
		{user: test.UserObserverPlus, object: team2InstallResult, action: write, allow: false},

		{user: test.UserGitOps, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserGitOps, object: team1InstallResult, action: write, allow: false},
		{user: test.UserGitOps, object: team2InstallResult, action: write, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1InstallResult, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team2InstallResult, action: write, allow: false},

		{user: test.UserTeamAdminTeam1, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1InstallResult, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team2InstallResult, action: write, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1InstallResult, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team2InstallResult, action: write, allow: false},

		{user: test.UserTeamObserverTeam1, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1InstallResult, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team2InstallResult, action: write, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: noTeamInstallResult, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1InstallResult, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team2InstallResult, action: write, allow: false},

		// Read permissions
		{user: nil, object: noTeamInstallResult, action: read, allow: false},
		{user: nil, object: team1InstallResult, action: read, allow: false},
		{user: nil, object: team2InstallResult, action: read, allow: false},

		{user: test.UserNoRoles, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserNoRoles, object: team1InstallResult, action: read, allow: false},
		{user: test.UserNoRoles, object: team2InstallResult, action: read, allow: false},

		{user: test.UserAdmin, object: noTeamInstallResult, action: read, allow: true},
		{user: test.UserAdmin, object: team1InstallResult, action: read, allow: true},
		{user: test.UserAdmin, object: team2InstallResult, action: read, allow: true},

		{user: test.UserMaintainer, object: noTeamInstallResult, action: read, allow: true},
		{user: test.UserMaintainer, object: team1InstallResult, action: read, allow: true},
		{user: test.UserMaintainer, object: team2InstallResult, action: read, allow: true},

		{user: test.UserObserver, object: noTeamInstallResult, action: read, allow: true},
		{user: test.UserObserver, object: team1InstallResult, action: read, allow: true},
		{user: test.UserObserver, object: team2InstallResult, action: read, allow: true},

		{user: test.UserObserverPlus, object: noTeamInstallResult, action: read, allow: true},
		{user: test.UserObserverPlus, object: team1InstallResult, action: read, allow: true},
		{user: test.UserObserverPlus, object: team2InstallResult, action: read, allow: true},

		{user: test.UserGitOps, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserGitOps, object: team1InstallResult, action: read, allow: false},
		{user: test.UserGitOps, object: team2InstallResult, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1InstallResult, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team2InstallResult, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1InstallResult, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: team2InstallResult, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1InstallResult, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team2InstallResult, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1InstallResult, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: team2InstallResult, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: noTeamInstallResult, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1InstallResult, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: team2InstallResult, action: read, allow: false},
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
		{user: nil, object: host, action: selectiveList, allow: false},
		{user: nil, object: host, action: selectiveRead, allow: false},
		{user: nil, object: hostTeam1, action: read, allow: false},
		{user: nil, object: hostTeam1, action: write, allow: false},
		{user: nil, object: hostTeam1, action: selectiveRead, allow: false},
		{user: nil, object: hostTeam2, action: read, allow: false},
		{user: nil, object: hostTeam2, action: write, allow: false},
		{user: nil, object: hostTeam2, action: selectiveRead, allow: false},

		// No host access if the user has no roles.
		{user: test.UserNoRoles, object: host, action: read, allow: false},
		{user: test.UserNoRoles, object: host, action: write, allow: false},
		{user: test.UserNoRoles, object: host, action: list, allow: false},
		{user: test.UserNoRoles, object: host, action: selectiveList, allow: false},
		{user: test.UserNoRoles, object: host, action: selectiveRead, allow: false},
		{user: test.UserNoRoles, object: hostTeam1, action: read, allow: false},
		{user: test.UserNoRoles, object: hostTeam1, action: write, allow: false},
		{user: test.UserNoRoles, object: hostTeam1, action: selectiveRead, allow: false},
		{user: test.UserNoRoles, object: hostTeam2, action: read, allow: false},
		{user: test.UserNoRoles, object: hostTeam2, action: write, allow: false},
		{user: test.UserNoRoles, object: hostTeam2, action: selectiveRead, allow: false},

		// Global observer can read all
		{user: test.UserObserver, object: host, action: read, allow: true},
		{user: test.UserObserver, object: host, action: write, allow: false},
		{user: test.UserObserver, object: host, action: list, allow: true},
		{user: test.UserObserver, object: host, action: selectiveList, allow: true},
		{user: test.UserObserver, object: host, action: selectiveRead, allow: true},
		{user: test.UserObserver, object: hostTeam1, action: read, allow: true},
		{user: test.UserObserver, object: hostTeam1, action: selectiveRead, allow: true},
		{user: test.UserObserver, object: hostTeam1, action: write, allow: false},
		{user: test.UserObserver, object: hostTeam2, action: read, allow: true},
		{user: test.UserObserver, object: hostTeam2, action: selectiveRead, allow: true},
		{user: test.UserObserver, object: hostTeam2, action: write, allow: false},

		// Global observer+ can read all
		{user: test.UserObserverPlus, object: host, action: read, allow: true},
		{user: test.UserObserverPlus, object: host, action: write, allow: false},
		{user: test.UserObserverPlus, object: host, action: list, allow: true},
		{user: test.UserObserverPlus, object: host, action: selectiveList, allow: true},
		{user: test.UserObserverPlus, object: host, action: selectiveRead, allow: true},
		{user: test.UserObserverPlus, object: hostTeam1, action: read, allow: true},
		{user: test.UserObserverPlus, object: hostTeam1, action: selectiveRead, allow: true},
		{user: test.UserObserverPlus, object: hostTeam1, action: write, allow: false},
		{user: test.UserObserverPlus, object: hostTeam2, action: read, allow: true},
		{user: test.UserObserverPlus, object: hostTeam2, action: selectiveRead, allow: true},
		{user: test.UserObserverPlus, object: hostTeam2, action: write, allow: false},

		// Global admin can read/write all
		{user: test.UserAdmin, object: host, action: read, allow: true},
		{user: test.UserAdmin, object: host, action: selectiveRead, allow: true},
		{user: test.UserAdmin, object: host, action: write, allow: true},
		{user: test.UserAdmin, object: host, action: list, allow: true},
		{user: test.UserAdmin, object: host, action: selectiveList, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: read, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: selectiveRead, allow: true},
		{user: test.UserAdmin, object: hostTeam1, action: write, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: read, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: selectiveRead, allow: true},
		{user: test.UserAdmin, object: hostTeam2, action: write, allow: true},

		// Global maintainer can read/write all
		{user: test.UserMaintainer, object: host, action: read, allow: true},
		{user: test.UserMaintainer, object: host, action: selectiveRead, allow: true},
		{user: test.UserMaintainer, object: host, action: write, allow: true},
		{user: test.UserMaintainer, object: host, action: list, allow: true},
		{user: test.UserMaintainer, object: host, action: selectiveList, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: read, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: selectiveRead, allow: true},
		{user: test.UserMaintainer, object: hostTeam1, action: write, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: read, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: selectiveRead, allow: true},
		{user: test.UserMaintainer, object: hostTeam2, action: write, allow: true},

		// Global GitOps can write and selectively read all.
		{user: test.UserGitOps, object: host, action: read, allow: false},
		{user: test.UserGitOps, object: host, action: write, allow: true},
		{user: test.UserGitOps, object: host, action: selectiveRead, allow: true},
		{user: test.UserGitOps, object: host, action: list, allow: false},
		{user: test.UserGitOps, object: host, action: selectiveList, allow: true},
		{user: test.UserGitOps, object: hostTeam1, action: read, allow: false},
		{user: test.UserGitOps, object: hostTeam1, action: write, allow: true},
		{user: test.UserGitOps, object: hostTeam1, action: selectiveRead, allow: true},
		{user: test.UserGitOps, object: hostTeam2, action: read, allow: false},
		{user: test.UserGitOps, object: hostTeam2, action: write, allow: true},
		{user: test.UserGitOps, object: hostTeam2, action: selectiveRead, allow: true},

		// Team observer can read only on appropriate team
		{user: teamObserver, object: host, action: read, allow: false},
		{user: teamObserver, object: host, action: selectiveRead, allow: false},
		{user: teamObserver, object: host, action: write, allow: false},
		{user: teamObserver, object: host, action: list, allow: true},
		{user: teamObserver, object: host, action: selectiveList, allow: true},
		{user: teamObserver, object: hostTeam1, action: read, allow: true},
		{user: teamObserver, object: hostTeam1, action: selectiveRead, allow: true},
		{user: teamObserver, object: hostTeam1, action: write, allow: false},
		{user: teamObserver, object: hostTeam2, action: read, allow: false},
		{user: teamObserver, object: hostTeam2, action: selectiveRead, allow: false},
		{user: teamObserver, object: hostTeam2, action: write, allow: false},

		// Team observer+ can read only on appropriate team
		{user: teamObserverPlus, object: host, action: read, allow: false},
		{user: teamObserverPlus, object: host, action: selectiveRead, allow: false},
		{user: teamObserverPlus, object: host, action: write, allow: false},
		{user: teamObserverPlus, object: host, action: list, allow: true},
		{user: teamObserverPlus, object: host, action: selectiveList, allow: true},
		{user: teamObserverPlus, object: hostTeam1, action: read, allow: true},
		{user: teamObserverPlus, object: hostTeam1, action: selectiveRead, allow: true},
		{user: teamObserverPlus, object: hostTeam1, action: write, allow: false},
		{user: teamObserverPlus, object: hostTeam2, action: read, allow: false},
		{user: teamObserverPlus, object: hostTeam2, action: selectiveRead, allow: false},
		{user: teamObserverPlus, object: hostTeam2, action: write, allow: false},

		// Team maintainer can read/write only on appropriate team
		{user: teamMaintainer, object: host, action: read, allow: false},
		{user: teamMaintainer, object: host, action: selectiveRead, allow: false},
		{user: teamMaintainer, object: host, action: write, allow: false},
		{user: teamMaintainer, object: host, action: list, allow: true},
		{user: teamMaintainer, object: host, action: selectiveList, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: read, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: selectiveRead, allow: true},
		{user: teamMaintainer, object: hostTeam1, action: write, allow: true},
		{user: teamMaintainer, object: hostTeam2, action: read, allow: false},
		{user: teamMaintainer, object: hostTeam2, action: write, allow: false},

		// Team admin can read/write only on appropriate team
		{user: teamAdmin, object: host, action: read, allow: false},
		{user: teamAdmin, object: host, action: selectiveRead, allow: false},
		{user: teamAdmin, object: host, action: write, allow: false},
		{user: teamAdmin, object: host, action: list, allow: true},
		{user: teamAdmin, object: host, action: selectiveList, allow: true},
		{user: teamAdmin, object: hostTeam1, action: read, allow: true},
		{user: teamAdmin, object: hostTeam1, action: write, allow: true},
		{user: teamAdmin, object: hostTeam2, action: read, allow: false},
		{user: teamAdmin, object: hostTeam2, action: write, allow: false},

		// Team GitOps can cannot read hosts, but it can write and selectively read them.
		{user: teamGitOps, object: host, action: read, allow: false},
		{user: teamGitOps, object: host, action: write, allow: false},
		{user: teamGitOps, object: host, action: selectiveRead, allow: false},
		{user: teamGitOps, object: hostTeam1, action: read, allow: false},
		{user: teamGitOps, object: hostTeam1, action: list, allow: false},
		{user: teamGitOps, object: hostTeam1, action: selectiveList, allow: true},
		{user: teamGitOps, object: hostTeam1, action: selectiveRead, allow: true},
		{user: teamGitOps, object: hostTeam1, action: write, allow: false},
		{user: teamGitOps, object: hostTeam2, action: read, allow: false},
		{user: teamGitOps, object: hostTeam2, action: write, allow: false},
		{user: teamGitOps, object: hostTeam2, action: selectiveRead, allow: false},
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

	globalQuery := &fleet.Query{
		ObserverCanRun: false,
	}
	globalQueryNoTargets := &fleet.TargetedQuery{
		Query: globalQuery,
	}

	globalQueryTargetedToTeam1 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1}},
		Query:       globalQuery,
	}
	globalQueryTargetedToTeam1AndTeam2 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2}},
		Query:       globalQuery,
	}
	globalQueryTargetedToTeam2 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{2}},
		Query:       globalQuery,
	}
	globalQueryTargetedToTeam1AndTeam2AndTeam3 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2, 3}},
		Query:       globalQuery,
	}

	globalObserverQuery := &fleet.Query{
		ObserverCanRun: true,
	}
	globalObserverQueryEmptyTargets := &fleet.TargetedQuery{
		Query: globalObserverQuery,
	}
	globalObserverQueryTargetedToTeam1 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1}},
		Query:       globalObserverQuery,
	}
	globalObserverQueryTargetedToTeam1AndTeam2 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2}},
		Query:       globalObserverQuery,
	}
	globalObserverQueryTargetedToTeam2 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{2}},
		Query:       globalObserverQuery,
	}
	globalObserverQueryTargetedToTeam1AndTeam2AndTeam3 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1, 2, 3}},
		Query:       globalObserverQuery,
	}

	teamAdminQuery := &fleet.Query{
		ID:             1,
		AuthorID:       ptr.Uint(teamAdmin.ID),
		ObserverCanRun: false,
		TeamID:         ptr.Uint(1),
	}
	teamMaintQuery := &fleet.Query{
		ID:             2,
		AuthorID:       ptr.Uint(teamMaintainer.ID),
		ObserverCanRun: false,
		TeamID:         ptr.Uint(1),
	}
	globalAdminQuery := &fleet.Query{ID: 3, AuthorID: ptr.Uint(test.UserAdmin.ID), ObserverCanRun: false}
	globalGitOpsQuery := &fleet.Query{ID: 4, AuthorID: ptr.Uint(test.UserGitOps.ID), ObserverCanRun: false}
	teamGitOpsQuery := &fleet.Query{
		ID: 5, AuthorID: ptr.Uint(teamGitOps.ID),
		ObserverCanRun: false,
		TeamID:         ptr.Uint(1),
	}
	observerQueryOnTeam3 := &fleet.Query{
		ID:             6,
		ObserverCanRun: true,
		TeamID:         ptr.Uint(3),
	}
	observerQueryOnTeam3TargetedToTeam3 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{3}},
		Query:       observerQueryOnTeam3,
	}
	observerQueryOnTeam3TargetedToTeam2 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{2}},
		Query:       observerQueryOnTeam3,
	}
	observerQueryOnTeam3TargetedToTeam1 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1}},
		Query:       observerQueryOnTeam3,
	}
	observerQueryOnTeam1 := &fleet.Query{
		ID:             7,
		ObserverCanRun: true,
		TeamID:         ptr.Uint(1),
	}
	observerQueryOnTeam1TargetedToTeam1 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{1}},
		Query:       observerQueryOnTeam1,
	}
	observerQueryOnTeam1TargetedToTeam2 := &fleet.TargetedQuery{
		HostTargets: fleet.HostTargets{TeamIDs: []uint{2}},
		Query:       observerQueryOnTeam1,
	}

	runTestCasesGroups(t, []tcGroup{
		{
			name: "no access",
			testCases: []authTestCase{
				{user: nil, object: globalQuery, action: read, allow: false},
				{user: nil, object: globalQuery, action: write, allow: false},
				{user: nil, object: teamAdminQuery, action: write, allow: false},
				{user: nil, object: globalQueryNoTargets, action: run, allow: false},
				{user: nil, object: globalQueryTargetedToTeam1, action: run, allow: false},
				{user: nil, object: globalQuery, action: runNew, allow: false},
				{user: nil, object: globalObserverQuery, action: read, allow: false},
				{user: nil, object: globalObserverQuery, action: write, allow: false},
				{user: nil, object: globalObserverQueryEmptyTargets, action: run, allow: false},
				{user: nil, object: globalObserverQueryTargetedToTeam1, action: run, allow: false},
				{user: nil, object: globalObserverQuery, action: runNew, allow: false},
			},
		},
		{
			name: "User with no roles cannot access queries",
			testCases: []authTestCase{
				{user: test.UserNoRoles, object: globalQuery, action: read, allow: false},
				{user: test.UserNoRoles, object: globalQuery, action: write, allow: false},
				{user: test.UserNoRoles, object: teamAdminQuery, action: write, allow: false},
				{user: test.UserNoRoles, object: globalQueryNoTargets, action: run, allow: false},
				{user: test.UserNoRoles, object: globalQueryTargetedToTeam1, action: run, allow: false},
				{user: test.UserNoRoles, object: globalQuery, action: runNew, allow: false},
				{user: test.UserNoRoles, object: globalObserverQuery, action: read, allow: false},
				{user: test.UserNoRoles, object: globalObserverQuery, action: write, allow: false},
				{user: test.UserNoRoles, object: globalObserverQueryEmptyTargets, action: run, allow: false},
				{user: test.UserNoRoles, object: globalObserverQueryTargetedToTeam1, action: run, allow: false},
				{user: test.UserNoRoles, object: globalObserverQuery, action: runNew, allow: false},
			},
		},
		{
			name: "Global observer can read",
			testCases: []authTestCase{
				{user: test.UserObserver, object: globalQuery, action: read, allow: true},
				{user: test.UserObserver, object: globalQuery, action: write, allow: false},
				{user: test.UserObserver, object: teamAdminQuery, action: write, allow: false},
				{user: test.UserObserver, object: globalQueryNoTargets, action: run, allow: false},
				{user: test.UserObserver, object: globalQueryTargetedToTeam1, action: run, allow: false},
				{user: test.UserObserver, object: globalQuery, action: runNew, allow: false},
				{user: test.UserObserver, object: globalObserverQuery, action: read, allow: true},
				{user: test.UserObserver, object: globalObserverQuery, action: write, allow: false},
				{user: test.UserObserver, object: globalObserverQueryEmptyTargets, action: run, allow: true},            // can run observer query
				{user: test.UserObserver, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},         // can run observer query
				{user: test.UserObserver, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: true}, // can run observer query
				{user: test.UserObserver, object: globalObserverQuery, action: runNew, allow: false},

				{user: test.UserObserver, object: observerQueryOnTeam3, action: read, allow: true},
				{user: test.UserObserver, object: observerQueryOnTeam3, action: write, allow: false},
				{user: test.UserObserver, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: true},
				{user: test.UserObserver, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: true},
			},
		},
		{
			name: "Global observer+ can read all queries, not write them, and can run any query",
			testCases: []authTestCase{
				{user: test.UserObserverPlus, object: globalQuery, action: read, allow: true},
				{user: test.UserObserverPlus, object: globalQuery, action: write, allow: false},
				{user: test.UserObserverPlus, object: teamAdminQuery, action: write, allow: false},
				{user: test.UserObserverPlus, object: globalQueryNoTargets, action: run, allow: true},
				{user: test.UserObserverPlus, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: test.UserObserverPlus, object: globalQuery, action: runNew, allow: true},
				{user: test.UserObserverPlus, object: globalObserverQuery, action: read, allow: true},
				{user: test.UserObserverPlus, object: globalObserverQuery, action: write, allow: false},
				{user: test.UserObserverPlus, object: globalObserverQueryEmptyTargets, action: run, allow: true},            // can run observer query
				{user: test.UserObserverPlus, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},         // can run observer query
				{user: test.UserObserverPlus, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: true}, // can run observer query
				{user: test.UserObserverPlus, object: globalObserverQuery, action: runNew, allow: true},

				{user: test.UserObserverPlus, object: observerQueryOnTeam3, action: read, allow: true},
				{user: test.UserObserverPlus, object: observerQueryOnTeam3, action: write, allow: false},
				{user: test.UserObserverPlus, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: true},
				{user: test.UserObserverPlus, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: true},
			},
		},
		{
			name: "Global maintainer can read/write/run any query",
			testCases: []authTestCase{
				{user: test.UserMaintainer, object: globalQuery, action: read, allow: true},
				{user: test.UserMaintainer, object: globalQuery, action: write, allow: true},
				{user: test.UserMaintainer, object: teamMaintQuery, action: write, allow: true},
				{user: test.UserMaintainer, object: globalAdminQuery, action: write, allow: true},
				{user: test.UserMaintainer, object: globalQueryNoTargets, action: run, allow: true},
				{user: test.UserMaintainer, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: test.UserMaintainer, object: globalQuery, action: runNew, allow: true},
				{user: test.UserMaintainer, object: globalObserverQuery, action: read, allow: true},
				{user: test.UserMaintainer, object: globalObserverQuery, action: write, allow: true},
				{user: test.UserMaintainer, object: globalObserverQueryEmptyTargets, action: run, allow: true},
				{user: test.UserMaintainer, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},
				{user: test.UserMaintainer, object: globalObserverQuery, action: runNew, allow: true},

				{user: test.UserMaintainer, object: observerQueryOnTeam3, action: read, allow: true},
				{user: test.UserMaintainer, object: observerQueryOnTeam3, action: write, allow: true},
				{user: test.UserMaintainer, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: true},
				{user: test.UserMaintainer, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: true},
			},
		},
		{
			name: "Global admin can read/write/run any query (on its team)",
			testCases: []authTestCase{
				{user: test.UserAdmin, object: globalQuery, action: read, allow: true},
				{user: test.UserAdmin, object: globalQuery, action: write, allow: true},
				{user: test.UserAdmin, object: teamMaintQuery, action: write, allow: true},
				{user: test.UserAdmin, object: globalAdminQuery, action: write, allow: true},
				{user: test.UserAdmin, object: globalQueryNoTargets, action: run, allow: true},
				{user: test.UserAdmin, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: test.UserAdmin, object: globalQuery, action: runNew, allow: true},
				{user: test.UserAdmin, object: globalObserverQuery, action: read, allow: true},
				{user: test.UserAdmin, object: globalObserverQuery, action: write, allow: true},
				{user: test.UserAdmin, object: globalObserverQueryEmptyTargets, action: run, allow: true},
				{user: test.UserAdmin, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},
				{user: test.UserAdmin, object: globalObserverQuery, action: runNew, allow: true},

				{user: test.UserAdmin, object: observerQueryOnTeam3, action: read, allow: true},
				{user: test.UserAdmin, object: observerQueryOnTeam3, action: write, allow: true},
				{user: test.UserAdmin, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: true},
				{user: test.UserAdmin, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: true},
			},
		},
		{
			name: "Global GitOps cannot run any query, but can read or write",
			testCases: []authTestCase{
				{user: test.UserGitOps, object: globalQuery, action: read, allow: true},
				{user: test.UserGitOps, object: globalQuery, action: write, allow: true},
				{user: test.UserGitOps, object: teamAdminQuery, action: write, allow: true},
				{user: test.UserGitOps, object: globalQueryNoTargets, action: run, allow: false},
				{user: test.UserGitOps, object: globalQueryTargetedToTeam1, action: run, allow: false},
				{user: test.UserGitOps, object: globalQuery, action: runNew, allow: false},
				{user: test.UserGitOps, object: globalObserverQuery, action: read, allow: true},
				{user: test.UserGitOps, object: globalObserverQuery, action: write, allow: true},
				{user: test.UserGitOps, object: globalObserverQueryEmptyTargets, action: run, allow: false},
				{user: test.UserGitOps, object: globalObserverQueryTargetedToTeam1, action: run, allow: false},
				{user: test.UserGitOps, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: false},
				{user: test.UserGitOps, object: globalObserverQuery, action: runNew, allow: false},
			},
		},
		{
			name: "Team observer can read and run observer_can_run only",
			testCases: []authTestCase{
				{user: teamObserver, object: globalQuery, action: read, allow: true},
				{user: teamObserver, object: globalQuery, action: write, allow: false},
				{user: teamObserver, object: teamAdminQuery, action: write, allow: false},
				{user: teamObserver, object: globalQueryNoTargets, action: run, allow: false},
				{user: teamObserver, object: globalQueryTargetedToTeam1, action: run, allow: false},
				{user: teamObserver, object: globalQuery, action: runNew, allow: false},
				{user: teamObserver, object: globalObserverQuery, action: read, allow: true},
				{user: teamObserver, object: globalObserverQuery, action: write, allow: false},
				{user: teamObserver, object: globalObserverQueryEmptyTargets, action: run, allow: true},             // can run observer query with no targeted team
				{user: teamObserver, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},          // can run observer query filtered to observed team
				{user: teamObserver, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: false}, // not filtered only to observed teams
				{user: teamObserver, object: globalObserverQueryTargetedToTeam2, action: run, allow: false},         // not filtered only to observed teams
				{user: teamObserver, object: globalObserverQuery, action: runNew, allow: false},

				{user: teamObserver, object: observerQueryOnTeam3, action: read, allow: false},
				{user: teamObserver, object: observerQueryOnTeam3, action: write, allow: false},
				{user: teamObserver, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: false},
				{user: teamObserver, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: false},
				{user: teamObserver, object: observerQueryOnTeam3TargetedToTeam1, action: run, allow: false},
				{user: teamObserver, object: observerQueryOnTeam1TargetedToTeam1, action: run, allow: true},
			},
		},
		{
			name: "Team observer+ can read all queries, not write them, and can run any query",
			testCases: []authTestCase{
				{user: teamObserverPlus, object: globalQuery, action: read, allow: true},
				{user: teamObserverPlus, object: globalQuery, action: write, allow: false},
				{user: teamObserverPlus, object: teamAdminQuery, action: write, allow: false},
				{user: teamObserverPlus, object: globalQueryNoTargets, action: run, allow: true},
				{user: teamObserverPlus, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: teamObserverPlus, object: globalQuery, action: runNew, allow: true},
				{user: teamObserverPlus, object: globalObserverQuery, action: read, allow: true},
				{user: teamObserverPlus, object: globalObserverQuery, action: write, allow: false},
				{user: teamObserverPlus, object: globalObserverQueryEmptyTargets, action: run, allow: true},             // can run observer query with no targeted team
				{user: teamObserverPlus, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},          // can run observer query filtered to observed team
				{user: teamObserverPlus, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: false}, // not filtered only to observed teams
				{user: teamObserverPlus, object: globalObserverQueryTargetedToTeam2, action: run, allow: false},         // not filtered only to observed teams
				{user: teamObserverPlus, object: globalObserverQuery, action: runNew, allow: true},

				{user: teamObserverPlus, object: observerQueryOnTeam3, action: read, allow: false},
				{user: teamObserverPlus, object: observerQueryOnTeam3, action: write, allow: false},
				{user: teamObserverPlus, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: false},
				{user: teamObserverPlus, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: false},
				{user: teamObserverPlus, object: observerQueryOnTeam3TargetedToTeam1, action: run, allow: false},
				{user: teamObserverPlus, object: observerQueryOnTeam1TargetedToTeam1, action: run, allow: true},
			},
		},
		{
			name: "Team maintainer can read/write/run queries filtered on their team(s)",
			testCases: []authTestCase{
				{user: teamMaintainer, object: globalQuery, action: read, allow: true},
				{user: teamMaintainer, object: globalQuery, action: write, allow: false}, // query belongs to global domain.
				{user: teamMaintainer, object: teamMaintQuery, action: write, allow: true},
				{user: teamMaintainer, object: teamAdminQuery, action: write, allow: true},
				{user: teamMaintainer, object: globalQueryNoTargets, action: run, allow: true},
				{user: teamMaintainer, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: teamMaintainer, object: globalQueryTargetedToTeam1AndTeam2, action: run, allow: false},
				{user: teamMaintainer, object: globalQueryTargetedToTeam2, action: run, allow: false},
				{user: teamMaintainer, object: globalQuery, action: runNew, allow: true},
				{user: teamMaintainer, object: globalObserverQuery, action: read, allow: true},
				{user: teamMaintainer, object: globalObserverQuery, action: write, allow: false}, // query belongs to global domain.
				{user: teamMaintainer, object: globalObserverQueryEmptyTargets, action: run, allow: true},
				{user: teamMaintainer, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},
				{user: teamMaintainer, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: false},
				{user: teamMaintainer, object: globalObserverQueryTargetedToTeam2, action: run, allow: false},
				{user: teamMaintainer, object: globalObserverQuery, action: runNew, allow: true},

				{user: teamMaintainer, object: observerQueryOnTeam3, action: read, allow: false},
				{user: teamMaintainer, object: observerQueryOnTeam3, action: write, allow: false},
				{user: teamMaintainer, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: false},
				{user: teamMaintainer, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: false},
				{user: teamMaintainer, object: observerQueryOnTeam3TargetedToTeam1, action: run, allow: false},
				{user: teamMaintainer, object: observerQueryOnTeam1TargetedToTeam1, action: run, allow: true},
			},
		},
		{
			name: "Team admin can read/write their own queries/run queries filtered on their team(s)",
			testCases: []authTestCase{
				{user: teamAdmin, object: globalQuery, action: read, allow: true},
				{user: teamAdmin, object: globalQuery, action: write, allow: false}, // query belongs to global domain.
				{user: teamAdmin, object: teamAdminQuery, action: write, allow: true},
				{user: teamAdmin, object: teamMaintQuery, action: write, allow: true},
				{user: teamAdmin, object: globalAdminQuery, action: write, allow: false}, // query belongs to global domain.
				{user: teamAdmin, object: globalQueryNoTargets, action: run, allow: true},
				{user: teamAdmin, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: teamAdmin, object: globalQueryTargetedToTeam1AndTeam2, action: run, allow: false},
				{user: teamAdmin, object: globalQueryTargetedToTeam2, action: run, allow: false},
				{user: teamAdmin, object: globalQuery, action: runNew, allow: true},
				{user: teamAdmin, object: globalObserverQuery, action: read, allow: true},
				{user: teamAdmin, object: globalObserverQuery, action: write, allow: false}, // observerQuery belongs to global domain.
				{user: teamAdmin, object: globalObserverQueryEmptyTargets, action: run, allow: true},
				{user: teamAdmin, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},
				{user: teamAdmin, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: false},
				{user: teamAdmin, object: globalObserverQueryTargetedToTeam2, action: run, allow: false},
				{user: teamAdmin, object: globalObserverQuery, action: runNew, allow: true},

				{user: teamAdmin, object: observerQueryOnTeam3, action: read, allow: false},
				{user: teamAdmin, object: observerQueryOnTeam3, action: write, allow: false},
				{user: teamAdmin, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: false},
				{user: teamAdmin, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: false},
				{user: teamAdmin, object: observerQueryOnTeam3TargetedToTeam1, action: run, allow: false},
				{user: teamAdmin, object: observerQueryOnTeam1TargetedToTeam1, action: run, allow: true},
			},
		},
		{
			name: "Team GitOps cannot read or run any query, but can create new or edit (write) queries authored by it.",
			testCases: []authTestCase{
				{user: teamGitOps, object: globalQuery, action: read, allow: false},
				{user: teamGitOps, object: globalQuery, action: write, allow: false}, // cannot create a global query
				{user: teamGitOps, object: teamAdminQuery, action: write, allow: true},
				{user: teamGitOps, object: teamGitOpsQuery, action: write, allow: true},
				{user: teamGitOps, object: globalGitOpsQuery, action: write, allow: false}, // cannot write a global query
				{user: teamGitOps, object: globalQueryNoTargets, action: run, allow: false},
				{user: teamGitOps, object: globalQueryTargetedToTeam1, action: run, allow: false},
				{user: teamGitOps, object: globalQuery, action: runNew, allow: false},
				{user: teamGitOps, object: globalObserverQueryEmptyTargets, action: run, allow: false},
				{user: teamGitOps, object: globalObserverQueryTargetedToTeam1, action: run, allow: false},
				{user: teamGitOps, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: false},
				{user: teamGitOps, object: globalObserverQueryTargetedToTeam2, action: run, allow: false},
				{user: teamGitOps, object: globalObserverQuery, action: runNew, allow: false},

				{user: teamGitOps, object: observerQueryOnTeam3, action: read, allow: false},
				{user: teamGitOps, object: observerQueryOnTeam3, action: write, allow: false},
				{user: teamGitOps, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: false},
				{user: teamGitOps, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: false},
				{user: teamGitOps, object: observerQueryOnTeam3TargetedToTeam1, action: run, allow: false},
				{user: teamGitOps, object: observerQueryOnTeam1TargetedToTeam1, action: run, allow: false},
			},
		},
		{
			name: "User admin on team 1, observer on team 2",
			testCases: []authTestCase{
				{user: twoTeamsAdminObs, object: globalQuery, action: read, allow: true},
				{user: twoTeamsAdminObs, object: globalQuery, action: write, allow: false}, // cannot write a global query
				{user: twoTeamsAdminObs, object: teamAdminQuery, action: write, allow: true},
				{user: twoTeamsAdminObs, object: teamMaintQuery, action: write, allow: true},
				{user: twoTeamsAdminObs, object: globalAdminQuery, action: write, allow: false}, // cannot write a global query
				{user: twoTeamsAdminObs, object: globalQueryNoTargets, action: run, allow: true},
				{user: twoTeamsAdminObs, object: globalQueryTargetedToTeam1, action: run, allow: true},
				{user: twoTeamsAdminObs, object: globalQueryTargetedToTeam1AndTeam2, action: run, allow: false}, // user is only observer on team 2
				{user: twoTeamsAdminObs, object: globalQueryTargetedToTeam2, action: run, allow: false},
				{user: twoTeamsAdminObs, object: globalQueryTargetedToTeam1AndTeam2AndTeam3, action: run, allow: false},
				{user: twoTeamsAdminObs, object: globalQuery, action: runNew, allow: true},
				{user: twoTeamsAdminObs, object: globalObserverQuery, action: read, allow: true},
				{user: twoTeamsAdminObs, object: globalObserverQuery, action: write, allow: false}, // cannot write a global query
				{user: twoTeamsAdminObs, object: globalObserverQueryEmptyTargets, action: run, allow: true},
				{user: twoTeamsAdminObs, object: globalObserverQueryTargetedToTeam1, action: run, allow: true},
				{user: twoTeamsAdminObs, object: globalObserverQueryTargetedToTeam1AndTeam2, action: run, allow: true}, // user is at least observer on both teams
				{user: twoTeamsAdminObs, object: globalObserverQueryTargetedToTeam2, action: run, allow: true},
				{user: twoTeamsAdminObs, object: globalObserverQueryTargetedToTeam1AndTeam2AndTeam3, action: run, allow: false}, // not member of team 3
				{user: twoTeamsAdminObs, object: globalObserverQuery, action: runNew, allow: true},

				{user: twoTeamsAdminObs, object: observerQueryOnTeam3, action: read, allow: false},
				{user: twoTeamsAdminObs, object: observerQueryOnTeam3, action: write, allow: false},
				{user: twoTeamsAdminObs, object: observerQueryOnTeam3TargetedToTeam3, action: run, allow: false},
				{user: twoTeamsAdminObs, object: observerQueryOnTeam3TargetedToTeam2, action: run, allow: false},
				{user: twoTeamsAdminObs, object: observerQueryOnTeam3TargetedToTeam1, action: run, allow: false},
				{user: twoTeamsAdminObs, object: observerQueryOnTeam1TargetedToTeam1, action: run, allow: true},
				{user: twoTeamsAdminObs, object: observerQueryOnTeam1TargetedToTeam2, action: run, allow: true},
			},
		},
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
		{user: test.UserGitOps, object: globalPolicy, action: read, allow: true},

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
		{user: test.UserGitOps, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamAdminTeam1, object: team1Policy, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam1, object: team1Policy, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamObserverTeam1, object: team1Policy, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamObserverPlusTeam1, object: team1Policy, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Policy, action: read, allow: true},

		{user: test.UserTeamGitOpsTeam1, object: team1Policy, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Policy, action: read, allow: true},

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

func TestAuthorizeMDMConfigProfile(t *testing.T) {
	t.Parallel()

	globalProfile := &fleet.MDMConfigProfileAuthz{}
	team1Profile := &fleet.MDMConfigProfileAuthz{
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
		{user: test.UserGitOps, object: globalProfile, action: read, allow: true},
		{user: test.UserGitOps, object: team1Profile, action: write, allow: true},
		{user: test.UserGitOps, object: team1Profile, action: read, allow: true},

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
		{user: test.UserTeamGitOpsTeam1, object: team1Profile, action: read, allow: true},

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

type tcGroup struct {
	name      string
	testCases []authTestCase
}

func runTestCases(t *testing.T, testCases []authTestCase) {
	runTestCasesGroups(t, []tcGroup{
		{
			name:      "all",
			testCases: testCases,
		},
	})
}

func runTestCasesGroups(t *testing.T, testCaseGroups []tcGroup) {
	t.Helper()

	for _, gg := range testCaseGroups {
		gg := gg
		t.Run(gg.name, func(t *testing.T) {
			for _, tt := range gg.testCases {
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
		})
	}
}

func TestAuthorizeMDMCommand(t *testing.T) {
	t.Parallel()

	globalCommand := &fleet.MDMCommandAuthz{}
	team1Command := &fleet.MDMCommandAuthz{
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

		{user: test.UserGitOps, object: globalCommand, action: write, allow: true},
		{user: test.UserGitOps, object: globalCommand, action: read, allow: false},
		{user: test.UserGitOps, object: team1Command, action: write, allow: true},
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
		{user: test.UserTeamGitOpsTeam1, object: team1Command, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Command, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalCommand, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalCommand, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Command, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Command, action: read, allow: false},
	})
}

func TestAuthorizeHostScriptResult(t *testing.T) {
	t.Parallel()

	globalScript := &fleet.HostScriptResult{}
	globalSavedScript := &fleet.HostScriptResult{ScriptID: ptr.Uint(1)}
	team1Script := &fleet.HostScriptResult{TeamID: ptr.Uint(1)}
	team1SavedScript := &fleet.HostScriptResult{TeamID: ptr.Uint(1), ScriptID: ptr.Uint(1)}

	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalScript, action: write, allow: false},
		{user: test.UserNoRoles, object: globalScript, action: read, allow: false},
		{user: test.UserNoRoles, object: globalSavedScript, action: write, allow: false},
		{user: test.UserNoRoles, object: globalSavedScript, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Script, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Script, action: read, allow: false},
		{user: test.UserNoRoles, object: team1SavedScript, action: write, allow: false},
		{user: test.UserNoRoles, object: team1SavedScript, action: read, allow: false},

		{user: test.UserAdmin, object: globalScript, action: write, allow: true},
		{user: test.UserAdmin, object: globalScript, action: read, allow: true},
		{user: test.UserAdmin, object: globalSavedScript, action: write, allow: true},
		{user: test.UserAdmin, object: globalSavedScript, action: read, allow: true},
		{user: test.UserAdmin, object: team1Script, action: write, allow: true},
		{user: test.UserAdmin, object: team1Script, action: read, allow: true},
		{user: test.UserAdmin, object: team1SavedScript, action: write, allow: true},
		{user: test.UserAdmin, object: team1SavedScript, action: read, allow: true},

		{user: test.UserMaintainer, object: globalScript, action: write, allow: true},
		{user: test.UserMaintainer, object: globalScript, action: read, allow: true},
		{user: test.UserMaintainer, object: globalSavedScript, action: write, allow: true},
		{user: test.UserMaintainer, object: globalSavedScript, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Script, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Script, action: read, allow: true},
		{user: test.UserMaintainer, object: team1SavedScript, action: write, allow: true},
		{user: test.UserMaintainer, object: team1SavedScript, action: read, allow: true},

		{user: test.UserObserver, object: globalScript, action: write, allow: false},
		{user: test.UserObserver, object: globalScript, action: read, allow: true},
		{user: test.UserObserver, object: globalSavedScript, action: write, allow: false},
		{user: test.UserObserver, object: globalSavedScript, action: read, allow: true},
		{user: test.UserObserver, object: team1Script, action: write, allow: false},
		{user: test.UserObserver, object: team1Script, action: read, allow: true},
		{user: test.UserObserver, object: team1SavedScript, action: write, allow: false},
		{user: test.UserObserver, object: team1SavedScript, action: read, allow: true},

		{user: test.UserObserverPlus, object: globalScript, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalScript, action: read, allow: true},
		{user: test.UserObserverPlus, object: globalSavedScript, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalSavedScript, action: read, allow: true},
		{user: test.UserObserverPlus, object: team1Script, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Script, action: read, allow: true},
		{user: test.UserObserverPlus, object: team1SavedScript, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1SavedScript, action: read, allow: true},

		{user: test.UserGitOps, object: globalScript, action: write, allow: false},
		{user: test.UserGitOps, object: globalScript, action: read, allow: false},
		{user: test.UserGitOps, object: globalSavedScript, action: write, allow: false},
		{user: test.UserGitOps, object: globalSavedScript, action: read, allow: false},
		{user: test.UserGitOps, object: team1Script, action: write, allow: false},
		{user: test.UserGitOps, object: team1Script, action: read, allow: false},
		{user: test.UserGitOps, object: team1SavedScript, action: write, allow: false},
		{user: test.UserGitOps, object: team1SavedScript, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Script, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Script, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1SavedScript, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1SavedScript, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Script, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1SavedScript, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Script, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Script, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1SavedScript, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1SavedScript, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Script, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1SavedScript, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Script, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1SavedScript, action: read, allow: true},

		{user: test.UserTeamObserverTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Script, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1SavedScript, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Script, action: read, allow: true},
		{user: test.UserTeamObserverPlusTeam1, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1SavedScript, action: read, allow: true},

		{user: test.UserTeamObserverPlusTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Script, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1SavedScript, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Script, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Script, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1SavedScript, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalSavedScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalSavedScript, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Script, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1SavedScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1SavedScript, action: read, allow: false},
	})
}

func TestAuthorizeScript(t *testing.T) {
	t.Parallel()

	globalScript := &fleet.Script{}
	team1Script := &fleet.Script{
		TeamID: ptr.Uint(1),
	}
	runTestCases(t, []authTestCase{
		{user: test.UserNoRoles, object: globalScript, action: write, allow: false},
		{user: test.UserNoRoles, object: globalScript, action: read, allow: false},
		{user: test.UserNoRoles, object: team1Script, action: write, allow: false},
		{user: test.UserNoRoles, object: team1Script, action: read, allow: false},

		{user: test.UserAdmin, object: globalScript, action: write, allow: true},
		{user: test.UserAdmin, object: globalScript, action: read, allow: true},
		{user: test.UserAdmin, object: team1Script, action: write, allow: true},
		{user: test.UserAdmin, object: team1Script, action: read, allow: true},

		{user: test.UserMaintainer, object: globalScript, action: write, allow: true},
		{user: test.UserMaintainer, object: globalScript, action: read, allow: true},
		{user: test.UserMaintainer, object: team1Script, action: write, allow: true},
		{user: test.UserMaintainer, object: team1Script, action: read, allow: true},

		{user: test.UserObserver, object: globalScript, action: write, allow: false},
		{user: test.UserObserver, object: globalScript, action: read, allow: true},
		{user: test.UserObserver, object: team1Script, action: write, allow: false},
		{user: test.UserObserver, object: team1Script, action: read, allow: true},

		{user: test.UserObserverPlus, object: globalScript, action: write, allow: false},
		{user: test.UserObserverPlus, object: globalScript, action: read, allow: true},
		{user: test.UserObserverPlus, object: team1Script, action: write, allow: false},
		{user: test.UserObserverPlus, object: team1Script, action: read, allow: true},

		{user: test.UserGitOps, object: globalScript, action: write, allow: true},
		{user: test.UserGitOps, object: globalScript, action: read, allow: false},
		{user: test.UserGitOps, object: team1Script, action: write, allow: true},
		{user: test.UserGitOps, object: team1Script, action: read, allow: false},

		{user: test.UserTeamAdminTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamAdminTeam1, object: team1Script, action: write, allow: true},
		{user: test.UserTeamAdminTeam1, object: team1Script, action: read, allow: true},

		{user: test.UserTeamAdminTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamAdminTeam2, object: team1Script, action: read, allow: false},

		{user: test.UserTeamMaintainerTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam1, object: team1Script, action: write, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: team1Script, action: read, allow: true},

		{user: test.UserTeamMaintainerTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamMaintainerTeam2, object: team1Script, action: read, allow: false},

		{user: test.UserTeamObserverTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverTeam1, object: team1Script, action: read, allow: true},

		{user: test.UserTeamObserverTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverTeam2, object: team1Script, action: read, allow: false},

		{user: test.UserTeamObserverPlusTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam1, object: team1Script, action: read, allow: true},

		{user: test.UserTeamObserverPlusTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamObserverPlusTeam2, object: team1Script, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam1, object: globalScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: globalScript, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: team1Script, action: write, allow: true},
		{user: test.UserTeamGitOpsTeam1, object: team1Script, action: read, allow: false},

		{user: test.UserTeamGitOpsTeam2, object: globalScript, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: globalScript, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Script, action: write, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: team1Script, action: read, allow: false},
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

func TestHostHealth(t *testing.T) {
	t.Parallel()

	hostHealth := &fleet.HostHealth{TeamID: ptr.Uint(1)}
	runTestCases(t, []authTestCase{
		{user: nil, object: hostHealth, action: read, allow: false},
		{user: test.UserGitOps, object: hostHealth, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam1, object: hostHealth, action: read, allow: false},
		{user: test.UserTeamGitOpsTeam2, object: hostHealth, action: read, allow: false},
		{user: test.UserAdmin, object: hostHealth, action: read, allow: true},
		{user: test.UserTeamAdminTeam1, object: hostHealth, action: read, allow: true},
		{user: test.UserTeamAdminTeam2, object: hostHealth, action: read, allow: false},
		{user: test.UserObserver, object: hostHealth, action: read, allow: true},
		{user: test.UserTeamObserverTeam1, object: hostHealth, action: read, allow: true},
		{user: test.UserTeamObserverTeam2, object: hostHealth, action: read, allow: false},
		{user: test.UserMaintainer, object: hostHealth, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam1, object: hostHealth, action: read, allow: true},
		{user: test.UserTeamMaintainerTeam2, object: hostHealth, action: read, allow: false},
	})
}
