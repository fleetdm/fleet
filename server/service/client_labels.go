package service

import (
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyLabels sends the list of Labels to be applied (upserted) to the
// Fleet instance.
func (c *Client) ApplyLabels(specs []*fleet.LabelSpec) error {
	req := applyLabelSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/labels"
	var responseBody applyLabelSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetLabel retrieves information about a label by name
func (c *Client) GetLabel(name string) (*fleet.LabelSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/labels/"+url.PathEscape(name)
	var responseBody getLabelSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// GetLabels retrieves the list of all LabelSpecs.
func (c *Client) GetLabels() ([]*fleet.LabelSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/labels"
	var responseBody getLabelSpecsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Specs, err
}

// DeleteLabel deletes the label with the matching name.
func (c *Client) DeleteLabel(name string) error {
	verb, path := "DELETE", "/api/latest/fleet/labels/"+url.PathEscape(name)
	var responseBody deleteLabelResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
