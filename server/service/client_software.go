package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListSoftwareVersions retrieves the software versions installed on hosts.
func (c *Client) ListSoftwareVersions(query string) ([]fleet.Software, error) {
	verb, path := "GET", "/api/latest/fleet/software" // TODO(mna): /versions
	// TODO(mna): adjust if the response struct has changed, when this gets merged: https://github.com/fleetdm/fleet/issues/15229
	var responseBody listSoftwareResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Software, nil
}

// ListSoftwareTitles retrieves the software titles installed on hosts.
func (c *Client) ListSoftwareTitles(query string) ([]fleet.Software, error) {
	verb, path := "GET", "/api/latest/fleet/software/titles"
	// TODO(mna): adjust when this gets merged: https://github.com/fleetdm/fleet/issues/15228
	var responseBody listSoftwareResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Software, nil
}
