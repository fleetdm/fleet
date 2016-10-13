package datastore

import (
	"sort"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *inmem) NewPack(pack *kolide.Pack) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	newPack := *pack

	for _, q := range orm.packs {
		if pack.Name == q.Name {
			return ErrExists
		}
	}

	newPack.ID = uint(len(orm.packs) + 1)
	orm.packs[newPack.ID] = &newPack

	return nil
}

func (orm *inmem) SavePack(pack *kolide.Pack) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.packs[pack.ID]; !ok {
		return ErrNotFound
	}

	orm.packs[pack.ID] = pack
	return nil
}

func (orm *inmem) DeletePack(pid uint) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if _, ok := orm.packs[pid]; !ok {
		return ErrNotFound
	}

	delete(orm.packs, pid)
	return nil
}

func (orm *inmem) Pack(id uint) (*kolide.Pack, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	pack, ok := orm.packs[id]
	if !ok {
		return nil, ErrNotFound
	}

	return pack, nil
}

func (orm *inmem) Packs(opt kolide.ListOptions) ([]*kolide.Pack, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// We need to sort by keys to provide reliable ordering
	keys := []int{}
	for k, _ := range orm.packs {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	packs := []*kolide.Pack{}
	for _, k := range keys {
		packs = append(packs, orm.packs[uint(k)])
	}

	// Apply limit/offset
	low, high := orm.getLimitOffsetSliceBounds(opt, len(packs))
	packs = packs[low:high]

	return packs, nil
}
