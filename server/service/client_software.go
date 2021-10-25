package service

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListSoftware retrieves the software running across hosts.
func (c *Client) ListSoftware(teamID *uint) ([]fleet.Software, error) {
	verb, path := "GET", "/api/v1/fleet/software"
	query := ""
	if teamID != nil {
		query = fmt.Sprintf("team_id=%d", *teamID)
	}
	var responseBody listSoftwareResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Software, nil
}
