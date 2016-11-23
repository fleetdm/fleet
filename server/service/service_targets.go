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

func (svc service) CountHostsInTargets(ctx context.Context, hostIDs []uint, labelIDs []uint) (total uint, online uint, err error) {
	hosts, err := svc.ds.ListUniqueHostsInLabels(labelIDs)
	if err != nil {
		return 0, 0, err
	}

	for _, id := range hostIDs {
		h, err := svc.ds.Host(id)
		if err != nil {
			return 0, 0, err
		}
		hosts = append(hosts, *h)
	}

	hostLookup := map[uint]bool{}
	online = uint(0)
	for _, host := range hosts {
		if !hostLookup[host.ID] {
			hostLookup[host.ID] = true
			if svc.HostStatus(ctx, host) == "online" {
				online++
			}
		}
	}

	return uint(len(hostLookup)), online, nil
}
