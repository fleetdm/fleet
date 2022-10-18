package service

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	verb, path := "POST", "/api/latest/fleet/targets"
	var responseBody searchTargetsResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	if err != nil {
		return nil, fmt.Errorf("SearchTargets: %s", err)
	}

	hosts := make([]*fleet.Host, len(responseBody.Targets.Hosts))
	for i, h := range responseBody.Targets.Hosts {
		hosts[i] = h.Host
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
