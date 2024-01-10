package allmulti

import (
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
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
