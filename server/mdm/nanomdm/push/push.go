// Package push defines interfaces, types, etc. related to MDM APNs
// push notifications.
package push

import (
	"context"
	"crypto/tls"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

type Response struct {
	Id  string
	Err error
}

// Pusher sends MDM APNs notifications to enrollments identified by a string.
type Pusher interface {
	Push(context.Context, []string) (map[string]*Response, error)
}

// PushProvider can send 'raw' MDM APNs push notifications.
//
// The non-error return type maps the string value of the push token
// (that is, the hex encoding of the bytes) to a pointer to a Response.
type PushProvider interface {
	Push(context.Context, []*mdm.Push) (map[string]*Response, error)
}

// PushProviderFactory generates a new PushProvider given a tls keypair
type PushProviderFactory interface {
	NewPushProvider(*tls.Certificate) (PushProvider, error)
}
