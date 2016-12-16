package inmem

import (
	"errors"
	"sort"
	"strings"
	"time"

	kolide_errors "github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewHost(host *kolide.Host) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, h := range d.hosts {
		if host.NodeKey == h.NodeKey || host.UUID == h.UUID {
			return nil, kolide_errors.ErrExists
		}
	}

	host.ID = d.nextID(host)
	d.hosts[host.ID] = host

	return host, nil
}

func (d *Datastore) SaveHost(host *kolide.Host) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.hosts[host.ID]; !ok {
		return kolide_errors.ErrNotFound
	}

	for _, nic := range host.NetworkInterfaces {
		if nic.ID == 0 {
			nic.ID = d.nextID(nic)
		}
		nic.HostID = host.ID
	}
	host.ResetPrimaryNetwork()
	d.hosts[host.ID] = host
	return nil
}

func (d *Datastore) DeleteHost(host *kolide.Host) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.hosts[host.ID]; !ok {
		return kolide_errors.ErrNotFound
	}

	delete(d.hosts, host.ID)
	return nil
}

func (d *Datastore) Host(id uint) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	host, ok := d.hosts[id]
	if !ok {
		return nil, kolide_errors.ErrNotFound
	}

	return host, nil
}

func (d *Datastore) ListHosts(opt kolide.ListOptions) ([]*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range d.hosts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	hosts := []*kolide.Host{}
	for _, k := range keys {
		hosts = append(hosts, d.hosts[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":                 "ID",
			"created_at":         "CreatedAt",
			"updated_at":         "UpdatedAt",
			"detail_update_time": "DetailUpdateTime",
			"hostname":           "HostName",
			"uuid":               "UUID",
			"platform":           "Platform",
			"osquery_version":    "OsqueryVersion",
			"os_version":         "OSVersion",
			"uptime":             "Uptime",
			"memory":             "PhysicalMemory",
			"mac":                "PrimaryMAC",
			"ip":                 "PrimaryIP",
		}
		if err := sortResults(hosts, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt, len(hosts))
	hosts = hosts[low:high]

	return hosts, nil
}

func (d *Datastore) EnrollHost(osQueryHostID string, nodeKeySize int) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if osQueryHostID == "" {
		return nil, errors.New("missing host identifier from osquery for host enrollment")
	}

	nodeKey, err := kolide.RandomText(nodeKeySize)
	if err != nil {
		return nil, err
	}

	host := kolide.Host{
		OsqueryHostID:    osQueryHostID,
		NodeKey:          nodeKey,
		DetailUpdateTime: time.Unix(0, 0).Add(24 * time.Hour),
	}

	host.CreatedAt = time.Now().UTC()
	host.UpdatedAt = host.CreatedAt

	for _, h := range d.hosts {
		if h.OsqueryHostID == osQueryHostID {
			host = *h
			break
		}
	}

	if host.ID == 0 {
		host.ID = d.nextID(host)
	}
	d.hosts[host.ID] = &host

	return &host, nil
}

func (d *Datastore) AuthenticateHost(nodeKey string) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, host := range d.hosts {
		if host.NodeKey == nodeKey {
			return host, nil
		}
	}

	return nil, kolide_errors.ErrNotFound
}

func (d *Datastore) MarkHostSeen(host *kolide.Host, t time.Time) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	host.UpdatedAt = t

	for _, h := range d.hosts {
		if h.ID == host.ID {
			h.UpdatedAt = t
			break
		}
	}
	return nil
}

func (d *Datastore) SearchHosts(query string, omit ...uint) ([]*kolide.Host, error) {
	omitLookup := map[uint]bool{}
	for _, o := range omit {
		omitLookup[o] = true
	}

	var results []*kolide.Host

	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, h := range d.hosts {
		if len(results) == 10 {
			break
		}

		if strings.Contains(h.HostName, query) && !omitLookup[h.ID] {
			results = append(results, h)
			continue
		}

		for _, nic := range h.NetworkInterfaces {
			if strings.Contains(nic.IPAddress, query) && !omitLookup[nic.HostID] {
				results = append(results, h)
				break
			}
		}
	}

	return results, nil
}

func (d *Datastore) DistributedQueriesForHost(host *kolide.Host) (map[uint]string, error) {
	// lookup of executions for this host
	hostExecutions := map[uint]kolide.DistributedQueryExecutionStatus{}
	for _, e := range d.distributedQueryExecutions {
		if e.HostID == host.ID {
			hostExecutions[e.DistributedQueryCampaignID] = e.Status
		}
	}

	// lookup of labels for this host (only including matching labels)
	hostLabels := map[uint]bool{}
	labels, err := d.ListLabelsForHost(host.ID)
	if err != nil {
		return nil, err
	}
	for _, l := range labels {
		hostLabels[l.ID] = true
	}

	queries := map[uint]string{} // map campaign ID -> query string
	for _, campaign := range d.distributedQueryCampaigns {
		if campaign.Status != kolide.QueryRunning {
			continue
		}
		for _, target := range d.distributedQueryCampaignTargets {
			if campaign.ID == target.DistributedQueryCampaignID &&
				((target.Type == kolide.TargetHost && target.TargetID == host.ID) ||
					(target.Type == kolide.TargetLabel && hostLabels[target.TargetID])) &&
				(hostExecutions[campaign.ID] == kolide.ExecutionWaiting) {
				queries[campaign.ID] = d.queries[campaign.QueryID].Query
			}
		}
	}

	return queries, nil
}
