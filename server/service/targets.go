package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func searchTargetsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.SearchTargetsRequest)

	results, err := svc.SearchTargets(ctx, req.MatchQuery, req.QueryID, req.Selected)
	if err != nil {
		return fleet.SearchTargetsResponse{Err: err}, nil
	}

	targets := &fleet.TargetsData{
		Hosts:  []*fleet.HostResponse{},
		Labels: []fleet.LabelSearchResult{},
		Teams:  []fleet.TeamSearchResult{},
	}

	for _, host := range results.Hosts {
		targets.Hosts = append(targets.Hosts, fleet.HostResponseForHostCheap(host))
	}

	for _, label := range results.Labels {
		targets.Labels = append(targets.Labels,
			fleet.LabelSearchResult{
				Label:       label,
				DisplayText: label.Name,
				Count:       label.HostCount,
			},
		)
	}

	for _, team := range results.Teams {
		targets.Teams = append(targets.Teams,
			fleet.TeamSearchResult{
				Team:        team,
				DisplayText: team.Name,
				Count:       team.HostCount,
			},
		)
	}

	metrics, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
	if err != nil {
		return fleet.SearchTargetsResponse{Err: err}, nil
	}

	return fleet.SearchTargetsResponse{
		Targets:                targets,
		TargetsCount:           metrics.TotalHosts,
		TargetsOnline:          metrics.OnlineHosts,
		TargetsOffline:         metrics.OfflineHosts,
		TargetsMissingInAction: metrics.MissingInActionHosts,
	}, nil
}

func (svc *Service) SearchTargets(ctx context.Context, matchQuery string, queryID *uint, targets fleet.HostTargets) (*fleet.TargetSearchResults, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Target{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	results := &fleet.TargetSearchResults{}

	hosts, err := svc.ds.SearchHosts(ctx, filter, matchQuery, targets.HostIDs...)
	if err != nil {
		return nil, err
	}

	results.Hosts = append(results.Hosts, hosts...)

	labels, err := svc.ds.SearchLabels(ctx, filter, matchQuery, targets.LabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	teams, err := svc.ds.SearchTeams(ctx, filter, matchQuery, targets.TeamIDs...)
	if err != nil {
		return nil, err
	}
	results.Teams = teams

	return results, nil
}

func (svc *Service) CountHostsInTargets(ctx context.Context, queryID *uint, targets fleet.HostTargets) (*fleet.TargetMetrics, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Target{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	metrics, err := svc.ds.CountHostsInTargets(ctx, filter, targets, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

func countTargetsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CountTargetsRequest)

	counts, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
	if err != nil {
		return fleet.SearchTargetsResponse{Err: err}, nil
	}

	return fleet.CountTargetsResponse{
		TargetsCount:   counts.TotalHosts,
		TargetsOnline:  counts.OnlineHosts,
		TargetsOffline: counts.OfflineHosts,
	}, nil
}
