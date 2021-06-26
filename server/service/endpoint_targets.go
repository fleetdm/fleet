package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Search Targets
////////////////////////////////////////////////////////////////////////////////

type searchTargetsRequest struct {
	// MatchQuery is the query SQL
	MatchQuery string `json:"query"`
	// QueryID is the ID of a saved query to run (used to determine if this is a
	// query that observers can run).
	QueryID  *uint             `json:"query_id"`
	Selected fleet.HostTargets `json:"selected"`
}

type hostSearchResult struct {
	HostResponse
	DisplayText string `json:"display_text"`
}

type labelSearchResult struct {
	*fleet.Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

type teamSearchResult struct {
	*fleet.Team
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

type targetsData struct {
	Hosts  []hostSearchResult  `json:"hosts"`
	Labels []labelSearchResult `json:"labels"`
	Teams  []teamSearchResult  `json:"teams"`
}

type searchTargetsResponse struct {
	Targets                *targetsData `json:"targets,omitempty"`
	TargetsCount           uint         `json:"targets_count"`
	TargetsOnline          uint         `json:"targets_online"`
	TargetsOffline         uint         `json:"targets_offline"`
	TargetsMissingInAction uint         `json:"targets_missing_in_action"`
	Err                    error        `json:"error,omitempty"`
}

func (r searchTargetsResponse) error() error { return r.Err }

func makeSearchTargetsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchTargetsRequest)

		results, err := svc.SearchTargets(ctx, req.MatchQuery, req.QueryID, req.Selected)
		if err != nil {
			return searchTargetsResponse{Err: err}, nil
		}

		targets := &targetsData{
			Hosts:  []hostSearchResult{},
			Labels: []labelSearchResult{},
			Teams:  []teamSearchResult{},
		}

		for _, host := range results.Hosts {
			targets.Hosts = append(targets.Hosts,
				hostSearchResult{
					HostResponse{
						Host:   host,
						Status: host.Status(time.Now()),
					},
					host.Hostname,
				},
			)
		}

		for _, label := range results.Labels {
			targets.Labels = append(targets.Labels,
				labelSearchResult{
					Label:       label,
					DisplayText: label.Name,
					Count:       label.HostCount,
				},
			)
		}

		for _, team := range results.Teams {
			targets.Teams = append(targets.Teams,
				teamSearchResult{
					Team:        team,
					DisplayText: team.Name,
					Count:       team.HostCount,
				},
			)
		}

		metrics, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
		if err != nil {
			return searchTargetsResponse{Err: err}, nil
		}

		return searchTargetsResponse{
			Targets:                targets,
			TargetsCount:           metrics.TotalHosts,
			TargetsOnline:          metrics.OnlineHosts,
			TargetsOffline:         metrics.OfflineHosts,
			TargetsMissingInAction: metrics.MissingInActionHosts,
		}, nil
	}
}
