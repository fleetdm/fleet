package apple_bm

import (
	"context"

	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
)

type key int

const abmKey key = 0

func NewContext(ctx context.Context, decryptedToken *depclient.OAuth1Tokens) context.Context {
	return context.WithValue(ctx, abmKey, decryptedToken)
}

func FromContext(ctx context.Context) (*depclient.OAuth1Tokens, bool) {
	tok, ok := ctx.Value(abmKey).(*depclient.OAuth1Tokens)
	return tok, ok
}
