package api

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
)

type DirectoryNonceService interface {
	NewNonce(ctx context.Context, identifier string) error
	GetDirectory(ctx context.Context, identifier string) (*types.Directory, error)
}
