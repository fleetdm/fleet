package fleet

import (
	"encoding/json"
	"time"
)

const (
	RoleAdmin      = "admin"
	RoleMaintainer = "maintainer"
	RoleObserver   = "observer"
)

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
	RoleAdmin:      true,
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
	// TeamID is the specific team id to filter by. If other criteria are
	// specified, they must met too (e.g. if a User is provided, that team ID
	// must be part of their teams).
	TeamID *uint
}

const (
	TeamKind = "team"
)

type TeamSpec struct {
	Name         string           `json:"name"`
	AgentOptions *json.RawMessage `json:"agent_options"`
	Secrets      []EnrollSecret   `json:"secrets"`
}
