package fleet

import (
	"context"
	"io"
)

// SoftwareInstallerStore is the interface to store and retrieve software
// installer files. Fleet supports storing to the local filesystem and to an
// S3 bucket.
type SoftwareInstallerStore interface {
	// TODO: the idea here is that the flow would look like this:
	// * User uploads software via API
	// * Server hashes the contents, extracts name and version
	// * Server saves metadata (hash, name, version, other stuff) to DB
	// * Server stores contents to storage location under that hash
	// And later on, to retrieve it:
	// * Server reads metadata from DB, gets hash
	// * Server retrieves contents from storage location using hash
	//
	// 'installerID' in the signature is really the hex-encoded hash.
	Get(ctx context.Context, installerID string) (io.ReadCloser, int64, error)
	Put(ctx context.Context, installerID string, content io.ReadSeeker) error
	// not strictly required, but could save an upload if we already have this
	// content.
	Exists(ctx context.Context, installerID string) (bool, error)
}
