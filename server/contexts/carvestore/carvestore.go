package carvestore

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const carveStoreKey key = 0

func NewContext(ctx context.Context, svc fleet.CarveBySessionIder) context.Context {
	return context.WithValue(ctx, carveStoreKey, svc)
}

func FromContext(ctx context.Context) fleet.CarveBySessionIder {
	svc, ok := ctx.Value(carveStoreKey).(fleet.CarveBySessionIder)
	if !ok {
		return nil
	}
	return svc
}
