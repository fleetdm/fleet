package datastore

import "github.com/kolide/kolide-ose/server/kolide"

func (orm *inmem) NewOrgInfo(info *kolide.OrgInfo) (*kolide.OrgInfo, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	orm.orginfo = info
	return info, nil
}

func (orm *inmem) OrgInfo() (*kolide.OrgInfo, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if orm.orginfo != nil {
		return orm.orginfo, nil
	}

	return nil, ErrNotFound
}

func (orm *inmem) SaveOrgInfo(info *kolide.OrgInfo) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	orm.orginfo = info
	return nil
}
