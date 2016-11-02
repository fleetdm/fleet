package datastore

import (
	"sort"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *inmem) NewPack(pack *kolide.Pack) error {
	newPack := *pack

	for _, q := range orm.packs {
		if pack.Name == q.Name {
			return ErrExists
		}
	}

	orm.mtx.Lock()
	newPack.ID = orm.nextID(pack)
	orm.packs[newPack.ID] = &newPack
	orm.mtx.Unlock()

	// TODO NewPack should return (*kolide.Pack, error) and this is a work around
	pack.ID = newPack.ID

	return nil
}

func (orm *inmem) SavePack(pack *kolide.Pack) error {
	if _, ok := orm.packs[pack.ID]; !ok {
		return ErrNotFound
	}

	orm.mtx.Lock()
	orm.packs[pack.ID] = pack
	orm.mtx.Unlock()

	return nil
}

func (orm *inmem) DeletePack(pid uint) error {
	if _, ok := orm.packs[pid]; !ok {
		return ErrNotFound
	}

	orm.mtx.Lock()
	delete(orm.packs, pid)
	orm.mtx.Unlock()

	return nil
}

func (orm *inmem) Pack(id uint) (*kolide.Pack, error) {
	orm.mtx.Lock()
	pack, ok := orm.packs[id]
	orm.mtx.Unlock()
	if !ok {
		return nil, ErrNotFound
	}

	return pack, nil
}

func (orm *inmem) ListPacks(opt kolide.ListOptions) ([]*kolide.Pack, error) {
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

func (orm *inmem) AddQueryToPack(qid uint, pid uint) error {
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

func (orm *inmem) ListQueriesInPack(pack *kolide.Pack) ([]*kolide.Query, error) {
	var queries []*kolide.Query

	orm.mtx.Lock()
	for _, packQuery := range orm.packQueries {
		queries = append(queries, orm.queries[packQuery.QueryID])
	}
	orm.mtx.Unlock()

	return queries, nil
}

func (orm *inmem) RemoveQueryFromPack(query *kolide.Query, pack *kolide.Pack) error {
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

func (orm *inmem) AddLabelToPack(lid uint, pid uint) error {
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

func (orm *inmem) ListLabelsForPack(pack *kolide.Pack) ([]*kolide.Label, error) {
	var labels []*kolide.Label

	orm.mtx.Lock()
	for _, pt := range orm.packTargets {
		if pt.Type == kolide.TargetLabel && pt.PackID == pack.ID {
			labels = append(labels, orm.labels[pt.TargetID])
		}
	}
	orm.mtx.Unlock()

	return labels, nil
}

func (orm *inmem) RemoveLabelFromPack(label *kolide.Label, pack *kolide.Pack) error {
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
