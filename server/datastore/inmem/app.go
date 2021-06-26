package inmem

import "github.com/fleetdm/fleet/server/fleet"

func (d *Datastore) NewAppConfig(info *fleet.AppConfig) (*fleet.AppConfig, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	info.ID = 1
	d.appConfig = info
	return info, nil
}

func (d *Datastore) AppConfig() (*fleet.AppConfig, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.appConfig != nil {
		return d.appConfig, nil
	}

	return nil, notFound("AppConfig")
}

func (d *Datastore) SaveAppConfig(info *fleet.AppConfig) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.appConfig = info
	return nil
}
