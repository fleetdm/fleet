package datastore

import (
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

func (orm *inmem) Packs() ([]*kolide.Pack, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	packs := []*kolide.Pack{}
	for _, pack := range orm.packs {
		packs = append(packs, pack)
	}

	return packs, nil
}
