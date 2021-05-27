package service

import (
	"context"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (svc service) SearchTargets(ctx context.Context, matchQuery string, queryID *uint, selectedHostIDs []uint, selectedLabelIDs []uint) (*kolide.TargetSearchResults, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
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

	hosts, err := svc.ds.SearchHosts(filter, matchQuery, selectedHostIDs...)
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		results.Hosts = append(results.Hosts, h)
	}

	labels, err := svc.ds.SearchLabels(filter, matchQuery, selectedLabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	return results, nil
}

func (svc service) CountHostsInTargets(ctx context.Context, queryID *uint, hostIDs []uint, labelIDs []uint) (*kolide.TargetMetrics, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
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

	metrics, err := svc.ds.CountHostsInTargets(filter, hostIDs, labelIDs, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}
