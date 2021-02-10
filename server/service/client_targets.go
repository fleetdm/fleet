package service

import (
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

// SearchTargets searches for the supplied targets in the Fleet instance.
func (c *Client) SearchTargets(query string, selectedHostIDs, selectedLabelIDs []uint) (*kolide.TargetSearchResults, error) {
	req := searchTargetsRequest{
		Query: query,
		Selected: struct {
			Labels []uint `json:"labels"`
			Hosts  []uint `json:"hosts"`
		}{
			Labels: selectedLabelIDs,
			Hosts:  selectedHostIDs,
		},
	}

	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/targets", "", req)
	if err != nil {
		return nil, errors.Wrap(err, "POST /api/v1/fleet/targets")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"SearchTargets received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody searchTargetsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode SearchTargets response")
	}

	if responseBody.Err != nil {
		return nil, errors.Errorf("SearchTargets: %s", responseBody.Err)
	}

	hosts := make([]kolide.Host, len(responseBody.Targets.Hosts))
	for i, h := range responseBody.Targets.Hosts {
		hosts[i] = h.HostResponse.Host
	}

	labels := make([]kolide.Label, len(responseBody.Targets.Labels))
	for i, h := range responseBody.Targets.Labels {
		labels[i] = h.Label
	}

	return &kolide.TargetSearchResults{
		Hosts:  hosts,
		Labels: labels,
	}, nil
}
