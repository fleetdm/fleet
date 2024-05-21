package mdm

import (
	"context"
)

type key int

const (
	refetchResultsRequest key = 0
)

func SetRefetchResultsRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, refetchResultsRequest, true)
}

func IsRefetchResultsRequest(ctx context.Context) bool {
	v, _ := ctx.Value(refetchResultsRequest).(bool)
	return v
}
