package allmulti

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// MultiAllStorage dispatches to multiple AllStorage instances.
// It returns results and errors from the first store and simply
// logs errors, if any, for the remaining.
type MultiAllStorage struct {
	logger log.Logger
	stores []storage.AllStorage
}

// New creates a new MultiAllStorage dispatcher.
func New(logger log.Logger, stores ...storage.AllStorage) *MultiAllStorage {
	if len(stores) < 1 {
		panic("must supply at least one store")
	}
	return &MultiAllStorage{logger: logger, stores: stores}
}

type returnCollector struct {
	storeNumber int
	returnValue interface{}
	err         error
}

type errRunner func(storage.AllStorage) (interface{}, error)

func (ms *MultiAllStorage) execStores(ctx context.Context, r errRunner) (interface{}, error) {
	retChan := make(chan *returnCollector)
	for i, store := range ms.stores {
		go func(n int, s storage.AllStorage) {
			val, err := r(s)
			retChan <- &returnCollector{
				storeNumber: n,
				returnValue: val,
				err:         err,
			}
		}(i, store)
	}
	var finalErr error
	var finalValue interface{}
	for range ms.stores {
		sErr := <-retChan
		if sErr.storeNumber == 0 {
			finalErr = sErr.err
			finalValue = sErr.returnValue
		} else if sErr.err != nil {
			ctxlog.Logger(ctx, ms.logger).Info(
				"n", sErr.storeNumber,
				"err", sErr.err,
			)
		}
	}
	return finalValue, finalErr
}

func (ms *MultiAllStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreAuthenticate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreTokenUpdate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.RetrieveTokenUpdateTally(ctx, id)
	})
	return val.(int), err
}

func (ms *MultiAllStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreUserAuthenticate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) Disable(r *mdm.Request) error {
	_, err := ms.execStores(r.Context, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.Disable(r)
	})
	return err
}

func (ms *MultiAllStorage) ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error) {
	doc, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.ExpandEmbeddedSecrets(ctx, document)
	})
	return doc.(string), err
}

func (ms *MultiAllStorage) BulkDeleteHostUserCommandsWithoutResults(ctx context.Context, commandToIDs map[string][]string) error {
	_, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.BulkDeleteHostUserCommandsWithoutResults(ctx, commandToIDs)
	})
	return err
}
