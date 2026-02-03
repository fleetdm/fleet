package carvestore

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const carveStoreKey key = 0

func NewContext(ctx context.Context, svc fleet.CarveStore) context.Context {
	return context.WithValue(ctx, carveStoreKey, svc)
}

func FromContext(ctx context.Context) fleet.CarveStore {
	svc, ok := ctx.Value(carveStoreKey).(fleet.CarveStore)
	if !ok {
		return nil
	}
	return svc
}
