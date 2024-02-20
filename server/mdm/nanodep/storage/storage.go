package storage

import (
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/http/api"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/sync"
)

// ErrNotFound is returned by AllStorage when a requested resource is not found.
var ErrNotFound = errors.New("resource not found")

// AllStorage represents all possible required storage used by NanoDEP.
type AllStorage interface {
	client.AuthTokensRetriever
	client.ConfigRetriever
	sync.AssignerProfileRetriever
	sync.CursorStorage
	api.AuthTokensStorer
	api.ConfigStorer
	api.TokenPKIStorer
	api.TokenPKIRetriever
	api.AssignerProfileStorer
}
