package inmem

import "github.com/kolide/kolide/server/kolide"

func (d *Datastore) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	info.ID = 1
	d.appConfig = info
	return info, nil
}

func (d *Datastore) AppConfig() (*kolide.AppConfig, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.appConfig != nil {
		return d.appConfig, nil
	}

	return nil, notFound("AppConfig")
}

func (d *Datastore) SaveAppConfig(info *kolide.AppConfig) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.appConfig = info
	return nil
}
