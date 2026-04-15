package fleet

import (
	"context"
	"io"
)

// FleetdInstallerStore is the interface for storing and retrieving cached
// fleetd installer packages (.pkg).
type FleetdInstallerStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Put(ctx context.Context, key string, content io.ReadSeeker) error
	Exists(ctx context.Context, key string) (bool, error)
}

// DownloadFleetdInstallerPayload contains the data needed to stream a fleetd
// installer package to the client.
type DownloadFleetdInstallerPayload struct {
	Filename  string
	Installer io.ReadCloser
	Size      int64
}
