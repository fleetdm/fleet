package failing

import (
	"context"
	"errors"
	"io"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// FailingOrgLogoStore is an implementation of fleet.OrgLogoStore that fails all
// operations. It is used when S3 is not configured and the local filesystem
// store could not be set up (e.g. a read-only root filesystem). Returning this
// instead of crashing at startup keeps Fleet bootable; custom org logos are
// simply unavailable for that deployment.
//
// It does not embed commonFailingStore because fleet.OrgLogoStore is keyed on
// fleet.OrgLogoMode (not string) and exposes Delete rather than Cleanup/Sign.
type FailingOrgLogoStore struct{}

var _ fleet.OrgLogoStore = (*FailingOrgLogoStore)(nil)

func NewFailingOrgLogoStore() *FailingOrgLogoStore {
	return &FailingOrgLogoStore{}
}

func (FailingOrgLogoStore) Put(_ context.Context, _ fleet.OrgLogoMode, _ io.ReadSeeker) error {
	return errors.New("org logo store not properly configured")
}

func (FailingOrgLogoStore) Get(_ context.Context, _ fleet.OrgLogoMode) (io.ReadCloser, int64, error) {
	return nil, 0, errors.New("org logo store not properly configured")
}

func (FailingOrgLogoStore) Delete(_ context.Context, _ fleet.OrgLogoMode) error {
	return errors.New("org logo store not properly configured")
}

func (FailingOrgLogoStore) Exists(_ context.Context, _ fleet.OrgLogoMode) (bool, error) {
	return false, errors.New("org logo store not properly configured")
}
