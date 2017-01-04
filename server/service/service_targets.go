package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
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
	hosts, err := svc.ds.ListUniqueHostsInLabels(labelIDs)
	if err != nil {
		return nil, err
	}

	for _, id := range hostIDs {
		h, err := svc.ds.Host(id)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, *h)
	}

	hostLookup := map[uint]bool{}

	result := &kolide.TargetMetrics{}

	for _, host := range hosts {
		if !hostLookup[host.ID] {
			hostLookup[host.ID] = true
			switch host.Status(svc.clock.Now().UTC()) {
			case kolide.StatusOnline:
				result.OnlineHosts++
			case kolide.StatusOffline:
				result.OfflineHosts++
			case kolide.StatusMIA:
				result.MissingInActionHosts++
			}
		}
	}

	result.TotalHosts = uint(len(hostLookup))

	return result, nil
}
