package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyLabels sends the list of Labels to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyLabels(specs []*fleet.LabelSpec) error {
	req := applyLabelSpecsRequest{Specs: specs}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/spec/labels", "", req)
	if err != nil {
		return fmt.Errorf("POST /api/v1/fleet/spec/labels: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"apply labels received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody applyLabelSpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode apply label spec response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("apply label spec: %s", responseBody.Err)
	}

	return nil
}

// GetLabel retrieves information about a label by name
func (c *Client) GetLabel(name string) (*fleet.LabelSpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/labels/"+url.PathEscape(name)
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/spec/labels: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return nil, notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get label received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getLabelSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get label spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get label spec: %s", responseBody.Err)
	}

	return responseBody.Spec, nil
}

// GetLabels retrieves the list of all Labels.
func (c *Client) GetLabels() ([]*fleet.LabelSpec, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/spec/labels", "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/spec/labels: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get labels received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getLabelSpecsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get label spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get label spec: %s", responseBody.Err)
	}

	return responseBody.Specs, nil
}

// DeleteLabel deletes the label with the matching name.
func (c *Client) DeleteLabel(name string) error {
	verb, path := "DELETE", "/api/v1/fleet/labels/"+url.PathEscape(name)
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
			"delete label received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody deleteLabelResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode get label spec response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("get label spec: %s", responseBody.Err)
	}

	return nil
}
