package s3

import "github.com/fleetdm/fleet/v4/server/config"

// stagedUploadPrefix must not string-prefix match softwareInstallersPrefix,
// bootstrapPackagePrefix, or softwareTitleIconPrefix (or vice versa):
// commonFileStore.Cleanup lists by byte-wise prefix, so an overlap would let one
// store's sweep delete the other's objects.
const stagedUploadPrefix = "uploads"

type StagedUploadStore struct {
	*commonFileStore
}

// NewStagedUploadStore creates a store for client uploads awaiting finalize,
// using the software installers S3 config.
func NewStagedUploadStore(config config.S3Config) (*StagedUploadStore, error) {
	s3store, err := newS3Store(config.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &StagedUploadStore{
		&commonFileStore{
			s3store:    s3store,
			pathPrefix: stagedUploadPrefix,
			fileLabel:  "staged upload",
		},
	}, nil
}
