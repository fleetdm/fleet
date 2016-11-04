package datastore

import "github.com/kolide/kolide-ose/server/kolide"

func (orm *inmem) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	info.ID = 1
	orm.orginfo = info
	return info, nil
}

func (orm *inmem) AppConfig() (*kolide.AppConfig, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if orm.orginfo != nil {
		return orm.orginfo, nil
	}

	return nil, ErrNotFound
}

func (orm *inmem) SaveAppConfig(info *kolide.AppConfig) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	orm.orginfo = info
	return nil
}
