package service

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// ApplyQuerySpecs sends the list of Queries to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyQuerySpecs(specs []*kolide.QuerySpec) error {
	req := applyQuerySpecsRequest{Specs: specs}
	response, err := c.Do("POST", "/api/v1/kolide/spec/queries", req)
	if err != nil {
		return errors.Wrap(err, "POST /api/v1/kolide/spec/queries")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("apply query spec got HTTP %d, expected 200", response.StatusCode)
	}

	var responseBody applyQuerySpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode apply query spec response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("apply query spec: %s", responseBody.Err)
	}

	return nil
}

// GetQuerySpecs retrieves the list of all Queries.
func (c *Client) GetQuerySpecs(specs []*kolide.QuerySpec) ([]*kolide.QuerySpec, error) {
	req := applyQuerySpecsRequest{Specs: specs}
	response, err := c.Do("GET", "/api/v1/kolide/spec/queries", req)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/kolide/spec/queries")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("get query spec got HTTP %d, expected 200", response.StatusCode)
	}

	var responseBody getQuerySpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get query spec response")
	}

	if responseBody.Err != nil {
		return nil, errors.Errorf("get query spec: %s", responseBody.Err)
	}

	return responseBody.Specs, nil
}

// DeleteQuery deletes the query with the matching name.
func (c *Client) DeleteQuery(name string) error {
	verb, path := "DELETE", "/api/v1/kolide/queries/"+url.QueryEscape(name)
	response, err := c.Do(verb, path, nil)
	if err != nil {
		return errors.Wrapf(err, "%s %s", verb, path)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("get query spec got HTTP %d, expected 200", response.StatusCode)
	}

	var responseBody deleteQueryResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode get query spec response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("get query spec: %s", responseBody.Err)
	}

	return nil
}
