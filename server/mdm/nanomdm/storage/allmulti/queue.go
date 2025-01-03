package allmulti

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

func (ms *MultiAllStorage) StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreCommandReport(r, report)
	})
	return err
}

func (ms *MultiAllStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
	val, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return s.RetrieveNextCommand(r, skipNotNow)
	})
	return val.(*mdm.CommandWithSubtype), err
}

func (ms *MultiAllStorage) ClearQueue(r *mdm.Request) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.ClearQueue(r)
	})
	return err
}

func (ms *MultiAllStorage) EnqueueCommand(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
	error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.EnqueueCommand(ctx, id, cmd)
	})
	return val.(map[string]error), err
}
