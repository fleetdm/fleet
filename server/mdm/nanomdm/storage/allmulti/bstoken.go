package allmulti

import (
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

func (ms *MultiAllStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreBootstrapToken(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	val, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return s.RetrieveBootstrapToken(r, msg)
	})
	return val.(*mdm.BootstrapToken), err
}
