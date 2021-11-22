package cached_mysql

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type cachedMysql struct {
	fleet.Datastore

	mu        sync.Mutex
	ac        *fleet.AppConfig
	acLastErr error
}

func New(ctx context.Context, ds fleet.Datastore) fleet.Datastore {
	cds := &cachedMysql{Datastore: ds}
	go cds.refresher(ctx)

	return cds
}

func (ds *cachedMysql) refresher(ctx context.Context) {
	for {
		select {
		case <-time.Tick(1 * time.Second):
			ac, err := ds.Datastore.AppConfig(ctx)
			ds.mu.Lock()
			ds.ac = ac
			ds.acLastErr = err
			ds.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (ds *cachedMysql) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	ac, err := ds.Datastore.NewAppConfig(ctx, info)
	if err != nil {
		return nil, err
	}

	ds.mu.Lock()
	ds.ac = ac
	ds.acLastErr = nil
	ds.mu.Unlock()

	return ac, nil
}

func (ds *cachedMysql) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	ds.mu.Lock()
	ac := ds.ac
	acLastErr := ds.acLastErr
	ds.mu.Unlock()

	if acLastErr != nil {
		return nil, acLastErr
	} else if ac == nil {
		return ds.Datastore.AppConfig(ctx)
	}

	return ac, nil
}

func (ds *cachedMysql) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	err := ds.Datastore.SaveAppConfig(ctx, info)
	if err != nil {
		return err
	}

	ds.mu.Lock()
	ds.ac = info
	ds.acLastErr = nil
	ds.mu.Unlock()

	return nil
}
