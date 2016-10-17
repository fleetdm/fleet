package datastore

import (
	"errors"
	"sort"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *inmem) NewHost(host *kolide.Host) (*kolide.Host, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, h := range orm.hosts {
		if host.NodeKey == h.NodeKey || host.UUID == h.UUID {
			return nil, ErrExists
		}
	}

	host.ID = orm.nextID(host)
	orm.hosts[host.ID] = host

	return host, nil
}

func (orm *inmem) SaveHost(host *kolide.Host) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.hosts[host.ID]; !ok {
		return ErrNotFound
	}

	orm.hosts[host.ID] = host
	return nil
}

func (orm *inmem) DeleteHost(host *kolide.Host) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.hosts[host.ID]; !ok {
		return ErrNotFound
	}

	delete(orm.hosts, host.ID)
	return nil
}

func (orm *inmem) Host(id uint) (*kolide.Host, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	host, ok := orm.hosts[id]
	if !ok {
		return nil, ErrNotFound
	}

	return host, nil
}

func (orm *inmem) ListHosts(opt kolide.ListOptions) ([]*kolide.Host, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range orm.hosts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	hosts := []*kolide.Host{}
	for _, k := range keys {
		hosts = append(hosts, orm.hosts[uint(k)])
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
	low, high := orm.getLimitOffsetSliceBounds(opt, len(hosts))
	hosts = hosts[low:high]

	return hosts, nil
}

func (orm *inmem) EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*kolide.Host, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if uuid == "" {
		return nil, errors.New("missing uuid for host enrollment")
	}

	host := kolide.Host{
		UUID:             uuid,
		HostName:         hostname,
		PrimaryIP:        ip,
		Platform:         platform,
		DetailUpdateTime: time.Unix(0, 0).Add(24 * time.Hour),
	}
	for _, h := range orm.hosts {
		if h.UUID == uuid {
			host = *h
			break
		}
	}

	var err error
	host.NodeKey, err = generateRandomText(nodeKeySize)
	if err != nil {
		return nil, err
	}

	if hostname != "" {
		host.HostName = hostname
	}
	if ip != "" {
		host.PrimaryIP = ip
	}
	if platform != "" {
		host.Platform = platform
	}

	if host.ID == 0 {
		host.ID = orm.nextID(host)
	}
	orm.hosts[host.ID] = &host

	return &host, nil
}

func (orm *inmem) AuthenticateHost(nodeKey string) (*kolide.Host, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, host := range orm.hosts {
		if host.NodeKey == nodeKey {
			return host, nil
		}
	}

	return nil, ErrNotFound
}

func (orm *inmem) MarkHostSeen(host *kolide.Host, t time.Time) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	host.UpdatedAt = t

	for _, h := range orm.hosts {
		if h.ID == host.ID {
			h.UpdatedAt = t
			break
		}
	}
	return nil
}
