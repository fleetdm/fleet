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
	results.Hosts = hosts

	labels, err := svc.ds.SearchLabels(query, selectedLabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	return results, nil
}

func (svc service) CountHostsInTargets(ctx context.Context, hosts []uint, labels []uint) (uint, error) {
	hostsInLabels, err := svc.ds.ListUniqueHostsInLabels(labels)
	if err != nil {
		return 0, err
	}

	hostLookup := map[uint]bool{}

	for _, host := range hosts {
		hostLookup[host] = true
	}

	for _, host := range hostsInLabels {
		if !hostLookup[host.ID] {
			hostLookup[host.ID] = true
		}
	}

	return uint(len(hostLookup)), nil
}
