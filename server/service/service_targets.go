package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
)

func (svc service) SearchTargets(ctx context.Context, query string, selectedHostIDs []uint, selectedLabelIDs []uint) (*kolide.TargetSearchResults, error) {
	results := &kolide.TargetSearchResults{}

	hosts, err := svc.ds.SearchHosts(query, selectedHostIDs...)
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		results.Hosts = append(results.Hosts, *h)
	}

	labels, err := svc.ds.SearchLabels(query, selectedLabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	return results, nil
}

func (svc service) CountHostsInTargets(ctx context.Context, hostIDs []uint, labelIDs []uint) (*kolide.TargetMetrics, error) {
	metrics, err := svc.ds.CountHostsInTargets(hostIDs, labelIDs, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}
