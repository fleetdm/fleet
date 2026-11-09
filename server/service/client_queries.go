package service

import (
	"fmt"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyQueries sends the list of Queries to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyQueries(specs []*fleet.QuerySpec) error {
	req := fleet.ApplyQuerySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/reports"
	var responseBody fleet.ApplyQuerySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetQuerySpec returns the query spec of a query by its team+name.
func (c *Client) GetQuerySpec(teamID *uint, name string) (*fleet.QuerySpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/reports/"+url.PathEscape(name)
	query := url.Values{}
	if teamID != nil {
		query.Set("fleet_id", fmt.Sprint(*teamID))
	}
	var responseBody fleet.GetQuerySpecResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	return responseBody.Spec, err
}

// GetQueries retrieves the list of all Queries.
func (c *Client) GetQueries(teamID *uint, name *string) ([]fleet.Query, error) {
	verb, path := "GET", "/api/latest/fleet/reports"
	query := url.Values{}
	if teamID != nil {
		query.Set("fleet_id", fmt.Sprint(*teamID))
	}
	if name != nil {
		query.Set("query", *name)
	}
	var responseBody fleet.ListQueriesResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, err
	}
	return responseBody.Queries, nil
}

// DeleteQuery deletes the query with the matching name.
func (c *Client) DeleteQuery(name string) error {
	verb, path := "DELETE", "/api/latest/fleet/reports/"+url.PathEscape(name)
	var responseBody fleet.DeleteQueryResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

// DeleteQueries deletes several queries.
func (c *Client) DeleteQueries(ids []uint) error {
	req := fleet.DeleteQueriesRequest{IDs: ids}
	verb, path := "POST", "/api/latest/fleet/reports/delete"
	var responseBody fleet.DeleteQueriesResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
