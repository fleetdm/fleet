package service

import (
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyPacks sends the list of Packs to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyPacks(specs []*fleet.PackSpec) error {
	req := applyPackSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/packs"
	var responseBody applyPackSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetPackSpec retrieves information about a pack in apply spec format.
func (c *Client) GetPackSpec(name string) (*fleet.PackSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/packs/"+url.PathEscape(name)
	var responseBody getPackSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// GetPacksSpecs retrieves the list of all Packs in apply specs format.
func (c *Client) GetPacksSpecs() ([]*fleet.PackSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/packs"
	var responseBody getPackSpecsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Specs, err
}

// ListPacks retrieves the list of all Packs.
func (c *Client) ListPacks() ([]*fleet.Pack, error) {
	verb, path := "GET", "/api/latest/fleet/packs"
	var responseBody listPacksResponse
	if err := c.authenticatedRequest(nil, verb, path, &responseBody); err != nil {
		return nil, err
	}

	packs := make([]*fleet.Pack, 0, len(responseBody.Packs))
	for _, pr := range responseBody.Packs {
		pack := pr.Pack
		packs = append(packs, &pack)
	}
	return packs, nil
}

// DeletePack deletes the pack with the matching name.
func (c *Client) DeletePack(name string) error {
	verb, path := "DELETE", "/api/latest/fleet/packs/"+url.PathEscape(name)
	var responseBody deletePackResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
