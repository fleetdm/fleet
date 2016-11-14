package service

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Search Targrets
////////////////////////////////////////////////////////////////////////////////

type searchTargetsRequest struct {
	Query    string `json:"query"`
	Selected struct {
		Labels []uint `json:"labels"`
		Hosts  []uint `json:"hosts"`
	} `json:"selected"`
}

type hostSearchResult struct {
	hostResponse
	DisplayText string `json:"display_text"`
}

type labelSearchResult struct {
	kolide.Label
	DisplayText string `json:"display_text"`
	Count       uint   `json:"count"`
}

type targetsData struct {
	Hosts  []hostSearchResult  `json:"hosts"`
	Labels []labelSearchResult `json:"labels"`
}

type searchTargetsResponse struct {
	Targets              *targetsData `json:"targets,omitempty"`
	SelectedTargetsCount uint         `json:"selected_targets_count,omitempty"`
	Err                  error        `json:"error,omitempty"`
}

func (r searchTargetsResponse) error() error { return r.Err }

func makeSearchTargetsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchTargetsRequest)

		results, err := svc.SearchTargets(ctx, req.Query, req.Selected.Hosts, req.Selected.Labels)
		if err != nil {
			return searchTargetsResponse{Err: err}, nil
		}

		count, err := svc.CountHostsInTargets(ctx, req.Selected.Hosts, req.Selected.Labels)
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
					hostResponse{host, svc.HostStatus(ctx, host)},
					host.HostName,
				},
			)
		}

		for _, label := range results.Labels {
			count, err := svc.CountHostsInTargets(ctx, nil, []uint{label.ID})
			if err != nil {
				return searchTargetsResponse{Err: err}, nil
			}
			targets.Labels = append(targets.Labels,
				labelSearchResult{
					label,
					label.Name,
					count,
				},
			)
		}

		return searchTargetsResponse{
			Targets:              targets,
			SelectedTargetsCount: count,
		}, nil
	}
}
