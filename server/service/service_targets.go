package service

import (
	"context"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (svc service) SearchTargets(ctx context.Context, query string, selectedHostIDs []uint, selectedLabelIDs []uint, includeObserver bool) (*kolide.TargetSearchResults, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}

	filter := kolide.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	results := &kolide.TargetSearchResults{}

	hosts, err := svc.ds.SearchHosts(query, selectedHostIDs...)
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		results.Hosts = append(results.Hosts, *h)
	}

	labels, err := svc.ds.SearchLabels(filter, query, selectedLabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	return results, nil
}

func (svc service) CountHostsInTargets(ctx context.Context, hostIDs []uint, labelIDs []uint, includeObserver bool) (*kolide.TargetMetrics, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}

	filter := kolide.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	metrics, err := svc.ds.CountHostsInTargets(filter, hostIDs, labelIDs, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}
