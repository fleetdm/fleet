package service

import (
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

// SearchTargets searches for the supplied targets in the Fleet instance.
func (c *Client) SearchTargets(query string, hostIDs, labelIDs []uint) (*fleet.TargetSearchResults, error) {
	req := searchTargetsRequest{
		MatchQuery: query,
		Selected: fleet.HostTargets{
			LabelIDs: labelIDs,
			HostIDs:  hostIDs,
			// TODO handle TeamIDs
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

	hosts := make([]*fleet.Host, len(responseBody.Targets.Hosts))
	for i, h := range responseBody.Targets.Hosts {
		hosts[i] = h.HostResponse.Host
	}

	labels := make([]*fleet.Label, len(responseBody.Targets.Labels))
	for i, h := range responseBody.Targets.Labels {
		labels[i] = h.Label
	}

	return &fleet.TargetSearchResults{
		Hosts:  hosts,
		Labels: labels,
	}, nil
}
