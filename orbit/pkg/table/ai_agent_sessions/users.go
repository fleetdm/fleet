package ai_agent_sessions

import "github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"

// userHome is a local account whose home directory may hold agent history.
type userHome struct {
	Username string
	HomeDir  string
}

// discoverUsers lists the accounts whose home directories should be searched.
// fleetd runs as root, so every user's home has to be enumerated; the shared
// homes package does that, naming each one after its real owner.
func discoverUsers() []userHome {
	all := homes.All()
	users := make([]userHome, 0, len(all))
	for _, h := range all {
		users = append(users, userHome{Username: h.Username, HomeDir: h.Dir})
	}
	return users
}
