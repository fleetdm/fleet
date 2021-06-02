package service

import (
	"context"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (svc Service) SearchTargets(ctx context.Context, matchQuery string, queryID *uint, targets kolide.HostTargets) (*kolide.TargetSearchResults, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Target{}, "read"); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, kolide.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(*queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := kolide.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	results := &kolide.TargetSearchResults{}

	hosts, err := svc.ds.SearchHosts(filter, matchQuery, targets.HostIDs...)
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		results.Hosts = append(results.Hosts, h)
	}

	labels, err := svc.ds.SearchLabels(filter, matchQuery, targets.LabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	teams, err := svc.ds.SearchTeams(filter, matchQuery, targets.TeamIDs...)
	if err != nil {
		return nil, err
	}
	results.Teams = teams

	return results, nil
}

func (svc Service) CountHostsInTargets(ctx context.Context, queryID *uint, targets kolide.HostTargets) (*kolide.TargetMetrics, error) {
	if err := svc.authz.Authorize(ctx, &kolide.Target{}, "read"); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, kolide.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(*queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := kolide.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	metrics, err := svc.ds.CountHostsInTargets(filter, targets, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}
