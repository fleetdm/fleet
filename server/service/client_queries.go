package service

import (
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyQueries sends the list of Queries to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyQueries(specs []*fleet.QuerySpec) error {
	req := applyQuerySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/v1/fleet/spec/queries"
	var responseBody applyQuerySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetQuery retrieves the list of all Queries.
func (c *Client) GetQuery(name string) (*fleet.QuerySpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/queries/"+url.PathEscape(name)
	var responseBody getQuerySpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// GetQueries retrieves the list of all Queries.
func (c *Client) GetQueries() ([]*fleet.QuerySpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/queries"
	var responseBody getQuerySpecsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Specs, err
}

// DeleteQuery deletes the query with the matching name.
func (c *Client) DeleteQuery(name string) error {
	verb, path := "DELETE", "/api/v1/fleet/queries/"+url.PathEscape(name)
	var responseBody deleteQueryResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
