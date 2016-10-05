package datastore

import (
	"errors"
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

	host.ID = uint(len(orm.hosts) + 1)
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

func (orm *inmem) Hosts() ([]*kolide.Host, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	hosts := []*kolide.Host{}
	for _, host := range orm.hosts {
		hosts = append(hosts, host)
	}

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
		IPAddress:        ip,
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
		host.IPAddress = ip
	}
	if platform != "" {
		host.Platform = platform
	}

	if host.ID == 0 {
		host.ID = uint(len(orm.hosts) + 1)
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
