package service

import (
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyPacks sends the list of Packs to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyPacks(specs []*fleet.PackSpec) error {
	req := applyPackSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/v1/fleet/spec/packs"
	var responseBody applyPackSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetPack retrieves information about a pack
func (c *Client) GetPack(name string) (*fleet.PackSpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/packs/"+url.PathEscape(name)
	var responseBody getPackSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// GetPacks retrieves the list of all Packs.
func (c *Client) GetPacks() ([]*fleet.PackSpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/packs"
	var responseBody getPackSpecsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Specs, err
}

// DeletePack deletes the pack with the matching name.
func (c *Client) DeletePack(name string) error {
	verb, path := "DELETE", "/api/v1/fleet/packs/"+url.PathEscape(name)
	var responseBody deletePackResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
