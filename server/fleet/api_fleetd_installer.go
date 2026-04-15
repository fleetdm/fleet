package fleet

// GetFleetdInstallerRequest is the request payload for the get fleetd installer endpoint.
type GetFleetdInstallerRequest struct {
	TeamID uint `url:"team_id"`
}
