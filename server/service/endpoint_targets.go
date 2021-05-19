package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Search Targets
////////////////////////////////////////////////////////////////////////////////

type searchTargetsRequest struct {
	Query    string `json:"query"`
	Selected struct {
		Labels []uint `json:"labels"`
		Hosts  []uint `json:"hosts"`
	} `json:"selected"`
}

type hostSearchResult struct {
	HostResponse
	DisplayText string `json:"display_text"`
}

type labelSearchResult struct {
	kolide.Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

type targetsData struct {
	Hosts  []hostSearchResult  `json:"hosts"`
	Labels []labelSearchResult `json:"labels"`
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

func makeSearchTargetsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchTargetsRequest)

		results, err := svc.SearchTargets(ctx, req.Query, req.Selected.Hosts, req.Selected.Labels)
		if err != nil {
			return searchTargetsResponse{Err: err}, nil
		}

		targets := &targetsData{
			Hosts:  []hostSearchResult{},
			Labels: []labelSearchResult{},
		}

		for _, host := range results.Hosts {
			targets.Hosts = append(targets.Hosts,
				hostSearchResult{
					HostResponse{
						Host:   host,
						Status: host.Status(time.Now()),
					},
					host.HostName,
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

		metrics, err := svc.CountHostsInTargets(ctx, req.Selected.Hosts, req.Selected.Labels)
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
