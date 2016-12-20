package inmem

import "github.com/kolide/kolide-ose/server/kolide"

func (d *Datastore) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	info.ID = 1
	d.orginfo = info
	return info, nil
}

func (d *Datastore) AppConfig() (*kolide.AppConfig, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.orginfo != nil {
		return d.orginfo, nil
	}

	return nil, notFound("AppConfig")
}

func (d *Datastore) SaveAppConfig(info *kolide.AppConfig) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.orginfo = info
	return nil
}
