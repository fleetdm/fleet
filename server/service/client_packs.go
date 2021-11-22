package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyPacks sends the list of Packs to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyPacks(specs []*fleet.PackSpec) error {
	req := applyPackSpecsRequest{Specs: specs}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/spec/packs", "", req)
	if err != nil {
		return fmt.Errorf("POST /api/v1/fleet/spec/packs: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"apply packs received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody applyPackSpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode apply pack spec response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("apply pack spec: %s", responseBody.Err)
	}

	return nil
}

// GetPack retrieves information about a pack
func (c *Client) GetPack(name string) (*fleet.PackSpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/packs/"+url.PathEscape(name)
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/spec/packs: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return nil, notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get pack received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getPackSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get pack spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get pack spec: %s", responseBody.Err)
	}

	return responseBody.Spec, nil
}

// GetPacks retrieves the list of all Packs.
func (c *Client) GetPacks() ([]*fleet.PackSpec, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/spec/packs", "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/spec/packs: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get packs received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getPackSpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get pack spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get pack spec: %s", responseBody.Err)
	}

	return responseBody.Specs, nil
}

// DeletePack deletes the pack with the matching name.
func (c *Client) DeletePack(name string) error {
	verb, path := "DELETE", "/api/v1/fleet/packs/"+url.PathEscape(name)
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
			"delete pack received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody deletePackResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode get pack spec response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("get pack spec: %s", responseBody.Err)
	}

	return nil
}
