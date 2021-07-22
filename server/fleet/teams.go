package fleet

import (
	"context"
	"encoding/json"
	"time"
)

const (
	RoleAdmin      = "admin"
	RoleMaintainer = "maintainer"
	RoleObserver   = "observer"
)

type TeamStore interface {
	// NewTeam creates a new Team object in the store.
	NewTeam(team *Team) (*Team, error)
	// SaveTeam saves any changes to the team.
	SaveTeam(team *Team) (*Team, error)
	// Team retrieves the Team by ID.
	Team(tid uint) (*Team, error)
	// Team deletes the Team by ID.
	DeleteTeam(tid uint) error
	// TeamByName retrieves the Team by Name.
	TeamByName(name string) (*Team, error)
	// ListTeams lists teams with the ordering and filters in the provided
	// options.
	ListTeams(filter TeamFilter, opt ListOptions) ([]*Team, error)
	// SearchTeams searches teams using the provided query and ommitting the
	// provided existing selection.
	SearchTeams(filter TeamFilter, matchQuery string, omit ...uint) ([]*Team, error)
	// TeamEnrollSecrets lists the enroll secrets for the team.
	TeamEnrollSecrets(teamID uint) ([]*EnrollSecret, error)
}

type TeamService interface {
	// NewTeam creates a new team.
	NewTeam(ctx context.Context, p TeamPayload) (*Team, error)
	// ModifyTeam modifies an existing team (besides agent options).
	ModifyTeam(ctx context.Context, id uint, payload TeamPayload) (*Team, error)
	// ModifyTeam modifies agent options for a team.
	ModifyTeamAgentOptions(ctx context.Context, id uint, options json.RawMessage) (*Team, error)
	// AddTeamUsers adds users to an existing team.
	AddTeamUsers(ctx context.Context, teamID uint, users []TeamUser) (*Team, error)
	// DeleteTeamUsers deletes users from an existing team.
	DeleteTeamUsers(ctx context.Context, teamID uint, users []TeamUser) (*Team, error)
	// DeleteTeam deletes an existing team.
	DeleteTeam(ctx context.Context, id uint) error
	// ListTeams lists teams with the ordering and filters in the provided
	// options.
	ListTeams(ctx context.Context, opt ListOptions) ([]*Team, error)
	// ListTeams lists users on the team with the provided list options.
	ListTeamUsers(ctx context.Context, teamID uint, opt ListOptions) ([]*User, error)
	// TeamEnrollSecrets lists the enroll secrets for the team.
	TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*EnrollSecret, error)
	// ApplyTeamSpecs applies the changes for each team as defined in the specs.
	ApplyTeamSpecs(ctx context.Context, specs []*TeamSpec) error
}

type TeamPayload struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Secrets     []*EnrollSecret `json:"secrets"`
	// Note AgentOptions must be set by a separate endpoint.
}

// Team is the data representation for the "Team" concept (group of hosts and
// group of users that can perform operations on those hosts).
type Team struct {
	// Directly in DB

	// ID is the database ID.
	ID uint `json:"id" db:"id"`
	// CreatedAt is the timestamp of the label creation.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// Name is the human friendly name of the team.
	Name string `json:"name" db:"name"`
	// Description is an optional description for the team.
	Description string `json:"description" db:"description"`
	// AgentOptions is the options for osquery and Orbit.
	AgentOptions *json.RawMessage `json:"agent_options" db:"agent_options"`

	// Derived from JOINs

	// UserCount is the count of users with explicit roles on this team.
	UserCount int `json:"user_count" db:"user_count"`
	// Users is the users that have a role on this team.
	Users []TeamUser `json:"users,omitempty"`
	// UserCount is the count of hosts assigned to this team.
	HostCount int `json:"host_count" db:"host_count"`
	// Hosts are the hosts assigned to the team.
	Hosts []Host `json:"hosts,omitempty"`
	// Secrets is the enroll secrets valid for this team.
	Secrets []*EnrollSecret `json:"secrets,omitempty"`
}

func (t Team) AuthzType() string {
	return "team"
}

// TeamUser is a user mapped to a team with a role.
type TeamUser struct {
	// User is the user object. At least ID must be specified for most uses.
	User
	// Role is the role the user has for the team.
	Role string `json:"role" db:"role"`
}

var teamRoles = map[string]bool{
	RoleObserver:   true,
	RoleMaintainer: true,
}

// ValidTeamRole returns whether the role provided is valid for a team user.
func ValidTeamRole(role string) bool {
	return teamRoles[role]
}

// ValidTeamRoles returns the list of valid roles for a team user.
func ValidTeamRoles() []string {
	var roles []string
	for role := range teamRoles {
		roles = append(roles, role)
	}
	return roles
}

var globalRoles = map[string]bool{
	RoleObserver:   true,
	RoleMaintainer: true,
	RoleAdmin:      true,
}

// ValidGlobalRole returns whether the role provided is valid for a global user.
func ValidGlobalRole(role string) bool {
	return globalRoles[role]
}

// ValidGlobalRoles returns the list of valid roles for a global user.
func ValidGlobalRoles() []string {
	var roles []string
	for role := range globalRoles {
		roles = append(roles, role)
	}
	return roles
}

// ValidateRole returns nil if the global and team roles combination is a valid
// one within fleet, or a fleet Error otherwise.
func ValidateRole(globalRole *string, teamUsers []UserTeam) error {
	if globalRole == nil || *globalRole == "" {
		if len(teamUsers) == 0 {
			return NewError(ErrNoRoleNeeded, "either global role or team role needs to be defined")
		}
		for _, t := range teamUsers {
			if !ValidTeamRole(t.Role) {
				return NewError(ErrNoRoleNeeded, "Team roles can be observer or maintainer")
			}
		}
		return nil
	}

	if len(teamUsers) > 0 {
		return NewError(ErrNoRoleNeeded, "Cannot specify both Global Role and Team Roles")
	}

	if !ValidGlobalRole(*globalRole) {
		return NewError(ErrNoRoleNeeded, "GlobalRole role can only be admin, observer, or maintainer.")
	}

	return nil
}

// TeamFilter is the filtering information passed to the datastore for queries
// that may be filtered by team.
type TeamFilter struct {
	// User is the user to filter by.
	User *User
	// IncludeObserver determines whether to include teams the user is an observer on.
	IncludeObserver bool
}

const (
	TeamKind = "team"
)

type TeamSpec struct {
	Name         string           `json:"name"`
	AgentOptions *json.RawMessage `json:"agent_options"`
	Secrets      []EnrollSecret   `json:"secrets"`
}
