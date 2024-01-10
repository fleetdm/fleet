package nanomdm

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

type (
	ctxKeyID   struct{}
	ctxKeyType struct{}
)

func newContextWithValues(ctx context.Context, r *mdm.Request) context.Context {
	newCtx := context.WithValue(ctx, ctxKeyID{}, r.ID)
	return context.WithValue(newCtx, ctxKeyType{}, r.Type)
}

func ctxKVs(ctx context.Context) (out []interface{}) {
	id, ok := ctx.Value(ctxKeyID{}).(string)
	if ok {
		out = append(out, "id", id)
	}
	eType, ok := ctx.Value(ctxKeyType{}).(mdm.EnrollType)
	if ok {
		out = append(out, "type", eType)
	}
	return
}
