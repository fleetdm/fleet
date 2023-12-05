package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListSoftware retrieves the software running across hosts.
func (c *Client) ListSoftware(query string) ([]fleet.Software, error) {
	verb, path := "GET", "/api/latest/fleet/software/versions"
	var responseBody listSoftwareResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Software, nil
}
