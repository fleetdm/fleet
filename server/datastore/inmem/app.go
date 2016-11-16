package inmem

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *Datastore) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	info.ID = 1
	orm.orginfo = info
	return info, nil
}

func (orm *Datastore) AppConfig() (*kolide.AppConfig, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if orm.orginfo != nil {
		return orm.orginfo, nil
	}

	return nil, errors.ErrNotFound
}

func (orm *Datastore) SaveAppConfig(info *kolide.AppConfig) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	orm.orginfo = info
	return nil
}
