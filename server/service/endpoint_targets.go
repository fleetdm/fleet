package service

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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
	hostResponse
	DisplayText string `json:"display_text"`
}

type labelSearchResult struct {
	kolide.Label
	DisplayText     string `json:"display_text"`
	Count           uint   `json:"count"`
	Online          uint   `json:"online"`
	Offline         uint   `json:"offline"`
	MissingInAction uint   `json:"missing_in_action"`
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

		onlineInterval, err := svc.ExpectedCheckinInterval(ctx)
		if err != nil {
			return searchTargetsResponse{Err: errors.Wrap(err, "getting expected check-in interval")}, nil
		}

		for _, host := range results.Hosts {
			targets.Hosts = append(targets.Hosts,
				hostSearchResult{
					hostResponse{
						Host:   host,
						Status: host.Status(time.Now(), onlineInterval),
					},
					host.HostName,
				},
			)
		}

		for _, label := range results.Labels {
			metrics, err := svc.CountHostsInTargets(ctx, nil, []uint{label.ID})
			if err != nil {
				return searchTargetsResponse{Err: err}, nil
			}
			targets.Labels = append(targets.Labels,
				labelSearchResult{
					Label:           label,
					DisplayText:     label.Name,
					Count:           metrics.TotalHosts,
					Online:          metrics.OnlineHosts,
					Offline:         metrics.OfflineHosts,
					MissingInAction: metrics.MissingInActionHosts,
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
