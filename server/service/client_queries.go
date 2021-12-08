package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyQueries sends the list of Queries to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyQueries(specs []*fleet.QuerySpec) error {
	req := applyQuerySpecsRequest{Specs: specs}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/spec/queries", "", req)
	if err != nil {
		return fmt.Errorf("POST /api/v1/fleet/spec/queries: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"apply queries received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody applyQuerySpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode apply query spec response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("apply query spec: %s", responseBody.Err)
	}

	return nil
}

// GetQuery retrieves the list of all Queries.
func (c *Client) GetQuery(name string) (*fleet.QuerySpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/queries/"+url.PathEscape(name)
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return nil, notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get query received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getQuerySpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get query spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get query spec: %s", responseBody.Err)
	}

	return responseBody.Spec, nil
}

// GetQueries retrieves the list of all Queries.
func (c *Client) GetQueries() ([]*fleet.QuerySpec, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/spec/queries", "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/spec/queries: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get queries received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getQuerySpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get query spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get query spec: %s", responseBody.Err)
	}

	return responseBody.Specs, nil
}

// DeleteQuery deletes the query with the matching name.
func (c *Client) DeleteQuery(name string) error {
	verb, path := "DELETE", "/api/v1/fleet/queries/"+url.PathEscape(name)
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"delete query received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody deleteQueryResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode get query spec response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("get query spec: %s", responseBody.Err)
	}

	return nil
}

func (c *Client) CreateQuery(name, query, description string) (*fleet.Query, error) {
	req := createQueryRequest{
		payload: fleet.QueryPayload{
			Name:        &name,
			Description: &description,
			Query:       &query,
		},
	}
	verb, path := "POST", "/api/v1/fleet/queries"
	var responseBody createQueryResponse
	err := c.authenticatedRequest(req.payload, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Query, nil
}
