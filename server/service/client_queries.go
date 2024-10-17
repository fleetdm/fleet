package service

import (
	"fmt"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyQueries sends the list of Queries to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyQueries(specs []*fleet.QuerySpec) error {
	req := applyQuerySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/queries"
	var responseBody applyQuerySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetQuerySpec returns the query spec of a query by its team+name.
func (c *Client) GetQuerySpec(teamID *uint, name string) (*fleet.QuerySpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/queries/"+url.PathEscape(name)
	query := url.Values{}
	if teamID != nil {
		query.Set("team_id", fmt.Sprint(*teamID))
	}
	var responseBody getQuerySpecResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	return responseBody.Spec, err
}

// GetQueries retrieves the list of all Queries.
func (c *Client) GetQueries(teamID *uint, name *string) ([]fleet.Query, error) {
	verb, path := "GET", "/api/latest/fleet/queries"
	query := url.Values{}
	if teamID != nil {
		query.Set("team_id", fmt.Sprint(*teamID))
	}
	if name != nil {
		query.Set("query", *name)
	}
	var responseBody listQueriesResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, err
	}
	return responseBody.Queries, nil
}

// DeleteQuery deletes the query with the matching name.
func (c *Client) DeleteQuery(name string) error {
	verb, path := "DELETE", "/api/latest/fleet/queries/"+url.PathEscape(name)
	var responseBody deleteQueryResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

// DeleteQueries deletes several queries.
func (c *Client) DeleteQueries(ids []uint) error {
	req := deleteQueriesRequest{IDs: ids}
	verb, path := "POST", "/api/latest/fleet/queries/delete"
	var responseBody deleteQueriesResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
