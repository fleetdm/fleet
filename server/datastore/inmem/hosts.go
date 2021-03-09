package inmem

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/patrickmn/sortutil"
)

func (d *Datastore) NewHost(host *kolide.Host) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, h := range d.hosts {
		if host.NodeKey == h.NodeKey || host.UUID == h.UUID {
			return nil, alreadyExists("Host", host.ID)
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
		return notFound("Host").WithID(host.ID)
	}

	d.hosts[host.ID] = host
	return nil
}

func (d *Datastore) DeleteHost(hid uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.hosts[hid]; ok {
		delete(d.hosts, hid)
	}

	return nil
}

func (d *Datastore) Host(id uint) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	host, ok := d.hosts[id]
	if !ok {
		return nil, notFound("Host").WithID(id)
	}

	return host, nil
}

func (d *Datastore) ListHosts(opt kolide.HostListOptions) ([]*kolide.Host, error) {
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
		if err := sortResults(hosts, opt.ListOptions, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt.ListOptions, len(hosts))
	hosts = hosts[low:high]

	// Filter additional info
	if len(opt.AdditionalFilters) > 0 {
		fieldsWanted := map[string]interface{}{}
		for _, field := range opt.AdditionalFilters {
			fieldsWanted[field] = true
		}
		for i, host := range hosts {
			addInfo := map[string]interface{}{}
			if err := json.Unmarshal(*host.Additional, &addInfo); err != nil {
				return nil, err
			}

			for k := range addInfo {
				if _, ok := fieldsWanted[k]; !ok {
					delete(addInfo, k)
				}
			}
			addInfoJSON := json.RawMessage{}
			addInfoJSON, err := json.Marshal(addInfo)
			if err != nil {
				return nil, err
			}
			host.Additional = &addInfoJSON
			hosts[i] = host
		}
	}

	return hosts, nil
}

func (d *Datastore) GenerateHostStatusStatistics(now time.Time) (online, offline, mia, new uint, err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, host := range d.hosts {
		if host.IsNew(now) {
			new++
		}

		status := host.Status(now)
		switch status {
		case kolide.StatusMIA:
			mia++
		case kolide.StatusOffline:
			offline++
		default:
			online++
		}
	}

	return online, offline, mia, new, nil
}

func (d *Datastore) EnrollHost(osQueryHostID, nodeKey, secretName string, cooldown time.Duration) (*kolide.Host, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if osQueryHostID == "" {
		return nil, errors.New("missing host identifier from osquery for host enrollment")
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

	return nil, notFound("AuthenticateHost")
}

func (d *Datastore) MarkHostSeen(host *kolide.Host, t time.Time) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, h := range d.hosts {
		if h.ID == host.ID {
			h.UpdatedAt = t
			h.SeenTime = t
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

		if (strings.Contains(h.HostName, query) || strings.Contains(h.UUID, query)) && !omitLookup[h.ID] {
			results = append(results, h)
			continue
		}

		for _, nic := range h.NetworkInterfaces {
			if strings.Contains(nic.IPAddress, query) && !omitLookup[nic.HostID] {
				results = append(results, h)

				break
			}
		}
		sortutil.AscByField(h.NetworkInterfaces, "ID")
	}

	return results, nil
}
