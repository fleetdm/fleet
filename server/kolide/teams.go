package kolide

import (
	"context"
	"time"
)

type TeamStore interface {
	// NewTeam creates a new Team object in the store.
	NewTeam(team *Team) (*Team, error)
	// Team retrieves the Team by ID.
	Team(tid uint) (*Team, error)
	// Team deletes the Team by ID.
	DeleteTeam(tid uint) error
	// TeamByName retrieves the Team by Name.
	TeamByName(name string) (*Team, error)
	// SaveTeam saves any changes to the team.
	SaveTeam(team *Team) (*Team, error)
	// ListTeams lists teams with the ordering and filters in the provided
	// options.
	ListTeams(opt ListOptions) ([]*Team, error)
}

type TeamService interface {
	// NewTeam creates a new team.
	NewTeam(ctx context.Context, p TeamPayload) (*Team, error)
	// ModifyTeam modifies an existing team.
	ModifyTeam(ctx context.Context, id uint, payload TeamPayload) (*Team, error)
	// DeleteTeam deletes an existing team.
	DeleteTeam(ctx context.Context, id uint) error
	// ListTeams lists teams with the ordering and filters in the provided
	// options.
	ListTeams(ctx context.Context, opt ListOptions) ([]*Team, error)
}

type TeamPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
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

	// Derived from JOINs

	// Users is the users that have a role on this team.
	Users []TeamUser `json:"users,omitempty"`
	// Hosts are the hosts assigned to the team.
	Hosts []Host `json:"hosts,omitempty"`
}

type TeamUser struct {
	// User is the user object
	User
	// Role is the role the user has for the team.
	Role string `json:"role" db:"role"`
}
