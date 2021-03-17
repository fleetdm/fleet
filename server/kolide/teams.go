package kolide

import "time"

type TeamStore interface {
	// NewTeam creates a new Team object in the store.
	NewTeam(team *Team) (*Team, error)
	// Team retrieves the Team by ID
	Team(tid uint) (*Team, error)
	// TeamByName retrieves the Team by Name
	TeamByName(name string) (*Team, error)
}

// Team is the data representation for the "Team" concept (group of hosts and
// group of users that can perform operations on those hosts).
type Team struct {
	// Directly in DB

	// ID is the database ID.
	ID        uint      `json:"id" db:"id"`
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
