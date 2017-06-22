package inmem

import (
	"sort"

	"github.com/kolide/fleet/server/kolide"
)

func (d *Datastore) PackByName(name string, opts ...kolide.OptionalArg) (*kolide.Pack, bool, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	for _, p := range d.packs {
		if p.Name == name {
			return p, true, nil
		}
	}
	return nil, false, nil
}

func (d *Datastore) NewPack(pack *kolide.Pack, opts ...kolide.OptionalArg) (*kolide.Pack, error) {
	newPack := *pack

	for _, q := range d.packs {
		if pack.Name == q.Name {
			return nil, alreadyExists("Pack", q.ID)
		}
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	newPack.ID = d.nextID(pack)
	d.packs[newPack.ID] = &newPack

	pack.ID = newPack.ID

	return pack, nil
}

func (d *Datastore) SavePack(pack *kolide.Pack) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.packs[pack.ID]; !ok {
		return notFound("Pack").WithID(pack.ID)
	}

	d.packs[pack.ID] = pack

	return nil
}

func (d *Datastore) DeletePack(pid uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if _, ok := d.packs[pid]; !ok {
		return notFound("Pack").WithID(pid)
	}

	delete(d.packs, pid)

	return nil
}

func (d *Datastore) Pack(id uint) (*kolide.Pack, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	pack, ok := d.packs[id]
	if !ok {
		return nil, notFound("Pack").WithID(id)
	}

	return pack, nil
}

func (d *Datastore) ListPacks(opt kolide.ListOptions) ([]*kolide.Pack, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range d.packs {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	packs := []*kolide.Pack{}
	for _, k := range keys {
		packs = append(packs, d.packs[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":         "ID",
			"created_at": "CreatedAt",
			"updated_at": "UpdatedAt",
			"name":       "Name",
			"platform":   "Platform",
		}
		if err := sortResults(packs, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt, len(packs))
	packs = packs[low:high]

	return packs, nil
}

func (d *Datastore) AddLabelToPack(lid, pid uint, opts ...kolide.OptionalArg) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, pt := range d.packTargets {
		if pt.PackID == pid && pt.Target.Type == kolide.TargetLabel && pt.Target.TargetID == lid {
			return nil
		}
	}
	pt := &kolide.PackTarget{
		PackID: pid,
		Target: kolide.Target{
			Type:     kolide.TargetLabel,
			TargetID: lid,
		},
	}
	pt.ID = d.nextID(pt)
	d.packTargets[pt.ID] = pt

	return nil
}

func (d *Datastore) AddHostToPack(hid, pid uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, pt := range d.packTargets {
		if pt.PackID == pid && pt.Target.Type == kolide.TargetHost && pt.Target.TargetID == hid {
			d.mtx.Unlock()
			return nil
		}
	}
	pt := &kolide.PackTarget{
		PackID: pid,
		Target: kolide.Target{
			Type:     kolide.TargetHost,
			TargetID: hid,
		},
	}
	pt.ID = d.nextID(pt)
	d.packTargets[pt.ID] = pt

	return nil
}

func (d *Datastore) ListLabelsForPack(pid uint) ([]*kolide.Label, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	var labels []*kolide.Label
	for _, pt := range d.packTargets {
		if pt.Type == kolide.TargetLabel && pt.PackID == pid {
			labels = append(labels, d.labels[pt.TargetID])
		}
	}

	return labels, nil
}

func (d *Datastore) RemoveLabelFromPack(lid, pid uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	var labelsToDelete []uint

	for _, pt := range d.packTargets {
		if pt.Type == kolide.TargetLabel && pt.TargetID == lid && pt.PackID == pid {
			labelsToDelete = append(labelsToDelete, pt.ID)
		}
	}

	for _, id := range labelsToDelete {
		delete(d.packTargets, id)
	}

	return nil
}

func (d *Datastore) RemoveHostFromPack(hid, pid uint) error {
	d.mtx.Lock()
	d.mtx.Unlock()

	var hostsToDelete []uint

	for _, pt := range d.packTargets {
		if pt.Type == kolide.TargetHost && pt.TargetID == hid && pt.PackID == pid {
			hostsToDelete = append(hostsToDelete, pt.ID)
		}
	}

	for _, id := range hostsToDelete {
		delete(d.packTargets, id)
	}

	return nil
}

func (d *Datastore) ListHostsInPack(pid uint, opt kolide.ListOptions) ([]uint, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	hosts := []*kolide.Host{}
	hostLookup := map[uint]bool{}

	for _, pt := range d.packTargets {
		if pt.PackID != pid {
			continue
		}

		switch pt.Type {
		case kolide.TargetHost:
			if !hostLookup[pt.TargetID] {
				hostLookup[pt.TargetID] = true
				hosts = append(hosts, d.hosts[pt.TargetID])
			}
		case kolide.TargetLabel:
			for _, lqe := range d.labelQueryExecutions {
				if lqe.LabelID == pt.TargetID && lqe.Matches && !hostLookup[lqe.HostID] {
					hostLookup[lqe.HostID] = true
					hosts = append(hosts, d.hosts[lqe.HostID])
				}
			}
		}
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

	results := make([]uint, len(hosts), len(hosts))
	for _, h := range hosts {
		results = append(results, h.ID)
	}

	return results, nil
}

func (d *Datastore) ListExplicitHostsInPack(pid uint, opt kolide.ListOptions) ([]uint, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	hosts := []*kolide.Host{}
	hostLookup := map[uint]bool{}

	for _, pt := range d.packTargets {
		if pt.PackID != pid {
			continue
		}

		if pt.Type == kolide.TargetHost {
			if !hostLookup[pt.TargetID] {
				hostLookup[pt.TargetID] = true
				hosts = append(hosts, d.hosts[pt.TargetID])
			}
		}
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

	results := make([]uint, len(hosts), len(hosts))
	for _, h := range hosts {
		results = append(results, h.ID)
	}

	return results, nil
}
