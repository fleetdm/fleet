package fleet

import (
	"context"
	"io"
	"time"
)

// StagedUploadURLExpiry is generous so a large upload on a slow link finishes;
// the URL can only write one random key of a pinned size.
const StagedUploadURLExpiry = 6 * time.Hour

// StagedUploadStore holds package bytes a client uploaded straight to the
// object store, until the package is registered with Fleet.
type StagedUploadStore interface {
	PresignPut(ctx context.Context, uploadID string, size int64, expiresIn time.Duration) (string, error)
	Get(ctx context.Context, uploadID string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, uploadID string) error
	Cleanup(ctx context.Context, usedUploadIDs []string, removeCreatedBefore time.Time) (int, error)
}

type StagedUploadTarget string

const (
	StagedUploadTargetBootstrapPackage StagedUploadTarget = "bootstrap_package"
	StagedUploadTargetSoftwarePackage  StagedUploadTarget = "software_package"
)

type StagedUpload struct {
	UploadID  string    `json:"upload_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
