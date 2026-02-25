package client

import (
	"fmt"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyLabels sends the list of Labels to be applied (upserted) to the
// Fleet instance. Use teamID = nil for global labels.
func (c *Client) ApplyLabels(
	specs []*fleet.LabelSpec,
	teamID *uint,
	moves []string,
) error {
	req := fleet.ApplyLabelSpecsRequest{TeamID: teamID, Specs: specs, NamesToMove: moves}
	verb, path := "POST", "/api/latest/fleet/spec/labels"
	var responseBody fleet.ApplyLabelSpecsResponse

	if teamID != nil {
		return c.authenticatedRequestWithQuery(
			req,
			verb,
			path,
			&responseBody,
			fmt.Sprintf("team_id=%d", *teamID),
		)
	}
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetLabel retrieves information about a label by name
func (c *Client) GetLabel(name string) (*fleet.LabelSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/labels/"+url.PathEscape(name)
	var responseBody fleet.GetLabelSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// GetLabels retrieves the list of all LabelSpecs.
func (c *Client) GetLabels(teamID uint) ([]*fleet.LabelSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/labels"
	var responseBody fleet.GetLabelSpecsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, fmt.Sprintf("team_id=%d", teamID))
	return responseBody.Specs, err
}

// DeleteLabel deletes the label with the matching name.
func (c *Client) DeleteLabel(name string) error {
	verb, path := "DELETE", "/api/latest/fleet/labels/"+url.PathEscape(name)
	var responseBody fleet.DeleteLabelResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
