package inmem

import (
	"sort"

	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *Datastore) NewPack(pack *kolide.Pack) (*kolide.Pack, error) {
	newPack := *pack

	for _, q := range orm.packs {
		if pack.Name == q.Name {
			return nil, errors.ErrExists
		}
	}

	orm.mtx.Lock()
	newPack.ID = orm.nextID(pack)
	orm.packs[newPack.ID] = &newPack
	orm.mtx.Unlock()

	pack.ID = newPack.ID

	return pack, nil
}

func (orm *Datastore) SavePack(pack *kolide.Pack) error {
	if _, ok := orm.packs[pack.ID]; !ok {
		return errors.ErrNotFound
	}

	orm.mtx.Lock()
	orm.packs[pack.ID] = pack
	orm.mtx.Unlock()

	return nil
}

func (orm *Datastore) DeletePack(pid uint) error {
	if _, ok := orm.packs[pid]; !ok {
		return errors.ErrNotFound
	}

	orm.mtx.Lock()
	delete(orm.packs, pid)
	orm.mtx.Unlock()

	return nil
}

func (orm *Datastore) Pack(id uint) (*kolide.Pack, error) {
	orm.mtx.Lock()
	pack, ok := orm.packs[id]
	orm.mtx.Unlock()
	if !ok {
		return nil, errors.ErrNotFound
	}

	return pack, nil
}

func (orm *Datastore) ListPacks(opt kolide.ListOptions) ([]*kolide.Pack, error) {
	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	orm.mtx.Lock()
	for k, _ := range orm.packs {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	packs := []*kolide.Pack{}
	for _, k := range keys {
		packs = append(packs, orm.packs[uint(k)])
	}
	orm.mtx.Unlock()

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
	low, high := orm.getLimitOffsetSliceBounds(opt, len(packs))
	packs = packs[low:high]

	return packs, nil
}

func (orm *Datastore) AddQueryToPack(qid uint, pid uint) error {
	packQuery := &kolide.PackQuery{
		PackID:  pid,
		QueryID: qid,
	}

	orm.mtx.Lock()
	packQuery.ID = orm.nextID(packQuery)
	orm.packQueries[packQuery.ID] = packQuery
	orm.mtx.Unlock()

	return nil
}

func (orm *Datastore) ListQueriesInPack(pack *kolide.Pack) ([]*kolide.Query, error) {
	var queries []*kolide.Query

	orm.mtx.Lock()
	for _, packQuery := range orm.packQueries {
		queries = append(queries, orm.queries[packQuery.QueryID])
	}
	orm.mtx.Unlock()

	return queries, nil
}

func (orm *Datastore) RemoveQueryFromPack(query *kolide.Query, pack *kolide.Pack) error {
	var packQueriesToDelete []uint

	orm.mtx.Lock()
	for _, packQuery := range orm.packQueries {
		if packQuery.QueryID == query.ID && packQuery.PackID == pack.ID {
			packQueriesToDelete = append(packQueriesToDelete, packQuery.ID)
		}
	}

	for _, packQueryToDelete := range packQueriesToDelete {
		delete(orm.packQueries, packQueryToDelete)
	}
	orm.mtx.Unlock()

	return nil
}

func (orm *Datastore) AddLabelToPack(lid uint, pid uint) error {
	pt := &kolide.PackTarget{
		PackID: pid,
		Target: kolide.Target{
			Type:     kolide.TargetLabel,
			TargetID: lid,
		},
	}

	orm.mtx.Lock()
	pt.ID = orm.nextID(pt)
	orm.packTargets[pt.ID] = pt
	orm.mtx.Unlock()

	return nil
}

func (orm *Datastore) ListLabelsForPack(pid uint) ([]*kolide.Label, error) {
	var labels []*kolide.Label

	orm.mtx.Lock()
	for _, pt := range orm.packTargets {
		if pt.Type == kolide.TargetLabel && pt.PackID == pid {
			labels = append(labels, orm.labels[pt.TargetID])
		}
	}
	orm.mtx.Unlock()

	return labels, nil
}

func (orm *Datastore) RemoveLabelFromPack(label *kolide.Label, pack *kolide.Pack) error {
	var labelsToDelete []uint

	orm.mtx.Lock()
	for _, pt := range orm.packTargets {
		if pt.Type == kolide.TargetLabel && pt.TargetID == label.ID && pt.PackID == pack.ID {
			labelsToDelete = append(labelsToDelete, pt.ID)
		}
	}

	for _, id := range labelsToDelete {
		delete(orm.packTargets, id)
	}
	orm.mtx.Unlock()

	return nil
}

func (orm *Datastore) ListHostsInPack(pid uint, opt kolide.ListOptions) ([]*kolide.Host, error) {
	hosts := []*kolide.Host{}
	hostLookup := map[uint]bool{}

	orm.mtx.Lock()
	for _, pt := range orm.packTargets {
		if pt.PackID != pid {
			continue
		}

		switch pt.Type {
		case kolide.TargetHost:
			if !hostLookup[pt.TargetID] {
				hostLookup[pt.TargetID] = true
				hosts = append(hosts, orm.hosts[pt.TargetID])
			}
		case kolide.TargetLabel:
			for _, lqe := range orm.labelQueryExecutions {
				if lqe.LabelID == pt.TargetID && lqe.Matches && !hostLookup[lqe.HostID] {
					hostLookup[lqe.HostID] = true
					hosts = append(hosts, orm.hosts[lqe.HostID])
				}
			}
		}
	}
	orm.mtx.Unlock()

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
